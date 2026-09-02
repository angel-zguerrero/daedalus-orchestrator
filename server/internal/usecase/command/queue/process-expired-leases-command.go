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
	gob.Register(ProcessExpiredLeasesCommand{})
	gob.Register(ProcessExpiredLeasesResult{})
}

// ProcessExpiredLeasesResult contains the statistics of processed expired leases.
type ProcessExpiredLeasesResult struct {
	ProcessedLeases  int
	DeletedMessages  int
	RequeuedMessages int
	Errors           []string
	Gauges           []models.QueueGauges
}

// ProcessExpiredLeasesCommand processes expired leases in a paginated manner.
// For each expired lease:
//   - If the message has reached MaxAttempts: delete both message and lease
//   - If the message has not reached MaxAttempts: increment Attempts, requeue message, delete lease
type ProcessExpiredLeasesCommand struct {
	Limit  int    // Number of leases to process in this batch
	Offset int    // Offset for pagination
	CF     string // Column family
	CFS    string // Column family sector
}

func (cmd *ProcessExpiredLeasesCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	result := &ProcessExpiredLeasesResult{
		Errors: make([]string, 0),
	}

	if cmd.Limit <= 0 {
		cmd.Limit = 100 // Default batch size
	}

	idFactory := &db.DeterministicIDGeneratorFactory{}

	// ── repositories ────────────────────────────────────────────────────────────

	leaseRepo, err := db.NewQueueMessageLeaseRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	queueMessageRepo, err := db.NewQueueMessageRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

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

	tenantSummaryRepo, err := db.NewTenantSummaryRepository(uow, idFactory)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	activeQueueRepo, err := db.NewActiveQueueRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	// ── 1. find expired leases ───────────────────────────────────────────────────

	expiredLeases, err := leaseRepo.FindExpiredLeases(cmd.Limit, cmd.Offset, now)
	if err != nil {
		commandResult.Error = fmt.Sprintf("failed to find expired leases: %s", err.Error())
		return *commandResult
	}

	if len(expiredLeases.Entities) == 0 {
		commandResult.Result = result
		return *commandResult
	}

	// ── 2. process each expired lease ────────────────────────────────────────────

	messagesDeleted := 0
	messagesRequeued := 0

	for _, lease := range expiredLeases.Entities {
		// Load the associated message
		message, err := queueMessageRepo.GetQueueMessageById(lease.QueueMessageID, now)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("failed to load message %s: %s", lease.QueueMessageID, err.Error()))
			continue
		}
		if message == nil {
			// Message already deleted, just clean up the lease
			if _, err = leaseRepo.Delete(lease.ID, now); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("failed to delete orphan lease %s: %s", lease.ID, err.Error()))
			} else {
				result.ProcessedLeases++
			}
			continue
		}

		// Load the queue to check MaxAttempts
		queue, err := queueRepo.GetQueueById(message.QueueID, now)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("failed to load queue %s: %s", message.QueueID, err.Error()))
			continue
		}
		if queue == nil {
			result.Errors = append(result.Errors, fmt.Sprintf("queue %s not found for message %s", message.QueueID, message.ID))
			continue
		}

		if !lease.IsDelivered {
			// The message was dequeued but never successfully delivered to the worker stream.
			message.Attempts--
			if message.Attempts < 0 {
				message.Attempts = 0
			}
		}

		// Check if message has reached max attempts
		if queue.MaxAttempts > 0 && message.Attempts >= queue.MaxAttempts {
			fmt.Printf("Message %s has reached max attempts (%d), deleting message and lease\n", message.ID, message.Attempts)
			// Delete message and lease
			if err := cmd.deleteMessageAndLease(queueMessageRepo, leaseRepo, queueRepo, queuePartitionRepo, message, &lease, queue, now); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("failed to delete message %s: %s", message.ID, err.Error()))
				continue
			}
			messagesDeleted++
			result.DeletedMessages++
		} else {
			// Requeue message and delete lease
			err = cmd.requeueMessageAndDeleteLease(queueMessageRepo, leaseRepo, queueRepo, queuePartitionRepo, activeQueueRepo, message, &lease, queue, now)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("failed to requeue message %s: %s", message.ID, err.Error()))
				continue
			}
			messagesRequeued++
			result.RequeuedMessages++
		}

		result.ProcessedLeases++
	}

	// ── 3. update tenant summary ─────────────────────────────────────────────────

	if messagesDeleted > 0 {
		err = tenantSummaryRepo.UpdateCounters(cmd.CFS, -messagesDeleted, 0, 0, 0, now)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("failed to update tenant summary: %s", err.Error()))
		}
	}

	// ── 4. Collect gauges for affected queues ────────────────────────────────────
	gaugesMap := make(map[string]models.QueueGauges)
	for _, lease := range expiredLeases.Entities {
		message, _ := queueMessageRepo.GetQueueMessageById(lease.QueueMessageID, now)
		if message != nil {
			queue, _ := queueRepo.GetQueueById(message.QueueID, now)
			if queue != nil {
				gaugesMap[queue.ID] = models.QueueGauges{
					QueueCode:  queue.Code,
					VNamespace: queue.VNamespace,
					Pending:    uint64(queue.MessagesCount),
					InProcess:  uint64(queue.CurrentDeliveringMessages),
				}
			}
		}
	}
	for _, g := range gaugesMap {
		result.Gauges = append(result.Gauges, g)
	}

	commandResult.Result = *result
	return *commandResult
}

