package queue

import (
	"deadalus-orch/server/internal/infrastructure/db"
	priorityqueue "deadalus-orch/server/internal/pkg/priority-queue"
	"deadalus-orch/server/internal/usecase/command"
	"deadalus-orch/shared/models"
	"encoding/gob"
	"fmt"
	"strings"
	"time"
)

func init() {
	gob.Register(DequeueCommand{})
	gob.Register(DequeueResult{})
	gob.Register(models.QueueMessageLease{})
}

// DequeueResult is the value returned in CommandResult.Result after a successful dequeue.
type DequeueResult struct {
	Message models.QueueMessage
	Lease   models.QueueMessageLease
}

// DequeueCommand dequeues the next available message from the specified queue for a
// JobWorker, using the threshold-based fair priority queue algorithm, and creates
// a QueueMessageLease to track the in-flight message.
type DequeueCommand struct {
	// QueueID is the internal ID of the target queue (NOT the queue code).
	QueueID string
	// JobWorkerID is the ID of the worker that claims the message.
	JobWorkerID string
	// LeaseDuration defines how long the lease is valid. Calculated by the BO
	// layer from config.GlobalConfiguration.MessageLeaseDuration.
	LeaseDuration time.Duration
	// JobWorkerCapacityPolicyIndex is the index of the capacity policy (from the
	// worker's capacityPolicies slice) that matched when this dequeue was triggered.
	// Stored in the lease so the SDK can update its per-policy in-flight counter.
	JobWorkerCapacityPolicyIndex int
	CF                           string
	CFS                          string
}

