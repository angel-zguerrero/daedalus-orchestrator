package command

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"encoding/gob"
	"time"
)

func init() {
	gob.Register(CreateColumnFamilyCommand{})
}

// CreateTenantInMasterCommand represents a command to authenticate a user.
type CreateColumnFamilyCommand struct {
	Name string
}

func (cmd *CreateTenantInMasterCommand) Execute(uow *db.UnitOfWork, now time.Time) CommandResult {
	commandResult := &CommandResult{}
	kvStore := uow.KVStore
	lastShardIdBytes, err := kvStore.Get(db.AdminFC, "last-shard-id", now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	return *commandResult
}
