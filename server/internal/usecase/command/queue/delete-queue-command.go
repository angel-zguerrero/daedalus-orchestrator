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

	bindingRepo, err := db.NewBindingRepository(uow, idFactory, cmd.CF, cmd.CFS)
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

	// Find and delete all bindings associated with this queue (with pagination)
	bindingCount := 0
	cursor := ""

	for {
		bindingsResult, err := bindingRepo.Find("QueueID = "+queue.ID, 100, cursor, now)
		if err != nil {
			commandResult.Error = "error retrieving queue bindings: " + err.Error()
			return *commandResult
		}

		if bindingsResult == nil || len(bindingsResult.Entities) == 0 {
			break
		}

		for _, binding := range bindingsResult.Entities {
			// Delete all routing headers associated with this binding
			headersResult, err := routingHeadersRepo.GetRoutingHeadersByBinding(binding.ID, now)
			if err != nil {
				commandResult.Error = "error retrieving binding headers: " + err.Error()
				return *commandResult
			}

			if headersResult != nil && len(headersResult.Entities) > 0 {
				for _, header := range headersResult.Entities {
					_, err := routingHeadersRepo.DeleteRoutingHeader(header.ID, now)
					if err != nil {
						commandResult.Error = "error deleting binding header: " + err.Error()
						return *commandResult
					}
				}
			}

			// Delete the binding
			_, err = bindingRepo.DeleteBinding(binding.ID, now)
			if err != nil {
				commandResult.Error = "error deleting binding: " + err.Error()
				return *commandResult
			}
			bindingCount++
		}

		// Update cursor for next page
		cursor = bindingsResult.Cursor
		if cursor == "" {
			break
		}
	}

	// Delete all messages in the queue before deleting the queue itself
	queueMessageRepo, err := db.NewQueueMessageRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	// Find and delete all messages in this queue (with pagination)
	messageCount := 0
	messageCursor := ""

	for {
		messagesResult, err := queueMessageRepo.Find("QueueID = "+queue.ID, 100, messageCursor, now)
		if err != nil {
			commandResult.Error = "error retrieving queue messages: " + err.Error()
			return *commandResult
		}

		if messagesResult == nil || len(messagesResult.Entities) == 0 {
			break
		}

		for _, message := range messagesResult.Entities {
			// Delete the message
			_, err = queueMessageRepo.Delete(message.ID, now)
			if err != nil {
				commandResult.Error = "error deleting queue message: " + err.Error()
				return *commandResult
			}
			messageCount++
		}

		// Update cursor for next page
		messageCursor = messagesResult.Cursor
		if messageCursor == "" {
			break
		}
	}

	// Delete all routing headers associated directly with this queue
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

	// Update tenant summary with a single operation
	// Decrease queue count by 1, binding count by bindingCount, and message count by messageCount
	err = tenantSummaryRepo.UpdateCounters(cmd.CFS, -messageCount, 0, -1, -bindingCount, now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	commandResult.Result = true
	return *commandResult
}
