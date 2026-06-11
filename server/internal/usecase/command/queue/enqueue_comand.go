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

	routingHeadersRepo, err := db.NewRoutingHeadersRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	stateRepo, err := db.NewTenantShardStateRepository(uow, idFactory, cmd.CF, cmd.CFS, "tenant_schema")
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	outboxRepo, err := db.NewOutboxEventRepository(uow, idFactory)
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
		message.Attempts = 0 // Initialize attempts counter

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

		// Check if queue has expired
		if queue.ExpireAt != nil && now.After(*queue.ExpireAt) {
			commandResult.Error = fmt.Sprintf("Queue %s has expired on %s", queue.Code, queue.ExpireAt.Format("2006-01-02 15:04:05"))
			return *commandResult
		}

		// Check if queue has reached maximum size
		if queue.MaxQueueSize > 0 && queue.MessagesCount >= queue.MaxQueueSize {
			commandResult.Error = fmt.Sprintf("Queue '%s' has reached its maximum size (%d messages). Cannot enqueue new messages.", queue.Code, queue.MaxQueueSize)
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

	// Seventh pass: Process headers for messages that have them
	for _, message := range processedMessages {
		// If this message has headers, upsert them
		if message.Headers != nil && len(message.Headers) > 0 {
			err = cmd.upsertQueueMessageHeaders(routingHeadersRepo, message, message.Headers, now)
			if err != nil {
				commandResult.Error = err.Error()
				return *commandResult
			}
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

	// Eighth pass: Update Tenant Shard State and generate Outbox Event if needed
	if totalMessagesCreated > 0 {
		tenantState, err := stateRepo.GetByTenantID(cmd.CFS, now)
		if err != nil {
			commandResult.Error = err.Error()
			return *commandResult
		}

		if tenantState == nil || !tenantState.HasMessages {
			// Create or update state
			if tenantState == nil {
				tenantState = &models.TenantShardState{
					ID:          cmd.CFS,
					HasMessages: true,
				}
			} else {
				tenantState.HasMessages = true
			}
			_, err = stateRepo.CreateOrUpdate(tenantState, now)
			if err != nil {
				commandResult.Error = err.Error()
				return *commandResult
			}

			// Generate Outbox Event
			event := &models.OutboxEvent{
				ID:        fmt.Sprintf("outbox_%s_%d", cmd.CFS, now.UnixNano()),
				EventType: models.EventTypeTenantActivated,
				TenantID:  cmd.CFS,
			}
			_, err = outboxRepo.CreateEvent(event, now)
			if err != nil {
				commandResult.Error = err.Error()
				return *commandResult
			}
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

// upsertQueueMessageHeaders creates or updates routing headers for a queue message
func (cmd *EnqueueCommand) upsertQueueMessageHeaders(routingHeadersRepo *db.RoutingHeadersRepository, message models.QueueMessage, headers map[string]string, now time.Time) error {
	// Get existing headers for this message
	existingHeaders, err := routingHeadersRepo.GetRoutingHeadersByMessage(message.ID, now)
	if err != nil {
		return err
	}

	// Create a map for quick lookup of existing headers
	existingByKey := make(map[string]*models.RoutingHeader)
	if existingHeaders != nil {
		for i := range existingHeaders.Entities {
			header := &existingHeaders.Entities[i]
			existingByKey[header.Key] = header
		}
	}

	// Process each header from input
	for key, value := range headers {
		if existingHeader, exists := existingByKey[key]; exists {
			// Update existing header if value changed
			if existingHeader.Value != value {
				existingHeader.Value = value
				_, err := routingHeadersRepo.UpdateRoutingHeader(existingHeader, now)
				if err != nil {
					return err
				}
			}
		} else {
			headerID := message.ID + "_" + key
			// Create new header
			routingHeader := &models.RoutingHeader{
				ID:             headerID,
				QueueMessageID: message.ID,
				Key:            key,
				Value:          value,
				HeaderType:     models.HeaderTypeQueueMessage,
				VNamespace:     message.VNamespace,
			}
			_, err := routingHeadersRepo.CreateRoutingHeader(routingHeader, now)
			if err != nil {
				return err
			}
		}
	}

	// Remove headers that are no longer in the input
	if existingHeaders != nil {
		for _, existingHeader := range existingHeaders.Entities {
			if _, stillExists := headers[existingHeader.Key]; !stillExists {
				_, err := routingHeadersRepo.DeleteRoutingHeader(existingHeader.ID, now)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}
