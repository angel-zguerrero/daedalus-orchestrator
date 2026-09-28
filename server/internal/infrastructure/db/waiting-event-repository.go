package db

import (
	"fmt"
	"time"

	models "deadalus-orch/shared/models"
)

type WaitingEventRepository struct {
	*Repository[models.WaitingEvent]
}

func NewWaitingEventRepository(uow *UnitOfWork, factory IDGeneratorFactory, cf, cfs string) (*WaitingEventRepository, error) {
	if uow == nil {
		return nil, fmt.Errorf("UnitOfWork is required")
	}
	repo, err := GetRepository[models.WaitingEvent](uow, cf, cfs, "admin_schema", factory)
	if err != nil {
		return nil, err
	}
	return &WaitingEventRepository{Repository: repo}, nil
}

func (r *WaitingEventRepository) CreateWaitingEvent(input *models.WaitingEvent, now time.Time) (string, error) {
	if input.WorkflowExecutionID == "" {
		return "", fmt.Errorf("WorkflowExecutionID is required")
	}
	if input.ExecutionTokenID == "" {
		return "", fmt.Errorf("ExecutionTokenID is required")
	}
	if input.TTL <= 0 {
		input.TTL = models.CalculateWorkflowExecutionTTL(0)
	}
	input.CreatedAt = now
	input.UpdatedAt = now
	return r.Create(input, now)
}

func (r *WaitingEventRepository) UpdateWaitingEvent(input *models.WaitingEvent, now time.Time) (bool, error) {
	input.UpdatedAt = now
	return r.Update(input, now)
}

func (r *WaitingEventRepository) GetWaitingEventByID(id string, now time.Time) (*models.WaitingEvent, error) {
	return r.FindByField("ID", id, now)
}

func (r *WaitingEventRepository) GetWaitingEventsByExecutionID(executionID string, now time.Time) ([]models.WaitingEvent, error) {
	query := fmt.Sprintf("WorkflowExecutionID = %s", executionID)
	res, err := r.Find(query, 1000, "", now)
	if err != nil {
		return nil, err
	}
	return res.Entities, nil
}

func (r *WaitingEventRepository) GetWaitingEventsByTokenID(tokenID string, now time.Time) ([]models.WaitingEvent, error) {
	query := fmt.Sprintf("ExecutionTokenID = %s", tokenID)
	res, err := r.Find(query, 1000, "", now)
	if err != nil {
		return nil, err
	}
	return res.Entities, nil
}

func (r *WaitingEventRepository) DeleteWaitingEvent(id string, now time.Time) (bool, error) {
	return r.Delete(id, now)
}
