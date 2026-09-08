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
	gob.Register(CreateScheduledJobCommand{})
}

type CreateScheduledJobCommand struct {
	ScheduledJob models.ScheduledJob
	CF           string
	CFS          string
}

func (cmd *CreateScheduledJobCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	if cmd.ScheduledJob.TenantID == "" {
		commandResult.Error = "TenantID is required"
		return *commandResult
	}

	if cmd.ScheduledJob.TargetType == "" || cmd.ScheduledJob.TargetID == "" {
		commandResult.Error = "TargetType and TargetID are required"
		return *commandResult
	}

	idFactory := &db.DeterministicIDGeneratorFactory{}

	repo, err := db.NewScheduledJobRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	job := cmd.ScheduledJob

	if job.ID == "" {
		defaultIDFactory := &db.DefaultIDGeneratorFactory{}
		job.ID = defaultIDFactory.GenerateID()
	}

	job.State = models.ScheduledJobIdle

	id, err := repo.CreateScheduledJob(&job, now)
	if err != nil {
		commandResult.Error = fmt.Sprintf("failed to create scheduled job: %s", err.Error())
		return *commandResult
	}

	job.ID = id
	commandResult.Result = job
	return *commandResult
}
