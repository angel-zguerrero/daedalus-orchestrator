package workflow_execution

import (
	"encoding/gob"
	"fmt"
	"time"

	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
)

func init() {
	gob.Register(ListWorkflowExecutionsCommand{})
}

type ListWorkflowExecutionsCommand struct {
	VNamespace           string
	WorkflowDefinitionID string
	Status               string
	PageSize             int
	Cursor               string
	CF                   string
	CFS                  string
}

func (cmd *ListWorkflowExecutionsCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	repo, err := db.NewWorkflowExecutionRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	res, err := repo.ListWorkflowExecutions(cmd.VNamespace, cmd.WorkflowDefinitionID, cmd.Status, cmd.PageSize, cmd.Cursor, now)
	if err != nil {
		commandResult.Error = fmt.Sprintf("failed to list workflow executions: %s", err.Error())
		return *commandResult
	}

	commandResult.Result = res
	return *commandResult
}
