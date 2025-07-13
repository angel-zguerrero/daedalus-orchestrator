package tenant_command

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"encoding/gob"
	"time"
)

func init() {
	gob.Register(DeleteTenantInMasterCommand{})
}

// DeleteTenantInMasterCommand represents a command to authenticate a user.
type DeleteTenantInMasterCommand struct {
	TenantId string
}

func (cmd *DeleteTenantInMasterCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	tenantInMasterRepo, err := db.NewTenantInMasterRepository(uow, idFactory) // Passing nil for IDGeneratorFactory
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	_, err = tenantInMasterRepo.DeleteTenantInMasterById(cmd.TenantId, now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	commandResult.Result = true

	return *commandResult
}
