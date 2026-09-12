package db

import (
	"fmt"
	"time"

	models "deadalus-orch/shared/models"
)

type WorkflowDefinitionRepository struct {
	*Repository[models.WorkflowDefinition]
}

func NewWorkflowDefinitionRepository(uow *UnitOfWork, factory IDGeneratorFactory, cf, cfs string) (*WorkflowDefinitionRepository, error) {
	if uow == nil {
		return nil, fmt.Errorf("UnitOfWork is required")
	}
	repo, err := GetRepository[models.WorkflowDefinition](uow, cf, cfs, "admin_schema", factory)
	if err != nil {
		return nil, err
	}
	return &WorkflowDefinitionRepository{Repository: repo}, nil
}

func (r *WorkflowDefinitionRepository) CreateWorkflowDefinition(input *models.WorkflowDefinition, now time.Time) (string, error) {
	if input.Code == "" {
		return "", fmt.Errorf("Code is required")
	}
	if input.Version <= 0 {
		input.Version = 1
	}
	if input.PayloadFormat == "" {
		input.PayloadFormat = models.WorkflowPayloadFormatJSON
	}
	input.CreatedAt = now
	input.UpdatedAt = now
	return r.Create(input, now)
}

func (r *WorkflowDefinitionRepository) UpdateWorkflowDefinition(input *models.WorkflowDefinition, now time.Time) (bool, error) {
	input.UpdatedAt = now
	return r.Update(input, now)
}

func (r *WorkflowDefinitionRepository) GetWorkflowDefinitionByID(id string, now time.Time) (*models.WorkflowDefinition, error) {
	return r.FindByField("ID", id, now)
}

func (r *WorkflowDefinitionRepository) GetWorkflowDefinitionByCode(code string, vnamespace string, now time.Time) (*models.WorkflowDefinition, error) {
	if vnamespace == "" {
		vnamespace = "default"
	}
	query := fmt.Sprintf("Code = %s & VNamespace = %s", code, vnamespace)
	res, err := r.Find(query, 1, "", now)
	if err != nil || len(res.Entities) == 0 {
		return nil, err
	}
	return &res.Entities[0], nil
}

func (r *WorkflowDefinitionRepository) ListWorkflowDefinitions(scope string, tenantID string, pageSize int, cursor string, now time.Time) (*FindResult[models.WorkflowDefinition], error) {
	var query string
	if scope != "" {
		query = fmt.Sprintf("Scope = %s", scope)
		if scope == string(models.WorkflowScopeTenant) && tenantID != "" {
			query += fmt.Sprintf(" & TenantID = %s", tenantID)
		}
	} else if tenantID != "" {
		query = fmt.Sprintf("TenantID = %s", tenantID)
	}

	if query == "" {
		query = "ID != 0"
	}

	if pageSize <= 0 {
		pageSize = 50
	}

	return r.Find(query, pageSize, cursor, now)
}

func (r *WorkflowDefinitionRepository) DeleteWorkflowDefinition(id string, now time.Time) (bool, error) {
	return r.Delete(id, now)
}
