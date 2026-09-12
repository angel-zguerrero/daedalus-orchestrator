package env_config

import (
	"encoding/gob"
	"fmt"
	"time"

	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
)

func init() {
	gob.Register(DeleteEnvVarCommand{})
}

type DeleteEnvVarCommand struct {
	VarID string
	CF    string
	CFS   string
}

func (cmd *DeleteEnvVarCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	if cmd.VarID == "" {
		commandResult.Error = "VarID is required"
		return *commandResult
	}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	repo, err := db.NewEnvVarRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	ok, err := repo.DeleteEnvVar(cmd.VarID, now)
	if err != nil || !ok {
		commandResult.Error = fmt.Sprintf("failed to delete env var: %v", err)
		return *commandResult
	}

	commandResult.Result = true
	return *commandResult
}
