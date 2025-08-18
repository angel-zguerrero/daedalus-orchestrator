package node_scheduler

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"encoding/gob"
	"time"
)

func init() {
	gob.Register(DeleteNodeSchedulerCommand{})
}

// DeleteNodeSchedulerCommand represents a command to authenticate a user.
type DeleteNodeSchedulerCommand struct {
	NodeSchedulerId string
}

func (cmd *DeleteNodeSchedulerCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	tenantInMasterRepo, err := db.NewNodeSchedulerRepository(uow, idFactory) // Passing nil for IDGeneratorFactory
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	_, err = tenantInMasterRepo.DeleteNodeSchedulerById(cmd.NodeSchedulerId, now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	commandResult.Result = true

	return *commandResult
}
