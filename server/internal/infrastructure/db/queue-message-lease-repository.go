package db

import (
	"fmt"
	"time"

	models "deadalus-orch/shared/models"
)

// QueueMessageLeaseRepository provides CRUD operations for QueueMessageLease entities.
type QueueMessageLeaseRepository struct {
	*Repository[models.QueueMessageLease]
}

// NewQueueMessageLeaseRepository creates a new QueueMessageLeaseRepository bound to the
// given UnitOfWork, column family (cf) and column family set (cfs).
func NewQueueMessageLeaseRepository(uow *UnitOfWork, factory IDGeneratorFactory, cf, cfs string) (*QueueMessageLeaseRepository, error) {
	if uow == nil {
		return nil, fmt.Errorf("UnitOfWork is required")
	}
	repo, err := GetRepository[models.QueueMessageLease](uow, cf, cfs, "admin_schema", factory)
	if err != nil {
		return nil, err
	}
	return &QueueMessageLeaseRepository{Repository: repo}, nil
}

// CreateQueueMessageLease persists a new lease record.
func (r *QueueMessageLeaseRepository) CreateQueueMessageLease(input *models.QueueMessageLease, now time.Time) (string, error) {
	return r.Create(input, now)
}

// GetQueueMessageLeaseByID retrieves a single lease by its primary key.
func (r *QueueMessageLeaseRepository) GetQueueMessageLeaseByID(id string, now time.Time) (*models.QueueMessageLease, error) {
	return r.FindByField("ID", id, now)
}

// GetQueueMessageLeaseByMessageID retrieves the active lease for a given queue message.
func (r *QueueMessageLeaseRepository) GetQueueMessageLeaseByMessageID(queueMessageID string, now time.Time) (*models.QueueMessageLease, error) {
	query := fmt.Sprintf("QueueMessageID = '%s'", queueMessageID)
	result, err := r.Find(query, 1, "", now)
	if err != nil {
		return nil, err
	}
	if len(result.Entities) == 0 {
		return nil, nil
	}
	return &result.Entities[0], nil
}

// UpdateQueueMessageLease updates an existing lease record.
func (r *QueueMessageLeaseRepository) UpdateQueueMessageLease(input *models.QueueMessageLease, now time.Time) (bool, error) {
	return r.Update(input, now)
}
