package queue

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"deadalus-orch/shared/models"
	"encoding/gob"
	"time"
)

func init() {
	gob.Register(MarkQueuesAsDrainCommand{})
}

type MarkQueuesAsDrainCommand struct {
	QueueIDs []string
	CF       string
	CFS      string
}

func (cmd *MarkQueuesAsDrainCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	if len(cmd.QueueIDs) == 0 {
		commandResult.Result = true
		return *commandResult
	}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	queueRepo, err := db.NewQueueRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	updatedCount := 0
	var updatedQueues []models.Queue

	for _, queueID := range cmd.QueueIDs {
		// Get the queue by ID
		queue, err := queueRepo.GetQueueById(queueID, now)
		if err != nil {
			commandResult.Error = "error retrieving queue with ID " + queueID + ": " + err.Error()
			return *commandResult
		}

		if queue == nil {
			// Queue not found, skip it
			continue
		}

		// Update the queue state to draining
		queue.State = models.QueueDraining
		queue.UpdatedAt = now

		// Update the queue in the repository
		updated, err := queueRepo.UpdateQueue(queue, now)
		if err != nil {
			commandResult.Error = "error updating queue state for ID " + queueID + ": " + err.Error()
			return *commandResult
		}

		if updated {
			updatedQueues = append(updatedQueues, *queue)
			updatedCount++
		}
	}

	commandResult.Result = updatedQueues
	return *commandResult
}
