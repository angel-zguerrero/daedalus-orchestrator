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
	gob.Register(BulkDequeueCommand{})
	gob.Register(BulkDequeueResult{})
}

type BulkDequeueResult struct {
	Results map[int]DequeueResult
}

type BulkDequeueRequest struct {
	QueueID                      string
	JobWorkerID                  string
	LeaseDuration                time.Duration
	JobWorkerCapacityPolicyIndex int
}

type BulkDequeueCommand struct {
	CF       string
	CFS      string
	Requests []BulkDequeueRequest
}

func (cmd *BulkDequeueCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	start := time.Now()
	defer func() {
		elapsed := time.Since(start)
		if elapsed > 50*time.Millisecond {
			fmt.Printf("[PERF] BulkDequeueCommand internal Execute took %v for %d requests\n", elapsed, len(cmd.Requests))
		}
	}()

	commandResult := &command.CommandResult{}

	if len(cmd.Requests) == 0 {
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

	cachedQueues := make(map[string]*models.Queue)
	cachedPartitions := make(map[string][]models.QueuePartition)

	results := make(map[int]DequeueResult)
	messagesToUpdate := make([]*models.QueueMessage, 0, len(cmd.Requests))
	leasesToCreate := make([]*models.QueueMessageLease, 0, len(cmd.Requests))
	validRequests := make([]int, 0, len(cmd.Requests))
	cachedMessages := make([]*models.QueueMessage, 0, len(cmd.Requests))
	cachedLeases := make([]*models.QueueMessageLease, 0, len(cmd.Requests))

	for i, req := range cmd.Requests {
		loopStart := time.Now()

		queue, ok := cachedQueues[req.QueueID]
		if !ok {
			q, err := queueRepo.GetQueueById(req.QueueID, now)
			if err != nil || q == nil {
				continue
			}
			queue = q
			cachedQueues[req.QueueID] = queue
		}

		if queue.State != models.QueueActive && queue.State != models.QueueDraining {
			continue
		}
		if queue.MaxDeliveringMessages > 0 && queue.CurrentDeliveringMessages >= queue.MaxDeliveringMessages {
			continue
		}

		partitions, ok := cachedPartitions[req.QueueID]
		if !ok {
			parts, err := queuePartitionRepo.GetNonEmptyPartitionsByQueueID(req.QueueID, now)
			if err != nil || len(parts) == 0 {
				continue
			}
			partitions = parts
			cachedPartitions[req.QueueID] = partitions
		}

		if len(partitions) == 0 {
			continue
		}

		t1 := time.Since(loopStart)

		// Recreate Priority Queue with the *current* state of the cached queue
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
		selectedTask := pq.Dequeue()
		if selectedTask == nil {
			continue
		}
		newCounts, newCurrentPriority := pq.GetState()
		queue.PQProcessedCounts = newCounts
		queue.PQCurrentPriority = newCurrentPriority

		t2 := time.Since(loopStart)

		var selectedPartition *models.QueuePartition
		for idx := range partitions {
			if partitions[idx].ID == selectedTask.ID {
				selectedPartition = &partitions[idx]
				break
			}
		}

		message, err := queueMessageRepo.GetQueueMessageById(selectedPartition.FirstQueueMessageID, now)
		if err != nil || message == nil {
			continue
		}

		t3 := time.Since(loopStart)

		nextMessageID := message.NextQueueMessageID

		selectedPartition.MessagesCount--
		if selectedPartition.MessagesCount <= 0 {
			selectedPartition.MessagesCount = 0
			selectedPartition.FirstQueueMessageID = ""
			selectedPartition.LastQueueMessageID = ""
		} else {
			selectedPartition.FirstQueueMessageID = nextMessageID
		}

		queue.MessagesCount--
		if queue.MessagesCount < 0 {
			queue.MessagesCount = 0
		}
		queue.CurrentDeliveringMessages++

		message.Attempts++
		messagesToUpdate = append(messagesToUpdate, message)

		leaseID := strings.ReplaceAll(message.ID+"-"+req.JobWorkerID, "-", "")
		lease := &models.QueueMessageLease{
			ID:                                leaseID,
			QueueMessageID:                    message.ID,
			WorkerID:                          req.JobWorkerID,
			LeaseStatus:                       models.QueueMessageLeaseStatusActive,
			LeaseUntil:                        now.Add(req.LeaseDuration),
			JobWorkerCapacityPolicyIndexMatch: req.JobWorkerCapacityPolicyIndex,
			CreatedAt:                         now,
		}
		leasesToCreate = append(leasesToCreate, lease)
		validRequests = append(validRequests, i)
		cachedMessages = append(cachedMessages, message)
		cachedLeases = append(cachedLeases, lease)
		
		t4 := time.Since(loopStart)
		if t4 > 2*time.Millisecond {
			fmt.Printf("[PERF] Loop iter took %v (getQueues: %v, pq: %v, getMessage: %v, rest: %v)\n", t4, t1, t2-t1, t3-t2, t4-t3)
		}
	}

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

	for idx, reqIdx := range validRequests {
		msg := cachedMessages[idx]
		ls := cachedLeases[idx]
		msg.NextQueueMessageID = ""
		results[reqIdx] = DequeueResult{
			Message:   *msg,
			Lease:     *ls,
			Pending:   uint64(cachedQueues[cmd.Requests[reqIdx].QueueID].MessagesCount),
			InProcess: uint64(cachedQueues[cmd.Requests[reqIdx].QueueID].CurrentDeliveringMessages),
		}
	}

	// Persist all modified queues and partitions
	for _, q := range cachedQueues {
		queueRepo.UpdateQueue(q, now)
	}
	for _, parts := range cachedPartitions {
		for _, p := range parts {
			queuePartitionRepo.UpdateQueuePartition(&p, now)
		}
	}

	commandResult.Result = BulkDequeueResult{Results: results}
	return *commandResult
}
