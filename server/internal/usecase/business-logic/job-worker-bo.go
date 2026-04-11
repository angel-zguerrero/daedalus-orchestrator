package business_logic

import (
	"context"
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	"deadalus-orch/server/internal/infrastructure/server/common"
	"deadalus-orch/server/internal/pkg/config"
	job_worker_command "deadalus-orch/server/internal/usecase/command/job-worker"
	queue_command "deadalus-orch/server/internal/usecase/command/queue"
	"deadalus-orch/shared/models"
	"errors"
	"fmt"
	"sync"
	"time"
)

// ClaimedMessage holds a dequeued message along with its lease and tenant info
type ClaimedMessage struct {
	Message    models.QueueMessage
	Lease      models.QueueMessageLease
	TenantCode string
}

type JobWorkerBO struct {
	Config     *common.ServerConfing
	stoppers   map[string]bool
	stoppersMu sync.Mutex
}

func NewJobWorkerBO(Config *common.ServerConfing) *JobWorkerBO {
	return &JobWorkerBO{
		Config:   Config,
		stoppers: make(map[string]bool),
	}
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
		bo.Config.Logger.Debug().Str("workerID", workerID).Msg("✅ ClaimWork stopper finished")
	}()

	stopperCtx, stopperCancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer stopperCancel()

	logger := bo.Config.Logger.With().Str("workerID", workerID).Logger()

	// Local claim counters per policy so we don't mutate the caller's map.
	claimedByPolicy := make(map[string]int, len(policies))
	for code, policy := range policies {
		claimedByPolicy[code] = policy.CurrentQueueMessages
	}

	// Returns true when every policy with a positive cap has been satisfied.
	allPoliciesSatisfied := func() bool {
		for code, policy := range policies {
			if policy.MaxQueueMessages > 0 && claimedByPolicy[code] < policy.MaxQueueMessages {
				return false
			}
		}
		return true
	}

	tenantBO := &TenantBO{Config: bo.Config}
	vnamespaceBO := &VNamespaceBO{Config: bo.Config}
	queueBO := &QueueBO{Config: bo.Config}

	for policyCode, policy := range policies {
		if allPoliciesSatisfied() {
			break
		}
		if policy.MaxQueueMessages > 0 && claimedByPolicy[policyCode] >= policy.MaxQueueMessages {
			continue
		}

		filter := policy.ClaimWorkFilter

		// ── Tenant pagination (DB-filtered) ───────────────────────────────────
		tenantCursor := ""
		const tenantPageSize = 50

	tenantLoop:
		for {
			if allPoliciesSatisfied() || (policy.MaxQueueMessages > 0 && claimedByPolicy[policyCode] >= policy.MaxQueueMessages) {
				break
			}

			paginateCtx, paginateCancel := context.WithTimeout(stopperCtx, 10*time.Second)
			tenantsResult, err := tenantBO.GetTenantsWithFilter(paginateCtx, filter, tenantCursor, tenantPageSize)
			paginateCancel()
			if err != nil {
				logger.Error().Err(err).Str("policy", policyCode).Msg("❌ Failed to paginate tenants during ClaimWork")
				break
			}

			//fmt.Printf("Policy %s: paginated %d tenants\n", policyCode, len(tenantsResult.Entities))

			for _, tenant := range tenantsResult.Entities {
				if allPoliciesSatisfied() || (policy.MaxQueueMessages > 0 && claimedByPolicy[policyCode] >= policy.MaxQueueMessages) {
					break tenantLoop
				}

				tenantNode := bo.getJobWorkerTenantNode(tenant)
				if tenantNode == nil {
					logger.Warn().Str("tenantCode", tenant.Code).Msg("No raft node found for tenant, skipping")
					continue
				}

				cf := db.ColumnFamilyPrefix + fmt.Sprintf("%d", tenant.ColumnFamilyIndex)
				cfs := tenant.ID

				// ── VNamespace pagination (DB-filtered) ───────────────────────
				vnsCursor := ""
				const vnsPageSize = 50

			vnsLoop:
				for {
					if allPoliciesSatisfied() || (policy.MaxQueueMessages > 0 && claimedByPolicy[policyCode] >= policy.MaxQueueMessages) {
						break
					}

					vnsCtx, vnsCancel := context.WithTimeout(stopperCtx, 10*time.Second)
					vnsResult, err := vnamespaceBO.GetVNamespacesWithFilter(vnsCtx, filter, vnsCursor, vnsPageSize, cf, cfs, &tenant, tenantNode)
					if tenant.Code == "QDBW9597" {
						fmt.Printf("Policy %s: paginated %d vnamespaces for tenant %s\n", policyCode, len(vnsResult.Entities), tenant.Code)
					}
					vnsCancel()
					if err != nil {
						logger.Error().Err(err).
							Str("policy", policyCode).
							Str("tenant", tenant.Code).
							Msg("❌ Failed to paginate vnamespaces during ClaimWork")
						break
					}

					for _, vns := range vnsResult.Entities {
						if allPoliciesSatisfied() || (policy.MaxQueueMessages > 0 && claimedByPolicy[policyCode] >= policy.MaxQueueMessages) {
							break vnsLoop
						}

						// ── Queue pagination (DB-filtered; MessagesCount > 0 guaranteed) ──
						queueCursor := ""
						const queuePageSize = 50

					queueLoop:
						for {
							if allPoliciesSatisfied() || (policy.MaxQueueMessages > 0 && claimedByPolicy[policyCode] >= policy.MaxQueueMessages) {
								break
							}

							qCtx, qCancel := context.WithTimeout(stopperCtx, 10*time.Second)
							queuesResult, err := queueBO.GetQueuesWithFilter(qCtx, filter, queueCursor, queuePageSize, vns.Name, cf, cfs, &tenant, tenantNode)
							qCancel()
							if err != nil {
								logger.Error().Err(err).
									Str("policy", policyCode).
									Str("tenant", tenant.Code).
									Str("vnamespace", vns.Name).
									Msg("❌ Failed to paginate queues during ClaimWork")
								break
							}

							for _, queue := range queuesResult.Entities {
								fmt.Printf("Policy %s: evaluating queue %s (MessagesCount=%d, CurrentDeliveringMessages=%d, MaxDeliveringMessages=%d) in vnamespace %s for tenant %s\n",
									policyCode, queue.Code, queue.MessagesCount, queue.CurrentDeliveringMessages, queue.MaxDeliveringMessages, vns.Name, tenant.Code)
								if allPoliciesSatisfied() || (policy.MaxQueueMessages > 0 && claimedByPolicy[policyCode] >= policy.MaxQueueMessages) {
									break queueLoop
								}

								// Respect the queue's own delivering-message cap (0 = unlimited).
								// MessagesCount > 0 is already guaranteed by the DB query.
								if queue.MaxDeliveringMessages > 0 && queue.CurrentDeliveringMessages >= queue.MaxDeliveringMessages {
									continue
								}

								// ── Dequeue message ──
								bo.dequeueMessage(stopperCtx, workerID, policyCode, &queue, &tenant, tenantNode, cf, cfs, messageChan)

								claimedByPolicy[policyCode]++
							}

							if queuesResult.Cursor == "" || len(queuesResult.Entities) < queuePageSize {
								break
							}
							queueCursor = queuesResult.Cursor
						}
					}

					if vnsResult.Cursor == "" || len(vnsResult.Entities) < vnsPageSize {
						break
					}
					vnsCursor = vnsResult.Cursor
				}
			}

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
	queue *models.Queue,
	tenant *models.TenantInMaster,
	tenantNode *dragonboat.RaftNode,
	cf, cfs string,
	messageChan chan<- ClaimedMessage,
) {
	bo.Config.Logger.Debug().
		Str("workerID", workerID).
		Str("policyCode", policyCode).
		Str("queueCode", queue.Code).
		Str("tenant", tenant.Code).
		Msg("📭 Dequeuing message from queue")

	// Crear el comando de dequeue
	dequeueCmd := &queue_command.DequeueCommand{
		QueueID:       queue.ID,
		JobWorkerID:   workerID,
		LeaseDuration: config.GlobalConfiguration.MessageLeaseDuration,
		CF:            cf,
		CFS:           cfs,
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
		bo.Config.Logger.Error().Err(err).
			Str("workerID", workerID).
			Str("queueCode", queue.Code).
			Str("tenant", tenant.Code).
			Msg("❌ Failed to dequeue message")
		return
	}

	fmt.Printf("✅ Dequeued message:\n")
	fmt.Printf("   Queue: %s (Tenant: %s)\n", queue.Code, tenant.Code)
	fmt.Printf("   Message ID: %s\n", result.Message.ID)
	fmt.Printf("   Priority: %d\n", result.Message.Priority)
	fmt.Printf("   Content: %s\n", string(result.Message.Content))
	fmt.Printf("   Parameters: %v\n", result.Message.Parameters)
	fmt.Printf("   Handler: %v\n", result.Message.Handler)
	fmt.Printf("   Lease ID: %s\n", result.Lease.ID)
	fmt.Printf("   Lease Until: %s\n", result.Lease.LeaseUntil.Format(time.RFC3339))
	fmt.Printf("   Worker: %s\n\n", workerID)

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
