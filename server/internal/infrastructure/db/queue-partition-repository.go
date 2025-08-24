package db

import (
	"fmt"
	"time"

	models "deadalus-orch/shared/models"
)

type QueuePartitionRepository struct {
	*Repository[models.QueuePartition]
}

func NewQueuePartitionRepository(uow *UnitOfWork, factory IDGeneratorFactory, cf, cfs string) (*QueuePartitionRepository, error) {
	if uow == nil {
		return nil, fmt.Errorf("UnitOfWork is required")
	}
	repo, err := GetRepository[models.QueuePartition](uow, cf, cfs, "admin_schema", factory)
	if err != nil {
		return nil, err
	}
	return &QueuePartitionRepository{Repository: repo}, nil
}

func (r *QueuePartitionRepository) CreateQueuePartition(input *models.QueuePartition, now time.Time) (string, error) {
	input.CreatedAt = now
	input.UpdatedAt = now
	return r.Create(input, now)
}

func (r *QueuePartitionRepository) UpdateQueuePartition(input *models.QueuePartition, now time.Time) (bool, error) {
	input.UpdatedAt = now
	return r.Update(input, now)
}

func (r *QueuePartitionRepository) GetQueuePartitionByQueueIDAndPriority(queueID string, priority int, now time.Time) (*models.QueuePartition, error) {
	query := fmt.Sprintf("QueueID = '%s' & Priority = %d", queueID, priority)
	result, err := r.Find(query, 1, "", now)
	if err != nil {
		return nil, err
	}
	if len(result.Entities) == 0 {
		return nil, nil
	}
	return &result.Entities[0], nil
}
