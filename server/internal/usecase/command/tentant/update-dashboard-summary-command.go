package tenant_command

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"deadalus-orch/shared/models"
	"encoding/gob"
	"time"
)

func init() {
	gob.Register(UpdateDashboardSummaryCommand{})
}

// UpdateDashboardSummaryCommand is a FSM write command that aggregates counters from all
// TenantInMaster records (paginated, 100 per batch) and writes the result as the
// global DashboardSummary in the master node.
type UpdateDashboardSummaryCommand struct {
	Summary models.DashboardSummary
}

func (cmd *UpdateDashboardSummaryCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	dashboardRepo, err := db.NewDashboardSummaryRepository(uow, idFactory)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	if err := dashboardRepo.UpsertDashboardSummary(&cmd.Summary, now); err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	commandResult.Result = cmd.Summary
	return *commandResult
}
