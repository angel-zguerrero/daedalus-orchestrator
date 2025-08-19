package app

import (
	"bytes"
	"context"
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	"deadalus-orch/server/internal/pkg/utils"
	commands "deadalus-orch/server/internal/usecase/command"
	general_command "deadalus-orch/server/internal/usecase/command/general"
	tenant_summary_command "deadalus-orch/server/internal/usecase/command/tenant-summary"
	tentant_command "deadalus-orch/server/internal/usecase/command/tentant"
	"deadalus-orch/shared/models"
	"encoding/gob"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
)

func (app *Application) StartTenantSummaryWorker(interval time.Duration) {
	app.TenantSummaryWorkerStopper.RunWorker(func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if !app.MasterNodeIsReady {
					log.Debug().Msg("⏳ TenantSummary worker is waiting for the master node to be ready")
					continue
				}

				if !app.MasterNodeIsLeader {
					log.Debug().Msg("⏳ TenantSummary worker is waiting for the master node to be leader")
					continue
				}

				select {
				case <-app.TenantSummaryWorkerStopper.ShouldStop():
					log.Info().Msg("🛑 TenantSummary worker received stop signal before execution")
					return
				default:
				}

				go func() {
					app.updateTenantSummaries()
				}()

			case <-app.TenantSummaryWorkerStopper.ShouldStop():
				log.Info().Msg("ℹ️  TenantSummary worker stopped gracefully")
				return
			}
		}
	})
}

func (app *Application) updateTenantSummaries() {
	now := time.Now()

	log.Info().Msg("🔄 Starting tenant summaries update process")

	// Process each TenantNode separately with its own stopper
	for _, tenantNode := range app.TenantNodes {
		select {
		case <-app.TenantSummaryWorkerStopper.ShouldStop():
			log.Info().Msg("🛑 Tenant summary worker received stop signal during processing")
			return
		default:
		}

		go func(node *dragonboat.RaftNode) {
			app.processTenantNode(node, now)
		}(tenantNode)
	}
}

func (app *Application) processTenantNode(tenantNode *dragonboat.RaftNode, now time.Time) {
	log.Info().Uint64("shard_id", tenantNode.ShardID).Msg("🏗️  Processing tenant node")

	// Get last update timestamp from KV store in this TenantNode
	lastUpdateTime, err := app.getLastUpdateAtFromTenantNode(tenantNode)
	if err != nil {
		log.Warn().Uint64("shard_id", tenantNode.ShardID).Err(err).Msg("⚠️  Failed to get last update time, using 24 hours ago")
		lastUpdateTime = now.Add(-24 * time.Hour)
	}

	if lastUpdateTime.IsZero() {
		log.Info().Uint64("shard_id", tenantNode.ShardID).Msg("ℹ️  No previous update time found, using 24 hours ago")
		lastUpdateTime = now.Add(-24 * time.Hour)
	}

	log.Info().Uint64("shard_id", tenantNode.ShardID).Time("last_update_at", lastUpdateTime).Msg("📅 Using last update time")

	// Process all tenant summaries from this node using pagination
	err = app.processTenantSummariesFromNode(tenantNode, lastUpdateTime, now)
	if err != nil {
		log.Err(err).Uint64("shard_id", tenantNode.ShardID).Msg("❌ Failed to process tenant summaries")
		return
	}

	// Update the last update timestamp in KV store for this node
	err = app.refreshLastUpdateAtInTenantNode(tenantNode, now)
	if err != nil {
		log.Err(err).Uint64("shard_id", tenantNode.ShardID).Msg("❌ Failed to refresh last update time")
		return
	}

	log.Info().Uint64("shard_id", tenantNode.ShardID).Msg("✅ Completed processing tenant node")
}

