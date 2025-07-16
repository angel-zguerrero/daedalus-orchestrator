package general_command

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"encoding/gob"
	"time"
)

func init() {
	gob.Register(CreateColumnFamilyCommand{})
}

// CreateTenantInMasterCommand represents a command to authenticate a user.
type CreateColumnFamilyCommand struct {
	Name  string
	IsTTL bool
}

func (cmd *CreateColumnFamilyCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}
	kvStore := uow.KVStore

	exists, _, err := kvStore.ExistsColumnFamily(cmd.Name)

	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	if exists {
		commandResult.Result = true
		return *commandResult
	}

	err = kvStore.CreateColumnFamily(cmd.Name, cmd.IsTTL)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}
	commandResult.Result = true
	return *commandResult
}
