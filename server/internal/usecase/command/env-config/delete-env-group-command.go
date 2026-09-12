package env_config

import (
	"encoding/gob"
	"fmt"
	"time"

	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
)

func init() {
	gob.Register(DeleteEnvGroupCommand{})
}

type DeleteEnvGroupCommand struct {
	GroupID string
	CF      string
	CFS     string
}

func (cmd *DeleteEnvGroupCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	if cmd.GroupID == "" {
		commandResult.Error = "GroupID is required"
		return *commandResult
	}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	groupRepo, err := db.NewEnvGroupRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	varRepo, err := db.NewEnvVarRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	// Clean up variables first
	_ = varRepo.DeleteEnvVarsByGroupID(cmd.GroupID, now)

	// Delete group
	ok, err := groupRepo.DeleteEnvGroup(cmd.GroupID, now)
	if err != nil || !ok {
		commandResult.Error = fmt.Sprintf("failed to delete env group: %v", err)
		return *commandResult
	}

	commandResult.Result = true
	return *commandResult
}