func (app *Application) processTenantSummariesFromNode(tenantNode *dragonboat.RaftNode, fromTime, now time.Time) error {
	cursor := ""
	pageSize := 100

	for {
		select {
		case <-app.TenantSummaryWorkerStopper.ShouldStop():
			log.Info().Uint64("shard_id", tenantNode.ShardID).Msg("🛑 Received stop signal during summary processing")
			return nil
		default:
		}

		// Paginate tenant summaries from this node
		paginateCommand := &tenant_summary_command.PaginateTenantUpdatedAtFromCommand{
			LastUpdatedAt: fromTime,
			Cursor:        cursor,
			PageSize:      pageSize,
		}

		queryCommand := &general_command.Query_Command{
			Command: &general_command.Repository_Command{
				CMD: paginateCommand,
			},
			Now: now.UnixNano(),
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		result, err := tenantNode.Read(ctx, *queryCommand)
		cancel()

		if err != nil {
			return err
		}

		buf := bytes.NewBuffer(result.([]byte))
		dec := gob.NewDecoder(buf)
		parsedResult := &commands.CommandResult{}
		if err := dec.Decode(parsedResult); err != nil {
			return err
		}

		if parsedResult.Error != "" {
			return fmt.Errorf("command error: %s", parsedResult.Error)
		}

		summariesResult := parsedResult.Result.(db.FindResult[models.TenantSummary])

		// Process each summary and update master node
		if len(summariesResult.Entities) > 0 {
			log.Debug().
				Uint64("shard_id", tenantNode.ShardID).
				Int("summaries_count", len(summariesResult.Entities)).
				Msg("📊 Processing batch of tenant summaries")

			// Since there's one summary per tenant, create a map by tenant code
			summariesById := make(map[string]models.TenantSummary)
			for _, summary := range summariesResult.Entities {
				// Use the most recent summary if there are duplicates
				if existing, exists := summariesById[summary.ID]; !exists || summary.UpdatedAt.After(existing.UpdatedAt) {
					summariesById[summary.ID] = summary
				}
			}

			// Update master node for each tenant ID
			for tenantID, summary := range summariesById {
				err = app.updateMasterWithSummary(tenantID, summary)
				if err != nil {
					log.Err(err).
						Str("tenant_id", tenantID).
						Uint64("shard_id", tenantNode.ShardID).
						Msg("❌ Failed to update tenant summary in master")
					continue
				}

				log.Debug().
					Str("tenant_id", tenantID).
					Int("exchanges", summary.ExchangesCount).
					Int("queues", summary.QueuesCount).
					Int("messages", summary.MessagesCount).
					Msg("✅ Updated tenant summary in master")
			}
		}

		// Check if we have more pages
		if summariesResult.Cursor == "" || len(summariesResult.Entities) < pageSize {
			break
		}
		cursor = summariesResult.Cursor
	}

	return nil
}

func (app *Application) getLastUpdateAtFromTenantNode(tenantNode *dragonboat.RaftNode) (time.Time, error) {
	getLastUpdateCommand := &tenant_summary_command.GetLastUpdateAtFromCommand{}

	queryCommand := &general_command.Query_Command{
		Command: &general_command.Repository_Command{
			CMD: getLastUpdateCommand,
		},
		Now: time.Now().UnixNano(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := tenantNode.Read(ctx, *queryCommand)
	if err != nil {
		return time.Time{}, err
	}

	buf := bytes.NewBuffer(result.([]byte))
	dec := gob.NewDecoder(buf)
	parsedResult := &commands.CommandResult{}
	if err := dec.Decode(parsedResult); err != nil {
		return time.Time{}, err
	}

	if parsedResult.Error != "" {
		return time.Time{}, fmt.Errorf("command error: %s", parsedResult.Error)
	}

	return parsedResult.Result.(time.Time), nil
}

func (app *Application) updateMasterWithSummary(_ string, summary models.TenantSummary) error {
	updateCommand := &tentant_command.UpdateTenantSummaryCommand{
		TenantSummaries: []models.TenantSummary{summary},
	}

	fsm_cmd := general_command.FSM_Command{
		Now:  utils.GetNowInInt(),
		Type: general_command.REPOSITORY_COMMAND,
		CMD:  updateCommand,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resultData, err := app.MasterNode.Write(ctx, fsm_cmd)
	if err != nil {
		return err
	}

	buf := bytes.NewBuffer(resultData.Data)
	dec := gob.NewDecoder(buf)
	parsedResult := &commands.CommandResult{}
	if err := dec.Decode(parsedResult); err != nil {
		return err
	}

	if parsedResult.Error != "" {
		return fmt.Errorf("command error: %s", parsedResult.Error)
	}

	return nil
}

func (app *Application) refreshLastUpdateAtInTenantNode(tenantNode *dragonboat.RaftNode, updateTime time.Time) error {
	refreshCommand := &tenant_summary_command.RefreshLastUpdateAtFromCommand{
		LastUpdateAtFrom: updateTime,
	}

	fsm_cmd := general_command.FSM_Command{
		Now:  utils.GetNowInInt(),
		Type: general_command.REPOSITORY_COMMAND,
		CMD:  refreshCommand,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resultData, err := tenantNode.Write(ctx, fsm_cmd)
	if err != nil {
		return err
	}

	buf := bytes.NewBuffer(resultData.Data)
	dec := gob.NewDecoder(buf)
	parsedResult := &commands.CommandResult{}
	if err := dec.Decode(parsedResult); err != nil {
		return err
	}

	if parsedResult.Error != "" {
		return fmt.Errorf("command error: %s", parsedResult.Error)
	}

	return nil
}
