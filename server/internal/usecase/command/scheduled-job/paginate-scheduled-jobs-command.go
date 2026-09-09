package scheduled_job

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"deadalus-orch/shared/models"
	"encoding/gob"
	"fmt"
	"time"
)

func init() {
	gob.Register(PaginateScheduledJobsCommand{})
}

type PaginateScheduledJobsCommand struct {
	PageSize   int
	Cursor     string
	VNamespace string
	CF         string
	CFS        string
}

func (cmd *PaginateScheduledJobsCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	if cmd.PageSize <= 0 {
		cmd.PageSize = 50
	}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	repo, err := db.NewScheduledJobRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	result, err := repo.PaginateScheduledJobs(cmd.PageSize, cmd.Cursor, cmd.VNamespace, now)
	if err != nil {
		commandResult.Error = fmt.Sprintf("failed to paginate scheduled jobs: %s", err.Error())
		return *commandResult
	}

	if result.Entities == nil {
		result.Entities = []models.ScheduledJob{}
	}

	commandResult.Result = *result
	return *commandResult
}