func (cmd *DequeueCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	if cmd.QueueID == "" {
		commandResult.Error = "QueueID is required"
		return *commandResult
	}
	if cmd.JobWorkerID == "" {
		commandResult.Error = "JobWorkerID is required"
		return *commandResult
	}
	if cmd.LeaseDuration <= 0 {
		commandResult.Error = "LeaseDuration must be positive"
		return *commandResult
	}

	idFactory := &db.DeterministicIDGeneratorFactory{}

	// ── repositories ────────────────────────────────────────────────────────────

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

	leaseRepo, err := db.NewQueueMessageLeaseRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	// ── 1. load queue ────────────────────────────────────────────────────────────

	queue, err := queueRepo.GetQueueById(cmd.QueueID, now)
	if err != nil {
		commandResult.Error = fmt.Sprintf("failed to load queue %s: %s", cmd.QueueID, err.Error())
		return *commandResult
	}
	if queue == nil {
		commandResult.Error = fmt.Sprintf("queue %s not found", cmd.QueueID)
		return *commandResult
	}

	// Only Active queues can deliver new messages.
	// Draining queues stop accepting new messages but still allow dequeue.
	if queue.State != models.QueueActive && queue.State != models.QueueDraining {
		commandResult.Error = fmt.Sprintf("queue '%s' is not available for dequeue (state: %s)", queue.Code, queue.State)
		return *commandResult
	}

	// Check max concurrent delivering messages (0 = unlimited).
	if queue.MaxDeliveringMessages > 0 && queue.CurrentDeliveringMessages >= queue.MaxDeliveringMessages {
		commandResult.Error = fmt.Sprintf("queue '%s' has reached the maximum number of in-flight messages (%d)", queue.Code, queue.MaxDeliveringMessages)
		return *commandResult
	}

	// ── 2. find non-empty partitions ─────────────────────────────────────────────

	partitions, err := queuePartitionRepo.GetNonEmptyPartitionsByQueueID(cmd.QueueID, now)
	if err != nil {
		commandResult.Error = fmt.Sprintf("failed to load partitions for queue %s: %s", cmd.QueueID, err.Error())
		return *commandResult
	}
	if len(partitions) == 0 {
		commandResult.Error = fmt.Sprintf("queue '%s' is empty", queue.Code)
		return *commandResult
	}

	// ── 3. threshold-based fair priority selection ────────────────────────────────
	//
	// Build an in-memory PriorityQueue, restore the persisted scheduler state
	// (PQProcessedCounts + PQCurrentPriority) so that the weighted cycling defined
	// by DesiredPriorityThresholds is maintained across independent Execute() calls,
	// then run a single Dequeue() to pick the next partition.
	// The updated state is written back into the queue record at step 6.

	thresholds := queue.DesiredPriorityThresholds
	if thresholds == nil {
		// Fallback: treat all priorities as "drain" (threshold 0).
		thresholds = make(map[int]int)
	}

	pq := priorityqueue.NewPriorityQueue(thresholds)
	for i, p := range partitions {
		pq.Enqueue(priorityqueue.Task{
			ID:       p.ID,
			Priority: p.Priority,
			Index:    i,
		})
	}

	// Restore persisted scheduler position so the weighted cycling continues
	// across independent DequeueCommand calls.
	pq.RestoreState(queue.PQProcessedCounts, queue.PQCurrentPriority)

	selectedTask := pq.Dequeue()
	if selectedTask == nil {
		// Should not happen since we already checked len(partitions) > 0.
		commandResult.Error = fmt.Sprintf("queue '%s' is empty (priority queue returned nil)", queue.Code)
		return *commandResult
	}

	// Capture the updated scheduler state — written to the queue record at step 6.
	newCounts, newCurrentPriority := pq.GetState()
	queue.PQProcessedCounts = newCounts
	queue.PQCurrentPriority = newCurrentPriority

	// Find the selected partition.
	var selectedPartition *models.QueuePartition
	for i := range partitions {
		if partitions[i].ID == selectedTask.ID {
			selectedPartition = &partitions[i]
			break
		}
	}
	if selectedPartition == nil {
		commandResult.Error = "internal error: selected partition not found in loaded set"
		return *commandResult
	}

	// ── 4. retrieve the first message of the partition ────────────────────────────

	message, err := queueMessageRepo.GetQueueMessageById(selectedPartition.FirstQueueMessageID, now)
	if err != nil {
		commandResult.Error = fmt.Sprintf("failed to load message %s: %s", selectedPartition.FirstQueueMessageID, err.Error())
		return *commandResult
	}
	if message == nil {
		commandResult.Error = fmt.Sprintf("message %s not found (partition data may be inconsistent)", selectedPartition.FirstQueueMessageID)
		return *commandResult
	}

	// ── 5. advance the partition ─────────────────────────────────────────────────

	// Keep a copy of the next pointer before we clear it on the returned message.
	nextMessageID := message.NextQueueMessageID

	updatedPartition := *selectedPartition
	updatedPartition.MessagesCount--
	if updatedPartition.MessagesCount <= 0 {
		updatedPartition.MessagesCount = 0
		updatedPartition.FirstQueueMessageID = ""
		updatedPartition.LastQueueMessageID = ""
	} else {
		updatedPartition.FirstQueueMessageID = nextMessageID
	}

	if _, err = queuePartitionRepo.UpdateQueuePartition(&updatedPartition, now); err != nil {
		commandResult.Error = fmt.Sprintf("failed to update partition %s: %s", selectedPartition.ID, err.Error())
		return *commandResult
	}

	// ── 6. update queue counters ─────────────────────────────────────────────────

	queue.MessagesCount--
	if queue.MessagesCount < 0 {
		queue.MessagesCount = 0
	}
	queue.CurrentDeliveringMessages++

	if _, err = queueRepo.UpdateQueue(queue, now); err != nil {
		commandResult.Error = fmt.Sprintf("failed to update queue %s counters: %s", cmd.QueueID, err.Error())
		return *commandResult
	}

	// ── 6.5. increment message attempts counter ──────────────────────────────────

	message.Attempts++
	if _, err = queueMessageRepo.UpdateQueueMessage(message, now); err != nil {
		commandResult.Error = fmt.Sprintf("failed to update message %s attempts: %s", message.ID, err.Error())
		return *commandResult
	}

	// ── 7. create the lease ──────────────────────────────────────────────────────

	leaseID := strings.ReplaceAll(message.ID+"-"+cmd.JobWorkerID, "-", "") // deterministic but unique per (message, worker)
	lease := &models.QueueMessageLease{
		ID:                                leaseID,
		QueueMessageID:                    message.ID,
		WorkerID:                          cmd.JobWorkerID,
		LeaseStatus:                       models.QueueMessageLeaseStatusActive,
		LeaseUntil:                        now.Add(cmd.LeaseDuration),
		JobWorkerCapacityPolicyIndexMatch: cmd.JobWorkerCapacityPolicyIndex,
	}

	if _, err = leaseRepo.CreateQueueMessageLease(lease, now); err != nil {
		commandResult.Error = fmt.Sprintf("failed to create lease for message %s: %s", message.ID, err.Error())
		return *commandResult
	}

	// ── 8. return result ─────────────────────────────────────────────────────────

	// Clear the linked-list pointer — the consumer only needs the message content.
	message.NextQueueMessageID = ""

	commandResult.Result = DequeueResult{
		Message: *message,
		Lease:   *lease,
	}
	return *commandResult
}
