package business_logic

import (
	"context"
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	"deadalus-orch/server/internal/infrastructure/server/common"
	"deadalus-orch/server/internal/pkg/config"
	"deadalus-orch/server/internal/pkg/utils"
	general_command "deadalus-orch/server/internal/usecase/command/general"
	job_worker_command "deadalus-orch/server/internal/usecase/command/job-worker"
	queue_command "deadalus-orch/server/internal/usecase/command/queue"
	tentant_command "deadalus-orch/server/internal/usecase/command/tentant"
	"deadalus-orch/shared/models"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// ClaimedMessage holds a dequeued message along with its lease and tenant info
type ClaimedMessage struct {
	Message    models.QueueMessage
	Lease      models.QueueMessageLease
	TenantCode string
	CF         string
	CFS        string
	TenantNode *dragonboat.RaftNode
}

// claimCursorKey identifies a pagination cursor stored in the registry.
// tenantID is empty for the tenant-level cursor; it is set for vnamespace-level cursors.
type claimCursorKey struct {
	policyCode string
	tenantID   string
}

// claimCursorRegistry stores pagination cursors between successive ClaimWork cycles.
// It is local to the connector node (not persisted in Raft) and safe for concurrent use.
type claimCursorRegistry struct {
	mu      sync.Mutex
	cursors map[claimCursorKey]string
}

func newClaimCursorRegistry() *claimCursorRegistry {
	return &claimCursorRegistry{
		cursors: make(map[claimCursorKey]string),
	}
}

// get returns the stored cursor for key, or "" if none is recorded.
func (r *claimCursorRegistry) get(key claimCursorKey) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.cursors[key]
}

// set persists cursor for key. An empty cursor means the list was exhausted;
// in that case the entry is deleted so the next cycle starts from the beginning.
func (r *claimCursorRegistry) set(key claimCursorKey, cursor string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if cursor == "" {
		delete(r.cursors, key)
	} else {
		r.cursors[key] = cursor
	}
}

type DequeueRequestMessage struct {
	QueueID                      string
	JobWorkerID                  string
	LeaseDuration                time.Duration
	JobWorkerCapacityPolicyIndex int
	CF                           string
	CFS                          string
	TenantNode                   *dragonboat.RaftNode
}

type DequeueResponseMessage struct {
	Result *queue_command.DequeueResult
}

type DequeueFunc func(ctx context.Context, req DequeueRequestMessage) (DequeueResponseMessage, error)
type HeartbeatFunc func(ctx context.Context, worker models.JobWorker, masterNode *dragonboat.RaftNode) error

type JobWorkerBO struct {
	Config         *common.ServerConfing
	stoppers       map[string]bool
	stoppersMu     sync.Mutex
	cursorRegistry *claimCursorRegistry
	lastHeartbeats map[string]time.Time
	heartbeatsMu   sync.Mutex
	dequeueFunc    DequeueFunc
	heartbeatFunc  HeartbeatFunc
}

func NewJobWorkerBO(Config *common.ServerConfing, dequeueFunc DequeueFunc, heartbeatFunc HeartbeatFunc) *JobWorkerBO {
	return &JobWorkerBO{
		Config:         Config,
		stoppers:       make(map[string]bool),
		cursorRegistry: newClaimCursorRegistry(),
		lastHeartbeats: make(map[string]time.Time),
		dequeueFunc:    dequeueFunc,
		heartbeatFunc:  heartbeatFunc,
	}
}

