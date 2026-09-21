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
	gob.Register(ListWorkflowVersionsCommand{})
	gob.Register(db.FindResult[models.WorkflowDefinitionVersion]{})
	gob.Register(models.WorkflowDefinitionVersion{})
}

type ListWorkflowVersionsCommand struct {
	WorkflowDefinitionID string
	PageSize             int
	Cursor               string
	CF                   string
	CFS                  string
}

func (cmd *ListWorkflowVersionsCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	if cmd.WorkflowDefinitionID == "" {
		commandResult.Error = "WorkflowDefinitionID is required"
		return *commandResult
	}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	repo, err := db.NewWorkflowDefinitionVersionRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	res, err := repo.ListVersions(cmd.WorkflowDefinitionID, cmd.PageSize, cmd.Cursor, now)
	if err != nil {
		commandResult.Error = fmt.Sprintf("failed to list workflow versions: %v", err)
		return *commandResult
	}

	if res != nil {
		commandResult.Result = *res
	} else {
		commandResult.Result = db.FindResult[models.WorkflowDefinitionVersion]{Entities: []models.WorkflowDefinitionVersion{}}
	}
	return *commandResult
}
