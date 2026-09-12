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
	gob.Register(CreateEnvGroupCommand{})
}

type CreateEnvGroupCommand struct {
	EnvGroup models.EnvGroup
	CF       string
	CFS      string
}

func (cmd *CreateEnvGroupCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	if cmd.EnvGroup.ID == "" {
		commandResult.Error = "ID is required and must be generated outside the command"
		return *commandResult
	}

	if cmd.EnvGroup.Code == "" {
		commandResult.Error = "Code is required"
		return *commandResult
	}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	repo, err := db.NewEnvGroupRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	id, err := repo.CreateEnvGroup(&cmd.EnvGroup, now)
	if err != nil {
		commandResult.Error = fmt.Sprintf("failed to create env group: %s", err.Error())
		return *commandResult
	}

	cmd.EnvGroup.ID = id
	commandResult.Result = cmd.EnvGroup
	return *commandResult
}