func (bo *JobWorkerBO) ClaimWork(ctx context.Context, workerId string, workerName string, Information map[string]string, ClaimWorkCapacityPolicies map[string]models.ClaimWorkCapacityPolicy, messageChan chan<- ClaimedMessage) error {
	logger := bo.Config.Logger.With().Str("workerID", workerId).Logger()

	// Check if heartbeat is needed (only every 5 seconds per worker)
	bo.heartbeatsMu.Lock()
	lastHB, exists := bo.lastHeartbeats[workerId]
	if !exists || time.Since(lastHB) > 5*time.Second {
		bo.lastHeartbeats[workerId] = time.Now()
		bo.heartbeatsMu.Unlock()

		worker := models.JobWorker{
			ID:                        workerId,
			Name:                      workerName,
			Information:               Information,
			ClaimWorkCapacityPolicies: ClaimWorkCapacityPolicies,
			ConnectionStatus:          models.JobWorkerConnectionStatusConnected,
		}

		if bo.heartbeatFunc != nil {
			if err := bo.heartbeatFunc(ctx, worker, bo.Config.MasterNode); err != nil {
				logger.Error().Err(err).Msg("❌ Failed to send heartbeat to buffer")
			}
		} else {
			upsertCmd := &job_worker_command.UpsertJobWorkerCommand{
				JobWorkers:     []models.JobWorker{worker},
				ApplyHeartbeat: true,
			}
			_, err := dragonboat.ExecuteRepositoryCommand[[]models.JobWorker](
				bo.Config.MasterNode,
				ctx,
				upsertCmd,
				config.GlobalConfiguration.ApiRaftTimeout,
				bo.Config.Logger,
				"upsert job worker",
			)
			if err != nil {
				logger.Error().Err(err).Msg("❌ Failed to upsert job worker in ClaimWork")
			}
		}
	} else {
		bo.heartbeatsMu.Unlock()
	}

	// Check if a stopper is already running for this worker.
	// If one is running there is already an active claim process in progress for this worker.
	bo.stoppersMu.Lock()
	if bo.stoppers[workerId] {
		bo.stoppersMu.Unlock()
		bo.Config.Logger.Debug().Str("workerID", workerId).Msg("ClaimWork stopper already running, skipping")
		return nil
	}
	bo.stoppers[workerId] = true
	bo.stoppersMu.Unlock()

	// Launch a dedicated stopper goroutine for this worker.
	// It will iterate all ClaimWorkCapacityPolicies, paginate tenants → vnamespaces → queues,
	// and dequeue messages until all policies are satisfied or pagination is exhausted.
	go bo.runClaimWorkStopper(ctx, workerId, ClaimWorkCapacityPolicies, messageChan)

	return nil
}

