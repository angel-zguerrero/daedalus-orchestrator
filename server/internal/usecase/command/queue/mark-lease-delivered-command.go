package queue

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"encoding/gob"
	"fmt"
	"time"
)

func init() {
	gob.Register(MarkLeaseDeliveredCommand{})
	gob.Register(MarkLeaseDeliveredResult{})
}

type MarkLeaseDeliveredResult struct {
	Success bool
}

type MarkLeaseDeliveredCommand struct {
	LeaseID string
	CF      string
	CFS     string
}

func (cmd *MarkLeaseDeliveredCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	if cmd.LeaseID == "" {
		commandResult.Error = "LeaseID is required"
		return *commandResult
	}

	idFactory := &db.DeterministicIDGeneratorFactory{}

	leaseRepo, err := db.NewQueueMessageLeaseRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	lease, err := leaseRepo.GetQueueMessageLeaseByID(cmd.LeaseID, now)
	if err != nil {
		commandResult.Error = fmt.Sprintf("failed to load lease %s: %s", cmd.LeaseID, err.Error())
		return *commandResult
	}
	if lease == nil {
		commandResult.Error = fmt.Sprintf("lease %s not found", cmd.LeaseID)
		return *commandResult
	}

	lease.IsDelivered = true

	if _, err := leaseRepo.UpdateQueueMessageLease(lease, now); err != nil {
		commandResult.Error = fmt.Sprintf("failed to update lease %s: %s", cmd.LeaseID, err.Error())
		return *commandResult
	}

	commandResult.Result = MarkLeaseDeliveredResult{Success: true}
	return *commandResult
}
