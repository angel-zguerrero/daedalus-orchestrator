package activity_template

import (
	"encoding/gob"
	"fmt"
	"time"

	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	models "deadalus-orch/shared/models"
)

func init() {
	gob.Register(GetActivityTemplateCommand{})
	gob.Register(GetActivityTemplateByCodeOrIDCommand{})
	gob.Register(ListActivityTemplatesCommand{})
	gob.Register(db.FindResult[models.ActivityTemplate]{})
	gob.Register(models.ActivityTemplate{})
}

type GetActivityTemplateCommand struct {
	TemplateID string
	CF         string
	CFS        string
}

func (cmd *GetActivityTemplateCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
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

	tpl, err := repo.GetActivityTemplateByID(cmd.TemplateID, now)
	if err != nil {
		commandResult.Error = fmt.Sprintf("failed to get activity template: %v", err)
		return *commandResult
	}

	if tpl != nil {
		commandResult.Result = *tpl
	} else {
		commandResult.Result = nil
	}
	return *commandResult
}

type GetActivityTemplateByCodeOrIDCommand struct {
	CodeOrID   string
	VNamespace string
	CF         string
	CFS        string
}

func (cmd *GetActivityTemplateByCodeOrIDCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	if cmd.CodeOrID == "" {
		commandResult.Error = "CodeOrID is required"
		return *commandResult
	}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	repo, err := db.NewActivityTemplateRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	tpl, _ := repo.ResolveByCodeOrID(cmd.CodeOrID, cmd.VNamespace, now)

	if tpl != nil {
		commandResult.Result = *tpl
	} else {
		commandResult.Result = models.ActivityTemplate{}
	}
	return *commandResult
}

type ListActivityTemplatesCommand struct {
	Scope          string
	TenantID       string
	VNamespace     string
	ActivityFamily string
	PageSize       int
	Cursor         string
	CF             string
	CFS            string
}

func (cmd *ListActivityTemplatesCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	repo, err := db.NewActivityTemplateRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	res, err := repo.ListActivityTemplates(cmd.Scope, cmd.TenantID, cmd.VNamespace, cmd.ActivityFamily, cmd.PageSize, cmd.Cursor, now)
	if err != nil {
		commandResult.Error = fmt.Sprintf("failed to list activity templates: %v", err)
		return *commandResult
	}

	if res != nil {
		commandResult.Result = *res
	} else {
		commandResult.Result = db.FindResult[models.ActivityTemplate]{Entities: []models.ActivityTemplate{}}
	}
	return *commandResult
}
