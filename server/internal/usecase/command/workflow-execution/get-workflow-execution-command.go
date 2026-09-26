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
	gob.Register(GetWorkflowExecutionCommand{})
	gob.Register(WorkflowExecutionDetail{})
}

type WorkflowExecutionDetail struct {
	Execution models.WorkflowExecution `json:"execution"`
	Tokens    []models.ExecutionToken  `json:"tokens"`
	Jobs      []models.WorkflowJob     `json:"jobs"`
}

type GetWorkflowExecutionCommand struct {
	ID  string
	CF  string
	CFS string
}

func (cmd *GetWorkflowExecutionCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	if cmd.ID == "" {
		commandResult.Error = "ID is required"
		return *commandResult
	}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	repo, err := db.NewWorkflowExecutionRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	exec, err := repo.GetWorkflowExecutionByID(cmd.ID, now)
	if err != nil || exec == nil {
		commandResult.Error = fmt.Sprintf("workflow execution with ID %s not found", cmd.ID)
		return *commandResult
	}

	tokenRepo, _ := db.NewExecutionTokenRepository(uow, idFactory, cmd.CF, cmd.CFS)
	jobRepo, _ := db.NewWorkflowJobRepository(uow, idFactory, cmd.CF, cmd.CFS)

	var tokens []models.ExecutionToken
	if tokenRepo != nil {
		tokens, _ = tokenRepo.GetTokensByExecutionID(cmd.ID, now)
		if exec.Status == models.WorkflowExecutionStatusCompleted {
			for i := range tokens {
				if tokens[i].Status == models.ExecutionTokenStatusWaiting {
					tokens[i].Status = models.ExecutionTokenStatusCompleted
					tokenRepo.UpdateExecutionToken(&tokens[i], now)
				}
			}
		}
	}

	var jobs []models.WorkflowJob
	if jobRepo != nil {
		jobs, _ = jobRepo.GetJobsByExecutionID(cmd.ID, now)
	}

	detail := WorkflowExecutionDetail{
		Execution: *exec,
		Tokens:    tokens,
		Jobs:      jobs,
	}

	commandResult.Result = detail
	return *commandResult
}
