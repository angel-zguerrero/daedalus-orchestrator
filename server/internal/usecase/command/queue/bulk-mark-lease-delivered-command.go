package queue

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"encoding/gob"
	"fmt"
	"time"
)

func init() {
	gob.Register(BulkMarkLeaseDeliveredCommand{})
	gob.Register(BulkMarkLeaseDeliveredResult{})
}

type BulkMarkLeaseDeliveredResult struct {
	Success bool
}

type BulkMarkLeaseDeliveredCommand struct {
	LeaseIDs []string
	CF       string
	CFS      string
}

func (cmd *BulkMarkLeaseDeliveredCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	start := time.Now()
	defer func() {
		elapsed := time.Since(start)
		if elapsed > 1*time.Millisecond {
			fmt.Printf("[PERF] BulkMarkLeaseDeliveredCommand took %v for %d leases\n", elapsed, len(cmd.LeaseIDs))
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

	for _, leaseID := range cmd.LeaseIDs {
		if leaseID == "" {
			continue
		}

		lease, err := leaseRepo.GetQueueMessageLeaseByID(leaseID, now)
		if err != nil || lease == nil {
			continue // ignore errors for missing leases, maybe they were already acked/deleted
		}

		if lease.IsDelivered {
			continue
		}

		lease.IsDelivered = true

		if _, err := leaseRepo.UpdateQueueMessageLease(lease, now); err != nil {
			commandResult.Error = fmt.Sprintf("failed to update lease %s: %s", leaseID, err.Error())
			return *commandResult
		}
	}

	commandResult.Result = BulkMarkLeaseDeliveredResult{
		Success: true,
	}
	return *commandResult
}
