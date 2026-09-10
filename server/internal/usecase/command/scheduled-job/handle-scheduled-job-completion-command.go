package scheduled_job

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"deadalus-orch/shared/models"
	"encoding/gob"
	"time"
)

func init() {
	gob.Register(HandleScheduledJobCompletionCommand{})
}

type HandleScheduledJobCompletionCommand struct {
	ScheduledJobID string
	ExecutionID    string
	QueueID        string
	Status         string
	CF             string
	CFS            string
}

func (cmd *HandleScheduledJobCompletionCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	if cmd.ScheduledJobID == "" {
		commandResult.Result = true
		return *commandResult
	}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	repo, err := db.NewScheduledJobRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	if cmd.ExecutionID != "" && cmd.QueueID != "" {
		trackerStatus := models.ScheduledJobTrackerCompleted
		if cmd.Status == string(models.ScheduledJobTrackerExhausted) {
			trackerStatus = models.ScheduledJobTrackerExhausted
		}
		err = repo.UpdateTrackerAndHandleCompletion(cmd.ScheduledJobID, cmd.ExecutionID, cmd.QueueID, trackerStatus, now)
		if err != nil {
			commandResult.Error = err.Error()
			return *commandResult
		}
		commandResult.Result = true
		return *commandResult
	}

	err = repo.HandleCompletion(cmd.ScheduledJobID, now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	commandResult.Result = true
	return *commandResult
}
