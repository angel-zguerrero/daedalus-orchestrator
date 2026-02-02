package node_scheduler

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"encoding/gob"
	"time"
)

func init() {
	gob.Register(GetNodeSchedulerBalancingStateCommand{})
}

type GetNodeSchedulerBalancingStateCommand struct {
}

func (cmd *GetNodeSchedulerBalancingStateCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	repo, err := db.NewNodeSchedulerBalancingRepository(uow, idFactory)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	state, err := repo.GetState(now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	if state == nil {
		commandResult.Result = nil
	} else {
		commandResult.Result = *state
	}

	return *commandResult
}
