package scheduled_job

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/pkg/utils"
	"deadalus-orch/server/internal/usecase/command"
	"deadalus-orch/shared/models"
	"encoding/gob"
	"fmt"
	"time"
)

func init() {
	gob.Register(HandleScheduledJobCompletionCommand{})
}

type HandleScheduledJobCompletionCommand struct {
	ScheduledJobID string
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

	job, err := repo.GetScheduledJobByID(cmd.ScheduledJobID, now)
	if err != nil {
		commandResult.Error = fmt.Sprintf("failed to get scheduled job %s: %s", cmd.ScheduledJobID, err.Error())
		return *commandResult
	}

	if job == nil {
		commandResult.Result = true
		return *commandResult
	}

	oldNextRunAt := job.NextRunAt

	if job.Type == models.ScheduledJobOneOff {
		_, err := repo.DeleteScheduledJob(job, now)
		if err != nil {
			commandResult.Error = fmt.Sprintf("failed to delete completed OneOff scheduled job %s: %s", job.ID, err.Error())
			return *commandResult
		}
		commandResult.Result = true
		return *commandResult
	}

	if job.Type == models.ScheduledJobRecurring {
		nextRunAt, err := utils.CalculateNextRunAt(
			job.Type,
			job.RunAt,
			job.RunAfter,
			job.Every,
			job.CronExpression,
			now,
		)
		if err != nil {
			commandResult.Error = fmt.Sprintf("failed to calculate next run time for recurring job %s: %s", job.ID, err.Error())
			return *commandResult
		}

		job.NextRunAt = nextRunAt
		job.State = models.ScheduledJobIdle

		_, err = repo.UpdateScheduledJobStateAndRunAt(job, &oldNextRunAt, now)
		if err != nil {
			commandResult.Error = fmt.Sprintf("failed to reset recurring scheduled job %s to idle: %s", job.ID, err.Error())
			return *commandResult
		}

		commandResult.Result = true
		return *commandResult
	}

	commandResult.Result = true
	return *commandResult
}
