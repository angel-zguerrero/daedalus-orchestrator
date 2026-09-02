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
		if summary.ID == "" {
			continue
		}
		// Find the existing tenant by ID, falling back to Code if needed
		tenant, err := tenantRepo.GetTenantInMasterByTenantId(summary.ID, now)
		if err != nil {
			commandResult.Error = err.Error()
			return *commandResult
		}
		if tenant == nil {
			tenant, err = tenantRepo.GetTenantInMasterByTenantCode(summary.ID, now)
			if err != nil {
				commandResult.Error = err.Error()
				return *commandResult
			}
		}

		if tenant == nil {
			continue // Skip if tenant doesn't exist
		}

		if tenant.ID == "" {
			tenant.ID = summary.ID
		}

		// Update the tenant with the summary counters
		tenant.ExchangesCount = summary.ExchangesCount
		tenant.QueuesCount = summary.QueuesCount
		tenant.BindingsCount = summary.BindingsCount
		tenant.MessagesCount = summary.MessagesCount

		// Self-healing: if the summary says we have messages but the master node thinks we don't,
		// we force it to active. We do NOT heal the other way (MessagesCount == 0 -> HasMessages = false)
		// because the summary is asynchronous and might be stale (e.g. read 0 just before a message was enqueued).
		if tenant.MessagesCount > 0 && !tenant.HasMessages {
			tenant.HasMessages = true
		}

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
