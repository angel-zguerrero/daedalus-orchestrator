package tenant_summary_command

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
	TenantSummary models.TenantSummary
}

func (cmd *UpdateTenantSummaryCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	tenantSummaryRepo, err := db.NewTenantSummaryRepository(uow, idFactory)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	// Update timestamps
	cmd.TenantSummary.UpdatedAt = now

	// Check if tenant summary exists
	existing, err := tenantSummaryRepo.GetTenantSummaryById(cmd.TenantSummary.ID, now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	if existing == nil {
		// Create new tenant summary if it doesn't exist
		cmd.TenantSummary.CreatedAt = now
		_, err = tenantSummaryRepo.CreateTenantSummary(&cmd.TenantSummary, now)
		if err != nil {
			commandResult.Error = err.Error()
			return *commandResult
		}
	} else {
		// Update existing tenant summary
		cmd.TenantSummary.CreatedAt = existing.CreatedAt // Preserve creation date
		_, err = tenantSummaryRepo.UpdateTenantSummary(&cmd.TenantSummary, now)
		if err != nil {
			commandResult.Error = err.Error()
			return *commandResult
		}
	}

	commandResult.Result = cmd.TenantSummary
	return *commandResult
}
