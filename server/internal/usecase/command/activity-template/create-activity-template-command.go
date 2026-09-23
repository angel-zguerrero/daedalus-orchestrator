package activity_template

import (
	"encoding/gob"
	"fmt"
	"time"

	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"deadalus-orch/shared/models"
)

func init() {
	gob.Register(CreateActivityTemplateCommand{})
}

type CreateActivityTemplateCommand struct {
	ActivityTemplate models.ActivityTemplate
	CF               string
	CFS              string
}

func (cmd *CreateActivityTemplateCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	if cmd.ActivityTemplate.ID == "" {
		commandResult.Error = "ID is required and must be generated outside the command"
		return *commandResult
	}

	if cmd.ActivityTemplate.Code == "" {
		commandResult.Error = "Code is required"
		return *commandResult
	}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	repo, err := db.NewActivityTemplateRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	if cmd.ActivityTemplate.RootActivity == "" {
		if canonical := db.NormalizeBuiltinActivityType(cmd.ActivityTemplate.ParentTemplateId); canonical != "" {
			cmd.ActivityTemplate.RootActivity = canonical
		} else if inferred := db.InferBaseActivityTypeFromPayload(cmd.ActivityTemplate.Payload); inferred != "" {
			cmd.ActivityTemplate.RootActivity = inferred
		}
	}

	id, err := repo.CreateActivityTemplate(&cmd.ActivityTemplate, now)
	if err != nil {
		commandResult.Error = fmt.Sprintf("failed to create activity template: %s", err.Error())
		return *commandResult
	}

	cmd.ActivityTemplate.ID = id
	commandResult.Result = cmd.ActivityTemplate
	return *commandResult
}
