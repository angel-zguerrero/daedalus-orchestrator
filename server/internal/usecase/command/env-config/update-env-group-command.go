package env_config

import (
	"encoding/gob"
	"fmt"
	"time"

	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"deadalus-orch/shared/models"
)

func init() {
	gob.Register(UpdateEnvGroupCommand{})
}

type UpdateEnvGroupCommand struct {
	EnvGroup models.EnvGroup
	CF       string
	CFS      string
}

func (cmd *UpdateEnvGroupCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	if cmd.EnvGroup.ID == "" {
		commandResult.Error = "ID is required"
		return *commandResult
	}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	repo, err := db.NewEnvGroupRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	existing, err := repo.GetEnvGroupByID(cmd.EnvGroup.ID, now)
	if err != nil || existing == nil {
		commandResult.Error = "env group not found"
		return *commandResult
	}

	existing.Name = cmd.EnvGroup.Name
	existing.Description = cmd.EnvGroup.Description

	ok, err := repo.UpdateEnvGroup(existing, now)
	if err != nil || !ok {
		commandResult.Error = fmt.Sprintf("failed to update env group: %v", err)
		return *commandResult
	}

	commandResult.Result = *existing
	return *commandResult
}
