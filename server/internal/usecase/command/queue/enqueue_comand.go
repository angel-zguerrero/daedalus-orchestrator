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
	gob.Register(EnqueueCommand{})
	gob.Register(models.QueuePartition{})
	gob.Register([]models.QueuePartition{})
	gob.Register(models.QueueMessage{})
	gob.Register([]models.QueueMessage{})
}

type EnqueueCommand struct {
	Message     models.QueueMessage
	MessageCode string
	VNamespace  string
	CF          string
	CFS         string
}

func (cmd *EnqueueCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	idFactory := &db.DeterministicIDGeneratorFactory{}

	// Initialize repositories
	queueRepo, err := db.NewQueueRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	queuePartitionRepo, err := db.NewQueuePartitionRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	queueMessageRepo, err := db.NewQueueMessageRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	// Get the queue
	queue, err := queueRepo.GetQueueByCode(cmd.MessageCode, cmd.VNamespace, now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	if queue == nil {
		commandResult.Error = "Queue not found"
		return *commandResult
	}

	if queue.State != models.QueueActive {
		commandResult.Error = "Queue is not active"
		return *commandResult
	}

	message := cmd.Message
	message.QueueID = queue.ID

	// Validate priority against DesiredPriorityThresholds
	if queue.DesiredPriorityThresholds == nil {
		commandResult.Error = "Queue has no priority thresholds configured"
		return *commandResult
	}

	_, priorityExists := queue.DesiredPriorityThresholds[message.Priority]
	if !priorityExists {
		commandResult.Error = fmt.Sprintf("Priority %d is not allowed for this queue. Allowed priorities: %v",
			message.Priority, getKeysFromMap(queue.DesiredPriorityThresholds))
		return *commandResult
	}

	// Find or create the queue partition for this priority
	partition, err := queuePartitionRepo.GetQueuePartitionByQueueIDAndPriority(queue.ID, message.Priority, now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	// Generate message ID if not provided
	if message.ID == "" {
		message.ID = idFactory.GenerateID()
	}

	if partition == nil {
		// Create new partition with MessagesCount = 1 since we can't update after creation in UoW
		partition = &models.QueuePartition{
			ID:                  queue.ID + "-p-" + fmt.Sprintf("%d", message.Priority),
			QueueID:             queue.ID,
			Priority:            message.Priority,
			MessagesCount:       1, // Start with 1 since we're adding the first message
			FirstQueueMessageID: message.ID,
			LastQueueMessageID:  message.ID,
		}

		_, err = queuePartitionRepo.CreateQueuePartition(partition, now)
		if err != nil {
			commandResult.Error = err.Error()
			return *commandResult
		}

		// Set the partition ID for the message
		message.QueuePartitionID = partition.ID

		// Create the message (this is the first message, so no chaining needed)
		_, err = queueMessageRepo.CreateQueueMessage(&message, now)
		if err != nil {
			commandResult.Error = err.Error()
			return *commandResult
		}
	} else {
		// Existing partition - handle message chaining and update
		// Set the partition ID for the message
		message.QueuePartitionID = partition.ID

		// Handle message chaining
		if partition.LastQueueMessageID != "" {
			// Find the last message and update its NextQueueMessageID
			lastMessage, err := queueMessageRepo.GetQueueMessageById(partition.LastQueueMessageID, now)
			if err != nil {
				commandResult.Error = err.Error()
				return *commandResult
			}

			if lastMessage != nil {
				// Update the last message to point to this new message
				lastMessage.NextQueueMessageID = message.ID

				_, err = queueMessageRepo.UpdateQueueMessage(lastMessage, now)
				if err != nil {
					commandResult.Error = err.Error()
					return *commandResult
				}
			}
		} else {
			// This should not happen in an existing partition, but handle it gracefully
			partition.FirstQueueMessageID = message.ID
		}

		// Create the message
		_, err = queueMessageRepo.CreateQueueMessage(&message, now)
		if err != nil {
			commandResult.Error = err.Error()
			return *commandResult
		}

		// Update partition with new last message and increment message count
		partition.LastQueueMessageID = message.ID
		partition.MessagesCount++

		_, err = queuePartitionRepo.UpdateQueuePartition(partition, now)
		if err != nil {
			commandResult.Error = err.Error()
			return *commandResult
		}
	}

	// Update queue message count
	queue.MessagesCount++
	_, err = queueRepo.UpdateQueue(queue, now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	commandResult.Result = message
	return *commandResult
}

// Helper function to get keys from map
func getKeysFromMap(m map[int]int) []int {
	keys := make([]int, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
