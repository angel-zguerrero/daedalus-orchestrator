package scheduled_job

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"encoding/gob"
	"fmt"
	"time"
)

func init() {
	gob.Register(DeleteScheduledJobCommand{})
}

type DeleteScheduledJobCommand struct {
	ID  string
	CF  string
	CFS string
}

func (cmd *DeleteScheduledJobCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	if cmd.ID == "" {
		commandResult.Error = "ID is required"
		return *commandResult
	}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	repo, err := db.NewScheduledJobRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	job, err := repo.GetScheduledJobByID(cmd.ID, now)
	if err != nil {
		commandResult.Error = fmt.Sprintf("failed to load scheduled job %s: %s", cmd.ID, err.Error())
		return *commandResult
	}
	if job == nil {
		commandResult.Result = true
		return *commandResult
	}

	deleted, err := repo.DeleteScheduledJob(job, now)
	if err != nil {
		commandResult.Error = fmt.Sprintf("failed to delete scheduled job %s: %s", cmd.ID, err.Error())
		return *commandResult
	}

	commandResult.Result = deleted
	return *commandResult
}
