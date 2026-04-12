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

// FindLeasesByQueueMessageIDs fetches the most recent lease for each given message ID
// and returns them as a map keyed by QueueMessageID.
func (r *QueueMessageLeaseRepository) FindLeasesByQueueMessageIDs(messageIDs []string, now time.Time) (map[string]*models.QueueMessageLease, error) {
	result := make(map[string]*models.QueueMessageLease, len(messageIDs))
	for _, id := range messageIDs {
		lease, err := r.GetQueueMessageLeaseByMessageID(id, now)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch lease for message %s: %w", id, err)
		}
		if lease != nil {
			result[id] = lease
		}
	}
	return result, nil
}

// FindExpiredLeases retrieves leases that have expired (LeaseUntil < now).
// Only returns active leases that have not yet been marked as expired.
// Results are paginated for efficient processing.
func (r *QueueMessageLeaseRepository) FindExpiredLeases(limit int, offset int, now time.Time) (*FindResult[models.QueueMessageLease], error) {
	query := fmt.Sprintf("LeaseStatus = '%s' & LeaseUntil < '%s'", models.QueueMessageLeaseStatusActive, now)

	// For pagination with offset, we'll need to fetch limit + offset records and skip the first offset
	// This is a limitation of the current Find implementation
	result, err := r.Find(query, limit, "", now)
	if err != nil {
		return nil, err
	}

	return &FindResult[models.QueueMessageLease]{
		Entities: result.Entities,
		Cursor:   result.Cursor,
	}, nil
}
