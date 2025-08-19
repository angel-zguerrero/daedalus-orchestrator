package tenant_summary_command

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"deadalus-orch/shared/models"
	"encoding/gob"
	"time"
)

func init() {
	gob.Register(PaginateTenantUpdatedAtFromCommand{})
	gob.Register(db.FindResult[models.TenantSummary]{})
}

type PaginateTenantUpdatedAtFromCommand struct {
	LastUpdatedAt time.Time
	PageSize      int
	Cursor        string
}

func (cmd *PaginateTenantUpdatedAtFromCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	tenantSummaryRepo, err := db.NewTenantSummaryRepository(uow, idFactory)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	findResult, err := tenantSummaryRepo.PaginateTenantUpdatedAtFrom(cmd.LastUpdatedAt, cmd.PageSize, cmd.Cursor, now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	commandResult.Result = *findResult
	return *commandResult
}
