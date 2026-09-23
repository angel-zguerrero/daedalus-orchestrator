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
	gob.Register(UpdateActivityTemplateCommand{})
}

type UpdateActivityTemplateCommand struct {
	ActivityTemplate models.ActivityTemplate
	CF               string
	CFS              string
}

func (cmd *UpdateActivityTemplateCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	if cmd.ActivityTemplate.ID == "" {
		commandResult.Error = "ID is required"
		return *commandResult
	}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	repo, err := db.NewActivityTemplateRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	existing, err := repo.GetActivityTemplateByID(cmd.ActivityTemplate.ID, now)
	if err != nil || existing == nil {
		commandResult.Error = "activity template not found"
		return *commandResult
	}

	if cmd.ActivityTemplate.Name != "" {
		existing.Name = cmd.ActivityTemplate.Name
	}
	existing.Description = cmd.ActivityTemplate.Description
	if cmd.ActivityTemplate.ActivityFamily != "" {
		existing.ActivityFamily = cmd.ActivityTemplate.ActivityFamily
	}
	if cmd.ActivityTemplate.ParentTemplateId != "" {
		existing.ParentTemplateId = cmd.ActivityTemplate.ParentTemplateId
	}

	if cmd.ActivityTemplate.RootActivity != "" {
		existing.RootActivity = cmd.ActivityTemplate.RootActivity
	} else if canonical := db.NormalizeBuiltinActivityType(existing.ParentTemplateId); canonical != "" {
		existing.RootActivity = canonical
	} else if canonicalRoot := db.NormalizeBuiltinActivityType(existing.RootActivity); canonicalRoot != "" {
		existing.RootActivity = canonicalRoot
	} else if inferred := db.InferBaseActivityTypeFromPayload(cmd.ActivityTemplate.Payload); inferred != "" {
		existing.RootActivity = inferred
	}

	if len(cmd.ActivityTemplate.Payload) > 0 {
		existing.Payload = cmd.ActivityTemplate.Payload
	}

	if existing.ParentTemplateId != "" {
		if parentTpl, err := repo.ResolveByCodeOrID(existing.ParentTemplateId, existing.VNamespace, now); err == nil && parentTpl != nil {
			dummyLeaf := &models.ActivityTemplate{Payload: existing.Payload}
			existing.Payload = db.EnrichTemplatePayloadWithAncestors([]*models.ActivityTemplate{parentTpl, dummyLeaf}, existing.Payload)
			if existing.RootActivity == "" {
				if canonicalParentRoot := db.NormalizeBuiltinActivityType(parentTpl.RootActivity); canonicalParentRoot != "" {
					existing.RootActivity = canonicalParentRoot
				} else if canonicalParentId := db.NormalizeBuiltinActivityType(parentTpl.ParentTemplateId); canonicalParentId != "" {
					existing.RootActivity = canonicalParentId
				}
			}
		}
	}

	existing.IsActive = cmd.ActivityTemplate.IsActive
	if cmd.ActivityTemplate.VNamespace != "" {
		existing.VNamespace = cmd.ActivityTemplate.VNamespace
	}

	ok, err := repo.UpdateActivityTemplate(existing, now)
	if err != nil || !ok {
		commandResult.Error = fmt.Sprintf("failed to update activity template: %v", err)
		return *commandResult
	}

	commandResult.Result = *existing
	return *commandResult
}
