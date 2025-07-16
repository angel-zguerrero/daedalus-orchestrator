package app

import (
	"bytes"
	"context"
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	"deadalus-orch/server/internal/pkg/config"
	"deadalus-orch/server/internal/pkg/utils"
	commands "deadalus-orch/server/internal/usecase/command"
	general_command "deadalus-orch/server/internal/usecase/command/general"
	tenant_command "deadalus-orch/server/internal/usecase/command/tentant"
	"deadalus-orch/shared/models"
	"encoding/gob"
	"time"

	"github.com/rs/zerolog/log"
)

func (app *Application) StartAssignTenants() {
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

		ctx, cancel := context.WithTimeout(context.Background(), config.GlobalConfiguration.ApiRaftTimeout)
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
		writeCtx, writeCancel := context.WithTimeout(context.Background(), config.GlobalConfiguration.ApiRaftTimeout)
		defer writeCancel()

		var assignableTenantCodes []string

		for _, tenant := range tenantsResult.Entities {
			var tenantNode *dragonboat.RaftNode
			for i := range app.TenantNodes {
				if app.TenantNodes[i].ShardID == uint64(tenant.ShardId) {
					tenantNode = app.TenantNodes[i]

					if tenant.Status == models.PendingForAssign {

						createColumnFamilyCommand := &general_command.CreateColumnFamilyCommand{
							Name:  "cf-n-" + tenant.ID,
							IsTTL: false,
						}

						ccfCmd := general_command.FSM_Command{
							Now:  utils.GetNowInInt(),
							Type: general_command.REPOSITORY_COMMAND,
							CMD:  createColumnFamilyCommand,
						}

						result, err = tenantNode.Write(writeCtx, ccfCmd)
						if err != nil {
							log.Fatal().Err(err).Str("Code", tenant.Code).Msg("Failed to create column family for tenant")
						}

						createColumnFamilyCommandTtl := &general_command.CreateColumnFamilyCommand{
							Name:  "cf-ttl-" + tenant.ID,
							IsTTL: false,
						}

						ccfCmdTtl := general_command.FSM_Command{
							Now:  utils.GetNowInInt(),
							Type: general_command.REPOSITORY_COMMAND,
							CMD:  createColumnFamilyCommandTtl,
						}

						result, err = tenantNode.Write(writeCtx, ccfCmdTtl)
						if err != nil {
							log.Fatal().Err(err).Str("Code", tenant.Code).Msg("Failed to create column family ttl for tenant")
						}

						assignableTenantCodes = append(assignableTenantCodes, tenant.Code)
					}

					if tenant.Status == models.PendingForDeletion {
						deleteColumnFamilyCommand := &general_command.DeleteColumnFamilyCommand{
							Name: tenant.ID,
						}

						ccfCmd := general_command.FSM_Command{
							Now:  utils.GetNowInInt(),
							Type: general_command.REPOSITORY_COMMAND,
							CMD:  deleteColumnFamilyCommand,
						}

						result, err = tenantNode.Write(writeCtx, ccfCmd)
						if err != nil {
							log.Fatal().Err(err).Str("Code", tenant.Code).Msg("Failed to delete column family")
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
