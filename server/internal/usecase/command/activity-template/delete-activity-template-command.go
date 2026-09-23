package activity_template

import (
	"encoding/gob"
	"fmt"
	"time"

	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
)

func init() {
	gob.Register(DeleteActivityTemplateCommand{})
}

type DeleteActivityTemplateCommand struct {
	TemplateID string
	CF         string
	CFS        string
}

func (cmd *DeleteActivityTemplateCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	if cmd.TemplateID == "" {
		commandResult.Error = "TemplateID is required"
		return *commandResult
	}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	repo, err := db.NewActivityTemplateRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	ok, err := repo.DeleteActivityTemplate(cmd.TemplateID, now)
	if err != nil || !ok {
		commandResult.Error = fmt.Sprintf("failed to delete activity template: %v", err)
		return *commandResult
	}

	commandResult.Result = true
	return *commandResult
}
