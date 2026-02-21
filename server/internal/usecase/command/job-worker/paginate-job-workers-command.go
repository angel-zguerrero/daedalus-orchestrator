package job_worker

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"deadalus-orch/shared/models"
	"encoding/gob"
	"time"
)

func init() {
	gob.Register(PaginateJobWorkersCommand{})
	gob.Register(db.FindResult[models.JobWorker]{})
}

// PaginateJobWorkersCommand represents a command to paginate job workers.
type PaginateJobWorkersCommand struct {
	Cursor   string
	PageSize int
	Q        string
}

func (cmd *PaginateJobWorkersCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	repo, err := db.NewJobWorkerRepository(uow, idFactory)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	found, err := repo.Paginate(cmd.Q, cmd.PageSize, cmd.Cursor, now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}
	commandResult.Result = found

	return *commandResult
}
