package queue

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"deadalus-orch/shared/models"
	"encoding/gob"
	"time"
)

func init() {
	gob.Register(PaginateQueuesWithFilterCommand{})
}

// PaginateQueuesWithFilterCommand paginates queues using the DB-level rules encoded in
// ClaimWorkFilter. Only queues with MessagesCount > 0 are returned. VNamespace filters
// are also applied at the DB level. ExcludeQueuePatterns are handled in the repository as
// an in-memory post-filter (no NOT LIKE support in the query DSL).
type PaginateQueuesWithFilterCommand struct {
	Filter     models.ClaimWorkFilter
	Cursor     string
	PageSize   int
	CF         string
	CFS        string
}

func (cmd *PaginateQueuesWithFilterCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	queueRepo, err := db.NewQueueRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	result, err := queueRepo.PaginateWithClaimWorkFilter(cmd.Filter, cmd.PageSize, cmd.Cursor, now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	commandResult.Result = *result
	return *commandResult
}
