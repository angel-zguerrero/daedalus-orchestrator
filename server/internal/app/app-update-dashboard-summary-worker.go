package app

import (
	"context"
	general_command "deadalus-orch/server/internal/usecase/command/general"
	tentant_command "deadalus-orch/server/internal/usecase/command/tentant"
	"deadalus-orch/server/internal/pkg/utils"
	commands "deadalus-orch/server/internal/usecase/command"
	"deadalus-orch/shared/models"
	"fmt"
	"time"

	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/infrastructure/dragonboat"

	"github.com/rs/zerolog/log"
)

func (app *Application) StartDashboardSummaryWorker(interval time.Duration) {
	app.DashboardSummaryWorkerStopper.RunWorker(func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if !app.MasterNodeIsReady {
					log.Debug().Msg("⏳ DashboardSummary worker is waiting for the master node to be ready")
					continue
				}

				if !app.MasterNodeIsLeader {
					log.Debug().Msg("⏳ DashboardSummary worker is waiting for the master node to be leader")
					continue
				}

				select {
				case <-app.DashboardSummaryWorkerStopper.ShouldStop():
					log.Info().Msg("🛑 DashboardSummary worker received stop signal before execution")
					return
				default:
				}

				go func() {
					app.updateDashboardSummary()
				}()

			case <-app.DashboardSummaryWorkerStopper.ShouldStop():
				log.Info().Msg("ℹ️  DashboardSummary worker stopped gracefully")
				return
			}
		}
	})
}

func (app *Application) updateDashboardSummary() {
	now := time.Now()

	log.Info().Msg("🔄 Starting dashboard summary update process")

	summary, err := app.aggregateDashboardSummary(now)
	if err != nil {
		log.Err(err).Msg("❌ Failed to aggregate dashboard summary")
		return
	}

	if err := app.writeDashboardSummaryToMaster(summary); err != nil {
		log.Err(err).Msg("❌ Failed to write dashboard summary to master node")
		return
	}

	log.Info().
		Int("tenants", summary.TenantsCount).
		Int("exchanges", summary.ExchangesCount).
		Int("queues", summary.QueuesCount).
		Int("bindings", summary.BindingsCount).
		Int("messages", summary.MessagesCount).
		Msg("✅ Dashboard summary updated")
}

// aggregateDashboardSummary paginates all TenantInMaster records (100 per batch) from the
// master node and accumulates the global counters.
func (app *Application) aggregateDashboardSummary(now time.Time) (models.DashboardSummary, error) {
	cursor := ""
	pageSize := 100

	summary := models.DashboardSummary{
		ID: models.DashboardSummaryID,
	}

	for {
		select {
		case <-app.DashboardSummaryWorkerStopper.ShouldStop():
			log.Info().Msg("🛑 DashboardSummary worker received stop signal during aggregation")
			return summary, nil
		default:
		}

		paginateCommand := &tentant_command.PaginateTenantsCommand{
			Cursor:   cursor,
			PageSize: pageSize,
			Q:        "",
		}

		queryCommand := &general_command.Query_Command{
			Command: &general_command.Repository_Command{
				CMD: paginateCommand,
			},
			Now: now.UnixNano(),
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		result, err := app.MasterNode.Read(ctx, *queryCommand)
		cancel()

		if err != nil {
			return summary, fmt.Errorf("failed to paginate tenants: %w", err)
		}

		tenantsResult, err := commands.DecodeCommandResult[db.FindResult[models.TenantInMaster]](result.([]byte))
		if err != nil {
			return summary, fmt.Errorf("failed to decode paginate result: %w", err)
		}

		for _, tenant := range tenantsResult.Entities {
			summary.TenantsCount++
			summary.ExchangesCount += tenant.ExchangesCount
			summary.QueuesCount += tenant.QueuesCount
			summary.BindingsCount += tenant.BindingsCount
			summary.MessagesCount += tenant.MessagesCount
		}

		log.Debug().
			Int("batch_size", len(tenantsResult.Entities)).
			Int("tenants_so_far", summary.TenantsCount).
			Msg("📊 Processing batch of tenants for dashboard summary")

		if tenantsResult.Cursor == "" {
			break
		}
		cursor = tenantsResult.Cursor
	}

	return summary, nil
}

func (app *Application) writeDashboardSummaryToMaster(summary models.DashboardSummary) error {
	updateCommand := &tentant_command.UpdateDashboardSummaryCommand{
		Summary: summary,
	}

	fsm_cmd := general_command.FSM_Command{
		Now:  utils.GetNowInInt(),
		Type: general_command.REPOSITORY_COMMAND,
		CMD:  updateCommand,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resultChan, err := app.MasterNode.Write(ctx, fsm_cmd)
	if err != nil {
		return err
	}

	// Wait for the result since we need to process it
	var writeResult dragonboat.WriteResult
	select {
	case writeResult = <-resultChan:
		if writeResult.Error != nil {
			return writeResult.Error
		}
	case <-ctx.Done():
		return ctx.Err()
	}

	_, err = commands.DecodeCommandResult[models.DashboardSummary](writeResult.Result.Data)
	return err
}
