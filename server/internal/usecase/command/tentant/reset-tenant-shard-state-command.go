package tenant_command

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"encoding/gob"
	"time"
)

func init() {
	gob.Register(ResetTenantShardStateCommand{})
}

type ResetTenantShardStateCommand struct {
	TenantID string
	CF       string
	CFS      string
}

func (cmd *ResetTenantShardStateCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	stateRepo, err := db.NewTenantShardStateRepository(uow, idFactory, cmd.CF, cmd.CFS, "tenant_schema")
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	state, err := stateRepo.GetByTenantID(cmd.TenantID, now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	if state == nil {
		// Nothing to reset if it doesn't exist.
		commandResult.Result = true
		return *commandResult
	}

	if !state.HasMessages {
		commandResult.Result = false // Already inactive
		return *commandResult
	}

	state.HasMessages = false
	_, err = stateRepo.CreateOrUpdate(state, now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	commandResult.Result = true
	return *commandResult
}
