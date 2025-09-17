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
	Messages []models.QueueMessage
	CF       string
	CFS      string
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

	tenantSummaryRepo, err := db.NewTenantSummaryRepository(uow, idFactory)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	// First pass: Group messages by QueueID and validate QueueIDs
	messagesByQueue := make(map[string][]models.QueueMessage)
	queuesCache := make(map[string]*models.Queue)

	for i := range cmd.Messages {
		message := &cmd.Messages[i]

		// Validate that QueueID is provided
		if message.QueueID == "" {
			commandResult.Error = "QueueID is required for all messages"
			return *commandResult
		}

		// Generate message ID if not provided
		if message.ID == "" {
			commandResult.Error = "Message ID is required for all messages"
			return *commandResult
		}

		message.ContentLength = int64(len(message.Content))

		// Group messages by QueueID
		if messagesByQueue[message.QueueID] == nil {
			messagesByQueue[message.QueueID] = make([]models.QueueMessage, 0)
		}
		messagesByQueue[message.QueueID] = append(messagesByQueue[message.QueueID], *message)
	}

	// Second pass: Validate queues and get their configurations
	for queueID := range messagesByQueue {
		queue, err := queueRepo.GetQueueById(queueID, now)
		if err != nil {
			commandResult.Error = err.Error()
			return *commandResult
		}

		if queue == nil {
			commandResult.Error = fmt.Sprintf("Queue with ID %s not found", queueID)
			return *commandResult
		}

		if queue.State != models.QueueActive {
			commandResult.Error = fmt.Sprintf("Queue %s is not active", queueID)
			return *commandResult
		}

		if queue.DesiredPriorityThresholds == nil {
			commandResult.Error = fmt.Sprintf("Queue %s has no priority thresholds configured", queueID)
			return *commandResult
		}

		queuesCache[queueID] = queue
	}

	// Third pass: Validate priorities for each queue and group by queue+priority
	messagesByQueueAndPriority := make(map[string]map[int][]models.QueueMessage)
	partitionUpdates := make(map[string]*models.QueuePartition)

	for queueID, messages := range messagesByQueue {
		queue := queuesCache[queueID]
		messagesByQueueAndPriority[queueID] = make(map[int][]models.QueueMessage)

		for _, message := range messages {
			// Validate priority against DesiredPriorityThresholds
			_, priorityExists := queue.DesiredPriorityThresholds[message.Priority]
			if !priorityExists {
				commandResult.Error = fmt.Sprintf("Priority %d is not allowed for queue %s. Allowed priorities: %v",
					message.Priority, queueID, getKeysFromMap(queue.DesiredPriorityThresholds))
				return *commandResult
			}

			// Group by priority within queue
			if messagesByQueueAndPriority[queueID][message.Priority] == nil {
				messagesByQueueAndPriority[queueID][message.Priority] = make([]models.QueueMessage, 0)
			}
			messagesByQueueAndPriority[queueID][message.Priority] = append(messagesByQueueAndPriority[queueID][message.Priority], message)
		}
	}

	// Fourth pass: Handle partitions and messages by queue and priority
	var processedMessages []models.QueueMessage

	for queueID, priorityGroups := range messagesByQueueAndPriority {
		for priority, messages := range priorityGroups {
			// Try to get existing partition (this read happens before any writes)
			existingPartition, err := queuePartitionRepo.GetQueuePartitionByQueueIDAndPriority(queueID, priority, now)
			if err != nil {
				commandResult.Error = err.Error()
				return *commandResult
			}

			var partition *models.QueuePartition
			isNewPartition := existingPartition == nil

			if isNewPartition {
				// Create new partition with the correct initial count
				partition = &models.QueuePartition{
					ID:                  queueID + "-p-" + fmt.Sprintf("%d", priority),
					QueueID:             queueID,
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
				partitionKey := fmt.Sprintf("%s-%d", queueID, priority)
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

				partitionUpdates[partitionKey] = partition
			}

			// Process messages for this queue+priority and set up chaining
			for i := range messages {
				messages[i].QueuePartitionID = partition.ID

				// Handle message chaining within this batch
				if i > 0 {
					// Link to previous message in this batch
					messages[i-1].NextQueueMessageID = messages[i].ID
				}
			}

			// Create all messages for this queue+priority
			for i := range messages {
				_, err = queueMessageRepo.CreateQueueMessage(&messages[i], now)
				if err != nil {
					commandResult.Error = err.Error()
					return *commandResult
				}
				processedMessages = append(processedMessages, messages[i])
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
	}

	// Fifth pass: Update existing partitions (separate from creation)
	for _, partition := range partitionUpdates {
		_, err = queuePartitionRepo.UpdateQueuePartition(partition, now)
		if err != nil {
			commandResult.Error = err.Error()
			return *commandResult
		}
	}

	// Sixth pass: Update queue message counts
	for queueID, messages := range messagesByQueue {
		queue := queuesCache[queueID]
		queue.MessagesCount += len(messages)
		_, err = queueRepo.UpdateQueue(queue, now)
		if err != nil {
			commandResult.Error = err.Error()
			return *commandResult
		}
	}

	// Update tenant summary with the total count of new messages created
	totalMessagesCreated := len(processedMessages)
	if totalMessagesCreated > 0 {
		err = tenantSummaryRepo.UpdateCounters(cmd.CFS, totalMessagesCreated, 0, 0, 0, now)
		if err != nil {
			commandResult.Error = err.Error()
			return *commandResult
		}
	}

	commandResult.Result = processedMessages
	return *commandResult
} // Helper function to get keys from map
func getKeysFromMap(m map[int]int) []int {
	keys := make([]int, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
