package tenant_command

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/pkg/config"
	"deadalus-orch/server/internal/pkg/utils"
	"deadalus-orch/server/internal/usecase/command"
	"deadalus-orch/shared/models"
	"encoding/gob"
	"time"
)

func init() {
	gob.Register(CreateTenantInMasterCommand{})
	gob.Register(models.TenantInMaster{})
	gob.Register([]models.TenantInMaster{})
}

// CreateTenantInMasterCommand represents a command to authenticate a user.
type CreateTenantInMasterCommand struct {
	Tenants []models.TenantInMaster
}

func (cmd *CreateTenantInMasterCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}
	kvStore := uow.KVStore

	// Obtener último shard ID
	lastShardIdBytes, err := kvStore.Get(db.AdminFC, "last-shard-id", now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}
	if lastShardIdBytes == nil {
		lastShardIdBytes, err = utils.IntToBytes(config.GlobalConfiguration.MaxTenants + 1)
		if err != nil {
			commandResult.Error = err.Error()
			return *commandResult
		}
	}
	lastShardId, err := utils.BytesToInt(lastShardIdBytes)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	tenantInMasterRepo, err := db.NewTenantInMasterRepository(uow, idFactory)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	var resultTenants []models.TenantInMaster

	for _, tenant := range cmd.Tenants {
		lastShardId++
		if lastShardId > config.GlobalConfiguration.MaxTenants+1 {
			lastShardId = 2
		}
		tenant.ShardId = lastShardId

		existing, err := tenantInMasterRepo.GetTenantInMasterByTenantCode(tenant.Code, now)
		if err != nil {
			commandResult.Error = err.Error()
			return *commandResult
		}

		if existing != nil {
			tenant.ID = existing.ID
			tenant.CreatedAt = existing.CreatedAt
			tenant.ShardId = existing.ShardId
			_, err = tenantInMasterRepo.UpdateTenantInMaster(&tenant, now)
		} else {
			_, err = tenantInMasterRepo.CreateTenantInMaster(&tenant, now)
		}

		if err != nil {
			commandResult.Error = err.Error()
			return *commandResult
		}
		resultTenants = append(resultTenants, tenant)
	}

	// Guardar el último shard ID actualizado
	nextShardIdInBytes, err := utils.IntToBytes(lastShardId)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}
	err = kvStore.Put(db.AdminFC, "last-shard-id", nextShardIdInBytes, 0, now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	commandResult.Result = resultTenants
	return *commandResult
}
