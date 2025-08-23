package queue

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"deadalus-orch/shared/models"
	"encoding/gob"
	"time"
)

func init() {
	gob.Register(PaginateQueuesCommand{})
	gob.Register(db.FindResult[models.Queue]{})
}

type PaginateQueuesCommand struct {
	Query      string
	Cursor     string
	PageSize   int
	VNamespace string
	CF         string
	CFS        string
}

func (cmd *PaginateQueuesCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	queueRepo, err := db.NewQueueRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	findResult, err := queueRepo.Paginate(cmd.Query, cmd.PageSize, cmd.Cursor, cmd.VNamespace, now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	commandResult.Result = *findResult
	return *commandResult
}
