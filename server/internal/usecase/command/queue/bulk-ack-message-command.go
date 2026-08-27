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
	gob.Register(BulkAckMessageCommand{})
	gob.Register(BulkAckMessageResult{})
}

type BulkAckMessageResult struct {
	Results []AckMessageResult
}

type BulkAckMessageCommand struct {
	LeaseIDs []string
	CF       string
	CFS      string
}

func (cmd *BulkAckMessageCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	start := time.Now()
	defer func() {
		elapsed := time.Since(start)
		if elapsed > 1*time.Millisecond {
			fmt.Printf("[PERF] BulkAckMessageCommand took %v for %d leases\n", elapsed, len(cmd.LeaseIDs))
		}
	}()

	commandResult := &command.CommandResult{}

	if len(cmd.LeaseIDs) == 0 {
		commandResult.Error = "LeaseIDs list is empty"
		return *commandResult
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

	tenantSummaryRepo, err := db.NewTenantSummaryRepository(uow, idFactory)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	// ── execute ─────────────────────────────────────────────────────────────────
	results := make([]AckMessageResult, len(cmd.LeaseIDs))
	cachedQueues := make(map[string]*models.Queue)
	var deletedCount int64

	for i, leaseID := range cmd.LeaseIDs {
		var res AckMessageResult
		res.Success = false

		if leaseID == "" {
			res.Message = "LeaseID is required"
			results[i] = res
			continue
		}

		lease, err := leaseRepo.GetQueueMessageLeaseByID(leaseID, now)
		if err != nil {
			res.Message = fmt.Sprintf("failed to load lease %s: %s", leaseID, err.Error())
			results[i] = res
			continue
		}
		if lease == nil {
			res.Message = fmt.Sprintf("lease %s not found", leaseID)
			results[i] = res
			continue
		}

		if lease.LeaseStatus != models.QueueMessageLeaseStatusActive {
			res.Message = fmt.Sprintf("lease %s is not active", leaseID)
			results[i] = res
			continue
		}

		message, err := queueMessageRepo.GetQueueMessageById(lease.QueueMessageID, now)
		if err != nil {
			res.Message = fmt.Sprintf("failed to load message %s: %s", lease.QueueMessageID, err.Error())
			results[i] = res
			continue
		}
		if message == nil {
			res.Message = fmt.Sprintf("message %s not found", lease.QueueMessageID)
			results[i] = res
			continue
		}

		// Use cached queue if available to prevent reading stale values from DB
		queue, ok := cachedQueues[message.QueueID]
		if !ok {
			q, err := queueRepo.GetQueueById(message.QueueID, now)
			if err != nil {
				res.Message = fmt.Sprintf("failed to load queue %s: %s", message.QueueID, err.Error())
				results[i] = res
				continue
			}
			if q == nil {
				res.Message = fmt.Sprintf("queue %s not found", message.QueueID)
				results[i] = res
				continue
			}
			queue = q
			cachedQueues[message.QueueID] = queue
		}

		// Decrement counters
		queue.CurrentDeliveringMessages--
		if queue.CurrentDeliveringMessages < 0 {
			queue.CurrentDeliveringMessages = 0
		}

		if _, err = leaseRepo.Delete(leaseID, now); err != nil {
			res.Message = fmt.Sprintf("failed to delete lease %s: %s", leaseID, err.Error())
			results[i] = res
			continue
		}

		if _, err = queueMessageRepo.Delete(message.ID, now); err != nil {
			res.Message = fmt.Sprintf("failed to delete message %s: %s", message.ID, err.Error())
			results[i] = res
			continue
		}

		deletedCount++

		var leaseCreatedAt time.Time = lease.CreatedAt
		if leaseCreatedAt.IsZero() {
			leaseCreatedAt = now
		}

		processingLatencyMs := float64(now.Sub(leaseCreatedAt).Milliseconds())
		queueLatencyMs := float64(leaseCreatedAt.Sub(message.CreatedAt).Milliseconds())

		res.Success = true
		res.Message = "Message acknowledged successfully"
		res.QueueCode = queue.Code
		res.VNamespace = queue.VNamespace
		res.ProcessingLatencyMs = processingLatencyMs
		res.QueueLatencyMs = queueLatencyMs
		results[i] = res
	}

	// Update all cached queues to disk
	for _, q := range cachedQueues {
		if _, err := queueRepo.UpdateQueue(q, now); err != nil {
			commandResult.Error = fmt.Sprintf("failed to update queue %s: %s", q.ID, err.Error())
			return *commandResult
		}
	}

	// Update metrics in results
	for i, res := range results {
		if res.Success {
			// Find the queue that matches to provide final numbers
			for _, q := range cachedQueues {
				if q.Code == res.QueueCode && q.VNamespace == res.VNamespace {
					results[i].Pending = uint64(q.MessagesCount)
					results[i].InProcess = uint64(q.CurrentDeliveringMessages)
					break
				}
			}
		}
	}

	// Update tenant summary
	if deletedCount > 0 {
		err = tenantSummaryRepo.UpdateCounters(cmd.CFS, int(-deletedCount), 0, 0, 0, now)
		if err != nil {
			commandResult.Error = fmt.Sprintf("failed to update tenant summary: %s", err.Error())
			return *commandResult
		}
	}

	commandResult.Result = BulkAckMessageResult{
		Results: results,
	}
	return *commandResult
}
