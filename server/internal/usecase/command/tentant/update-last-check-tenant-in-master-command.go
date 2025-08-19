package tenant_command

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"encoding/gob"
	"fmt"
	"time"
)

func init() {
	gob.Register(UpdateLastCheckTenantInMasterCommand{})
}

// UpdateLastCheckTenantInMasterCommand represents a command to update the LastCheckUpdatedAt field for one or more tenants.
type UpdateLastCheckTenantInMasterCommand struct {
	TenantCodes        []string
	LastCheckUpdatedAt time.Time
}

func (cmd *UpdateLastCheckTenantInMasterCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	tenantInMasterRepo, err := db.NewTenantInMasterRepository(uow, idFactory)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	for _, tenantCode := range cmd.TenantCodes {
		tenantInMaster, err := tenantInMasterRepo.GetTenantInMasterByTenantCode(tenantCode, now)
		if err != nil {
			commandResult.Error = fmt.Sprintf("error retrieving tenant '%s': %s", tenantCode, err.Error())
			return *commandResult
		}

		if tenantInMaster == nil {
			commandResult.Error = fmt.Sprintf("tenant '%s' not found", tenantCode)
			return *commandResult
		}

		// Only update the LastCheckUpdatedAt field, leave all other fields as they are
		tenantInMaster.LastCheckUpdatedAt = cmd.LastCheckUpdatedAt

		if _, err := tenantInMasterRepo.UpdateTenantInMaster(tenantInMaster, now); err != nil {
			commandResult.Error = fmt.Sprintf("error updating tenant '%s': %s", tenantCode, err.Error())
			return *commandResult
		}
	}

	commandResult.Result = true
	return *commandResult
}
