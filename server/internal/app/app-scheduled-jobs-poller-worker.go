package app

import (
	"context"
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	"deadalus-orch/server/internal/pkg/utils"
	"deadalus-orch/server/internal/usecase/command"
	general_command "deadalus-orch/server/internal/usecase/command/general"
	scheduled_job_command "deadalus-orch/server/internal/usecase/command/scheduled-job"
	tenant_command "deadalus-orch/server/internal/usecase/command/tentant"
	"deadalus-orch/shared/models"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

func (app *Application) StartScheduledJobsPollerWorker(interval time.Duration, batchSize int) {
	var pollerLock sync.Mutex

	app.ScheduledJobsPollerStopper.RunWorker(func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if !pollerLock.TryLock() {
					log.Warn().Msg("⏳ Skipping scheduled jobs polling: previous execution still in progress")
					continue
				}

				go func() {
					defer pollerLock.Unlock()

					if !app.MasterNodeIsReady {
						return
					}

					if !app.MasterNodeIsLeader {
						return
					}

					select {
					case <-app.ScheduledJobsPollerStopper.ShouldStop():
						log.Info().Msg("🛑 Scheduled jobs poller received stop signal before starting")
						return
					default:
					}

					app.processAllTenantsScheduledJobs(batchSize)
				}()

			case <-app.ScheduledJobsPollerStopper.ShouldStop():
				log.Info().Msg("ℹ️ Scheduled jobs poller worker stopped gracefully")
				return
			}
		}
	})
}

func (app *Application) processAllTenantsScheduledJobs(batchSize int) {
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

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		result, err := app.MasterNode.Read(ctx, *queryCommand)
		cancel()

		if err != nil {
			log.Err(err).Msg("❌ Failed to paginate tenants for scheduled jobs polling")
			return
		}

		tenantsResult, err := command.DecodeCommandResult[db.FindResult[models.TenantInMaster]](result.([]byte))
		if err != nil {
			log.Err(err).Msg("❌ Failed to decode tenant pagination result for scheduled jobs polling")
			return
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

			select {
			case <-app.ScheduledJobsPollerStopper.ShouldStop():
				log.Info().Msg("🛑 Scheduled jobs poller stopped during tenant processing")
				return
			default:
			}

			app.processTenantScheduledJobs(tenantNode, &tenant, cf, cfs, batchSize)
		}

		if tenantsResult.Cursor == "" || len(tenantsResult.Entities) < pageSize {
			break
		}
		cursor = tenantsResult.Cursor
	}
}

func (app *Application) processTenantScheduledJobs(
	node *dragonboat.RaftNode,
	tenant *models.TenantInMaster,
	cf, cfs string,
	batchSize int,
) {
	cmd := general_command.FSM_Command{
		Now:  utils.GetNowInInt(),
		Type: general_command.REPOSITORY_COMMAND,
		CMD: scheduled_job_command.ProcessDueScheduledJobsCommand{
			BatchSize: batchSize,
			CF:        cf,
			CFS:       cfs,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resultChan, err := node.Write(ctx, cmd)
	if err != nil {
		log.Err(err).
			Str("tenant", tenant.Code).
			Str("node", strconv.FormatUint(node.ShardID, 10)).
			Msg("❌ Failed to start scheduled jobs processing on tenant node")
		return
	}

	select {
	case writeResult := <-resultChan:
		if writeResult.Error != nil {
			log.Err(writeResult.Error).
				Str("tenant", tenant.Code).
				Str("node", strconv.FormatUint(node.ShardID, 10)).
				Msg("❌ Failed to process scheduled jobs on tenant node")
			return
		}

		res, err := command.DecodeCommandResult[*scheduled_job_command.ProcessDueScheduledJobsResult](writeResult.Result.Data)
		if err != nil {
			log.Err(err).
				Str("tenant", tenant.Code).
				Str("node", strconv.FormatUint(node.ShardID, 10)).
				Msg("❌ Failed to decode scheduled jobs command result")
			return
		}

		if res != nil && res.ProcessedJobs > 0 {
			log.Info().
				Str("tenant", tenant.Code).
				Str("node", strconv.FormatUint(node.ShardID, 10)).
				Int("processedJobs", res.ProcessedJobs).
				Int("dispatchedMsgs", res.DispatchedMsgs).
				Msg("✅ Processed due scheduled jobs on tenant node")

			if app.MetricsCollector != nil {
				for _, gauge := range res.Gauges {
					app.MetricsCollector.UpdateGauges(tenant.Code, gauge.QueueCode, gauge.VNamespace, gauge.Pending, gauge.InProcess)
				}
				for _, detail := range res.DispatchedDetails {
					app.MetricsCollector.RecordPublish(tenant.Code, detail.QueueCode, detail.VNamespace, detail.Count)
				}
			}
		}

	case <-ctx.Done():
		log.Warn().
			Str("tenant", tenant.Code).
			Str("node", strconv.FormatUint(node.ShardID, 10)).
			Msg("⏱️ Scheduled jobs processing timed out on tenant node")
	}
}
