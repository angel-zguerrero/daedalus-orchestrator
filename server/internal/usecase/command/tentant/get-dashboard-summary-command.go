package tenant_command

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"deadalus-orch/shared/models"
	"encoding/gob"
	"fmt"
	"time"
)

func init() {
	gob.Register(GetDashboardSummaryCommand{})
	gob.Register(models.DashboardSummary{})
}

// GetDashboardSummaryCommand is a read-only query command executed against the master node
// that retrieves the single global DashboardSummary record.
type GetDashboardSummaryCommand struct{}

func (cmd *GetDashboardSummaryCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	dashboardRepo, err := db.NewDashboardSummaryRepository(uow, idFactory)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	summary, err := dashboardRepo.GetDashboardSummary(now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	if summary == nil {
		commandResult.Error = fmt.Sprintf("dashboard summary not found")
		return *commandResult
	}

	commandResult.Result = *summary
	return *commandResult
}
