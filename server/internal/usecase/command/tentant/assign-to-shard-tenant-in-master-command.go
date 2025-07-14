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
	gob.Register(AssignToShardTenantInMasterCommand{})
}

// AssignToShardTenantInMasterCommand represents a command to assign one or more tenants to a shard.
type AssignToShardTenantInMasterCommand struct {
	TenantCodes []string
}

func (cmd *AssignToShardTenantInMasterCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
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

		tenantInMaster.Status = models.Assigned
		if _, err := tenantInMasterRepo.UpdateTenantInMaster(tenantInMaster, now); err != nil {
			commandResult.Error = fmt.Sprintf("error updating tenant '%s': %s", tenantCode, err.Error())
			return *commandResult
		}
	}

	commandResult.Result = true
	return *commandResult
}
