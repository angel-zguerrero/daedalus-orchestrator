package business_logic

import (
	"context"
	"errors"
	"strings"
	"time"

	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	"deadalus-orch/server/internal/infrastructure/server/common"
	"deadalus-orch/server/internal/pkg/config"
	activity_template_command "deadalus-orch/server/internal/usecase/command/activity-template"
	"deadalus-orch/shared/models"

	"github.com/google/uuid"
)

type ActivityTemplateBO struct {
	Config *common.ServerConfing
}

func NewActivityTemplateBO(config *common.ServerConfing) *ActivityTemplateBO {
	return &ActivityTemplateBO{
		Config: config,
	}
}

func (bo *ActivityTemplateBO) resolveRaftNode(scope models.ActivityTemplateScope, tenantNode *dragonboat.RaftNode) (*dragonboat.RaftNode, string, string, error) {
	if scope == models.ActivityTemplateScopeGlobal {
		return bo.Config.MasterNode, db.AdminFC, db.AdminFCSector, nil
	}
	if tenantNode == nil {
		return nil, "", "", errors.New("tenant node is required for tenant scope")
	}
	return tenantNode, "", "", nil
}

func (bo *ActivityTemplateBO) CreateActivityTemplate(
	ctx context.Context,
	scope models.ActivityTemplateScope,
	tenantID string,
	code string,
	name string,
	description string,
	activityFamily string,
	parentTemplateId string,
	rootActivity string,
	payload []byte,
	isActive bool,
	vnamespace string,
	cf, cfs string,
	tenantNode *dragonboat.RaftNode,
) (models.ActivityTemplate, error) {
	code = strings.TrimSpace(code)
	name = strings.TrimSpace(name)
	parentTemplateId = strings.TrimSpace(parentTemplateId)
	rootActivity = strings.TrimSpace(rootActivity)

	if code == "" && name == "" {
		return models.ActivityTemplate{}, errors.New("activity template name or code is required")
	}

	if code == "" {
		code = toKebabCase(name)
	}
	if name == "" {
		name = code
	}
	if vnamespace == "" {
		vnamespace = "default"
	}
	if activityFamily == "" {
		activityFamily = "default"
	}

	node, targetCF, targetCFS, err := bo.resolveRaftNode(scope, tenantNode)
	if err != nil {
		return models.ActivityTemplate{}, err
	}
	if scope == models.ActivityTemplateScopeTenant {
		targetCF = cf
		targetCFS = cfs
	}

	// Determine RootActivity in O(1):
	// 1. If parentTemplateId is a built-in connector (Redis, HTTP, Log), RootActivity is that built-in connector.
	// 2. Otherwise, if inheriting from an existing custom ActivityTemplate (e.g. Activity 1), copy Activity 1's RootActivity
	//    and inherit Activity 1's properties/locks into the new template's payload once at creation time.
	if canonical := db.NormalizeBuiltinActivityType(parentTemplateId); canonical != "" {
		rootActivity = canonical
	} else {
		if canonicalRoot := db.NormalizeBuiltinActivityType(rootActivity); canonicalRoot != "" {
			rootActivity = canonicalRoot
		}
		if parentTpl := bo.resolveParentByCodeOrID(ctx, parentTemplateId, targetCF, targetCFS, tenantNode); parentTpl != nil {
			if rootActivity == "" {
				if canonicalParentRoot := db.NormalizeBuiltinActivityType(parentTpl.RootActivity); canonicalParentRoot != "" {
					rootActivity = canonicalParentRoot
				} else if canonicalParentId := db.NormalizeBuiltinActivityType(parentTpl.ParentTemplateId); canonicalParentId != "" {
					rootActivity = canonicalParentId
				} else {
					rootActivity = db.InferBaseActivityTypeFromPayload(parentTpl.Payload)
				}
			}
			// Bake parent's properties and locks into child's payload at creation time so no runtime chain walk is needed
			dummyLeaf := &models.ActivityTemplate{Payload: payload}
			payload = db.EnrichTemplatePayloadWithAncestors([]*models.ActivityTemplate{parentTpl, dummyLeaf}, payload)
		}
		if rootActivity == "" {
			rootActivity = db.InferBaseActivityTypeFromPayload(payload)
		}
	}

	templateID := uuid.New().String()

	tpl := models.ActivityTemplate{
		ID:               templateID,
		Code:             code,
		Name:             name,
		Description:      description,
		ActivityFamily:   activityFamily,
		ParentTemplateId: parentTemplateId,
		RootActivity:     rootActivity,
		Payload:          payload,
		IsActive:         isActive,
		Scope:            scope,
		TenantID:         tenantID,
		VNamespace:       vnamespace,
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
	}

	cmd := &activity_template_command.CreateActivityTemplateCommand{
		ActivityTemplate: tpl,
		CF:               targetCF,
		CFS:              targetCFS,
	}

	timeout := config.GlobalConfiguration.ApiRaftTimeout
	created, err := dragonboat.ExecuteRepositoryCommand[models.ActivityTemplate](
		node,
		ctx,
		cmd,
		timeout,
		bo.Config.Logger,
		"create activity template",
	)
	if err != nil {
		return models.ActivityTemplate{}, err
	}

	return created, nil
}

