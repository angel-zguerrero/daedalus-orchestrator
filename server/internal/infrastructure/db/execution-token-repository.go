package db

import (
	"fmt"
	"time"

	models "deadalus-orch/shared/models"
)

type ExecutionTokenRepository struct {
	*Repository[models.ExecutionToken]
}

func NewExecutionTokenRepository(uow *UnitOfWork, factory IDGeneratorFactory, cf, cfs string) (*ExecutionTokenRepository, error) {
	if uow == nil {
		return nil, fmt.Errorf("UnitOfWork is required")
	}
	repo, err := GetRepository[models.ExecutionToken](uow, cf, cfs, "admin_schema", factory)
	if err != nil {
		return nil, err
	}
	return &ExecutionTokenRepository{Repository: repo}, nil
}

func (r *ExecutionTokenRepository) CreateExecutionToken(input *models.ExecutionToken, now time.Time) (string, error) {
	if input.WorkflowExecutionID == "" {
		return "", fmt.Errorf("WorkflowExecutionID is required")
	}
	if input.Status == "" {
		input.Status = models.ExecutionTokenStatusActive
	}
	input.CreatedAt = now
	input.UpdatedAt = now
	return r.Create(input, now)
}

func (r *ExecutionTokenRepository) UpdateExecutionToken(input *models.ExecutionToken, now time.Time) (bool, error) {
	input.UpdatedAt = now
	return r.Update(input, now)
}

func (r *ExecutionTokenRepository) GetExecutionTokenByID(id string, now time.Time) (*models.ExecutionToken, error) {
	return r.FindByField("ID", id, now)
}

func (r *ExecutionTokenRepository) GetTokensByExecutionID(executionID string, now time.Time) ([]models.ExecutionToken, error) {
	query := fmt.Sprintf("WorkflowExecutionID = %s", executionID)
	res, err := r.Find(query, 1000, "", now)
	if err != nil {
		return nil, err
	}
	return res.Entities, nil
}

func (r *ExecutionTokenRepository) GetActiveTokensByExecutionID(executionID string, now time.Time) ([]models.ExecutionToken, error) {
	query := fmt.Sprintf("WorkflowExecutionID = %s & Status = %s", executionID, string(models.ExecutionTokenStatusActive))
	res, err := r.Find(query, 1000, "", now)
	if err != nil {
		return nil, err
	}
	return res.Entities, nil
}

func (r *ExecutionTokenRepository) DeleteExecutionToken(id string, now time.Time) (bool, error) {
	return r.Delete(id, now)
}
