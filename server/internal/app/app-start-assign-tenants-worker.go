package app

import (
	"bytes"
	"context"
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	"deadalus-orch/server/internal/pkg/utils"
	commands "deadalus-orch/server/internal/usecase/command"
	general_command "deadalus-orch/server/internal/usecase/command/general"
	tenant_command "deadalus-orch/server/internal/usecase/command/tentant"
	"deadalus-orch/shared/models"
	"encoding/gob"
	"fmt"
	"strconv"
	"time"

	"github.com/rs/zerolog/log"
)

func (app *Application) StartAssignTenantsWorker(interval time.Duration) {
	app.AssignTenantsStopper.RunWorker(func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if !app.MasterNodeIsReady {
					log.Warn().Msg("⏳ Assign tenants worker is waiting for the master node to be ready")
					continue
				}

				if !app.MasterNodeIsLeader {
					log.Warn().Msg("⏳ Only leader can assign tenants")
					continue
				}

				select {
				case <-app.AssignTenantsStopper.ShouldStop():
					log.Info().Msg("🛑 Assign tenants worker received stop signal before starting")
					return
				default:
				}

				app.AssignTenants()

			case <-app.AssignTenantsStopper.ShouldStop():
				log.Info().Msg("ℹ️  Assign tenants worker stopped gracefully")
				return
			}
		}
	})
}

func (app *Application) AssignTenants() {
	cursor := ""
	pageSize := 10

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

		ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
		defer cancel()

		result, err := app.MasterNode.Read(ctx, *queryCommand)
		if err != nil {
			log.Fatal().Err(err).Msg("Paginate tenants command failed")
			return
		}

		buf := bytes.NewBuffer(result.([]byte))
		dec := gob.NewDecoder(buf)
		parsedResult := &commands.CommandResult{}
		if err := dec.Decode(parsedResult); err != nil {
			log.Fatal().Err(err).Msg("Paginate tenants command failed (decode)")
			return
		}

		if parsedResult.Error != "" {
			log.Fatal().Str("error", parsedResult.Error).Msg("Paginate tenants command failed (business error)")
			return
		}

		tenantsResult := parsedResult.Result.(db.FindResult[models.TenantInMaster])
		writeCtx, writeCancel := context.WithTimeout(context.Background(), time.Hour)
		defer writeCancel()

		var assignableTenantCodes []string

		for _, tenant := range tenantsResult.Entities {
			var tenantNode *dragonboat.RaftNode
			for i := range app.TenantNodes {
				if app.TenantNodes[i].ShardID == uint64(tenant.ShardId) {
					tenantNode = app.TenantNodes[i]

					if tenant.Status == models.PendingForAssign {
						assignableTenantCodes = append(assignableTenantCodes, tenant.Code)
					}

					if tenant.Status == models.PendingForDeletion {

						deleteColumnFamilyCommandSector := &general_command.DeleteColumnFamilySectorCommand{
							ColumnFamily:       db.ColumnFamilyPrefix + strconv.Itoa(tenant.ColumnFamilyIndex),
							ColumnFamilySector: tenant.ID,
						}

						ccfCmd := general_command.FSM_Command{
							Now:  utils.GetNowInInt(),
							Type: general_command.REPOSITORY_COMMAND,
							CMD:  deleteColumnFamilyCommandSector,
						}

						_, err = tenantNode.Write(writeCtx, ccfCmd)
						if err != nil {
							log.Fatal().Err(err).Str("Code", tenant.Code).Msg("Failed to delete tenant")
							return
						}

						// Delete from TTL column family
						deleteColumnFamilyTTLCommandSector := &general_command.DeleteColumnFamilySectorCommand{
							ColumnFamily:       db.ColumnFamilyTTLPrefix + strconv.Itoa(tenant.ColumnFamilyIndex),
							ColumnFamilySector: tenant.ID,
						}

						ccfTTLCmd := general_command.FSM_Command{
							Now:  utils.GetNowInInt(),
							Type: general_command.REPOSITORY_COMMAND,
							CMD:  deleteColumnFamilyTTLCommandSector,
						}

						_, err = tenantNode.Write(writeCtx, ccfTTLCmd)
						if err != nil {
							log.Fatal().Err(err).Str("Code", tenant.Code).Msg("Failed to delete tenant")
							return
						}

						deleteTenantInMasterCommand := &tenant_command.DeleteTenantInMasterCommand{
							TenantId: tenant.ID,
						}

						atstCmd := general_command.FSM_Command{
							Now:  utils.GetNowInInt(),
							Type: general_command.REPOSITORY_COMMAND,
							CMD:  deleteTenantInMasterCommand,
						}

						result, err = app.MasterNode.Write(writeCtx, atstCmd)
						if err != nil {
							log.Fatal().Err(err).Str("Code", tenant.Code).Msg("Failed to delete tenant")
						}
						fmt.Printf("Tenant %s deleted successfully\n", tenant.Code)
					}

					break
				}
			}
			app.TenantNodesDictionary[tenant.ID] = tenantNode
		}

		if len(assignableTenantCodes) > 0 {
			assignCmd := &tenant_command.AssignToShardTenantInMasterCommand{
				TenantCodes: assignableTenantCodes,
			}

			atstCmd := general_command.FSM_Command{
				Now:  utils.GetNowInInt(),
				Type: general_command.REPOSITORY_COMMAND,
				CMD:  assignCmd,
			}

			result, err = app.MasterNode.Write(writeCtx, atstCmd)
			if err != nil {
				log.Fatal().Err(err).Strs("Codes", assignableTenantCodes).Msg("Failed to assign tenants to shard")
			}

			buf = bytes.NewBuffer(result.([]byte))
			dec = gob.NewDecoder(buf)
			if err := dec.Decode(parsedResult); err != nil || parsedResult.Error != "" {
				log.Fatal().
					Strs("Codes", assignableTenantCodes).
					Err(err).
					Str("commandError", parsedResult.Error).
					Msg("Shard assignment failed for one or more tenants")
			}
		}

		if tenantsResult.Cursor == "" {
			break
		}

		cursor = tenantsResult.Cursor
	}
}
