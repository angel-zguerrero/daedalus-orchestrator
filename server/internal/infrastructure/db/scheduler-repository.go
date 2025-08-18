package db

import (
	"fmt"
	"time"

	models "deadalus-orch/shared/models"
)

type NodeSchedulerRepository struct {
	*Repository[models.NodeScheduler]
}

func NewNodeSchedulerRepository(uow *UnitOfWork, factory IDGeneratorFactory) (*NodeSchedulerRepository, error) {
	if uow == nil {
		return nil, fmt.Errorf("UnitOfWork is required")
	}
	repo, err := GetRepository[models.NodeScheduler](uow, MasterEventFC, MasterEventFCSector, "admin_event_schema", factory)
	if err != nil {
		return nil, err
	}
	return &NodeSchedulerRepository{Repository: repo}, nil
}

func (r *NodeSchedulerRepository) CreateNodeScheduler(input *models.NodeScheduler, now time.Time) (string, error) {
	input.CreatedAt = now
	input.UpdatedAt = now
	return r.Create(input, now)
}

func (r *NodeSchedulerRepository) UpdateNodeScheduler(input *models.NodeScheduler, now time.Time) (bool, error) {
	input.UpdatedAt = now
	return r.Update(input, now)
}

func (r *NodeSchedulerRepository) GetNodeSchedulerByName(name string, now time.Time) (*models.NodeScheduler, error) {
	return r.FindByField("Name", name, now)
}
func (r *NodeSchedulerRepository) GetNodeSchedulerById(id string, now time.Time) (*models.NodeScheduler, error) {
	return r.FindByField("ID", id, now)
}

func (r *NodeSchedulerRepository) Paginate(q string, pageSize int, cursor string, now time.Time) (*FindResult[models.NodeScheduler], error) {
	if q == "" {
		return r.Find("ID != 0", pageSize, cursor, now) // ID != 0 Workaround
	} else {
		return r.Find("Name LIKE *"+q+"*", pageSize, cursor, now) // ID != 0 Workaround
	}
}

func (r *NodeSchedulerRepository) DeleteNodeSchedulerById(id string, now time.Time) (bool, error) {
	return r.Delete(id, now)
}