// runClaimWorkStopper is the goroutine that drives the claim-work process for a single JobWorker.
// It terminates when all ClaimWorkCapacityPolicies are satisfied or all pagination is exhausted,
// after which a subsequent ClaimWork call is allowed to spawn a new stopper.
func (bo *JobWorkerBO) runClaimWorkStopper(ctx context.Context, workerID string, policies map[string]models.ClaimWorkCapacityPolicy, messageChan chan<- ClaimedMessage) {
	// Always release the stopper slot when the goroutine exits so the next ClaimWork call
	// can spawn a new one.
	defer func() {
		bo.stoppersMu.Lock()
		bo.stoppers[workerID] = false
		bo.stoppersMu.Unlock()
	}()

	stopperCtx, stopperCancel := context.WithTimeout(ctx, 5*time.Minute)
	defer stopperCancel()

	logger := bo.Config.Logger.With().Str("workerID", workerID).Logger()

	// Local claim counters per policy so we don't mutate the caller's map.
	claimedByPolicy := make(map[string]int, len(policies))
	for code, policy := range policies {
		claimedByPolicy[code] = policy.CurrentQueueMessages
	}

	// Protects concurrent reads and writes to claimedByPolicy across shard goroutines
	var claimMu sync.Mutex

	// Returns true when every policy with a positive cap has been satisfied.
	// MaxQueueMessages == 0 means unlimited, so that policy is never considered satisfied.
	// Must be called with claimMu held.
	allPoliciesSatisfied := func() bool {
		for code, policy := range policies {
			if policy.MaxQueueMessages == 0 {
				return false // unlimited policy is never satisfied
			}
			if claimedByPolicy[code] < policy.MaxQueueMessages {
				return false
			}
		}
		return true
	}

	tenantBO := &TenantBO{Config: bo.Config}
	queueBO := &QueueBO{Config: bo.Config}

	for policyCode, policy := range policies {
		claimMu.Lock()
		if allPoliciesSatisfied() || (policy.MaxQueueMessages > 0 && claimedByPolicy[policyCode] >= policy.MaxQueueMessages) {
			claimMu.Unlock()
			continue
		}
		claimMu.Unlock()

		// Derive the numeric index from the policy code ("policy-N" → N).
		policyIndex := 0
		fmt.Sscanf(policyCode, "policy-%d", &policyIndex)

		filter := policy.ClaimWorkFilter

		// ── Tenant pagination (DB-filtered) ───────────────────────────────────
		// Resume from the cursor saved by the previous ClaimWork cycle for this
		// worker+policy pair, enabling fair round-robin rotation across all tenants.
		tenantKey := claimCursorKey{policyCode: policyCode}
		tenantCursor := bo.cursorRegistry.get(tenantKey)
		const tenantPageSize = 50

		for {
			claimMu.Lock()
			if allPoliciesSatisfied() || (policy.MaxQueueMessages > 0 && claimedByPolicy[policyCode] >= policy.MaxQueueMessages) {
				claimMu.Unlock()
				break
			}
			claimMu.Unlock()

			paginateCtx, paginateCancel := context.WithTimeout(stopperCtx, 10*time.Second)
			tenantsResult, err := tenantBO.GetTenantsWithFilter(paginateCtx, filter, tenantCursor, tenantPageSize)
			paginateCancel()
			if err != nil {
				logger.Error().Err(err).Str("policy", policyCode).Msg("❌ Failed to paginate tenants during ClaimWork")
				break
			}

			if len(tenantsResult.Entities) == 0 {
				bo.cursorRegistry.set(tenantKey, "")
				break
			}

			// Group the tenants in the current page by their Shard/RaftNode.
			type tenantGroup struct {
				node    *dragonboat.RaftNode
				tenants []models.TenantInMaster
			}
			groupsByShard := make(map[uint64]*tenantGroup)

			for _, t := range tenantsResult.Entities {
				node := bo.getJobWorkerTenantNode(t)
				if node == nil {
					logger.Warn().Str("tenantCode", t.Code).Msg("No raft node found for tenant, skipping")
					continue
				}
				shardID := node.ShardID
				if groupsByShard[shardID] == nil {
					groupsByShard[shardID] = &tenantGroup{
						node:    node,
						tenants: make([]models.TenantInMaster, 0),
					}
				}
				groupsByShard[shardID].tenants = append(groupsByShard[shardID].tenants, t)
			}

			var wg sync.WaitGroup

			// Process each shard concurrently.
			for _, group := range groupsByShard {
				wg.Add(1)
				go func(g *tenantGroup) {
					defer wg.Done()

				shardTenantLoop:
					for _, tenant := range g.tenants {
						claimMu.Lock()
						if allPoliciesSatisfied() || (policy.MaxQueueMessages > 0 && claimedByPolicy[policyCode] >= policy.MaxQueueMessages) {
							claimMu.Unlock()
							break shardTenantLoop
						}
						claimMu.Unlock()

						cf := db.ColumnFamilyPrefix + fmt.Sprintf("%d", tenant.ColumnFamilyIndex)
						cfs := tenant.ID

						// ── Queue pagination (DB-filtered with VNamespace rules) ──────────
						// Resume queue iteration from the last saved position for this tenant.
						tenantQueueKey := claimCursorKey{policyCode: policyCode, tenantID: tenant.ID}
						queueCursor := bo.cursorRegistry.get(tenantQueueKey)
						const queuePageSize = 50

					queueLoop:
						for {
							claimMu.Lock()
							if allPoliciesSatisfied() || (policy.MaxQueueMessages > 0 && claimedByPolicy[policyCode] >= policy.MaxQueueMessages) {
								claimMu.Unlock()
								break
							}
							claimMu.Unlock()

							qCtx, qCancel := context.WithTimeout(stopperCtx, 10*time.Second)
							queuesResult, err := queueBO.GetQueuesWithFilter(qCtx, filter, queueCursor, queuePageSize, cf, cfs, &tenant, g.node)
							qCancel()
							if err != nil {
								logger.Error().Err(err).
									Str("policy", policyCode).
									Str("tenant", tenant.Code).
									Msg("❌ Failed to paginate queues during ClaimWork")
								break
							}

							var allQueues []models.Queue = queuesResult.Entities

							if len(allQueues) == 0 {
								break queueLoop
							}

							// ── Round-robin drain: cycle through all queues until the policy
							// is satisfied or a full round yields no new messages. ─────────────
							for {
								claimMu.Lock()
								if allPoliciesSatisfied() || (policy.MaxQueueMessages > 0 && claimedByPolicy[policyCode] >= policy.MaxQueueMessages) {
									claimMu.Unlock()
									break queueLoop
								}
								claimMu.Unlock()

								claimedInRound := 0
								for i := range allQueues {
									queue := &allQueues[i]

									claimMu.Lock()
									if allPoliciesSatisfied() || (policy.MaxQueueMessages > 0 && claimedByPolicy[policyCode] >= policy.MaxQueueMessages) {
										claimMu.Unlock()
										break queueLoop
									}

									// Respect the queue's own delivering-message cap (0 = unlimited).
									// MessagesCount > 0 is already guaranteed by the DB query.
									if queue.MaxDeliveringMessages > 0 && queue.CurrentDeliveringMessages >= queue.MaxDeliveringMessages {
										claimMu.Unlock()
										continue
									}

									// Optimistically reserve the slot
									if policy.MaxQueueMessages > 0 {
										claimedByPolicy[policyCode]++
									}
									claimMu.Unlock()

									// ── Dequeue message ──
									if bo.dequeueMessage(stopperCtx, workerID, policyCode, policyIndex, queue, &tenant, g.node, cf, cfs, messageChan) {
										queue.CurrentDeliveringMessages++
										claimedInRound++
									} else {
										// Failed to dequeue, refund the slot
										claimMu.Lock()
										if policy.MaxQueueMessages > 0 {
											claimedByPolicy[policyCode]--
										}
										claimMu.Unlock()
									}
								}
								if claimedInRound == 0 {
									break // No queue delivered a message this round; all queues exhausted.
								}
							}

							bo.cursorRegistry.set(tenantQueueKey, queuesResult.Cursor)
							if queuesResult.Cursor == "" || len(queuesResult.Entities) < queuePageSize {
								break
							}
							queueCursor = queuesResult.Cursor
						}
					}
				}(group)
			}

			// Wait for all shard goroutines to finish processing this page of tenants
			wg.Wait()

			// Persist the cursor so the next cycle resumes from here.
			// set("") automatically removes the entry, causing the next cycle to wrap around.
			bo.cursorRegistry.set(tenantKey, tenantsResult.Cursor)
			if tenantsResult.Cursor == "" || len(tenantsResult.Entities) < tenantPageSize {
				break
			}
			tenantCursor = tenantsResult.Cursor
		}
	}
}

