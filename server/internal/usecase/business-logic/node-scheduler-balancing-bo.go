package business_logic

import (
	"context"
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	"deadalus-orch/server/internal/infrastructure/server/common"
	"deadalus-orch/server/internal/pkg/config"
	node_scheduler "deadalus-orch/server/internal/usecase/command/node-scheduler"
	"deadalus-orch/shared/models"
	"fmt"
)

type NodeSchedulerBalancingBO struct {
	Config *common.ServerConfing
}

func NewNodeSchedulerBalancingBO(Config *common.ServerConfing) *NodeSchedulerBalancingBO {
	return &NodeSchedulerBalancingBO{
		Config: Config,
	}
}

func (bo *NodeSchedulerBalancingBO) GetState(ctx context.Context) (*models.NodeSchedulerBalancingState, error) {
	getStateCommand := &node_scheduler.GetNodeSchedulerBalancingStateCommand{}

	state, err := dragonboat.ExecuteRepositoryQuery[models.NodeSchedulerBalancingState](
		bo.Config.MasterNode,
		ctx,
		getStateCommand,
		config.GlobalConfiguration.ApiRaftTimeout,
		bo.Config.Logger,
		"get node scheduler balancing state",
	)
	if err != nil {
		return nil, fmt.Errorf("get node scheduler balancing state command failed: %w", err)
	}

	return &state, nil
}

func (bo *NodeSchedulerBalancingBO) UpsertState(ctx context.Context, state models.NodeSchedulerBalancingState) error {
	upsertCommand := &node_scheduler.UpsertNodeSchedulerBalancingStateCommand{
		State: state,
	}

	_, err := dragonboat.ExecuteRepositoryCommand[bool](
		bo.Config.MasterNode,
		ctx,
		upsertCommand,
		config.GlobalConfiguration.ApiRaftTimeout,
		bo.Config.Logger,
		"upsert node scheduler balancing state",
	)
	return err
}

