package tenant_command

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"deadalus-orch/shared/models"
	"encoding/gob"
	"time"
)

func init() {
	gob.Register(PaginateTenantsWithFilterCommand{})
	gob.Register(models.ClaimWorkFilter{})
}

// PaginateTenantsWithFilterCommand paginates tenants using the DB-level rules encoded in
// ClaimWorkFilter, pushing inclusion lists, exact exclusions, and the MessagesCount > 0
// guard down to the repository layer.
type PaginateTenantsWithFilterCommand struct {
	Filter   models.ClaimWorkFilter
	Cursor   string
	PageSize int
}

func (cmd *PaginateTenantsWithFilterCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	tenantRepo, err := db.NewTenantInMasterRepository(uow, idFactory)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	result, err := tenantRepo.PaginateWithClaimWorkFilter(cmd.Filter, cmd.PageSize, cmd.Cursor, now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	commandResult.Result = *result
	return *commandResult
}
