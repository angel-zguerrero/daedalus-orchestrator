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
	gob.Register(AckMessageCommand{})
	gob.Register(AckMessageResult{})
}

// AckMessageResult is the value returned in CommandResult.Result after a successful ack.
type AckMessageResult struct {
	Success bool
	Message string
}

// AckMessageCommand acknowledges a message by deleting its lease and decrementing
// the queue's CurrentDeliveringMessages counter.
type AckMessageCommand struct {
	// LeaseID is the ID of the lease to acknowledge.
	LeaseID string
	CF      string
	CFS     string
}

func (cmd *AckMessageCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	if cmd.LeaseID == "" {
		commandResult.Error = "LeaseID is required"
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

	// ── 1. load lease ────────────────────────────────────────────────────────────

	lease, err := leaseRepo.GetQueueMessageLeaseByID(cmd.LeaseID, now)
	if err != nil {
		commandResult.Error = fmt.Sprintf("failed to load lease %s: %s", cmd.LeaseID, err.Error())
		return *commandResult
	}
	if lease == nil {
		commandResult.Error = fmt.Sprintf("lease %s not found", cmd.LeaseID)
		return *commandResult
	}

	// Verify the lease is active
	if lease.LeaseStatus != models.QueueMessageLeaseStatusActive {
		commandResult.Error = fmt.Sprintf("lease %s is not active (status: %s)", cmd.LeaseID, lease.LeaseStatus)
		return *commandResult
	}

	// ── 2. load message to get queue ID ──────────────────────────────────────────

	message, err := queueMessageRepo.GetQueueMessageById(lease.QueueMessageID, now)
	if err != nil {
		commandResult.Error = fmt.Sprintf("failed to load message %s: %s", lease.QueueMessageID, err.Error())
		return *commandResult
	}
	if message == nil {
		commandResult.Error = fmt.Sprintf("message %s not found", lease.QueueMessageID)
		return *commandResult
	}

	// ── 3. load queue ────────────────────────────────────────────────────────────

	queue, err := queueRepo.GetQueueById(message.QueueID, now)
	if err != nil {
		commandResult.Error = fmt.Sprintf("failed to load queue %s: %s", message.QueueID, err.Error())
		return *commandResult
	}
	if queue == nil {
		commandResult.Error = fmt.Sprintf("queue %s not found", message.QueueID)
		return *commandResult
	}

	// ── 4. decrement queue delivering messages counter ───────────────────────────

	queue.CurrentDeliveringMessages--
	if queue.CurrentDeliveringMessages < 0 {
		queue.CurrentDeliveringMessages = 0
	}

	if _, err = queueRepo.UpdateQueue(queue, now); err != nil {
		commandResult.Error = fmt.Sprintf("failed to update queue %s: %s", message.QueueID, err.Error())
		return *commandResult
	}

	// ── 5. delete the lease ──────────────────────────────────────────────────────

	if _, err = leaseRepo.Delete(cmd.LeaseID, now); err != nil {
		commandResult.Error = fmt.Sprintf("failed to delete lease %s: %s", cmd.LeaseID, err.Error())
		return *commandResult
	}

	// ── 6. delete the message ────────────────────────────────────────────────────

	if _, err = queueMessageRepo.Delete(message.ID, now); err != nil {
		commandResult.Error = fmt.Sprintf("failed to delete message %s: %s", message.ID, err.Error())
		return *commandResult
	}

	// ── 7. update tenant summary ─────────────────────────────────────────────────

	err = tenantSummaryRepo.UpdateCounters(cmd.CFS, -1, 0, 0, 0, now)
	if err != nil {
		commandResult.Error = fmt.Sprintf("failed to update tenant summary: %s", err.Error())
		return *commandResult
	}

	// ── 8. return result ─────────────────────────────────────────────────────────

	commandResult.Result = AckMessageResult{
		Success: true,
		Message: "Message acknowledged successfully",
	}
	return *commandResult
}