// dequeueMessage dequeues a message from the queue and sends it through the channel.
func (bo *JobWorkerBO) dequeueMessage(
	ctx context.Context,
	workerID, policyCode string,
	policyIndex int,
	queue *models.Queue,
	tenant *models.TenantInMaster,
	tenantNode *dragonboat.RaftNode,
	cf, cfs string,
	messageChan chan<- ClaimedMessage,
) bool {

	req := DequeueRequestMessage{
		QueueID:                      queue.ID,
		JobWorkerID:                  workerID,
		LeaseDuration:                config.GlobalConfiguration.MessageLeaseDuration,
		JobWorkerCapacityPolicyIndex: policyIndex,
		CF:                           cf,
		CFS:                          cfs,
		TenantNode:                   tenantNode,
	}

	conf, err := bo.dequeueFunc(ctx, req)
	if err != nil {
		if strings.Contains(err.Error(), "is empty") {
			bo.Config.Logger.Debug().
				Str("workerID", workerID).
				Str("queueCode", queue.Code).
				Str("tenant", tenant.Code).
				Msg("Queue became empty before dequeue could complete")
		} else {
			bo.Config.Logger.Error().Err(err).
				Str("workerID", workerID).
				Str("queueCode", queue.Code).
				Str("tenant", tenant.Code).
				Msg("❌ Failed to dequeue message")
		}
		return false
	}
	if conf.Result == nil {
		return false
	}

	result := conf.Result

	// Send the claimed message through the channel
	claimedMsg := ClaimedMessage{
		Message:    result.Message,
		Lease:      result.Lease,
		TenantCode: tenant.Code,
		CF:         cf,
		CFS:        cfs,
		TenantNode: tenantNode,
	}

	if bo.Config.MetricsCollector != nil {
		bo.Config.MetricsCollector.UpdateGauges(tenant.Code, queue.Code, queue.VNamespace, result.Pending, result.InProcess)
	}

	select {
	case messageChan <- claimedMsg:
		bo.Config.Logger.Debug().Str("messageID", result.Message.ID).Msg("📤 Sent message to stream")

		if bo.Config.MetricsCollector != nil {
			bo.Config.MetricsCollector.RecordDelivery(tenant.Code, queue.Code, queue.VNamespace, 1)
		}
		return true

	case <-ctx.Done():
		bo.Config.Logger.Warn().Str("messageID", result.Message.ID).Msg("⚠️ Context cancelled, message not sent")
	}
	return true
}

// getJobWorkerTenantNode resolves the RaftNode that owns the given tenant's shard.
func (bo *JobWorkerBO) getJobWorkerTenantNode(tenant models.TenantInMaster) *dragonboat.RaftNode {
	bo.Config.TenantNodesLock.Lock()
	defer bo.Config.TenantNodesLock.Unlock()
	for i := range bo.Config.TenantNodes {
		if bo.Config.TenantNodes[i].ShardID == uint64(tenant.ShardId) {
			return bo.Config.TenantNodes[i]
		}
	}
	return nil
}

