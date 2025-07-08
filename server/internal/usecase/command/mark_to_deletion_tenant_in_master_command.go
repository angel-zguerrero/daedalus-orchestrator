package command

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/shared/models"
	"encoding/gob"
	"time"
)

func init() {
	gob.Register(MarkToDeletionTenantInMasterCommand{})
}

// MarkToDeletionTenantInMasterCommand represents a command to authenticate a user.
type MarkToDeletionTenantInMasterCommand struct {
	TenantId string
}

func (cmd *MarkToDeletionTenantInMasterCommand) Execute(uow *db.UnitOfWork, now time.Time) CommandResult {
	commandResult := &CommandResult{}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	tenantInMasterRepo, err := db.NewTenantInMasterRepository(uow, idFactory) // Passing nil for IDGeneratorFactory
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	tenantInMasterFound, err := tenantInMasterRepo.GetTenantInMasterByTenantId(cmd.TenantId, now)

	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	if tenantInMasterFound == nil {
		commandResult.Error = "tenant not found"
		return *commandResult
	}

	tenantInMasterFound.Status = models.PendingForDeletion
	_, err = tenantInMasterRepo.UpdateTenantInMaster(*tenantInMasterFound, now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	commandResult.Result = true

	return *commandResult
}
