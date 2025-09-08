package queue

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"encoding/gob"
	"fmt"
	"time"
)

func init() {
	gob.Register(DeleteQueueCommand{})
}

type DeleteQueueCommand struct {
	Code       string
	VNamespace string
	CF         string
	CFS        string
}

func (cmd *DeleteQueueCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	queueRepo, err := db.NewQueueRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	tenantSummaryRepo, err := db.NewTenantSummaryRepository(uow, idFactory)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	routingHeadersRepo, err := db.NewRoutingHeadersRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	fmt.Println("Attempting to delete queue with code:", cmd.Code, "in vNamespace:", cmd.VNamespace)
	// First find the queue by code
	queue, err := queueRepo.GetQueueByCode(cmd.Code, cmd.VNamespace, now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	if queue == nil {
		commandResult.Error = "queue not found"
		return *commandResult
	}

	// Delete all routing headers associated with this queue
	headersResult, err := routingHeadersRepo.GetRoutingHeadersByQueue(queue.ID, now)
	if err != nil {
		commandResult.Error = "error retrieving queue headers: " + err.Error()
		return *commandResult
	}

	if headersResult != nil && len(headersResult.Entities) > 0 {
		for _, header := range headersResult.Entities {
			_, err := routingHeadersRepo.DeleteRoutingHeader(header.ID, now)
			if err != nil {
				commandResult.Error = "error deleting queue header: " + err.Error()
				return *commandResult
			}
		}
	}

	// Now delete the queue by ID
	deleted, err := queueRepo.DeleteQueueById(queue.ID, now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	if !deleted {
		commandResult.Error = "queue not found or could not be deleted"
		return *commandResult
	}

	err = tenantSummaryRepo.DecreaseQueueCount(cmd.CFS, 1, now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	commandResult.Result = true
	return *commandResult
}