func (bo *JobWorkerBO) GetJobWorker(ctx context.Context, jobWorkerID string) (models.JobWorker, error) {
	findJobWorkerCommand := &job_worker_command.PaginateJobWorkersCommand{
		Q:        jobWorkerID,
		PageSize: 1,
	}

	findResult, err := dragonboat.ExecuteRepositoryQuery[db.FindResult[models.JobWorker]](
		bo.Config.MasterNode,
		ctx,
		findJobWorkerCommand,
		config.GlobalConfiguration.ApiRaftTimeout,
		bo.Config.Logger,
		"find jobWorker",
	)

	if err != nil {
		return models.JobWorker{}, err
	}

	if len(findResult.Entities) == 0 {
		return models.JobWorker{}, errors.New("JobWorker not found")
	}

	return findResult.Entities[0], nil
}

func (bo *JobWorkerBO) GetJobWorkers(ctx context.Context, q string, status string, cursor string, pageSize int) (db.FindResult[models.JobWorker], error) {
	paginateJobWorkersCommand := &job_worker_command.PaginateJobWorkersCommand{
		Cursor:           cursor,
		PageSize:         pageSize,
		Q:                q,
		ConnectionStatus: models.JobWorkerConnectionStatus(status),
	}

	findResult, err := dragonboat.ExecuteRepositoryQuery[db.FindResult[models.JobWorker]](
		bo.Config.MasterNode,
		ctx,
		paginateJobWorkersCommand,
		config.GlobalConfiguration.ApiRaftTimeout,
		bo.Config.Logger,
		"paginate jobWorkers",
	)
	if err != nil {
		return db.FindResult[models.JobWorker]{}, fmt.Errorf("paginate jobWorkers failed: %w", err)
	}

	if findResult.Entities == nil {
		findResult.Entities = []models.JobWorker{}
	}

	return findResult, nil
}

func (bo *JobWorkerBO) BulkUpsertJobWorker(ctx context.Context, jobWorkers []*models.JobWorker) ([]models.JobWorker, error) {
	if len(jobWorkers) == 0 {
		return nil, errors.New("no jobWorkers provided")
	}

	upsertCmd := &job_worker_command.UpsertJobWorkerCommand{
		JobWorkers: make([]models.JobWorker, len(jobWorkers)),
	}
	for i, jw := range jobWorkers {
		upsertCmd.JobWorkers[i] = *jw
	}

	created, err := dragonboat.ExecuteRepositoryCommand[[]models.JobWorker](
		bo.Config.MasterNode,
		ctx,
		upsertCmd,
		config.GlobalConfiguration.ApiRaftTimeout*time.Duration(len(jobWorkers)),
		bo.Config.Logger,
		"bulk upsert jobWorkers",
	)
	if err != nil {
		return nil, err
	}

	return created, nil
}

func (bo *JobWorkerBO) ReviewJobWorkersConnectionStatus(ctx context.Context) {
	// Paginate through all connected job workers to update their connection status
	pageSize := 100
	cursor := ""
	allJobWorkers := []*models.JobWorker{}

	statusConnected := string(models.JobWorkerConnectionStatusConnected)

	for {
		paginateCtx, paginateCancel := context.WithTimeout(ctx, 10*time.Second)
		findResult, err := bo.GetJobWorkers(paginateCtx, "", statusConnected, cursor, pageSize)
		paginateCancel()
		if err != nil {
			bo.Config.Logger.Error().Err(err).Msg("❌ Failed to paginate JobWorkers during heartbeat review")
			break
		}

		for _, jw := range findResult.Entities {
			jwCopy := jw
			allJobWorkers = append(allJobWorkers, &jwCopy)
		}

		if findResult.Cursor == "" || len(findResult.Entities) < pageSize {
			break
		}
		cursor = findResult.Cursor
	}

	if len(allJobWorkers) > 0 {
		upsertCtx, upsertCancel := context.WithTimeout(ctx, 30*time.Second)
		_, err := bo.BulkUpsertJobWorker(upsertCtx, allJobWorkers)
		upsertCancel()
		if err != nil {
			bo.Config.Logger.Error().Err(err).Msg("❌ Failed to update JobWorkers connection status")
		} else {
			bo.Config.Logger.Debug().Int("count", len(allJobWorkers)).Msg("✅ Updated connection status for existing JobWorkers")
		}
	}
}

