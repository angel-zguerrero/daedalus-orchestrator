package command

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/shared/models"
	"encoding/gob"
	"time"
)

func init() {
	gob.Register(AssignToShardTenantInMasterCommand{})
}

// AssignToShardTenantInMasterCommand represents a command to authenticate a user.
type AssignToShardTenantInMasterCommand struct {
	TenantCode string
}

func (cmd *AssignToShardTenantInMasterCommand) Execute(uow *db.UnitOfWork, now time.Time) CommandResult {
	commandResult := &CommandResult{}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	tenantInMasterRepo, err := db.NewTenantInMasterRepository(uow, idFactory) // Passing nil for IDGeneratorFactory
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	tenantInMasterFound, err := tenantInMasterRepo.GetTenantInMasterByTenantCode(cmd.TenantCode, now)

	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	if tenantInMasterFound == nil {
		commandResult.Error = "tenant not found"
		return *commandResult
	}

	tenantInMasterFound.Status = models.Assigned
	_, err = tenantInMasterRepo.UpdateTenantInMaster(tenantInMasterFound, now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	commandResult.Result = true

	return *commandResult
}
