package app

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	"deadalus-orch/server/internal/pkg/config"
	"deadalus-orch/server/internal/usecase/command"
	general_command "deadalus-orch/server/internal/usecase/command/general"
	queue_command "deadalus-orch/server/internal/usecase/command/queue"
	tenant_command "deadalus-orch/server/internal/usecase/command/tentant"
	workflow_execution_command "deadalus-orch/server/internal/usecase/command/workflow-execution"
	"deadalus-orch/shared/models"

	"github.com/rs/zerolog/log"
)

type ExecutionTokenMessage struct {
	ExecutionID string `json:"executionId"`
	TokenID     string `json:"tokenId"`
}

func (app *Application) StartWorkflowExecutionWorker(interval time.Duration, batchSize int) {
	var workerLock sync.Mutex

	app.WorkflowExecutionStopper.RunWorker(func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if !workerLock.TryLock() {
					continue
				}

				go func() {
					defer workerLock.Unlock()

					if !app.MasterNodeIsReady || !app.MasterNodeIsLeader {
						return
					}

					select {
					case <-app.WorkflowExecutionStopper.ShouldStop():
						return
					default:
					}

					app.processWorkflowExecutionQueues(batchSize)
				}()

			case <-app.WorkflowExecutionStopper.ShouldStop():
				log.Info().Msg("ℹ️ Workflow execution worker stopped gracefully")
				return
			}
		}
	})
}

func (app *Application) processWorkflowExecutionQueues(batchSize int) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 1. Process MasterNode global execution queues
	app.processExecutionQueuesForNode(ctx, app.MasterNode, db.AdminFC, db.AdminFCSector, batchSize)

	// 2. Process TenantNodes execution queues
	cursor := ""
	pageSize := 50
	for {
		paginateTenantsCommand := &tenant_command.PaginateTenantsCommand{
			Cursor:   cursor,
			PageSize: pageSize,
		}

		queryCommand := &general_command.Query_Command{
			Command: &general_command.Repository_Command{
				CMD: paginateTenantsCommand,
			},
			Now: time.Now().UnixNano(),
		}

		result, err := app.MasterNode.Read(ctx, *queryCommand)
		if err != nil {
			break
		}

		tenantsResult, err := command.DecodeCommandResult[db.FindResult[models.TenantInMaster]](result.([]byte))
		if err != nil || len(tenantsResult.Entities) == 0 {
			break
		}

		for _, tenant := range tenantsResult.Entities {
			var tenantNode *dragonboat.RaftNode
			for i := range app.TenantNodes {
				if app.TenantNodes[i].ShardID == uint64(tenant.ShardId) {
					tenantNode = app.TenantNodes[i]
					break
				}
			}
			if tenantNode == nil {
				continue
			}

			cf := db.ColumnFamilyPrefix + fmt.Sprintf("%d", tenant.ColumnFamilyIndex)
			cfs := tenant.ID

			app.processExecutionQueuesForNode(ctx, tenantNode, cf, cfs, batchSize)
		}

		if tenantsResult.Cursor == "" || len(tenantsResult.Entities) < pageSize {
			break
		}
		cursor = tenantsResult.Cursor
	}
}

func (app *Application) processExecutionQueuesForNode(
	ctx context.Context,
	node *dragonboat.RaftNode,
	cf, cfs string,
	batchSize int,
) {
	if node == nil {
		return
	}

	cursor := ""
	for {
		pagCmd := &queue_command.PaginateQueuesCommand{
			PageSize:  100,
			QueueType: models.WorkflowExecutionQueue,
			Cursor:    cursor,
			CF:        cf,
			CFS:       cfs,
		}

		res, err := dragonboat.ExecuteRepositoryQuery[db.FindResult[models.Queue]](
			node,
			ctx,
			pagCmd,
			config.GlobalConfiguration.ApiRaftTimeout,
			log.Logger,
			"paginate execution queues",
		)
		if err != nil || len(res.Entities) == 0 {
			break
		}

		for _, q := range res.Entities {
			if q.Type != models.WorkflowExecutionQueue || q.State != models.QueueActive || q.MessagesCount == 0 {
				continue
			}

			deqCmd := &queue_command.DequeueCommand{
				QueueID:       q.ID,
				JobWorkerID:   "WorkflowExecutionWorker-1",
				LeaseDuration: 30 * time.Second,
				CF:            cf,
				CFS:           cfs,
			}

			deqRes, err := dragonboat.ExecuteRepositoryCommand[queue_command.DequeueResult](
				node,
				ctx,
				deqCmd,
				config.GlobalConfiguration.ApiRaftTimeout,
				log.Logger,
				"dequeue execution token message",
			)
			if err != nil || deqRes.Message.ID == "" {
				if err != nil {
					log.Debug().Err(err).Str("queueID", q.ID).Msg("⚠️ Failed to dequeue execution token message")
				}
				continue
			}

			log.Info().
				Str("queueID", q.ID).
				Str("messageID", deqRes.Message.ID).
				Msg("📥 Dequeued workflow execution token message")

			var tokenMsg ExecutionTokenMessage
			if err := json.Unmarshal(deqRes.Message.Content, &tokenMsg); err == nil && tokenMsg.ExecutionID != "" && tokenMsg.TokenID != "" {
				advanceCmd := &workflow_execution_command.AdvanceTokenCommand{
					ExecutionID: tokenMsg.ExecutionID,
					TokenID:     tokenMsg.TokenID,
					CF:          cf,
					CFS:         cfs,
				}
				_, advanceErr := dragonboat.ExecuteRepositoryCommand[models.WorkflowExecution](
					node,
					ctx,
					advanceCmd,
					config.GlobalConfiguration.ApiRaftTimeout,
					log.Logger,
					"advance token",
				)
				if advanceErr != nil {
					log.Error().Err(advanceErr).
						Str("executionID", tokenMsg.ExecutionID).
						Str("tokenID", tokenMsg.TokenID).
						Msg("❌ Failed to advance workflow execution token")
				} else {
					log.Info().
						Str("executionID", tokenMsg.ExecutionID).
						Str("tokenID", tokenMsg.TokenID).
						Msg("🚀 Successfully advanced workflow execution token")
				}

				// Ack message
				ackCmd := &queue_command.AckMessageCommand{
					LeaseID: deqRes.Lease.ID,
					CF:      cf,
					CFS:     cfs,
				}
				_, ackErr := dragonboat.ExecuteRepositoryCommand[queue_command.AckMessageResult](
					node,
					ctx,
					ackCmd,
					config.GlobalConfiguration.ApiRaftTimeout,
					log.Logger,
					"ack token message",
				)
				if ackErr != nil {
					log.Warn().Err(ackErr).Str("leaseID", deqRes.Lease.ID).Msg("⚠️ Failed to ack token message")
				}
			} else {
				log.Error().
					Str("content", string(deqRes.Message.Content)).
					Msg("❌ Failed to unmarshal ExecutionTokenMessage or missing fields")
			}
		}

		if res.Cursor == "" || len(res.Entities) < 100 {
			break
		}
		cursor = res.Cursor
	}
}
