package tenant_command

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"encoding/gob"
	"time"
)

func init() {
	gob.Register(MarkTenantInactiveCommand{})
}

type MarkTenantInactiveCommand struct {
	TenantID string
}

func (cmd *MarkTenantInactiveCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	tenantRepo, err := db.NewTenantInMasterRepository(uow, idFactory)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	tenant, err := tenantRepo.GetTenantInMasterByTenantId(cmd.TenantID, now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	if tenant == nil {
		commandResult.Error = "Tenant not found"
		return *commandResult
	}

	if !tenant.HasMessages {
		commandResult.Result = false // Already inactive
		return *commandResult
	}

	tenant.HasMessages = false
	success, err := tenantRepo.UpdateTenantInMaster(tenant, now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	commandResult.Result = success
	return *commandResult
}
