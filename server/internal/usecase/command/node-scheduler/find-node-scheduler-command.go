package node_scheduler

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"encoding/gob"
	"time"
)

func init() {
	gob.Register(FindNodeSchedulerCommand{})
}

// FindNodeSchedulerCommand represents a command to authenticate a user.
type FindNodeSchedulerCommand struct {
	NodeSchedulerID   string
	NodeSchedulerName string
}

func (cmd *FindNodeSchedulerCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	nodeSchedulerRepo, err := db.NewNodeSchedulerRepository(uow, idFactory) // Passing nil for IDGeneratorFactory
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	if cmd.NodeSchedulerID != "" {
		nodeSchedulerFound, err := nodeSchedulerRepo.GetNodeSchedulerById(cmd.NodeSchedulerID, now)

		if err != nil {
			commandResult.Error = err.Error()
			return *commandResult
		}
		commandResult.Result = nodeSchedulerFound

		return *commandResult
	} else {
		nodeSchedulerFound, err := nodeSchedulerRepo.GetNodeSchedulerByName(cmd.NodeSchedulerName, now)

		if err != nil {
			commandResult.Error = err.Error()
			return *commandResult
		}
		commandResult.Result = nodeSchedulerFound

		return *commandResult
	}

}
