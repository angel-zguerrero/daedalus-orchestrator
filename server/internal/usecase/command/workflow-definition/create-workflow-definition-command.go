package workflow_definition

import (
	"encoding/gob"
	"fmt"
	"time"

	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"deadalus-orch/shared/models"
)

func init() {
	gob.Register(CreateWorkflowDefinitionCommand{})
}

type CreateWorkflowDefinitionCommand struct {
	WorkflowDefinition models.WorkflowDefinition
	CF                 string
	CFS                string
}

func (cmd *CreateWorkflowDefinitionCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	if cmd.WorkflowDefinition.ID == "" {
		commandResult.Error = "ID is required and must be generated outside the command"
		return *commandResult
	}

	if cmd.WorkflowDefinition.Code == "" {
		commandResult.Error = "Code is required"
		return *commandResult
	}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	repo, err := db.NewWorkflowDefinitionRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	id, err := repo.CreateWorkflowDefinition(&cmd.WorkflowDefinition, now)
	if err != nil {
		commandResult.Error = fmt.Sprintf("failed to create workflow definition: %s", err.Error())
		return *commandResult
	}

	cmd.WorkflowDefinition.ID = id
	commandResult.Result = cmd.WorkflowDefinition
	return *commandResult
}
