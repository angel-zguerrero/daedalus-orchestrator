package app

import (
	"bytes"
	"context"
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	"deadalus-orch/server/internal/pkg/utils"
	"deadalus-orch/server/internal/usecase/command"
	general_command "deadalus-orch/server/internal/usecase/command/general"
	queue_command "deadalus-orch/server/internal/usecase/command/queue"
	tenant_command "deadalus-orch/server/internal/usecase/command/tentant"
	"deadalus-orch/shared/models"
	"encoding/gob"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

func (app *Application) StartNodeClearExpiredLeasesWorker(interval time.Duration, leaseBatchSize int) {
	var cleaningLock sync.Mutex

	app.NodeClearExpiredLeasesStopper.RunWorker(func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if !cleaningLock.TryLock() {
					log.Warn().Msg("⏳ Skipping expired leases cleanup: previous execution still in progress")
					continue
				}

				go func() {
					defer cleaningLock.Unlock()

					if !app.MasterNodeIsReady {
						log.Warn().Msg("⏳ Expired leases cleaner is waiting for the master node to be ready")
						return
					}

					if !app.MasterNodeIsLeader {
						log.Warn().Msg("⏳ Only leader can process expired leases")
						return
					}

					select {
					case <-app.NodeClearExpiredLeasesStopper.ShouldStop():
						log.Info().Msg("🛑 Expired leases cleaner received stop signal before starting")
						return
					default:
					}

					app.clearAllTenantsExpiredLeases(leaseBatchSize)
				}()

			case <-app.NodeClearExpiredLeasesStopper.ShouldStop():
				log.Info().Msg("ℹ️  Expired leases cleaner worker stopped gracefully")
				return
			}
		}
	})
}

func (app *Application) clearAllTenantsExpiredLeases(leaseBatchSize int) {
	// Paginate through all tenants
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
			log.Err(err).Msg("❌ Failed to paginate tenants for expired leases cleanup")
			return
		}

		buf := bytes.NewBuffer(result.([]byte))
		dec := gob.NewDecoder(buf)
		parsedResult := &command.CommandResult{}
		if err := dec.Decode(parsedResult); err != nil {
			log.Err(err).Msg("❌ Failed to decode tenant pagination result")
			return
		}

		if parsedResult.Error != "" {
			log.Error().Str("error", parsedResult.Error).Msg("❌ Tenant pagination command returned error")
			return
		}

		tenantsResult, ok := parsedResult.Result.(db.FindResult[models.TenantInMaster])
		if !ok {
			log.Warn().Str("type", fmt.Sprintf("%T", parsedResult.Result)).Msg("⚠️ Unexpected result type from tenant pagination")
			return
		}

		// Process each tenant
		for _, tenant := range tenantsResult.Entities {
			// Find the tenant node by ShardID
			var tenantNode *dragonboat.RaftNode
			for i := range app.TenantNodes {
				if app.TenantNodes[i].ShardID == uint64(tenant.ShardId) {
					tenantNode = app.TenantNodes[i]
					break
				}
			}

			if tenantNode == nil {
				log.Warn().
					Str("tenant", tenant.Code).
					Int("shardId", tenant.ShardId).
					Msg("⚠️ Tenant node not found for shard, skipping")
				continue
			}

			// Calculate CF and CFS for this tenant
			cf := db.ColumnFamilyPrefix + fmt.Sprintf("%d", tenant.ColumnFamilyIndex)
			cfs := tenant.ID

			// Check if we should stop
			select {
			case <-app.NodeClearExpiredLeasesStopper.ShouldStop():
				log.Info().Msg("🛑 Expired leases cleaner stopped during tenant processing")
				return
			default:
			}

			// Clear expired leases for this tenant
			app.clearTenantExpiredLeases(tenantNode, &tenant, cf, cfs, leaseBatchSize)
		}

		// Check if there are more tenants to process
		if tenantsResult.Cursor == "" || len(tenantsResult.Entities) < pageSize {
			break
		}
		cursor = tenantsResult.Cursor
	}
}

