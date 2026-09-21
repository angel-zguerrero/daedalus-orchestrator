package db

import (
	"fmt"
	"strings"
	"time"

	models "deadalus-orch/shared/models"
)

type WorkflowExecutionRepository struct {
	*Repository[models.WorkflowExecution]
}

func NewWorkflowExecutionRepository(uow *UnitOfWork, factory IDGeneratorFactory, cf, cfs string) (*WorkflowExecutionRepository, error) {
	if uow == nil {
		return nil, fmt.Errorf("UnitOfWork is required")
	}
	repo, err := GetRepository[models.WorkflowExecution](uow, cf, cfs, "admin_schema", factory)
	if err != nil {
		return nil, err
	}
	return &WorkflowExecutionRepository{Repository: repo}, nil
}

func (r *WorkflowExecutionRepository) CreateWorkflowExecution(input *models.WorkflowExecution, now time.Time) (string, error) {
	if input.WorkflowDefinitionID == "" {
		return "", fmt.Errorf("WorkflowDefinitionID is required")
	}
	if input.Status == "" {
		input.Status = models.WorkflowExecutionStatusPending
	}
	input.CreatedAt = now
	input.UpdatedAt = now
	return r.Create(input, now)
}

func (r *WorkflowExecutionRepository) UpdateWorkflowExecution(input *models.WorkflowExecution, now time.Time) (bool, error) {
	input.UpdatedAt = now
	return r.Update(input, now)
}

func (r *WorkflowExecutionRepository) GetWorkflowExecutionByID(id string, now time.Time) (*models.WorkflowExecution, error) {
	return r.FindByField("ID", id, now)
}

func (r *WorkflowExecutionRepository) ListWorkflowExecutions(vnamespace string, workflowDefinitionID string, status string, pageSize int, cursor string, now time.Time) (*FindResult[models.WorkflowExecution], error) {
	var conditions []string

	if vnamespace != "" {
		conditions = append(conditions, fmt.Sprintf("VNamespace = %s", vnamespace))
	}
	if workflowDefinitionID != "" {
		conditions = append(conditions, fmt.Sprintf("WorkflowDefinitionID = %s", workflowDefinitionID))
	}
	if status != "" {
		conditions = append(conditions, fmt.Sprintf("Status = %s", status))
	}

	var query string
	if len(conditions) == 0 {
		query = "ID != 0"
	} else {
		query = strings.Join(conditions, " & ")
	}

	if pageSize <= 0 {
		pageSize = 50
	}

	return r.Find(query, pageSize, cursor, now)
}

func (r *WorkflowExecutionRepository) DeleteWorkflowExecution(id string, now time.Time) (bool, error) {
	return r.Delete(id, now)
}

func (r *WorkflowExecutionRepository) GetActiveExecutionByBusinessKey(vnamespace string, workflowDefinitionID string, executionKey string, now time.Time) (*models.WorkflowExecution, error) {
	if executionKey == "" {
		return nil, nil
	}
	var conditions []string
	if vnamespace != "" {
		conditions = append(conditions, fmt.Sprintf("VNamespace = %s", vnamespace))
	}
	if workflowDefinitionID != "" {
		conditions = append(conditions, fmt.Sprintf("WorkflowDefinitionID = %s", workflowDefinitionID))
	}
	conditions = append(conditions, fmt.Sprintf("ExecutionKey = %s", executionKey))

	query := strings.Join(conditions, " & ")
	res, err := r.Find(query, 100, "", now)
	if err != nil || res == nil {
		return nil, err
	}

	for i := range res.Entities {
		exec := &res.Entities[i]
		if exec.Status == models.WorkflowExecutionStatusRunning || exec.Status == models.WorkflowExecutionStatusPending {
			return exec, nil
		}
	}
	return nil, nil
}
