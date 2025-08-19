package tenant_summary_command

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"deadalus-orch/shared/models"
	"encoding/gob"
	"fmt"
	"time"
)

func init() {
	gob.Register(GetTenantSummaryCommand{})
	gob.Register(models.TenantSummary{})
	gob.Register([]models.TenantSummary{})
}

type GetTenantSummaryCommand struct {
	TenantId string
}

func (cmd *GetTenantSummaryCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	fmt.Println("Executing GetTenantSummaryCommand for TenantId:", cmd.TenantId)

	idFactory := &db.DeterministicIDGeneratorFactory{}
	tenantSummaryRepo, err := db.NewTenantSummaryRepository(uow, idFactory)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	// Get tenant summary by tenant ID - only look in tenant-summaries table
	tenantSummary, err := tenantSummaryRepo.GetTenantSummaryById(cmd.TenantId, now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	if tenantSummary == nil {
		commandResult.Error = fmt.Sprintf("tenant summary not found for tenant ID: %s", cmd.TenantId)
		return *commandResult
	}

	commandResult.Result = *tenantSummary
	return *commandResult
}
