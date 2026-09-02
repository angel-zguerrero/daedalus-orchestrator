package queue

import (
	"deadalus-orch/server/internal/infrastructure/db"
	priorityqueue "deadalus-orch/server/internal/pkg/priority-queue"
	"deadalus-orch/server/internal/usecase/command"
	"deadalus-orch/shared/models"
	"encoding/gob"
	"strings"
	"time"
)

func init() {
	gob.Register(BatchDequeueCommand{})
	gob.Register(BatchDequeueResult{})
}

type BatchDequeueResult struct {
	Results []DequeueResult
}

type BatchDequeueCommand struct {
	CF                           string
	CFS                          string
	QueueID                      string
	JobWorkerID                  string
	LeaseDuration                time.Duration
	JobWorkerCapacityPolicyIndex int
	Count                        int
}

func (cmd *BatchDequeueCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	if cmd.Count <= 0 {
		return *commandResult
	}

	idFactory := &db.DeterministicIDGeneratorFactory{}

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

	activeQueueRepo, err := db.NewActiveQueueRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	queue, err := queueRepo.GetQueueById(cmd.QueueID, now)
	if err != nil || queue == nil {
		commandResult.Error = "queue not found"
		return *commandResult
	}

	if queue.State != models.QueueActive && queue.State != models.QueueDraining {
		commandResult.Error = "queue not active or draining"
		return *commandResult
	}

	partitions, err := queuePartitionRepo.GetNonEmptyPartitionsByQueueID(cmd.QueueID, now)
	if err != nil || len(partitions) == 0 {
		return *commandResult
	}

	// Recreate Priority Queue
	thresholds := queue.DesiredPriorityThresholds
	if thresholds == nil {
		thresholds = make(map[int]int)
	}
	pq := priorityqueue.NewPriorityQueue(thresholds)
	for idx, p := range partitions {
		if p.MessagesCount > 0 {
			pq.Enqueue(priorityqueue.Task{
				ID:       p.ID,
				Priority: p.Priority,
				Index:    idx,
			})
		}
	}

	pq.RestoreState(queue.PQProcessedCounts, queue.PQCurrentPriority)

	limit := cmd.Count
	if queue.MaxDeliveringMessages > 0 {
		available := queue.MaxDeliveringMessages - queue.CurrentDeliveringMessages
		if available < limit {
			limit = available
		}
	}
	if limit <= 0 {
		return *commandResult
	}

	results := make([]DequeueResult, 0, limit)
	messagesToUpdate := make([]*models.QueueMessage, 0, limit)
	leasesToCreate := make([]*models.QueueMessageLease, 0, limit)
	partitionsToUpdateMap := make(map[string]*models.QueuePartition)

	for i := 0; i < limit; i++ {
		selectedTask := pq.Dequeue()
		if selectedTask == nil {
			break
		}

		var selectedPartition *models.QueuePartition
		for idx := range partitions {
			if partitions[idx].ID == selectedTask.ID {
				selectedPartition = &partitions[idx]
				break
			}
		}

		message, err := queueMessageRepo.GetQueueMessageById(selectedPartition.FirstQueueMessageID, now)
		if err != nil || message == nil {
			break
		}

		nextMessageID := message.NextQueueMessageID

		selectedPartition.MessagesCount--
		if selectedPartition.MessagesCount <= 0 {
			selectedPartition.MessagesCount = 0
			selectedPartition.FirstQueueMessageID = ""
			selectedPartition.LastQueueMessageID = ""
		} else {
			selectedPartition.FirstQueueMessageID = nextMessageID
			// Re-enqueue the partition to the priority queue.
			// Since we only extracted 1 message in this iteration, we must put the partition 
			// back into the queue so the batch loop can continue extracting its remaining messages.
			pq.Enqueue(priorityqueue.Task{
				ID:       selectedPartition.ID,
				Priority: selectedPartition.Priority,
				Index:    selectedTask.Index,
			})
		}
		partitionsToUpdateMap[selectedPartition.ID] = selectedPartition

		queue.MessagesCount--
		if queue.MessagesCount <= 0 {
			queue.MessagesCount = 0
			// ACTIVE QUEUE REGISTRY: Queue is now empty, remove from ActiveQueues
			_, err = activeQueueRepo.DeleteActiveQueue(queue.ID, now)
			if err != nil {
				commandResult.Error = err.Error()
				return *commandResult
			}
		}
		queue.CurrentDeliveringMessages++

		message.Attempts++
		messagesToUpdate = append(messagesToUpdate, message)

		leaseID := strings.ReplaceAll(message.ID+"-"+cmd.JobWorkerID, "-", "")
		lease := &models.QueueMessageLease{
			ID:                                leaseID,
			QueueMessageID:                    message.ID,
			WorkerID:                          cmd.JobWorkerID,
			LeaseStatus:                       models.QueueMessageLeaseStatusActive,
			LeaseUntil:                        now.Add(cmd.LeaseDuration),
			JobWorkerCapacityPolicyIndexMatch: cmd.JobWorkerCapacityPolicyIndex,
			CreatedAt:                         now,
		}
		leasesToCreate = append(leasesToCreate, lease)

		msgCopy := *message
		msgCopy.NextQueueMessageID = "" // Hide linked list implementation details

		results = append(results, DequeueResult{
			Message:   msgCopy,
			Lease:     *lease,
			Pending:   uint64(queue.MessagesCount),
			InProcess: uint64(queue.CurrentDeliveringMessages),
		})
	}

	newCounts, newCurrentPriority := pq.GetState()
	queue.PQProcessedCounts = newCounts
	queue.PQCurrentPriority = newCurrentPriority

	if len(messagesToUpdate) > 0 {
		_, err = queueMessageRepo.BulkUpdateQueueMessage(messagesToUpdate, now)
		if err != nil {
			commandResult.Error = err.Error()
			return *commandResult
		}
	}

	if len(leasesToCreate) > 0 {
		_, err = leaseRepo.BulkCreateQueueMessageLease(leasesToCreate, now)
		if err != nil {
			commandResult.Error = err.Error()
			return *commandResult
		}
	}

	queueRepo.UpdateQueue(queue, now)
	for _, p := range partitionsToUpdateMap {
		queuePartitionRepo.UpdateQueuePartition(p, now)
	}

	commandResult.Result = BatchDequeueResult{Results: results}
	return *commandResult
}
