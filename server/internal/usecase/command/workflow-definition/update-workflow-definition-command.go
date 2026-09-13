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
	gob.Register(UpdateWorkflowDefinitionCommand{})
}

type UpdateWorkflowDefinitionCommand struct {
	WorkflowDefinition models.WorkflowDefinition
	CF                 string
	CFS                string
}

func (cmd *UpdateWorkflowDefinitionCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	if cmd.WorkflowDefinition.ID == "" {
		commandResult.Error = "ID is required"
		return *commandResult
	}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	repo, err := db.NewWorkflowDefinitionRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	existing, err := repo.GetWorkflowDefinitionByID(cmd.WorkflowDefinition.ID, now)
	if err != nil || existing == nil {
		commandResult.Error = "workflow definition not found"
		return *commandResult
	}

	existing.Name = cmd.WorkflowDefinition.Name
	existing.Description = cmd.WorkflowDefinition.Description
	if cmd.WorkflowDefinition.Version > 0 {
		existing.Version = cmd.WorkflowDefinition.Version
	}
	if len(cmd.WorkflowDefinition.Payload) > 0 {
		existing.Payload = cmd.WorkflowDefinition.Payload
	}
	if cmd.WorkflowDefinition.PayloadFormat != "" {
		existing.PayloadFormat = cmd.WorkflowDefinition.PayloadFormat
	}
	existing.MaxDurationSeconds = cmd.WorkflowDefinition.MaxDurationSeconds
	existing.IsActive = cmd.WorkflowDefinition.IsActive
	if cmd.WorkflowDefinition.VNamespace != "" {
		existing.VNamespace = cmd.WorkflowDefinition.VNamespace
	}

	ok, err := repo.UpdateWorkflowDefinition(existing, now)
	if err != nil || !ok {
		commandResult.Error = fmt.Sprintf("failed to update workflow definition: %v", err)
		return *commandResult
	}

	commandResult.Result = *existing
	return *commandResult
}
