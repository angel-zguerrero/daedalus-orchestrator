package command

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/shared/models"
	"encoding/gob"
	"time"
)

func init() {
	gob.Register(PaginateTenantsCommand{})
	gob.Register(db.FindResult[models.TenantInMaster]{})

}

// PaginateTenantsCommand represents a command to authenticate a user.
type PaginateTenantsCommand struct {
	Cursor   string
	PageSize int
}

func (cmd *PaginateTenantsCommand) Execute(uow *db.UnitOfWork, now time.Time) CommandResult {
	commandResult := &CommandResult{}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	tenantInMasterRepo, err := db.NewTenantInMasterRepository(uow, idFactory) // Passing nil for IDGeneratorFactory
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	tenantInMasterFound, err := tenantInMasterRepo.PaginateTenant(cmd.PageSize, cmd.Cursor, now)

	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}
	commandResult.Result = tenantInMasterFound

	return *commandResult
}
