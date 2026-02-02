package node_scheduler

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"deadalus-orch/shared/models"
	"encoding/gob"
	"time"
)

func init() {
	gob.Register(UpsertNodeSchedulerBalancingStateCommand{})
}

type UpsertNodeSchedulerBalancingStateCommand struct {
	State models.NodeSchedulerBalancingState
}

func (cmd *UpsertNodeSchedulerBalancingStateCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	repo, err := db.NewNodeSchedulerBalancingRepository(uow, idFactory)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	_, err = repo.UpsertState(&cmd.State, now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	commandResult.Result = true
	return *commandResult
}