func (bo *ActivityTemplateBO) UpdateActivityTemplate(
	ctx context.Context,
	scope models.ActivityTemplateScope,
	id string,
	name string,
	description string,
	activityFamily string,
	parentTemplateId string,
	rootActivity string,
	payload []byte,
	isActive bool,
	vnamespace string,
	cf, cfs string,
	tenantNode *dragonboat.RaftNode,
) (models.ActivityTemplate, error) {
	node, targetCF, targetCFS, err := bo.resolveRaftNode(scope, tenantNode)
	if err != nil {
		return models.ActivityTemplate{}, err
	}
	if scope == models.ActivityTemplateScopeTenant {
		targetCF = cf
		targetCFS = cfs
	}

	if canonical := db.NormalizeBuiltinActivityType(parentTemplateId); canonical != "" {
		rootActivity = canonical
	} else {
		if canonicalRoot := db.NormalizeBuiltinActivityType(rootActivity); canonicalRoot != "" {
			rootActivity = canonicalRoot
		}
		if parentTpl := bo.resolveParentByCodeOrID(ctx, parentTemplateId, targetCF, targetCFS, tenantNode); parentTpl != nil {
			if rootActivity == "" {
				if canonicalParentRoot := db.NormalizeBuiltinActivityType(parentTpl.RootActivity); canonicalParentRoot != "" {
					rootActivity = canonicalParentRoot
				} else if canonicalParentId := db.NormalizeBuiltinActivityType(parentTpl.ParentTemplateId); canonicalParentId != "" {
					rootActivity = canonicalParentId
				} else {
					rootActivity = db.InferBaseActivityTypeFromPayload(parentTpl.Payload)
				}
			}
			if len(payload) > 0 {
				dummyLeaf := &models.ActivityTemplate{Payload: payload}
				payload = db.EnrichTemplatePayloadWithAncestors([]*models.ActivityTemplate{parentTpl, dummyLeaf}, payload)
			}
		}
		if rootActivity == "" && len(payload) > 0 {
			rootActivity = db.InferBaseActivityTypeFromPayload(payload)
		}
	}

	tpl := models.ActivityTemplate{
		ID:               id,
		Name:             name,
		Description:      description,
		ActivityFamily:   activityFamily,
		ParentTemplateId: parentTemplateId,
		RootActivity:     rootActivity,
		Payload:          payload,
		IsActive:         isActive,
		VNamespace:       vnamespace,
	}

	cmd := &activity_template_command.UpdateActivityTemplateCommand{
		ActivityTemplate: tpl,
		CF:               targetCF,
		CFS:              targetCFS,
	}

	timeout := config.GlobalConfiguration.ApiRaftTimeout
	updated, err := dragonboat.ExecuteRepositoryCommand[models.ActivityTemplate](
		node,
		ctx,
		cmd,
		timeout,
		bo.Config.Logger,
		"update activity template",
	)
	if err != nil {
		return models.ActivityTemplate{}, err
	}

	return updated, nil
}

func (bo *ActivityTemplateBO) DeleteActivityTemplate(
	ctx context.Context,
	scope models.ActivityTemplateScope,
	templateID string,
	cf, cfs string,
	tenantNode *dragonboat.RaftNode,
) error {
	node, targetCF, targetCFS, err := bo.resolveRaftNode(scope, tenantNode)
	if err != nil {
		return err
	}
	if scope == models.ActivityTemplateScopeTenant {
		targetCF = cf
		targetCFS = cfs
	}

	cmd := &activity_template_command.DeleteActivityTemplateCommand{
		TemplateID: templateID,
		CF:         targetCF,
		CFS:        targetCFS,
	}

	timeout := config.GlobalConfiguration.ApiRaftTimeout
	_, err = dragonboat.ExecuteRepositoryCommand[bool](
		node,
		ctx,
		cmd,
		timeout,
		bo.Config.Logger,
		"delete activity template",
	)
	return err
}

