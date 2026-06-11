package tenant_command

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"deadalus-orch/shared/models"
	"encoding/gob"
	"time"
)

func init() {
	gob.Register(UpdateTenantSummaryCommand{})
}

type UpdateTenantSummaryCommand struct {
	TenantSummaries []models.TenantSummary
}

func (cmd *UpdateTenantSummaryCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	tenantRepo, err := db.NewTenantInMasterRepository(uow, idFactory)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	for _, summary := range cmd.TenantSummaries {
		// Find the existing tenant by ID (match TenantSummary.ID with TenantInMaster.ID)
		tenant, err := tenantRepo.GetTenantInMasterByTenantId(summary.ID, now)
		if err != nil {
			commandResult.Error = err.Error()
			return *commandResult
		}

		if tenant == nil {
			continue // Skip if tenant doesn't exist
		}

		// Update the tenant with the summary counters
		tenant.ExchangesCount = summary.ExchangesCount
		tenant.QueuesCount = summary.QueuesCount
		tenant.BindingsCount = summary.BindingsCount
		tenant.MessagesCount = summary.MessagesCount
		tenant.HasMessages = summary.HasMessages

		// Save the updated tenant
		_, err = tenantRepo.UpdateTenantInMaster(tenant, now)
		if err != nil {
			commandResult.Error = err.Error()
			return *commandResult
		}
	}

	commandResult.Result = len(cmd.TenantSummaries)
	return *commandResult
}
