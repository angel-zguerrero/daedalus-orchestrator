package business_logic

import (
	"context"
	"errors"
	"strings"

	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	"deadalus-orch/server/internal/infrastructure/server/common"
	"deadalus-orch/server/internal/pkg/config"
	workflow_execution_command "deadalus-orch/server/internal/usecase/command/workflow-execution"
	"deadalus-orch/shared/models"

	"github.com/google/uuid"
)

type WorkflowExecutionBO struct {
	Config *common.ServerConfing
}

func NewWorkflowExecutionBO(config *common.ServerConfing) *WorkflowExecutionBO {
	return &WorkflowExecutionBO{
		Config: config,
	}
}

func (bo *WorkflowExecutionBO) resolveRaftNode(scope models.WorkflowScope, tenantNode *dragonboat.RaftNode) (*dragonboat.RaftNode, string, string, error) {
	if scope == models.WorkflowScopeGlobal {
		return bo.Config.MasterNode, db.AdminFC, db.AdminFCSector, nil
	}
	if tenantNode == nil {
		return nil, "", "", errors.New("tenant node is required for tenant scope")
	}
	return tenantNode, "", "", nil
}

func (bo *WorkflowExecutionBO) StartExecution(
	ctx context.Context,
	scope models.WorkflowScope,
	workflowDefinitionID string,
	executionKey string,
	input map[string]interface{},
	vnamespace string,
	cf, cfs string,
	tenantNode *dragonboat.RaftNode,
) (models.WorkflowExecution, error) {
	node, targetCF, targetCFS, err := bo.resolveRaftNode(scope, tenantNode)
	if err != nil {
		return models.WorkflowExecution{}, err
	}
	if scope == models.WorkflowScopeTenant {
		targetCF = cf
		targetCFS = cfs
	}

	execID := strings.ReplaceAll(uuid.New().String(), "-", "")
	tokenID := strings.ReplaceAll(uuid.New().String(), "-", "")

	cmd := &workflow_execution_command.StartWorkflowExecutionCommand{
		ExecutionID:          execID,
		InitialTokenID:       tokenID,
		WorkflowDefinitionID: workflowDefinitionID,
		ExecutionKey:         executionKey,
		Input:                input,
		VNamespace:           vnamespace,
		CF:                   targetCF,
		CFS:                  targetCFS,
	}

	timeout := config.GlobalConfiguration.ApiRaftTimeout
	created, err := dragonboat.ExecuteRepositoryCommand[models.WorkflowExecution](
		node,
		ctx,
		cmd,
		timeout,
		bo.Config.Logger,
		"start workflow execution",
	)
	if err != nil {
		return models.WorkflowExecution{}, err
	}

	return created, nil
}

func (bo *WorkflowExecutionBO) AdvanceToken(
	ctx context.Context,
	scope models.WorkflowScope,
	executionID string,
	tokenID string,
	outputData map[string]interface{},
	cf, cfs string,
	tenantNode *dragonboat.RaftNode,
) (models.WorkflowExecution, error) {
	node, targetCF, targetCFS, err := bo.resolveRaftNode(scope, tenantNode)
	if err != nil {
		return models.WorkflowExecution{}, err
	}
	if scope == models.WorkflowScopeTenant {
		targetCF = cf
		targetCFS = cfs
	}

	cmd := &workflow_execution_command.AdvanceTokenCommand{
		ExecutionID: executionID,
		TokenID:     tokenID,
		OutputData:  outputData,
		CF:          targetCF,
		CFS:         targetCFS,
	}

	timeout := config.GlobalConfiguration.ApiRaftTimeout
	updated, err := dragonboat.ExecuteRepositoryCommand[models.WorkflowExecution](
		node,
		ctx,
		cmd,
		timeout,
		bo.Config.Logger,
		"advance workflow token",
	)
	if err != nil {
		return models.WorkflowExecution{}, err
	}

	return updated, nil
}

func (bo *WorkflowExecutionBO) CompleteJob(
	ctx context.Context,
	scope models.WorkflowScope,
	jobID string,
	workerID string,
	outputData map[string]interface{},
	jobError string,
	cf, cfs string,
	tenantNode *dragonboat.RaftNode,
) (interface{}, error) {
	node, targetCF, targetCFS, err := bo.resolveRaftNode(scope, tenantNode)
	if err != nil {
		return nil, err
	}
	if scope == models.WorkflowScopeTenant {
		targetCF = cf
		targetCFS = cfs
	}

	cmd := &workflow_execution_command.CompleteJobCommand{
		JobID:      jobID,
		WorkerID:   workerID,
		OutputData: outputData,
		Error:      jobError,
		CF:         targetCF,
		CFS:        targetCFS,
	}

	timeout := config.GlobalConfiguration.ApiRaftTimeout
	res, err := dragonboat.ExecuteRepositoryCommand[interface{}](
		node,
		ctx,
		cmd,
		timeout,
		bo.Config.Logger,
		"complete workflow job",
	)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (bo *WorkflowExecutionBO) GetExecution(
	ctx context.Context,
	scope models.WorkflowScope,
	id string,
	cf, cfs string,
	tenantNode *dragonboat.RaftNode,
) (*workflow_execution_command.WorkflowExecutionDetail, error) {
	node, targetCF, targetCFS, err := bo.resolveRaftNode(scope, tenantNode)
	if err != nil {
		return nil, err
	}
	if scope == models.WorkflowScopeTenant {
		targetCF = cf
		targetCFS = cfs
	}

	cmd := &workflow_execution_command.GetWorkflowExecutionCommand{
		ID:  id,
		CF:  targetCF,
		CFS: targetCFS,
	}

	timeout := config.GlobalConfiguration.ApiRaftTimeout
	detail, err := dragonboat.ExecuteRepositoryQuery[workflow_execution_command.WorkflowExecutionDetail](
		node,
		ctx,
		cmd,
		timeout,
		bo.Config.Logger,
		"get workflow execution detail",
	)
	if err != nil {
		return nil, err
	}
	if detail.Execution.ID == "" {
		return nil, nil
	}
	return &detail, nil
}

func (bo *WorkflowExecutionBO) ListExecutions(
	ctx context.Context,
	scope models.WorkflowScope,
	vnamespace string,
	workflowDefinitionID string,
	status string,
	pageSize int,
	cursor string,
	cf, cfs string,
	tenantNode *dragonboat.RaftNode,
) (*db.FindResult[models.WorkflowExecution], error) {
	node, targetCF, targetCFS, err := bo.resolveRaftNode(scope, tenantNode)
	if err != nil {
		return nil, err
	}
	if scope == models.WorkflowScopeTenant {
		targetCF = cf
		targetCFS = cfs
	}

	cmd := &workflow_execution_command.ListWorkflowExecutionsCommand{
		VNamespace:           vnamespace,
		WorkflowDefinitionID: workflowDefinitionID,
		Status:               status,
		PageSize:             pageSize,
		Cursor:               cursor,
		CF:                   targetCF,
		CFS:                  targetCFS,
	}

	timeout := config.GlobalConfiguration.ApiRaftTimeout
	res, err := dragonboat.ExecuteRepositoryQuery[db.FindResult[models.WorkflowExecution]](
		node,
		ctx,
		cmd,
		timeout,
		bo.Config.Logger,
		"list workflow executions",
	)
	if err != nil {
		return nil, err
	}
	return &res, nil
}
