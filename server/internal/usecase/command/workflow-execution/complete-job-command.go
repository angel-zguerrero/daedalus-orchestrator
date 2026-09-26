package workflow_execution

import (
	"context"
	"encoding/gob"
	"fmt"
	"strings"
	"time"

	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/pkg/activity"
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

	isScriptJob := strings.EqualFold(job.ActivityType, "io.camunda.connectors.ScriptTask.v1") ||
		strings.EqualFold(job.ActivityType, "scriptTask") ||
		(job.Input != nil && job.Input["script"] != nil)

	// If a ScriptTask job is completed without pre-computed OutputData (e.g. direct command execution),
	// execute the isolated JavaScript engine so its mandatory return is evaluated and mapped.
	if cmd.Error == "" && cmd.OutputData == nil && isScriptJob {
		execCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		scriptExec := &activity.ScriptExecutor{}
		out, execErr := scriptExec.Execute(execCtx, job.Input)
		cancel()
		if execErr != nil {
			cmd.Error = execErr.Error()
		} else {
			cmd.OutputData = out
		}
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

	// Determine state output mapping: if resultVariable is defined on the job, map the result to resultVariable
	stateOutputData := cmd.OutputData
	if cmd.Error == "" && job.Input != nil {
		resultVar := ""
		for _, rk := range []string{"resultVariable", "camunda:resultVariable", "outputVariable"} {
			if rv, ok := job.Input[rk].(string); ok && strings.TrimSpace(rv) != "" {
				resultVar = strings.TrimSpace(rv)
				break
			}
		}
		if resultVar != "" && cmd.OutputData != nil {
			if mappedVal, exists := cmd.OutputData[resultVar]; exists {
				stateOutputData = map[string]interface{}{
					resultVar: mappedVal,
				}
			} else if resVal, hasRes := cmd.OutputData["result"]; hasRes {
				stateOutputData = map[string]interface{}{
					resultVar: resVal,
				}
			} else if respVal, hasResp := cmd.OutputData["response"]; hasResp {
				stateOutputData = map[string]interface{}{
					resultVar: respVal,
				}
			}
		}
	}

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
				OutputData:  stateOutputData,
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
