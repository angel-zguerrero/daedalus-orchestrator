package workflow_definition

import (
	"encoding/gob"
	"fmt"
	"time"

	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
)

func init() {
	gob.Register(DeleteWorkflowDefinitionCommand{})
}

type DeleteWorkflowDefinitionCommand struct {
	WorkflowID string
	CF         string
	CFS        string
}

func (cmd *DeleteWorkflowDefinitionCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
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

	ok, err := repo.DeleteWorkflowDefinition(cmd.WorkflowID, now)
	if err != nil || !ok {
		commandResult.Error = fmt.Sprintf("failed to delete workflow definition: %v", err)
		return *commandResult
	}

	commandResult.Result = true
	return *commandResult
}
