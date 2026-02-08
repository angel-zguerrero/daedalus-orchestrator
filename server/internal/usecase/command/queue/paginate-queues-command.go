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
	gob.Register(models.RoutingHeader{})
}

type PaginateQueuesCommand struct {
	Query            string
	Cursor           string
	PageSize         int
	VNamespace       string
	IncludeHeaders   bool
	CF               string
	CFS              string
	SupervisionState models.QueueSupervisionState
}

func (cmd *PaginateQueuesCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	queueRepo, err := db.NewQueueRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	var findResult *db.FindResult[models.Queue]
	if cmd.SupervisionState != "" {
		findResult, err = queueRepo.PaginateBySupervisionState(cmd.Query, cmd.SupervisionState, cmd.PageSize, cmd.Cursor, cmd.VNamespace, now)
		if err != nil {
			commandResult.Error = err.Error()
			return *commandResult
		}
	} else {
		findResult, err = queueRepo.Paginate(cmd.Query, cmd.PageSize, cmd.Cursor, cmd.VNamespace, now)
		if err != nil {
			commandResult.Error = err.Error()
			return *commandResult
		}
	}

	// If headers are requested, populate them using RoutingHeader
	if cmd.IncludeHeaders && findResult.Entities != nil {
		routingHeadersRepo, err := db.NewRoutingHeadersRepository(uow, idFactory, cmd.CF, cmd.CFS)
		if err != nil {
			commandResult.Error = err.Error()
			return *commandResult
		}

		// Populate headers for each queue
		for i := range findResult.Entities {
			queue := &findResult.Entities[i]

			// Get headers for this queue using QueueID
			if headersResult, err := routingHeadersRepo.GetRoutingHeadersByQueue(queue.ID, now); err == nil && headersResult != nil {
				headers := make(map[string]string)
				for _, header := range headersResult.Entities {
					headers[header.Key] = header.Value
				}
				queue.Headers = headers
			}
		}
	}

	commandResult.Result = *findResult
	return *commandResult
}
