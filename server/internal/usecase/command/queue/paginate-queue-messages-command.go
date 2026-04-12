package queue

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"deadalus-orch/shared/models"
	"encoding/gob"
	"time"
)

func init() {
	gob.Register(PaginateQueueMessagesCommand{})
	gob.Register(db.FindResult[models.QueueMessage]{})
}

type PaginateQueueMessagesCommand struct {
	QueueID  string
	Cursor   string
	PageSize int
	CF       string
	CFS      string
}

func (cmd *PaginateQueueMessagesCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	queueMessageRepo, err := db.NewQueueMessageRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	findResult, err := queueMessageRepo.PaginateByQueueID(cmd.QueueID, cmd.PageSize, cmd.Cursor, now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	if findResult.Entities == nil {
		findResult.Entities = []models.QueueMessage{}
	}

	commandResult.Result = *findResult
	return *commandResult
}
