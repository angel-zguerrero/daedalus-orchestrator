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
	gob.Register(SaveEnvVarCommand{})
}

type SaveEnvVarCommand struct {
	EnvVar models.EnvVar
	CF     string
	CFS    string
}

func (cmd *SaveEnvVarCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	if cmd.EnvVar.ID == "" {
		commandResult.Error = "ID is required and must be generated outside the command"
		return *commandResult
	}

	if cmd.EnvVar.GroupID == "" {
		commandResult.Error = "GroupID is required"
		return *commandResult
	}

	if cmd.EnvVar.Key == "" {
		commandResult.Error = "Key is required"
		return *commandResult
	}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	repo, err := db.NewEnvVarRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	// Check if var already exists by ID or by Key within GroupID
	existing, _ := repo.GetEnvVarByID(cmd.EnvVar.ID, now)
	if existing == nil {
		existingByKey, _ := repo.GetEnvVarByKey(cmd.EnvVar.GroupID, cmd.EnvVar.Key, now)
		if existingByKey != nil {
			existing = existingByKey
		}
	}

	if existing != nil {
		existing.Value = cmd.EnvVar.Value
		existing.Description = cmd.EnvVar.Description
		ok, err := repo.UpdateEnvVar(existing, now)
		if err != nil || !ok {
			commandResult.Error = fmt.Sprintf("failed to update env var: %v", err)
			return *commandResult
		}
		commandResult.Result = *existing
		return *commandResult
	}

	id, err := repo.CreateEnvVar(&cmd.EnvVar, now)
	if err != nil {
		commandResult.Error = fmt.Sprintf("failed to create env var: %s", err.Error())
		return *commandResult
	}

	cmd.EnvVar.ID = id
	commandResult.Result = cmd.EnvVar
	return *commandResult
}
