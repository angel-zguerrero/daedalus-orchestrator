package queue_command

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"encoding/gob"
	"fmt"
	"time"
)

func init() {
	gob.Register(FindQueueCommand{})
}

type FindQueueCommand struct {
	ID  string
	CF  string
	CFS string
}

func (cmd *FindQueueCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	fmt.Println("Executing FindQueueCommand for ID:", cmd.ID, "CF:", cmd.CF, "CFS:", cmd.CFS)
	queueRepo, err := db.NewQueueRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	queue, err := queueRepo.GetQueueById(cmd.ID, now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	if queue == nil {
		commandResult.Error = "queue not found"
		return *commandResult
	}

	commandResult.Result = *queue
	return *commandResult
}
