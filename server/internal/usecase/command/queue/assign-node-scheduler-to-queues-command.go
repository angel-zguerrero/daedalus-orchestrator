package queue

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"deadalus-orch/shared/models"
	"encoding/gob"
	"time"
)

func init() {
	gob.Register(AssignNodeSchedulerToQueuesCommand{})
}

type AssignNodeSchedulerToQueuesCommand struct {
	Queues []models.Queue
	CF     string
	CFS    string
}

func (cmd *AssignNodeSchedulerToQueuesCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	queueRepo, err := db.NewQueueRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	var resultQueues []models.Queue

	for _, queue := range cmd.Queues {
		// Validate that ID is not empty
		if queue.ID == "" {
			commandResult.Error = "Queue ID is required for update"
			return *commandResult
		}

		// Get existing queue to ensure it exists
		existing, err := queueRepo.GetQueueById(queue.ID, now)
		if err != nil {
			commandResult.Error = err.Error()
			return *commandResult
		}

		if existing == nil {
			commandResult.Error = "Queue with ID " + queue.ID + " not found"
			return *commandResult
		}

		// Preserve immutable fields from existing queue
		queue.Code = existing.Code
		queue.Type = existing.Type
		queue.VNamespace = existing.VNamespace
		queue.CreatedAt = existing.CreatedAt
		queue.Name = existing.Name
		queue.State = existing.State
		queue.MessagesCount = existing.MessagesCount
		queue.DefaultQueueMessageTTL = existing.DefaultQueueMessageTTL
		queue.DefaultQueueMessageDelayTime = existing.DefaultQueueMessageDelayTime
		queue.QueueExpires = existing.QueueExpires
		queue.ExpireAt = existing.ExpireAt
		queue.AllowDuplicated = existing.AllowDuplicated
		queue.MaxAttempts = existing.MaxAttempts
		queue.DesiredPriorityThresholds = existing.DesiredPriorityThresholds
		queue.PriorityThresholds = existing.PriorityThresholds
		queue.MaxQueueSize = existing.MaxQueueSize
		queue.DeadLetterExchangeId = existing.DeadLetterExchangeId
		queue.DeadLetterExchangeRoutingKeyOrPattern = existing.DeadLetterExchangeRoutingKeyOrPattern

		// Update the queue
		_, err = queueRepo.UpdateQueue(&queue, now)
		if err != nil {
			commandResult.Error = err.Error()
			return *commandResult
		}

		resultQueues = append(resultQueues, queue)
	}

	commandResult.Result = resultQueues
	return *commandResult
}
