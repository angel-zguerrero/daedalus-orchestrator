package db

import (
	"fmt"
	"time"

	models "deadalus-orch/shared/models"
)

type QueueMessageRepository struct {
	*Repository[models.QueueMessage]
}

func NewQueueMessageRepository(uow *UnitOfWork, factory IDGeneratorFactory, cf, cfs string) (*QueueMessageRepository, error) {
	if uow == nil {
		return nil, fmt.Errorf("UnitOfWork is required")
	}
	repo, err := GetRepository[models.QueueMessage](uow, cf, cfs, "admin_schema", factory)
	if err != nil {
		return nil, err
	}
	return &QueueMessageRepository{Repository: repo}, nil
}

func (r *QueueMessageRepository) CreateQueueMessage(input *models.QueueMessage, now time.Time) (string, error) {
	input.CreatedAt = now
	input.UpdatedAt = now
	return r.Create(input, now)
}

func (r *QueueMessageRepository) UpdateQueueMessage(input *models.QueueMessage, now time.Time) (bool, error) {
	input.UpdatedAt = now
	return r.Update(input, now)
}

func (r *QueueMessageRepository) GetQueueMessageById(id string, now time.Time) (*models.QueueMessage, error) {
	return r.FindByField("ID", id, now)
}

func (r *QueueMessageRepository) PaginateByQueueID(queueID string, pageSize int, cursor string, now time.Time) (*FindResult[models.QueueMessage], error) {
	filter := fmt.Sprintf("QueueID = '%s'", queueID)
	return r.Find(filter, pageSize, cursor, now)
}