func (app *Application) clearTenantExpiredLeases(
	node *dragonboat.RaftNode,
	tenant *models.TenantInMaster,
	cf, cfs string,
	leaseBatchSize int,
) {
	offset := 0
	limit := leaseBatchSize

	for {
		cmd := general_command.FSM_Command{
			Now:  utils.GetNowInInt(),
			Type: general_command.REPOSITORY_COMMAND,
			CMD: queue_command.ProcessExpiredLeasesCommand{
				Limit:  limit,
				Offset: offset,
				CF:     cf,
				CFS:    cfs,
			},
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)

		resultChan, err := node.Write(ctx, cmd)
		if err != nil {
			log.Err(err).
				Str("tenant", tenant.Code).
				Str("node", strconv.FormatUint(node.ShardID, 10)).
				Msg("❌ Failed to start expired leases processing on tenant node")
			cancel()
			return
		}

		// Wait for result to determine if there are more leases to process
		var processedCount int
		shouldContinue := false

		select {
		case writeResult := <-resultChan:
			cancel()
			if writeResult.Error != nil {
				log.Err(writeResult.Error).
					Str("tenant", tenant.Code).
					Str("node", strconv.FormatUint(node.ShardID, 10)).
					Msg("❌ Failed to process expired leases on tenant node")
				return
			}

			// Decode the result
			buf := bytes.NewBuffer(writeResult.Result.Data)
			dec := gob.NewDecoder(buf)
			var commandResult command.CommandResult
			if err := dec.Decode(&commandResult); err != nil {
				log.Err(err).
					Str("tenant", tenant.Code).
					Str("node", strconv.FormatUint(node.ShardID, 10)).
					Msg("❌ Failed to decode expired leases command result")
				return
			}

			if commandResult.Error != "" {
				log.Error().
					Str("tenant", tenant.Code).
					Str("node", strconv.FormatUint(node.ShardID, 10)).
					Str("error", commandResult.Error).
					Msg("❌ Command returned error")
				return
			}

			// Try to cast the result
			if result, ok := commandResult.Result.(queue_command.ProcessExpiredLeasesResult); ok {
				processedCount = result.ProcessedLeases

				if processedCount > 0 {
					log.Info().
						Str("tenant", tenant.Code).
						Str("node", strconv.FormatUint(node.ShardID, 10)).
						Int("processed", result.ProcessedLeases).
						Int("deleted", result.DeletedMessages).
						Int("requeued", result.RequeuedMessages).
						Int("errors", len(result.Errors)).
						Msg("✅ Processed expired leases on tenant node")

					// Log errors if any
					for _, errMsg := range result.Errors {
						log.Warn().
							Str("tenant", tenant.Code).
							Str("node", strconv.FormatUint(node.ShardID, 10)).
							Str("error", errMsg).
							Msg("⚠️ Error processing expired lease")
					}
				}

				// If we processed fewer leases than the limit, we've reached the end
				if processedCount >= limit {
					shouldContinue = true
					offset += limit
				}
			} else {
				log.Warn().
					Str("tenant", tenant.Code).
					Str("node", strconv.FormatUint(node.ShardID, 10)).
					Str("type", fmt.Sprintf("%T", commandResult.Result)).
					Msg("⚠️ Unexpected result type from ProcessExpiredLeasesCommand")
				return
			}

		case <-ctx.Done():
			cancel()
			log.Warn().
				Str("tenant", tenant.Code).
				Str("node", strconv.FormatUint(node.ShardID, 10)).
				Msg("⏱️ Expired leases processing timed out on tenant node")
			return
		}

		// Check if we should stop
		select {
		case <-app.NodeClearExpiredLeasesStopper.ShouldStop():
			log.Info().Msg("🛑 Expired leases cleaner stopped during pagination")
			return
		default:
		}

		// If we didn't process any leases or shouldn't continue, we're done
		if processedCount == 0 || !shouldContinue {
			return
		}
	}
}