// deleteMessageAndLease deletes both the message and its lease when MaxAttempts is reached.
func (cmd *ProcessExpiredLeasesCommand) deleteMessageAndLease(
	messageRepo *db.QueueMessageRepository,
	leaseRepo *db.QueueMessageLeaseRepository,
	queueRepo *db.QueueRepository,
	partitionRepo *db.QueuePartitionRepository,
	message *models.QueueMessage,
	lease *models.QueueMessageLease,
	queue *models.Queue,
	now time.Time,
) error {
	fmt.Printf("Deleting message %s and lease %s for queue %s due to max attempts reached\n", message.ID, lease.ID, queue.ID)
	// Delete the lease first
	if _, err := leaseRepo.Delete(lease.ID, now); err != nil {
		return fmt.Errorf("failed to delete lease: %w", err)
	}

	// Delete the message
	if _, err := messageRepo.Delete(message.ID, now); err != nil {
		return fmt.Errorf("failed to delete message: %w", err)
	}

	// Decrement delivering messages counter
	queue.CurrentDeliveringMessages--
	if queue.CurrentDeliveringMessages < 0 {
		queue.CurrentDeliveringMessages = 0
	}

	if _, err := queueRepo.UpdateQueue(queue, now); err != nil {
		return fmt.Errorf("failed to update queue: %w", err)
	}

	return nil
}

// requeueMessageAndDeleteLease requeues the message at the front of its partition,
// increments attempts, and deletes the lease.
func (cmd *ProcessExpiredLeasesCommand) requeueMessageAndDeleteLease(
	messageRepo *db.QueueMessageRepository,
	leaseRepo *db.QueueMessageLeaseRepository,
	queueRepo *db.QueueRepository,
	partitionRepo *db.QueuePartitionRepository,
	activeQueueRepo *db.ActiveQueueRepository,
	message *models.QueueMessage,
	lease *models.QueueMessageLease,
	queue *models.Queue,
	now time.Time,
) error {
	fmt.Printf("Requeuing message %s and deleting lease %s for queue %s\n", message.ID, lease.ID, queue.ID)

	// 1. Delete the lease
	if _, err := leaseRepo.Delete(lease.ID, now); err != nil {
		return fmt.Errorf("failed to delete lease: %w", err)
	}

	// 2. Prepare the message to be first in the chain

	message.NextQueueMessageID = ""

	// 3. Re-link the message at the front of its partition's linked list
	if message.QueuePartitionID != "" {
		partition, err := partitionRepo.FindByField("ID", message.QueuePartitionID, now)
		if err != nil {
			return fmt.Errorf("failed to load partition: %w", err)
		}
		if partition != nil {
			if partition.MessagesCount > 0 && partition.FirstQueueMessageID != "" {
				// Partition has messages: this message points to the current first
				message.NextQueueMessageID = partition.FirstQueueMessageID
			} else {
				// Partition is empty: this message is also the last
				partition.LastQueueMessageID = message.ID
			}

			// This message becomes the new first in the partition
			partition.FirstQueueMessageID = message.ID
			partition.MessagesCount++

			if _, err := partitionRepo.UpdateQueuePartition(partition, now); err != nil {
				return fmt.Errorf("failed to update partition: %w", err)
			}
		}
	}

	// 4. Update the message (with corrected NextQueueMessageID)
	if _, err := messageRepo.UpdateQueueMessage(message, now); err != nil {
		return fmt.Errorf("failed to update message: %w", err)
	}

	// 5. Update queue counters: message returns to available pool
	// ACTIVE QUEUE REGISTRY: Add to ActiveQueues unconditionally to avoid race conditions
	if _, err := activeQueueRepo.PutActiveQueue(queue, now); err != nil {
		return fmt.Errorf("failed to put active queue: %w", err)
	}
	queue.MessagesCount++
	queue.CurrentDeliveringMessages--
	if queue.CurrentDeliveringMessages < 0 {
		queue.CurrentDeliveringMessages = 0
	}

	if _, err := queueRepo.UpdateQueue(queue, now); err != nil {
		return fmt.Errorf("failed to update queue: %w", err)
	}

	return nil
}
