package db

import (
	"fmt"
	"time"

	models "deadalus-orch/shared/models"
)

type WorkflowJobRepository struct {
	*Repository[models.WorkflowJob]
}

func NewWorkflowJobRepository(uow *UnitOfWork, factory IDGeneratorFactory, cf, cfs string) (*WorkflowJobRepository, error) {
	if uow == nil {
		return nil, fmt.Errorf("UnitOfWork is required")
	}
	repo, err := GetRepository[models.WorkflowJob](uow, cf, cfs, "admin_schema", factory)
	if err != nil {
		return nil, err
	}
	return &WorkflowJobRepository{Repository: repo}, nil
}

func (r *WorkflowJobRepository) CreateWorkflowJob(input *models.WorkflowJob, now time.Time) (string, error) {
	if input.WorkflowExecutionID == "" {
		return "", fmt.Errorf("WorkflowExecutionID is required")
	}
	if input.Status == "" {
		input.Status = models.WorkflowJobStatusPending
	}
	input.CreatedAt = now
	input.UpdatedAt = now
	return r.Create(input, now)
}

func (r *WorkflowJobRepository) UpdateWorkflowJob(input *models.WorkflowJob, now time.Time) (bool, error) {
	input.UpdatedAt = now
	return r.Update(input, now)
}

func (r *WorkflowJobRepository) GetWorkflowJobByID(id string, now time.Time) (*models.WorkflowJob, error) {
	return r.FindByField("ID", id, now)
}

func (r *WorkflowJobRepository) GetPendingJobsByExecutionID(executionID string, now time.Time) ([]models.WorkflowJob, error) {
	query := fmt.Sprintf("WorkflowExecutionID = %s & Status = %s", executionID, string(models.WorkflowJobStatusPending))
	res, err := r.Find(query, 1000, "", now)
	if err != nil {
		return nil, err
	}
	return res.Entities, nil
}

func (r *WorkflowJobRepository) GetJobsByExecutionID(executionID string, now time.Time) ([]models.WorkflowJob, error) {
	query := fmt.Sprintf("WorkflowExecutionID = %s", executionID)
	res, err := r.Find(query, 1000, "", now)
	if err != nil {
		return nil, err
	}
	return res.Entities, nil
}

func (r *WorkflowJobRepository) DeleteWorkflowJob(id string, now time.Time) (bool, error) {
	return r.Delete(id, now)
}
