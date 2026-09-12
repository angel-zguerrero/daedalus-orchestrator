package workflow_definition

import (
	"encoding/gob"
	"fmt"
	"time"

	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	models "deadalus-orch/shared/models"
)

func init() {
	gob.Register(GetWorkflowDefinitionCommand{})
	gob.Register(ListWorkflowDefinitionsCommand{})
	gob.Register(db.FindResult[models.WorkflowDefinition]{})
	gob.Register(models.WorkflowDefinition{})
}

type GetWorkflowDefinitionCommand struct {
	WorkflowID string
	CF         string
	CFS        string
}

func (cmd *GetWorkflowDefinitionCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	if cmd.WorkflowID == "" {
		commandResult.Error = "WorkflowID is required"
		return *commandResult
	}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	repo, err := db.NewWorkflowDefinitionRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	wf, err := repo.GetWorkflowDefinitionByID(cmd.WorkflowID, now)
	if err != nil {
		commandResult.Error = fmt.Sprintf("failed to get workflow definition: %v", err)
		return *commandResult
	}

	if wf != nil {
		commandResult.Result = *wf
	} else {
		commandResult.Result = nil
	}
	return *commandResult
}

type ListWorkflowDefinitionsCommand struct {
	Scope    string
	TenantID string
	PageSize int
	Cursor   string
	CF       string
	CFS      string
}

func (cmd *ListWorkflowDefinitionsCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	repo, err := db.NewWorkflowDefinitionRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	res, err := repo.ListWorkflowDefinitions(cmd.Scope, cmd.TenantID, cmd.PageSize, cmd.Cursor, now)
	if err != nil {
		commandResult.Error = fmt.Sprintf("failed to list workflow definitions: %v", err)
		return *commandResult
	}

	if res != nil {
		commandResult.Result = *res
	} else {
		commandResult.Result = db.FindResult[models.WorkflowDefinition]{Entities: []models.WorkflowDefinition{}}
	}
	return *commandResult
}