func (bo *ActivityTemplateBO) GetActivityTemplate(
	ctx context.Context,
	scope models.ActivityTemplateScope,
	templateID string,
	cf, cfs string,
	tenantNode *dragonboat.RaftNode,
) (*models.ActivityTemplate, error) {
	node, targetCF, targetCFS, err := bo.resolveRaftNode(scope, tenantNode)
	if err != nil {
		return nil, err
	}
	if scope == models.ActivityTemplateScopeTenant {
		targetCF = cf
		targetCFS = cfs
	}

	cmd := &activity_template_command.GetActivityTemplateCommand{
		TemplateID: templateID,
		CF:         targetCF,
		CFS:        targetCFS,
	}

	timeout := config.GlobalConfiguration.ApiRaftTimeout
	tpl, err := dragonboat.ExecuteRepositoryQuery[models.ActivityTemplate](
		node,
		ctx,
		cmd,
		timeout,
		bo.Config.Logger,
		"get activity template",
	)
	if err != nil {
		return nil, err
	}
	if tpl.ID == "" {
		return nil, nil
	}
	bo.EnrichTemplateWithAncestors(ctx, &tpl, cf, cfs, tenantNode)
	return &tpl, nil
}

func (bo *ActivityTemplateBO) ListActivityTemplates(
	ctx context.Context,
	scope models.ActivityTemplateScope,
	tenantID string,
	vnamespace string,
	activityFamily string,
	pageSize int,
	cursor string,
	cf, cfs string,
	tenantNode *dragonboat.RaftNode,
) (*db.FindResult[models.ActivityTemplate], error) {
	node, targetCF, targetCFS, err := bo.resolveRaftNode(scope, tenantNode)
	if err != nil {
		return nil, err
	}
	if scope == models.ActivityTemplateScopeTenant {
		targetCF = cf
		targetCFS = cfs
	}

	cmd := &activity_template_command.ListActivityTemplatesCommand{
		Scope:          string(scope),
		TenantID:       tenantID,
		VNamespace:     vnamespace,
		ActivityFamily: activityFamily,
		PageSize:       pageSize,
		Cursor:         cursor,
		CF:             targetCF,
		CFS:            targetCFS,
	}

	timeout := config.GlobalConfiguration.ApiRaftTimeout
	res, err := dragonboat.ExecuteRepositoryQuery[db.FindResult[models.ActivityTemplate]](
		node,
		ctx,
		cmd,
		timeout,
		bo.Config.Logger,
		"list activity templates",
	)
	if err != nil {
		return nil, err
	}
	for i := range res.Entities {
		bo.EnrichTemplateWithAncestors(ctx, &res.Entities[i], cf, cfs, tenantNode)
	}
	return &res, nil
}

func (bo *ActivityTemplateBO) resolveParentByCodeOrID(
	ctx context.Context,
	codeOrID string,
	cf, cfs string,
	tenantNode *dragonboat.RaftNode,
) *models.ActivityTemplate {
	trimmed := strings.TrimSpace(codeOrID)
	if trimmed == "" || db.IsBuiltinActivityType(trimmed) {
		return nil
	}
	timeout := config.GlobalConfiguration.ApiRaftTimeout

	if tenantNode != nil && cf != "" && cfs != "" {
		cmd := &activity_template_command.GetActivityTemplateByCodeOrIDCommand{
			CodeOrID: trimmed,
			CF:       cf,
			CFS:      cfs,
		}
		if res, err := dragonboat.ExecuteRepositoryQuery[models.ActivityTemplate](tenantNode, ctx, cmd, timeout, bo.Config.Logger, "resolve parent template in tenant"); err == nil && res.ID != "" {
			return &res
		}
	}

	if bo.Config != nil && bo.Config.MasterNode != nil {
		cmd := &activity_template_command.GetActivityTemplateByCodeOrIDCommand{
			CodeOrID: trimmed,
			CF:       db.AdminFC,
			CFS:      db.AdminFCSector,
		}
		if res, err := dragonboat.ExecuteRepositoryQuery[models.ActivityTemplate](bo.Config.MasterNode, ctx, cmd, timeout, bo.Config.Logger, "resolve parent template in global"); err == nil && res.ID != "" {
			return &res
		}
	}

	return nil
}

// ensureRootActivity ensures RootActivity is populated on a template without walking a recursive chain.
func (bo *ActivityTemplateBO) EnrichTemplateWithAncestors(
	ctx context.Context,
	tpl *models.ActivityTemplate,
	cf, cfs string,
	tenantNode *dragonboat.RaftNode,
) {
	if tpl == nil {
		return
	}
	if tpl.RootActivity == "" {
		if canonical := db.NormalizeBuiltinActivityType(tpl.ParentTemplateId); canonical != "" {
			tpl.RootActivity = canonical
		} else if inferred := db.InferBaseActivityTypeFromPayload(tpl.Payload); inferred != "" {
			tpl.RootActivity = inferred
		}
	}
}

