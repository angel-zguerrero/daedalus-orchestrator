package business_logic

import (
	"context"
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	"deadalus-orch/server/internal/infrastructure/server/common"
	"deadalus-orch/server/internal/pkg/config"
	job_worker_command "deadalus-orch/server/internal/usecase/command/job-worker"
	queue_command "deadalus-orch/server/internal/usecase/command/queue"
	tentant_command "deadalus-orch/server/internal/usecase/command/tentant"
	general_command "deadalus-orch/server/internal/usecase/command/general"
	"deadalus-orch/shared/models"
	"deadalus-orch/server/internal/pkg/utils"
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
}

// claimCursorKey identifies a pagination cursor stored in the registry.
// tenantID is empty for the tenant-level cursor; it is set for vnamespace-level cursors.
type claimCursorKey struct {
	workerID   string
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

// evictWorker removes all cursors belonging to workerID.
func (r *claimCursorRegistry) evictWorker(workerID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for key := range r.cursors {
		if key.workerID == workerID {
			delete(r.cursors, key)
		}
	}
}

type JobWorkerBO struct {
	Config         *common.ServerConfing
	stoppers        map[string]bool
	stoppersMu      sync.Mutex
	cursorRegistry  *claimCursorRegistry
}

func NewJobWorkerBO(Config *common.ServerConfing) *JobWorkerBO {
	return &JobWorkerBO{
		Config:         Config,
		stoppers:        make(map[string]bool),
		cursorRegistry:  newClaimCursorRegistry(),
	}
}

// EvictWorkerCursors removes all cached pagination cursors for the given worker.
// Call this when a worker's gRPC stream closes so stale cursors don't accumulate.
func (bo *JobWorkerBO) EvictWorkerCursors(workerID string) {
	bo.cursorRegistry.evictWorker(workerID)
}

func (bo *JobWorkerBO) ClaimWork(ctx context.Context, workerId string, workerName string, Information map[string]string, ClaimWorkCapacityPolicies map[string]models.ClaimWorkCapacityPolicy, messageChan chan<- ClaimedMessage) error {
	// Upsert the JobWorker: update LastHeartbeat and TTL on every ClaimWork call
	upsertCmd := &job_worker_command.UpsertJobWorkerCommand{
		JobWorkers: []models.JobWorker{
			{
				ID:                        workerId,
				Name:                      workerName,
				Information:               Information,
				ClaimWorkCapacityPolicies: ClaimWorkCapacityPolicies,
				ConnectionStatus:          models.JobWorkerConnectionStatusConnected,
			},
		},
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
		return fmt.Errorf("failed to upsert job worker: %w", err)
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
	go bo.runClaimWorkStopper(workerId, ClaimWorkCapacityPolicies, messageChan)

	return nil
}

// runClaimWorkStopper is the goroutine that drives the claim-work process for a single JobWorker.
// It terminates when all ClaimWorkCapacityPolicies are satisfied or all pagination is exhausted,
// after which a subsequent ClaimWork call is allowed to spawn a new stopper.
func (bo *JobWorkerBO) runClaimWorkStopper(workerID string, policies map[string]models.ClaimWorkCapacityPolicy, messageChan chan<- ClaimedMessage) {
	// Always release the stopper slot when the goroutine exits so the next ClaimWork call
	// can spawn a new one.
	defer func() {
		bo.stoppersMu.Lock()
		bo.stoppers[workerID] = false
		bo.stoppersMu.Unlock()
	}()

	stopperCtx, stopperCancel := context.WithTimeout(context.Background(), 5*time.Minute)
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
		tenantKey := claimCursorKey{workerID: workerID, policyCode: policyCode}
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
						tenantQueueKey := claimCursorKey{workerID: workerID, policyCode: policyCode, tenantID: tenant.ID}
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

							if len(allQueues) == 0 && queueCursor == "" {
								// No queues match this policy. Let's check if the tenant is completely empty.
								emptyFilter := models.ClaimWorkFilter{}
								qCtx2, qCancel2 := context.WithTimeout(stopperCtx, 2*time.Second)
								allActive, err2 := queueBO.GetQueuesWithFilter(qCtx2, emptyFilter, "", 1, cf, cfs, &tenant, g.node)
								qCancel2()
								
								if err2 == nil && (allActive.Entities == nil || len(allActive.Entities) == 0) {
									bo.deactivateTenant(tenant.ID, g.node, cf, cfs)
								}
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

	// Crear el comando de dequeue
	dequeueCmd := &queue_command.DequeueCommand{
		QueueID:                      queue.ID,
		JobWorkerID:                  workerID,
		LeaseDuration:                config.GlobalConfiguration.MessageLeaseDuration,
		JobWorkerCapacityPolicyIndex: policyIndex,
		CF:                           cf,
		CFS:                          cfs,
	}

	// Ejecutar el comando en el nodo de tenant correspondiente
	result, err := dragonboat.ExecuteRepositoryCommand[queue_command.DequeueResult](
		tenantNode,
		ctx,
		dequeueCmd,
		config.GlobalConfiguration.ApiRaftTimeout,
		bo.Config.Logger,
		"dequeue message",
	)

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

	// Send the claimed message through the channel
	claimedMsg := ClaimedMessage{
		Message:    result.Message,
		Lease:      result.Lease,
		TenantCode: tenant.Code,
	}

	select {
	case messageChan <- claimedMsg:
		bo.Config.Logger.Debug().Str("messageID", result.Message.ID).Msg("📤 Sent message to stream")
	default:
		bo.Config.Logger.Warn().Str("messageID", result.Message.ID).Msg("⚠️ Message channel full or closed, message not sent")
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

	// Fire and forget to Master
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, err := bo.Config.MasterNode.Write(ctx, fsmCmd)
		if err != nil {
			bo.Config.Logger.Debug().Err(err).Str("tenant", tenantID).Msg("Failed to mark tenant inactive in Master")
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

	// Fire and forget to Shard
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, err := tenantNode.Write(ctx, fsmResetCmd)
		if err != nil {
			bo.Config.Logger.Debug().Err(err).Str("tenant", tenantID).Msg("Failed to reset tenant shard state")
		}
	}()
}
