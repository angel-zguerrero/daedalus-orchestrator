package queue

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"encoding/gob"
	"time"
)

func init() {
	gob.Register(FindQueueByIDCommand{})
}

type FindQueueByIDCommand struct {
	ID             string
	VNamespace     string
	IncludeHeaders bool
	CF             string
	CFS            string
}

func (cmd *FindQueueByIDCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	idFactory := &db.DeterministicIDGeneratorFactory{}
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

	// If headers are requested, populate them using RoutingHeader
	if cmd.IncludeHeaders {
		routingHeadersRepo, err := db.NewRoutingHeadersRepository(uow, idFactory, cmd.CF, cmd.CFS)
		if err != nil {
			commandResult.Error = err.Error()
			return *commandResult
		}

		// Get headers for this queue using QueueID
		if headersResult, err := routingHeadersRepo.GetRoutingHeadersByQueue(queue.ID, now); err == nil && headersResult != nil {
			headers := make(map[string]string)
			for _, header := range headersResult.Entities {
				headers[header.Key] = header.Value
			}
			queue.Headers = headers
		}
	}

	commandResult.Result = *queue
	return *commandResult
}
