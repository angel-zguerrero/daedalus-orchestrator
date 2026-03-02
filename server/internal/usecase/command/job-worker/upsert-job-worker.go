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
	JobWorkers     []models.JobWorker
	ApplyHeartbeat bool
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

		// Look for existing JobWorker by ID (primary upsert strategy)
		existing, err := repo.GetJobWorkerById(jobWorker.ID, now)
		if err != nil {
			commandResult.Error = err.Error()
			return *commandResult
		}

		// Set TTL from configuration (converted to seconds)
		// Using NodeSchedulerTTL as the shared TTL until a dedicated JobWorkerTTL is defined
		jobWorker.TTL = config.GlobalConfiguration.NodeSchedulerTTL * 60

		if existing != nil {
			// Keep immutable fields from existing record
			jobWorker.ID = existing.ID
			jobWorker.Name = existing.Name
			jobWorker.CreatedAt = existing.CreatedAt

			if cmd.ApplyHeartbeat {
				jobWorker.LastHeartbeat = now
			}

			// Merge Information: existing values are overridden by incoming ones
			if existing.Information != nil && jobWorker.Information == nil {
				jobWorker.Information = existing.Information
			}

			// Heartbeat logic:
			// If LastHeartbeat in command is zero, use the existing one
			if jobWorker.LastHeartbeat.IsZero() {
				jobWorker.LastHeartbeat = existing.LastHeartbeat
			}

			// If last heartbeat is more than 1 minute ago, mark as disconnected
			if jobWorker.LastHeartbeat.UnixNano() < now.Add(-1*time.Minute).UnixNano() {
				jobWorker.ConnectionStatus = models.JobWorkerConnectionStatusDisconnected
			} else {
				// We don't want to force "connected" if the command didn't intend to.
				// But usually an upsert with a fresh heartbeat means connected.
				// If the command came with a fresh heartbeat (not zero), it's connected.
				if !jobWorker.LastHeartbeat.IsZero() {
					jobWorker.ConnectionStatus = models.JobWorkerConnectionStatusConnected
				} else {
					jobWorker.ConnectionStatus = existing.ConnectionStatus
				}
			}

			_, err = repo.UpdateJobWorker(&jobWorker, now)
		} else {
			// New worker initialization
			if jobWorker.LastHeartbeat.IsZero() {
				jobWorker.LastHeartbeat = now
			}
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
