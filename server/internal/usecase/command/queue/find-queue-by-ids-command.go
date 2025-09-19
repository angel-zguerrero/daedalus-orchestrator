package queue

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"deadalus-orch/shared/models"
	"encoding/gob"
	"time"
)

func init() {
	gob.Register(FindQueueByIDsCommand{})
}

type FindQueueByIDsCommand struct {
	IDs            []string
	VNamespace     string
	IncludeHeaders bool
	CF             string
	CFS            string
}

func (cmd *FindQueueByIDsCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	queueRepo, err := db.NewQueueRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	var queues []models.Queue
	var routingHeadersRepo *db.RoutingHeadersRepository

	// Initialize routing headers repository if headers are requested
	if cmd.IncludeHeaders {
		routingHeadersRepo, err = db.NewRoutingHeadersRepository(uow, idFactory, cmd.CF, cmd.CFS)
		if err != nil {
			commandResult.Error = err.Error()
			return *commandResult
		}
	}

	// Iterate over each ID and get the queue
	for _, id := range cmd.IDs {
		queue, err := queueRepo.GetQueueById(id, now)
		if err != nil {
			// Skip queues that can't be found or have errors, continue with the rest
			continue
		}

		if queue == nil {
			// Skip nil queues, continue with the rest
			continue
		}

		// If headers are requested, populate them using RoutingHeader
		if cmd.IncludeHeaders && routingHeadersRepo != nil {
			if headersResult, err := routingHeadersRepo.GetRoutingHeadersByQueue(queue.ID, now); err == nil && headersResult != nil {
				headers := make(map[string]string)
				for _, header := range headersResult.Entities {
					headers[header.Key] = header.Value
				}
				queue.Headers = headers
			}
		}

		queues = append(queues, *queue)
	}

	commandResult.Result = queues
	return *commandResult
}
