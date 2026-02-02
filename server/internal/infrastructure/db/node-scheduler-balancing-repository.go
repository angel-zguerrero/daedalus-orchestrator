package db

import (
	"fmt"
	"time"

	models "deadalus-orch/shared/models"
)

type NodeSchedulerBalancingRepository struct {
	*Repository[models.NodeSchedulerBalancingState]
}

func NewNodeSchedulerBalancingRepository(uow *UnitOfWork, factory IDGeneratorFactory) (*NodeSchedulerBalancingRepository, error) {
	if uow == nil {
		return nil, fmt.Errorf("UnitOfWork is required")
	}
	repo, err := GetRepository[models.NodeSchedulerBalancingState](uow, AdminFC, AdminFCSector, "admin_schema", factory)
	if err != nil {
		return nil, err
	}
	return &NodeSchedulerBalancingRepository{Repository: repo}, nil
}

func (r *NodeSchedulerBalancingRepository) UpsertState(input *models.NodeSchedulerBalancingState, now time.Time) (bool, error) {
	input.UpdatedAt = now
	existing, err := r.GetState(now)
	if err != nil {
		return false, err
	}

	if existing == nil {
		input.CreatedAt = now
		_, err = r.Create(input, now)
		return true, err
	}

	return r.Update(input, now)
}

func (r *NodeSchedulerBalancingRepository) GetState(now time.Time) (*models.NodeSchedulerBalancingState, error) {
	return r.FindByField("ID", models.NodeSchedulerBalancingStateID, now)
}
