package business_logic

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	"deadalus-orch/server/internal/infrastructure/server/common"
	"deadalus-orch/server/internal/pkg/config"
	workflow_definition_command "deadalus-orch/server/internal/usecase/command/workflow-definition"
	"deadalus-orch/shared/models"

	"github.com/google/uuid"
)

type WorkflowDefinitionBO struct {
	Config *common.ServerConfing
}

func NewWorkflowDefinitionBO(config *common.ServerConfing) *WorkflowDefinitionBO {
	return &WorkflowDefinitionBO{
		Config: config,
	}
}

func (bo *WorkflowDefinitionBO) resolveRaftNode(scope models.WorkflowScope, tenantNode *dragonboat.RaftNode) (*dragonboat.RaftNode, string, string, error) {
	if scope == models.WorkflowScopeGlobal {
		return bo.Config.MasterNode, db.AdminFC, db.AdminFCSector, nil
	}
	if tenantNode == nil {
		return nil, "", "", errors.New("tenant node is required for tenant scope")
	}
	return tenantNode, "", "", nil
}

func (bo *WorkflowDefinitionBO) CreateWorkflow(
	ctx context.Context,
	scope models.WorkflowScope,
	tenantID string,
	code string,
	name string,
	description string,
	version int32,
	payload []byte,
	payloadFormat models.WorkflowPayloadFormat,
	maxDurationSeconds int32,
	isActive bool,
	vnamespace string,
	cf, cfs string,
	tenantNode *dragonboat.RaftNode,
) (models.WorkflowDefinition, error) {
	code = strings.TrimSpace(code)
	name = strings.TrimSpace(name)

	if code == "" && name == "" {
		return models.WorkflowDefinition{}, errors.New("workflow name or code is required")
	}

	if code == "" {
		code = toKebabCase(name)
	}
	if name == "" {
		name = code
	}
	if version <= 0 {
		version = 1
	}
	if payloadFormat == "" {
		payloadFormat = models.WorkflowPayloadFormatJSON
	}
	if vnamespace == "" {
		vnamespace = "default"
	}

	node, targetCF, targetCFS, err := bo.resolveRaftNode(scope, tenantNode)
	if err != nil {
		return models.WorkflowDefinition{}, err
	}
	if scope == models.WorkflowScopeTenant {
		targetCF = cf
		targetCFS = cfs
	}

	workflowID := uuid.New().String()
	randToken := uuid.New().String()[:8]

	execQueueID := strings.ReplaceAll(uuid.New().String(), "-", "")
	actQueueID := strings.ReplaceAll(uuid.New().String(), "-", "")

	execQueueCode := fmt.Sprintf("wf-exec-%s-%s", code, randToken)
	execQueueName := fmt.Sprintf("%s Execution Queue (%s)", name, randToken)
	actQueueCode := fmt.Sprintf("wf-act-%s-%s", code, randToken)
	actQueueName := fmt.Sprintf("%s Activity Queue (%s)", name, randToken)

	execQueue := models.Queue{
		ID:                   execQueueID,
		Code:                 execQueueCode,
		Name:                 execQueueName,
		Type:                 models.WorkflowExecutionQueue,
		WorkflowDefinitionID: workflowID,
		VNamespace:           vnamespace,
		State:                models.QueueActive,
		AllowDuplicated:      true,
		MaxAttempts:          1,
	}

	actQueue := models.Queue{
		ID:                   actQueueID,
		Code:                 actQueueCode,
		Name:                 actQueueName,
		Type:                 models.WorkflowActivityQueue,
		WorkflowDefinitionID: workflowID,
		VNamespace:           vnamespace,
		State:                models.QueueActive,
		AllowDuplicated:      true,
		MaxAttempts:          1,
	}

	wf := models.WorkflowDefinition{
		ID:                 workflowID,
		Code:               code,
		Name:               name,
		Description:        description,
		Version:            version,
		Payload:            payload,
		PayloadFormat:      payloadFormat,
		MaxDurationSeconds: maxDurationSeconds,
		IsActive:           isActive,
		Scope:              scope,
		TenantID:           tenantID,
		VNamespace:         vnamespace,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}

	cmd := &workflow_definition_command.CreateWorkflowDefinitionCommand{
		WorkflowDefinition: wf,
		ExecQueue:          execQueue,
		ActQueue:           actQueue,
		CF:                 targetCF,
		CFS:                targetCFS,
	}

	timeout := config.GlobalConfiguration.ApiRaftTimeout
	created, err := dragonboat.ExecuteRepositoryCommand[models.WorkflowDefinition](
		node,
		ctx,
		cmd,
		timeout,
		bo.Config.Logger,
		"create workflow definition",
	)
	if err != nil {
		return models.WorkflowDefinition{}, err
	}

	return created, nil
}

