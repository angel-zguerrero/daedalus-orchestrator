package workflow_execution

import (
	"encoding/gob"
	"fmt"
	"strings"
	"time"

	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"deadalus-orch/shared/models"
)

func init() {
	gob.Register(HandleJobFailureCommand{})
	gob.Register(HandleJobFailureResult{})
}

type HandleJobFailureCommand struct {
	JobID    string
	WorkerID string
	ErrorMsg string
	CF       string
	CFS      string
}

type HandleJobFailureResult struct {
	JobID           string `json:"jobId"`
	RetriesExceeded bool   `json:"retriesExceeded"`
	CurrentRetries  int32  `json:"currentRetries"`
	MaxRetries      int32  `json:"maxRetries"`
}

func (cmd *HandleJobFailureCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	if cmd.JobID == "" {
		commandResult.Error = "JobID is required"
		return *commandResult
	}

	idFactory := &db.DeterministicIDGeneratorFactory{}

	jobRepo, err := db.NewWorkflowJobRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	job, err := jobRepo.GetWorkflowJobByID(cmd.JobID, now)
	if err != nil || job == nil {
		commandResult.Error = fmt.Sprintf("job not found: %s", cmd.JobID)
		return *commandResult
	}

	if job.Status == models.WorkflowJobStatusCompleted || job.Status == models.WorkflowJobStatusFailed {
		res := HandleJobFailureResult{
			JobID:           job.ID,
			RetriesExceeded: job.Status == models.WorkflowJobStatusFailed,
			CurrentRetries:  job.Retries,
			MaxRetries:      job.MaxRetries,
		}
		commandResult.Result = res
		return *commandResult
	}

	job.Retries++
	job.Error = cmd.ErrorMsg

	if job.MaxRetries <= 0 {
		job.MaxRetries = 3
	}

	jobRepo.UpdateWorkflowJob(job, now)

	isFatalError := strings.Contains(cmd.ErrorMsg, "no native executor found") ||
		strings.Contains(cmd.ErrorMsg, "unsupported") ||
		strings.Contains(cmd.ErrorMsg, "not found in model")

	if job.Retries >= job.MaxRetries || isFatalError {
		job.Status = models.WorkflowJobStatusFailed
		job.CompletedAt = &now
		jobRepo.UpdateWorkflowJob(job, now)

		// Update the corresponding ExecutionToken to Cancelled and the
		// WorkflowExecution to Failed. We do this directly here because
		// CompleteJobCommand short-circuits when it sees job.Status == Failed.
		tokenRepo, tokErr := db.NewExecutionTokenRepository(uow, idFactory, cmd.CF, cmd.CFS)
		if tokErr == nil {
			token, tokGetErr := tokenRepo.GetExecutionTokenByID(job.ExecutionTokenID, now)
			if tokGetErr == nil && token != nil {
				token.Status = models.ExecutionTokenStatusCancelled
				tokenRepo.UpdateExecutionToken(token, now)
			}
		}

		execRepo, execErr := db.NewWorkflowExecutionRepository(uow, idFactory, cmd.CF, cmd.CFS)
		if execErr == nil {
			execution, execGetErr := execRepo.GetWorkflowExecutionByID(job.WorkflowExecutionID, now)
			if execGetErr == nil && execution != nil && execution.Status == models.WorkflowExecutionStatusRunning {
				execution.Status = models.WorkflowExecutionStatusFailed
				execution.Error = cmd.ErrorMsg
				execution.CompletedAt = &now
				execRepo.UpdateWorkflowExecution(execution, now)
			}
		}

		res := HandleJobFailureResult{
			JobID:           job.ID,
			RetriesExceeded: true,
			CurrentRetries:  job.Retries,
			MaxRetries:      job.MaxRetries,
		}
		commandResult.Result = res
		return *commandResult
	}

	job.Status = models.WorkflowJobStatusPending
	job.AssignedWorkerID = ""
	jobRepo.UpdateWorkflowJob(job, now)

	res := HandleJobFailureResult{
		JobID:           job.ID,
		RetriesExceeded: false,
		CurrentRetries:  job.Retries,
		MaxRetries:      job.MaxRetries,
	}
	commandResult.Result = res
	return *commandResult
}
