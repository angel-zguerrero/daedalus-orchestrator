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
	gob.Register(BulkSaveEnvVarsCommand{})
}

type BulkSaveEnvVarsCommand struct {
	GroupID string
	EnvVars []models.EnvVar
	CF      string
	CFS     string
}

func (cmd *BulkSaveEnvVarsCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	if cmd.GroupID == "" {
		commandResult.Error = "GroupID is required"
		return *commandResult
	}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	repo, err := db.NewEnvVarRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	// Clean existing vars for this group to replace with new state
	_ = repo.DeleteEnvVarsByGroupID(cmd.GroupID, now)

	var saved []models.EnvVar
	for _, v := range cmd.EnvVars {
		if v.Key == "" {
			continue
		}
		if v.ID == "" {
			commandResult.Error = fmt.Sprintf("ID is required for key %s and must be generated outside the command", v.Key)
			return *commandResult
		}
		v.GroupID = cmd.GroupID
		v.GroupIDComp = cmd.GroupID
		id, err := repo.CreateEnvVar(&v, now)
		if err != nil {
			commandResult.Error = fmt.Sprintf("failed to save env var %s: %v", v.Key, err)
			return *commandResult
		}
		v.ID = id
		saved = append(saved, v)
	}

	commandResult.Result = saved
	return *commandResult
}
