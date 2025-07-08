package command

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/pkg/config"
	"deadalus-orch/server/internal/pkg/utils"
	"deadalus-orch/shared/models"
	"encoding/gob"
	"time"
)

func init() {
	gob.Register(CreateTenantInMasterCommand{})
	gob.Register(models.TenantInMaster{})
}

// CreateTenantInMasterCommand represents a command to authenticate a user.
type CreateTenantInMasterCommand struct {
	TenantId   string
	TenantCode string
}

func (cmd *CreateTenantInMasterCommand) Execute(uow *db.UnitOfWork, now time.Time) CommandResult {
	commandResult := &CommandResult{}
	kvStore := uow.KVStore
	lastShardIdBytes, err := kvStore.Get(db.AdminFC, "last-shard-id", now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}
	if lastShardIdBytes == nil {
		lastShardIdBytes, err = utils.IntToBytes(0)
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
	lastShardId++
	if lastShardId > config.GlobalConfiguration.MaxTenants {
		lastShardId = 0
	}

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
	idFactory := &db.DeterministicIDGeneratorFactory{}
	tenantInMasterRepo, err := db.NewTenantInMasterRepository(uow, idFactory) // Passing nil for IDGeneratorFactory
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	tenantInMasterFound, err := tenantInMasterRepo.GetTenantInMasterByTenantCode(cmd.TenantCode)

	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	if tenantInMasterFound != nil {
		commandResult.Result = tenantInMasterFound
		return *commandResult
	}

	tenantInMaster := models.TenantInMaster{
		ID:      cmd.TenantId,
		Code:    cmd.TenantCode,
		ShardId: lastShardId,
	}
	_, err = tenantInMasterRepo.CreateTenantInMaster(tenantInMaster, now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	commandResult.Result = tenantInMaster

	return *commandResult
}