func (bo *NodeSchedulerBalancingBO) BalanceNodeSchedulers(ctx context.Context, tenantNodes []*dragonboat.RaftNode, supervisionState models.QueueSupervisionState, lastIndices map[int]int, balancingId string) (map[int]int, error) {
	nodeSchedulerBO := NewNodeSchedulerBO(bo.Config)
	queueBO := NewQueueBO(bo.Config)

	if lastIndices == nil {
		lastIndices = make(map[int]int)
	}

	bo.Config.Logger.Info().
		Str("supervisionState", string(supervisionState)).
		Msg("⚖️ Starting Node Scheduler balancing process...")

	// Iterate through each TenantNode
	for tenantNodeIndex, tenantNode := range tenantNodes {
		bo.Config.Logger.Info().
			Int("tenantNodeIndex", tenantNodeIndex).
			Uint64("shardID", tenantNode.ShardID).
			Msg("⚖️ Processing TenantNode")

		// Get NodeSchedulers assigned to this TenantNode

		nodeSchedulersCursor := ""
		nodeSchedulersPageSize := 100
		var assignedNodeSchedulers []models.NodeScheduler

		for {
			findResult, err := nodeSchedulerBO.GetNodeSchedulersUsingAssignedTenantNodeIndex(ctx, "", nodeSchedulersCursor, nodeSchedulersPageSize, tenantNodeIndex, balancingId)
			if err != nil {
				return lastIndices, fmt.Errorf("failed to fetch node schedulers for TenantNode %d: %w", tenantNodeIndex, err)
			}

			assignedNodeSchedulers = append(assignedNodeSchedulers, findResult.Entities...)

			if findResult.Cursor == "" || len(findResult.Entities) < nodeSchedulersPageSize {
				break
			}
			nodeSchedulersCursor = findResult.Cursor
		}

		nodeSchedulerCount := len(assignedNodeSchedulers)
		if nodeSchedulerCount == 0 {
			bo.Config.Logger.Warn().
				Int("tenantNodeIndex", tenantNodeIndex).
				Msg("⚠️ No NodeSchedulers assigned to this TenantNode, skipping")
			continue
		}

		bo.Config.Logger.Info().
			Int("tenantNodeIndex", tenantNodeIndex).
			Int("nodeSchedulerCount", nodeSchedulerCount).
			Msg("⚖️ Found NodeSchedulers for TenantNode")

		// Calculate dynamic page size for queues based on NodeScheduler count
		// We want to distribute queues evenly, so we fetch in batches that are multiples of nodeSchedulerCount
		queuesPageSize := nodeSchedulerCount * 10 // Fetch 10 queues per NodeScheduler at a time
		if queuesPageSize > 1000 {
			queuesPageSize = 1000 // Cap at reasonable limit
		}

		queuesCursor := ""
		totalQueuesProcessed := 0

		// Resume from last index for this tenant node
		nodeSchedulerIndex := lastIndices[tenantNodeIndex]

		// First, get all tenants to iterate through them
		tenantBO := NewTenantBO(bo.Config)
		tenantsCursor := ""
		tenantsPageSize := 100

		// Paginate through all tenants
		for {
			tenantsResult, err := tenantBO.GetTenants(ctx, "", tenantsCursor, tenantsPageSize)
			if err != nil {
				return lastIndices, fmt.Errorf("failed to fetch tenants for TenantNode %d: %w", tenantNodeIndex, err)
			}

			if len(tenantsResult.Entities) == 0 {
				break
			}

			// For each tenant, fetch and distribute its queues
			for _, tenant := range tenantsResult.Entities {
				// Skip if tenant is not assigned to this TenantNode
				if tenant.ShardId != int(tenantNode.ShardID) {
					continue
				}

				// Calculate cf and cfs for this tenant
				cf := db.ColumnFamilyPrefix + fmt.Sprintf("%d", tenant.ColumnFamilyIndex)
				cfs := tenant.ID

				// Paginate through all queues for this tenant
				queuesCursor = ""
				for {

					var findResult db.FindResult[models.Queue]
					var err error
					if supervisionState == "" {
						findResult, err = queueBO.GetQueues(ctx, "", queuesCursor, queuesPageSize, "", false, cf, cfs, &tenant, tenantNode, false)
						if err != nil {
							bo.Config.Logger.Warn().
								Err(err).
								Str("tenantId", tenant.ID).
								Int("tenantNodeIndex", tenantNodeIndex).
								Msg("⚠️ Failed to fetch queues for tenant, skipping")
							break
						}
					} else {
						findResult, err = queueBO.GetQueuesBySupervisionState(ctx, "", queuesCursor, queuesPageSize, "", false, cf, cfs, &tenant, tenantNode, supervisionState, false)
						if err != nil {
							bo.Config.Logger.Warn().
								Err(err).
								Str("tenantId", tenant.ID).
								Int("tenantNodeIndex", tenantNodeIndex).
								Msg("⚠️ Failed to fetch queues for tenant, skipping")
							break
						}
					}

					if len(findResult.Entities) == 0 {
						break
					}

					// Distribute queues among NodeSchedulers in round-robin fashion
					var queuesToUpdate []models.Queue
					for _, queue := range findResult.Entities {
						// If supervisionState is Supervised, skip queues that are already supervised
						if supervisionState == models.Supervised && queue.NodeSchedulerQueueSupervisionState == models.Supervised {
							continue
						}

						// Assign NodeSchedulerSupervisorId in round-robin
						nodeSchedulerIndex++
						assignedNodeScheduler := assignedNodeSchedulers[nodeSchedulerIndex%nodeSchedulerCount]
						queue.NodeSchedulerSupervisorId = assignedNodeScheduler.ID
						queue.NodeSchedulerQueueSupervisionState = models.Supervised
						queuesToUpdate = append(queuesToUpdate, queue)
					}

					// Bulk update the queues
					if len(queuesToUpdate) > 0 {
						_, err = queueBO.AssignNodeSchedulerToQueues(ctx, queuesToUpdate, cf, cfs, tenantNode)
						if err != nil {
							bo.Config.Logger.Warn().
								Err(err).
								Str("tenantId", tenant.ID).
								Int("tenantNodeIndex", tenantNodeIndex).
								Msg("⚠️ Failed to update queues for tenant, skipping")
							break
						}
						totalQueuesProcessed += len(queuesToUpdate)
					}

					if findResult.Cursor == "" || len(findResult.Entities) < queuesPageSize {
						break
					}
					queuesCursor = findResult.Cursor
				}
			}

			if tenantsResult.Cursor == "" || len(tenantsResult.Entities) < tenantsPageSize {
				break
			}
			tenantsCursor = tenantsResult.Cursor
		}

		// Store last index
		lastIndices[tenantNodeIndex] = nodeSchedulerIndex

		bo.Config.Logger.Info().
			Int("tenantNodeIndex", tenantNodeIndex).
			Int("totalQueuesProcessed", totalQueuesProcessed).
			Int("nodeSchedulerCount", nodeSchedulerCount).
			Msg("⚖️ Queues distributed among NodeSchedulers")

		// Update all NodeSchedulers to running status
		var nodeSchedulersToUpdate []*models.NodeScheduler
		for i := range assignedNodeSchedulers {
			ns := assignedNodeSchedulers[i]
			ns.RunningStatus = models.NodeSchedulerRunningStatusRunning
			nodeSchedulersToUpdate = append(nodeSchedulersToUpdate, &ns)
		}

		if len(nodeSchedulersToUpdate) > 0 {
			_, err := nodeSchedulerBO.UpdateRunningStatusNodeScheduler(ctx, nodeSchedulersToUpdate, models.NodeSchedulerRunningStatusRunning)
			if err != nil {
				return lastIndices, fmt.Errorf("failed to update NodeSchedulers status for TenantNode %d: %w", tenantNodeIndex, err)
			}

			bo.Config.Logger.Info().
				Int("tenantNodeIndex", tenantNodeIndex).
				Int("count", len(nodeSchedulersToUpdate)).
				Msg("✅ NodeSchedulers set to running status")
		}
	}

	bo.Config.Logger.Info().Msg("✅ Node Scheduler balancing completed successfully")
	return lastIndices, nil
}
