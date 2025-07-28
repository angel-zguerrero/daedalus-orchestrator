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

type CreateTenantInMasterCommand struct {
	Tenants []models.TenantInMaster
}

func (cmd *CreateTenantInMasterCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}
	kvStore := uow.KVStore

	lastShardIdBytes, err := kvStore.Get(db.AdminFC, db.AdminFCSector, "last-shard-id", now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}
	if lastShardIdBytes == nil {
		lastShardIdBytes, err = utils.IntToBytes(config.GlobalConfiguration.MaxShards + 1)
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

	lastCFIndexBytes, err := kvStore.Get(db.AdminFC, db.AdminFCSector, "last-cf-index", now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}
	if lastCFIndexBytes == nil {
		lastCFIndexBytes, err = utils.IntToBytes(0)
		if err != nil {
			commandResult.Error = err.Error()
			return *commandResult
		}
	}
	lastCFIndex, err := utils.BytesToInt(lastCFIndexBytes)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	shardLoopCounterBytes, err := kvStore.Get(db.AdminFC, db.AdminFCSector, "cf-shard-loop-counter", now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}
	var shardLoopCounter int
	if shardLoopCounterBytes == nil {
		shardLoopCounter = 0
	} else {
		shardLoopCounter, err = utils.BytesToInt(shardLoopCounterBytes)
		if err != nil {
			commandResult.Error = err.Error()
			return *commandResult
		}
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
		if lastShardId > config.GlobalConfiguration.MaxShards+1 {
			lastShardId = 2
		}
		tenant.ShardId = lastShardId

		tenant.ColumnFamilyIndex = lastCFIndex

		shardLoopCounter++
		if shardLoopCounter >= config.GlobalConfiguration.MaxShards {
			shardLoopCounter = 0
			lastCFIndex++
			if lastCFIndex >= config.GlobalConfiguration.MaxColumnFamilies {
				lastCFIndex = 0
			}
		}

		// Upsert
		existing, err := tenantInMasterRepo.GetTenantInMasterByTenantCode(tenant.Code, now)
		if err != nil {
			commandResult.Error = err.Error()
			return *commandResult
		}

		if existing != nil {
			tenant.ID = existing.ID
			tenant.CreatedAt = existing.CreatedAt
			tenant.ShardId = existing.ShardId
			tenant.ColumnFamilyIndex = existing.ColumnFamilyIndex
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

	nextShardIdInBytes, err := utils.IntToBytes(lastShardId)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}
	err = kvStore.Put(db.AdminFC, db.AdminFCSector, "last-shard-id", nextShardIdInBytes, 0, now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	nextCFIndexBytes, err := utils.IntToBytes(lastCFIndex)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}
	err = kvStore.Put(db.AdminFC, db.AdminFCSector, "last-cf-index", nextCFIndexBytes, 0, now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	nextLoopCounterBytes, err := utils.IntToBytes(shardLoopCounter)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}
	err = kvStore.Put(db.AdminFC, db.AdminFCSector, "cf-shard-loop-counter", nextLoopCounterBytes, 0, now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	commandResult.Result = resultTenants
	return *commandResult
}