// AckMessage acknowledges a message by executing the AckMessage command on the appropriate tenant node.
func (bo *JobWorkerBO) AckMessage(ctx context.Context, leaseID, tenantCode string) error {
	if leaseID == "" {
		return errors.New("leaseID is required")
	}
	if tenantCode == "" {
		return errors.New("tenantCode is required")
	}

	// Get the tenant to determine the correct node
	tenantBO := NewTenantBO(bo.Config)
	tenant, tenantNode, _, err := tenantBO.GetTenant(ctx, tenantCode)
	if err != nil {
		return fmt.Errorf("failed to get tenant %s: %w", tenantCode, err)
	}

	// Verify we have a valid tenant node
	if tenantNode == nil {
		return fmt.Errorf("failed to get node for tenant %s", tenantCode)
	}

	// Determine CF and CFS based on tenant (same pattern as dequeue)
	cf := db.ColumnFamilyPrefix + fmt.Sprintf("%d", tenant.ColumnFamilyIndex)
	cfs := tenant.ID

	// Execute the AckMessage command
	ackCmd := &queue_command.AckMessageCommand{
		LeaseID: leaseID,
		CF:      cf,
		CFS:     cfs,
	}

	result, err := dragonboat.ExecuteRepositoryCommand[queue_command.AckMessageResult](
		tenantNode,
		ctx,
		ackCmd,
		config.GlobalConfiguration.ApiRaftTimeout,
		bo.Config.Logger,
		"ack message",
	)

	if err != nil {
		return fmt.Errorf("failed to ack message: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("ack message failed: %s", result.Message)
	}

	bo.Config.Logger.Debug().
		Str("leaseID", leaseID).
		Str("tenantCode", tenantCode).
		Msg("✅ Message acknowledged successfully")

	if bo.Config.MetricsCollector != nil {
		bo.Config.MetricsCollector.RecordAck(tenantCode, result.QueueCode, result.VNamespace, 1)
		bo.Config.MetricsCollector.RecordLatency(tenantCode, result.QueueCode, result.VNamespace, uint64(result.ProcessingLatencyMs))
		bo.Config.MetricsCollector.UpdateGauges(tenantCode, result.QueueCode, result.VNamespace, result.Pending, result.InProcess)
	}

	return nil
}

func (bo *JobWorkerBO) deactivateTenant(tenantID string, tenantNode *dragonboat.RaftNode, cf string, cfs string) {
	// 1. Send MarkTenantInactiveCommand to Master Node
	inactiveCmd := &tentant_command.MarkTenantInactiveCommand{
		TenantID: tenantID,
	}
	fsmCmd := general_command.FSM_Command{
		Now:  utils.GetNowInInt(),
		Type: general_command.REPOSITORY_COMMAND,
		CMD:  inactiveCmd,
	}

	// Fire and wait to Master
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		resultChan, err := bo.Config.MasterNode.Write(ctx, fsmCmd)
		if err != nil {
			bo.Config.Logger.Debug().Err(err).Str("tenant", tenantID).Msg("Failed to start mark tenant inactive operation")
			return
		}
		select {
		case res := <-resultChan:
			if res.Error != nil {
				bo.Config.Logger.Debug().Err(res.Error).Str("tenant", tenantID).Msg("Failed to mark tenant inactive in Master")
			}
		case <-ctx.Done():
			bo.Config.Logger.Debug().Err(ctx.Err()).Str("tenant", tenantID).Msg("Timeout marking tenant inactive in Master")
		}
	}()

	// 2. Send ResetTenantShardStateCommand to Shard Node
	resetCmd := &tentant_command.ResetTenantShardStateCommand{
		TenantID: tenantID,
		CF:       cf,
		CFS:      cfs,
	}
	fsmResetCmd := general_command.FSM_Command{
		Now:  utils.GetNowInInt(),
		Type: general_command.REPOSITORY_COMMAND,
		CMD:  resetCmd,
	}

	// Fire and wait to Shard
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		resultChan, err := tenantNode.Write(ctx, fsmResetCmd)
		if err != nil {
			bo.Config.Logger.Debug().Err(err).Str("tenant", tenantID).Msg("Failed to start reset tenant shard state operation")
			return
		}
		select {
		case res := <-resultChan:
			if res.Error != nil {
				bo.Config.Logger.Debug().Err(res.Error).Str("tenant", tenantID).Msg("Failed to reset tenant shard state")
			}
		case <-ctx.Done():
			bo.Config.Logger.Debug().Err(ctx.Err()).Str("tenant", tenantID).Msg("Timeout resetting tenant shard state")
		}
	}()
}
