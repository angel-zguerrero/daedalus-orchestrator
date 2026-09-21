package workflow_execution

import (
	"encoding/gob"
	"fmt"
	"time"

	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"deadalus-orch/shared/models"
)

func init() {
	gob.Register(CompleteJobCommand{})
}

type CompleteJobCommand struct {
	JobID      string
	WorkerID   string
	OutputData map[string]interface{}
	Error      string
	CF         string
	CFS        string
}

func (cmd *CompleteJobCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
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

	tokenRepo, err := db.NewExecutionTokenRepository(uow, idFactory, cmd.CF, cmd.CFS)
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
		commandResult.Result = job
		return *commandResult
	}

	if cmd.Error != "" {
		job.Status = models.WorkflowJobStatusFailed
		job.Error = cmd.Error
	} else {
		job.Status = models.WorkflowJobStatusCompleted
		job.Output = cmd.OutputData
		job.CompletedAt = &now
	}
	job.AssignedWorkerID = cmd.WorkerID
	jobRepo.UpdateWorkflowJob(job, now)

	// Update corresponding ExecutionToken back to Active if job succeeded
	token, err := tokenRepo.GetExecutionTokenByID(job.ExecutionTokenID, now)
	if err == nil && token != nil {
		if cmd.Error == "" {
			token.Status = models.ExecutionTokenStatusActive
			tokenRepo.UpdateExecutionToken(token, now)

			// Advance token to next BPMN node
			advanceCmd := &AdvanceTokenCommand{
				ExecutionID: job.WorkflowExecutionID,
				TokenID:     token.ID,
				OutputData:  cmd.OutputData,
				CF:          cmd.CF,
				CFS:         cmd.CFS,
			}
			return advanceCmd.Execute(uow, now)
		} else {
			token.Status = models.ExecutionTokenStatusCancelled
			tokenRepo.UpdateExecutionToken(token, now)

			// Update WorkflowExecution status to FAILED if no active tokens remain
			activeTokens, _ := tokenRepo.GetActiveTokensByExecutionID(job.WorkflowExecutionID, now)
			if len(activeTokens) == 0 {
				execRepo, err := db.NewWorkflowExecutionRepository(uow, idFactory, cmd.CF, cmd.CFS)
				if err == nil {
					execution, err := execRepo.GetWorkflowExecutionByID(job.WorkflowExecutionID, now)
					if err == nil && execution != nil {
						execution.Status = models.WorkflowExecutionStatusFailed
						execution.Error = cmd.Error
						execRepo.UpdateWorkflowExecution(execution, now)
					}
				}
			}
		}
	}

	commandResult.Result = job
	return *commandResult
}
