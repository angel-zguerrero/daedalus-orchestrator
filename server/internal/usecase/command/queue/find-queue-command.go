package queue

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"deadalus-orch/shared/models"
	"encoding/gob"
	"fmt"
	"time"
)

func init() {
	gob.Register(FindQueueCommand{})
	gob.Register(models.RoutingHeader{})
}

type FindQueueCommand struct {
	Code           string
	VNamespace     string
	IncludeHeaders bool
	CF             string
	CFS            string
}

func (cmd *FindQueueCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	fmt.Println("Executing FindQueueCommand for Code:", cmd.Code, "VNamespace:", cmd.VNamespace, "CF:", cmd.CF, "CFS:", cmd.CFS)
	queueRepo, err := db.NewQueueRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	queue, err := queueRepo.GetQueueByCode(cmd.Code, cmd.VNamespace, now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	if queue == nil {

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
