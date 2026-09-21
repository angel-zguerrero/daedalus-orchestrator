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
	gob.Register(GetWorkflowVersionCommand{})
	gob.Register(models.WorkflowDefinitionVersion{})
}

type GetWorkflowVersionCommand struct {
	WorkflowDefinitionID string
	Version              int32
	CF                   string
	CFS                  string
}

func (cmd *GetWorkflowVersionCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	if cmd.WorkflowDefinitionID == "" {
		commandResult.Error = "WorkflowDefinitionID is required"
		return *commandResult
	}
	if cmd.Version <= 0 {
		commandResult.Error = "Version must be greater than 0"
		return *commandResult
	}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	repo, err := db.NewWorkflowDefinitionVersionRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	ver, err := repo.GetVersionByNumber(cmd.WorkflowDefinitionID, cmd.Version, now)
	if err != nil {
		commandResult.Error = fmt.Sprintf("failed to get workflow version: %v", err)
		return *commandResult
	}

	if ver != nil {
		commandResult.Result = *ver
	} else {
		commandResult.Result = nil
	}
	return *commandResult
}
