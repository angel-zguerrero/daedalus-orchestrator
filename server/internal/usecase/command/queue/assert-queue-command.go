package queue_command

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"deadalus-orch/shared/models"
	"encoding/gob"
	"time"
)

func init() {
	gob.Register(AssertQueueCommand{})
	gob.Register(models.Queue{})
	gob.Register([]models.Queue{})
}

type AssertQueueCommand struct {
	Queues []models.Queue
	CF     string
	CFS    string
}

func (cmd *AssertQueueCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	queueRepo, err := db.NewQueueRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}
	vNamespaceRepo, err := db.NewVNamespaceRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	var resultQueues []models.Queue

	for _, queue := range cmd.Queues {

		// Validate that code is not empty
		if queue.Code == "" {
			commandResult.Error = "Queue code is required"
			return *commandResult
		}

		// Validate that VNamespace is not empty
		if queue.VNamespace == "" {
			queue.VNamespace = "default"
		}

		// Look for existing queue by code (primary upsert strategy)
		existing, err := queueRepo.GetQueueByCode(queue.Code, now)
		if err != nil {
			commandResult.Error = err.Error()
			return *commandResult
		}

		if existing != nil {
			// Update: preserve the existing code and other immutable fields
			queue.ID = existing.ID
			queue.Code = existing.Code // Frontend cannot edit code
			queue.Type = existing.Type
			queue.VNamespace = existing.VNamespace
			queue.CreatedAt = existing.CreatedAt

			_, err = queueRepo.UpdateQueue(&queue, now)
		} else {
			// For new queues, generate ID first if empty
			if queue.ID == "" {
				queue.ID = idFactory.GenerateID()
			}

			// Upsert VNamespace if it exists (now that we have an ID)
			if queue.VNamespace != "" {
				existingVNamespace, err := vNamespaceRepo.GetVNamespaceByName(queue.VNamespace, now)
				if err != nil {
					commandResult.Error = err.Error()
					return *commandResult
				}

				if existingVNamespace == nil {
					// Create new VNamespace
					vNamespace := models.VNamespace{
						ID:   queue.ID, // Use Queue ID as VNamespace ID
						Name: queue.VNamespace,
					}
					_, err = vNamespaceRepo.CreateVNamespace(&vNamespace, now)
					if err != nil {
						commandResult.Error = err.Error()
						return *commandResult
					}
				}
			}

			_, err = queueRepo.CreateQueue(&queue, now)
		}

		if err != nil {
			commandResult.Error = err.Error()
			return *commandResult
		}
		resultQueues = append(resultQueues, queue)
	}

	commandResult.Result = resultQueues
	return *commandResult
}
