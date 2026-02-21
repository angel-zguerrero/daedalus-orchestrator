package job_worker

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/pkg/config"
	"deadalus-orch/server/internal/usecase/command"
	"deadalus-orch/shared/models"
	"encoding/gob"
	"time"
)

func init() {
	gob.Register(UpsertJobWorkerCommand{})
	gob.Register(models.JobWorker{})
	gob.Register([]models.JobWorker{})
}

type UpsertJobWorkerCommand struct {
	JobWorkers []models.JobWorker
}

func (cmd *UpsertJobWorkerCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	repo, err := db.NewJobWorkerRepository(uow, idFactory)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	var resultJobWorkers []models.JobWorker

	for _, jobWorker := range cmd.JobWorkers {

		// Validate that name is not empty
		if jobWorker.Name == "" {
			commandResult.Error = "JobWorker name is required"
			return *commandResult
		}

		// Look for existing JobWorker by name (primary upsert strategy)
		existing, err := repo.GetJobWorkerByName(jobWorker.Name, now)
		if err != nil {
			commandResult.Error = err.Error()
			return *commandResult
		}

		// Set TTL from configuration (converted to seconds)
		// Using NodeSchedulerTTL as the shared TTL until a dedicated JobWorkerTTL is defined
		jobWorker.TTL = config.GlobalConfiguration.NodeSchedulerTTL * 60

		// Set LastHeartbeat to now (always refreshed on ClaimWork)
		jobWorker.LastHeartbeat = now

		if existing != nil {
			// Keep immutable fields from existing record
			jobWorker.ID = existing.ID
			jobWorker.Name = existing.Name
			jobWorker.CreatedAt = existing.CreatedAt

			// Merge Information: existing values are overridden by incoming ones
			if existing.Information != nil && jobWorker.Information == nil {
				jobWorker.Information = existing.Information
			}

			jobWorker.ConnectionStatus = models.JobWorkerConnectionStatusConnected

			_, err = repo.UpdateJobWorker(&jobWorker, now)
		} else {
			jobWorker.ConnectionStatus = models.JobWorkerConnectionStatusConnected
			_, err = repo.CreateJobWorker(&jobWorker, now)
		}

		if err != nil {
			commandResult.Error = err.Error()
			return *commandResult
		}

		resultJobWorkers = append(resultJobWorkers, jobWorker)
	}

	commandResult.Result = resultJobWorkers
	return *commandResult
}
