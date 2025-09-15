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
	gob.Register(GetTenantSummaryCommand{})
}

type GetTenantSummaryCommand struct {
	TenantCode string
}

func (cmd *GetTenantSummaryCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	fmt.Println("Executing GetTenantSummaryCommand for TenantId:", cmd.TenantCode)

	idFactory := &db.DeterministicIDGeneratorFactory{}
	tenantRepo, err := db.NewTenantInMasterRepository(uow, idFactory)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	// Get tenant by ID
	tenant, err := tenantRepo.GetTenantInMasterByTenantCode(cmd.TenantCode, now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	if tenant == nil {
		commandResult.Error = fmt.Sprintf("tenant not found for ID: %s", cmd.TenantCode)
		return *commandResult
	}

	// Create a TenantSummary object with the data from TenantInMaster
	tenantSummary := models.TenantSummary{
		ID:             tenant.ID,
		ExchangesCount: tenant.ExchangesCount,
		QueuesCount:    tenant.QueuesCount,
		MessagesCount:  tenant.MessagesCount,
		CreatedAt:      tenant.CreatedAt,
		UpdatedAt:      tenant.UpdatedAt,
	}

	commandResult.Result = tenantSummary
	return *commandResult
}
