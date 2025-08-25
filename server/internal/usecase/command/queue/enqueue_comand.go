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
	Messages   []models.QueueMessage
	QueueCode  string
	VNamespace string
	CF         string
	CFS        string
}

func (cmd *EnqueueCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	// Validate input
	if len(cmd.Messages) == 0 {
		commandResult.Error = "No messages provided"
		return *commandResult
	}

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
	queue, err := queueRepo.GetQueueByCode(cmd.QueueCode, cmd.VNamespace, now)
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

	// Validate queue has priority thresholds configured
	if queue.DesiredPriorityThresholds == nil {
		commandResult.Error = "Queue has no priority thresholds configured"
		return *commandResult
	}

	var processedMessages []models.QueueMessage

	// First pass: Count messages per priority and prepare message data
	messagesByPriority := make(map[int][]models.QueueMessage)
	partitionUpdates := make(map[int]*models.QueuePartition)

	for i := range cmd.Messages {
		message := &cmd.Messages[i] // Use pointer to modify in place
		message.QueueID = queue.ID

		// Validate priority against DesiredPriorityThresholds
		_, priorityExists := queue.DesiredPriorityThresholds[message.Priority]
		if !priorityExists {
			commandResult.Error = fmt.Sprintf("Priority %d is not allowed for this queue. Allowed priorities: %v",
				message.Priority, getKeysFromMap(queue.DesiredPriorityThresholds))
			return *commandResult
		}

		// Generate message ID if not provided
		if message.ID == "" {
			message.ID = idFactory.GenerateID()
		}

		// Group messages by priority
		if messagesByPriority[message.Priority] == nil {
			messagesByPriority[message.Priority] = make([]models.QueueMessage, 0)
		}
		messagesByPriority[message.Priority] = append(messagesByPriority[message.Priority], *message)
	}

	// Second pass: Handle partitions and messages by priority
	for priority, messages := range messagesByPriority {
		// Try to get existing partition (this read happens before any writes)
		existingPartition, err := queuePartitionRepo.GetQueuePartitionByQueueIDAndPriority(queue.ID, priority, now)
		if err != nil {
			commandResult.Error = err.Error()
			return *commandResult
		}

		var partition *models.QueuePartition
		isNewPartition := existingPartition == nil

		if isNewPartition {
			// Create new partition with the correct initial count
			partition = &models.QueuePartition{
				ID:                  queue.ID + "-p-" + fmt.Sprintf("%d", priority),
				QueueID:             queue.ID,
				Priority:            priority,
				MessagesCount:       len(messages), // Start with total count for this batch
				FirstQueueMessageID: messages[0].ID,
				LastQueueMessageID:  messages[len(messages)-1].ID,
			}

			_, err = queuePartitionRepo.CreateQueuePartition(partition, now)
			if err != nil {
				commandResult.Error = err.Error()
				return *commandResult
			}
		} else {
			// Prepare update for existing partition
			partition = &models.QueuePartition{
				ID:                  existingPartition.ID,
				QueueID:             existingPartition.QueueID,
				Priority:            existingPartition.Priority,
				MessagesCount:       existingPartition.MessagesCount + len(messages), // Increment by batch size
				FirstQueueMessageID: existingPartition.FirstQueueMessageID,
				LastQueueMessageID:  messages[len(messages)-1].ID, // Update to last message in this batch
				CreatedAt:           existingPartition.CreatedAt,
			}

			// If this was an empty partition, set the first message
			if existingPartition.MessagesCount == 0 {
				partition.FirstQueueMessageID = messages[0].ID
			}

			partitionUpdates[priority] = partition
		}

		// Process messages for this priority and set up chaining
		for i, message := range messages {
			msg := message // Create a copy
			msg.QueuePartitionID = partition.ID

			// Handle message chaining within this batch
			if i > 0 {
				// Link to previous message in this batch
				messages[i-1].NextQueueMessageID = msg.ID
			}

			processedMessages = append(processedMessages, msg)
		}

		// Create all messages for this priority
		for i := range messages {
			_, err = queueMessageRepo.CreateQueueMessage(&messages[i], now)
			if err != nil {
				commandResult.Error = err.Error()
				return *commandResult
			}
		}

		// Handle chaining to existing partition's last message (if this is an existing partition)
		if !isNewPartition && existingPartition.LastQueueMessageID != "" {
			// Update the last existing message to point to our first message
			lastMessage, err := queueMessageRepo.GetQueueMessageById(existingPartition.LastQueueMessageID, now)
			if err != nil {
				commandResult.Error = err.Error()
				return *commandResult
			}
			if lastMessage != nil {
				lastMessage.NextQueueMessageID = messages[0].ID
				_, err = queueMessageRepo.UpdateQueueMessage(lastMessage, now)
				if err != nil {
					commandResult.Error = err.Error()
					return *commandResult
				}
			}
		}
	}

	// Third pass: Update existing partitions (separate from creation)
	for _, partition := range partitionUpdates {
		_, err = queuePartitionRepo.UpdateQueuePartition(partition, now)
		if err != nil {
			commandResult.Error = err.Error()
			return *commandResult
		}
	}

	// Update queue message count
	queue.MessagesCount += len(cmd.Messages)
	_, err = queueRepo.UpdateQueue(queue, now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	commandResult.Result = processedMessages
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
