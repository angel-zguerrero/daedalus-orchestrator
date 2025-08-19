package tenant_command

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"encoding/gob"
	"time"
)

func init() {
	gob.Register(UpdateLastCheckTenantInMasterCommand{})
}

type UpdateLastCheckTenantInMasterCommand struct {
	TenantId           string
	LastCheckUpdatedAt time.Time
}

func (cmd *UpdateLastCheckTenantInMasterCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	tenantRepo, err := db.NewTenantInMasterRepository(uow, idFactory)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	// Find the existing tenant by ID
	tenant, err := tenantRepo.GetTenantInMasterByTenantId(cmd.TenantId, now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	if tenant == nil {
		commandResult.Error = "Tenant not found"
		return *commandResult
	}

	// Update the LastCheckUpdatedAt field
	tenant.LastCheckUpdatedAt = cmd.LastCheckUpdatedAt

	// Save the updated tenant
	_, err = tenantRepo.UpdateTenantInMaster(tenant, now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	commandResult.Result = "Tenant last check time updated successfully"
	return *commandResult
}
