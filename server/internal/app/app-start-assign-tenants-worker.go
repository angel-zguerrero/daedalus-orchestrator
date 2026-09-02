package app

import (
	"context"
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	"deadalus-orch/server/internal/pkg/utils"
	commands "deadalus-orch/server/internal/usecase/command"
	general_command "deadalus-orch/server/internal/usecase/command/general"
	tenant_command "deadalus-orch/server/internal/usecase/command/tentant"
	"deadalus-orch/shared/models"
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
			//log.Fatal().Err(err).Msg("Paginate tenants command failed")

			fmt.Println("Paginate tenants command failed", err)
			return
		}

		tenantsResult, err := commands.DecodeCommandResult[db.FindResult[models.TenantInMaster]](result.([]byte))
		if err != nil {
			fmt.Println("Paginate tenants command failed (decode)", err)
			return
		}
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

						resultChan, err := tenantNode.Write(writeCtx, ccfCmd)
						if err != nil {
							//log.Fatal().Err(err).Str("Code", tenant.Code).Msg("Failed to delete tenant")
							fmt.Println("Failed to start delete tenant operation", err)
							return
						}
						// Wait for result since this is sequential
						select {
						case writeResult := <-resultChan:
							if writeResult.Error != nil {
								//log.Fatal().Err(writeResult.Error).Str("Code", tenant.Code).Msg("Failed to delete tenant")
								fmt.Println("Failed to delete tenant", writeResult.Error)
								return
							}
						case <-writeCtx.Done():
							fmt.Println("Delete tenant operation timed out", writeCtx.Err())
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

						resultChan, err = tenantNode.Write(writeCtx, ccfTTLCmd)
						if err != nil {
							//log.Fatal().Err(err).Str("Code", tenant.Code).Msg("Failed to delete tenant")
							fmt.Println("Failed to start delete tenant TTL operation", err)
							return
						}
						// Wait for result since this is sequential
						select {
						case writeResult := <-resultChan:
							if writeResult.Error != nil {
								//log.Fatal().Err(writeResult.Error).Str("Code", tenant.Code).Msg("Failed to delete tenant")
								fmt.Println("Failed to delete tenant", writeResult.Error)
								return
							}
						case <-writeCtx.Done():
							fmt.Println("Delete tenant TTL operation timed out", writeCtx.Err())
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

						resultChan, err = app.MasterNode.Write(writeCtx, atstCmd)
						if err != nil {
							//log.Fatal().Err(err).Str("Code", tenant.Code).Msg("Failed to delete tenant")
							fmt.Println("Failed to start delete tenant from master operation", err)
							return
						}
						// Wait for result since this is sequential
						select {
						case writeResult := <-resultChan:
							if writeResult.Error != nil {
								//log.Fatal().Err(writeResult.Error).Str("Code", tenant.Code).Msg("Failed to delete tenant")
								fmt.Println("Failed to delete tenant", writeResult.Error)
								return
							}
						case <-writeCtx.Done():
							fmt.Println("Delete tenant from master operation timed out", writeCtx.Err())
							return
						}
						fmt.Printf("Tenant %s deleted successfully\n", tenant.Code)
					}

					break
				}
			}
			app.TenantNodesDictionary[tenant.ID] = tenantNode
			app.TenantNodesDictionary[tenant.Code] = tenantNode
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

			resultChan, err := app.MasterNode.Write(writeCtx, atstCmd)
			if err != nil {
				//log.Fatal().Err(err).Strs("Codes", assignableTenantCodes).Msg("Failed to assign tenants to shard")
				fmt.Println("Failed to start assign tenants to shard operation", err)
				return
			}

			// Wait for result since we need to process it
			select {
			case writeResult := <-resultChan:
				if writeResult.Error != nil {
					//log.Fatal().Err(writeResult.Error).Strs("Codes", assignableTenantCodes).Msg("Failed to assign tenants to shard")
					fmt.Println("Failed to assign tenants to shard", writeResult.Error)
					return
				}

				_, err := commands.DecodeCommandResult[string](writeResult.Result.Data)
				if err != nil {
					fmt.Println("Shard assignment failed for one or more tenants", err)
					return
				}
			case <-writeCtx.Done():
				fmt.Println("Assign tenants to shard operation timed out", writeCtx.Err())
				return
			}
		}

		if tenantsResult.Cursor == "" {
			break
		}

		cursor = tenantsResult.Cursor
	}
}
