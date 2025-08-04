package vnamespace_command

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"deadalus-orch/shared/models"
	"encoding/gob"
	"time"
)

func init() {
	gob.Register(PaginateVNamespacesCommand{})
	gob.Register(db.FindResult[models.VNamespace]{})
}

type PaginateVNamespacesCommand struct {
	Query    string
	Cursor   string
	PageSize int
	CF       string
	CFS      string
}

func (cmd *PaginateVNamespacesCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	vNamespaceRepo, err := db.NewVNamespaceRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	findResult, err := vNamespaceRepo.Paginate(cmd.Query, cmd.PageSize, cmd.Cursor, now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	commandResult.Result = *findResult
	return *commandResult
}
