package vnamespace_command

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"deadalus-orch/shared/models"
	"encoding/gob"
	"time"
)

func init() {
	gob.Register(PaginateVNamespacesWithFilterCommand{})
}

// PaginateVNamespacesWithFilterCommand paginates vnamespaces using the DB-level rules encoded in
// ClaimWorkFilter, pushing inclusion lists and exact exclusions down to the repository layer.
type PaginateVNamespacesWithFilterCommand struct {
	Filter   models.ClaimWorkFilter
	Cursor   string
	PageSize int
	CF       string
	CFS      string
}

func (cmd *PaginateVNamespacesWithFilterCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	vnsRepo, err := db.NewVNamespaceRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	result, err := vnsRepo.PaginateWithClaimWorkFilter(cmd.Filter, cmd.PageSize, cmd.Cursor, now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	commandResult.Result = *result
	return *commandResult
}