func (bo *WorkflowDefinitionBO) UpdateWorkflow(
	ctx context.Context,
	scope models.WorkflowScope,
	id string,
	name string,
	description string,
	version int32,
	payload []byte,
	payloadFormat models.WorkflowPayloadFormat,
	maxDurationSeconds int32,
	isActive bool,
	vnamespace string,
	cf, cfs string,
	tenantNode *dragonboat.RaftNode,
) (models.WorkflowDefinition, error) {
	node, targetCF, targetCFS, err := bo.resolveRaftNode(scope, tenantNode)
	if err != nil {
		return models.WorkflowDefinition{}, err
	}
	if scope == models.WorkflowScopeTenant {
		targetCF = cf
		targetCFS = cfs
	}

	wf := models.WorkflowDefinition{
		ID:                 id,
		Name:               name,
		Description:        description,
		Version:            version,
		Payload:            payload,
		PayloadFormat:      payloadFormat,
		MaxDurationSeconds: maxDurationSeconds,
		IsActive:           isActive,
		VNamespace:         vnamespace,
	}

	cmd := &workflow_definition_command.UpdateWorkflowDefinitionCommand{
		WorkflowDefinition: wf,
		CF:                 targetCF,
		CFS:                targetCFS,
	}

	timeout := config.GlobalConfiguration.ApiRaftTimeout
	updated, err := dragonboat.ExecuteRepositoryCommand[models.WorkflowDefinition](
		node,
		ctx,
		cmd,
		timeout,
		bo.Config.Logger,
		"update workflow definition",
	)
	if err != nil {
		return models.WorkflowDefinition{}, err
	}

	return updated, nil
}

func (bo *WorkflowDefinitionBO) DeleteWorkflow(
	ctx context.Context,
	scope models.WorkflowScope,
	workflowID string,
	cf, cfs string,
	tenantNode *dragonboat.RaftNode,
) error {
	node, targetCF, targetCFS, err := bo.resolveRaftNode(scope, tenantNode)
	if err != nil {
		return err
	}
	if scope == models.WorkflowScopeTenant {
		targetCF = cf
		targetCFS = cfs
	}

	cmd := &workflow_definition_command.DeleteWorkflowDefinitionCommand{
		WorkflowID: workflowID,
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
		"delete workflow definition",
	)
	return err
}

func (bo *WorkflowDefinitionBO) GetWorkflow(
	ctx context.Context,
	scope models.WorkflowScope,
	workflowID string,
	cf, cfs string,
	tenantNode *dragonboat.RaftNode,
) (*models.WorkflowDefinition, error) {
	node, targetCF, targetCFS, err := bo.resolveRaftNode(scope, tenantNode)
	if err != nil {
		return nil, err
	}
	if scope == models.WorkflowScopeTenant {
		targetCF = cf
		targetCFS = cfs
	}

	cmd := &workflow_definition_command.GetWorkflowDefinitionCommand{
		WorkflowID: workflowID,
		CF:         targetCF,
		CFS:        targetCFS,
	}

	timeout := config.GlobalConfiguration.ApiRaftTimeout
	wf, err := dragonboat.ExecuteRepositoryQuery[models.WorkflowDefinition](
		node,
		ctx,
		cmd,
		timeout,
		bo.Config.Logger,
		"get workflow definition",
	)
	if err != nil {
		return nil, err
	}
	if wf.ID == "" {
		return nil, nil
	}
	return &wf, nil
}

func (bo *WorkflowDefinitionBO) ListWorkflows(
	ctx context.Context,
	scope models.WorkflowScope,
	tenantID string,
	vnamespace string,
	pageSize int,
	cursor string,
	cf, cfs string,
	tenantNode *dragonboat.RaftNode,
) (*db.FindResult[models.WorkflowDefinition], error) {
	node, targetCF, targetCFS, err := bo.resolveRaftNode(scope, tenantNode)
	if err != nil {
		return nil, err
	}
	if scope == models.WorkflowScopeTenant {
		targetCF = cf
		targetCFS = cfs
	}

	cmd := &workflow_definition_command.ListWorkflowDefinitionsCommand{
		Scope:      string(scope),
		TenantID:   tenantID,
		VNamespace: vnamespace,
		PageSize:   pageSize,
		Cursor:     cursor,
		CF:         targetCF,
		CFS:        targetCFS,
	}

	timeout := config.GlobalConfiguration.ApiRaftTimeout
	res, err := dragonboat.ExecuteRepositoryQuery[db.FindResult[models.WorkflowDefinition]](
		node,
		ctx,
		cmd,
		timeout,
		bo.Config.Logger,
		"list workflow definitions",
	)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (bo *WorkflowDefinitionBO) GetWorkflowQueues(
	ctx context.Context,
	scope models.WorkflowScope,
	workflowID string,
	cf, cfs string,
	tenantNode *dragonboat.RaftNode,
) ([]models.Queue, error) {
	node, targetCF, targetCFS, err := bo.resolveRaftNode(scope, tenantNode)
	if err != nil {
		return nil, err
	}
	if scope == models.WorkflowScopeTenant {
		targetCF = cf
		targetCFS = cfs
	}

	cmd := &workflow_definition_command.GetWorkflowQueuesCommand{
		WorkflowID: workflowID,
		CF:         targetCF,
		CFS:        targetCFS,
	}

	timeout := config.GlobalConfiguration.ApiRaftTimeout
	queues, err := dragonboat.ExecuteRepositoryQuery[[]models.Queue](
		node,
		ctx,
		cmd,
		timeout,
		bo.Config.Logger,
		"get workflow queues",
	)
	if err != nil {
		return nil, err
	}
	return queues, nil
}
