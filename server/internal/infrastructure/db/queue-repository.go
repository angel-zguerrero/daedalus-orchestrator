package db

import (
	"fmt"
	"strings"
	"time"

	models "deadalus-orch/shared/models"
)

type QueueRepository struct {
	*Repository[models.Queue]
}

func NewQueueRepository(uow *UnitOfWork, factory IDGeneratorFactory, cf, cfs string) (*QueueRepository, error) {
	if uow == nil {
		return nil, fmt.Errorf("UnitOfWork is required")
	}
	repo, err := GetRepository[models.Queue](uow, cf, cfs, "admin_schema", factory)
	if err != nil {
		return nil, err
	}
	return &QueueRepository{Repository: repo}, nil
}

func (r *QueueRepository) CreateQueue(input *models.Queue, now time.Time) (string, error) {
	// Validate QueueType
	if !isValidQueueType(input.Type) {
		return "", fmt.Errorf("invalid queue type: %s. Valid types are: standard", input.Type)
	}

	// Validate DefaultQueueMessageTTL must be >= 0
	if input.DefaultQueueMessageTTL < 0 {
		return "", fmt.Errorf("DefaultQueueMessageTTL cannot be negative, got: %d", input.DefaultQueueMessageTTL)
	}

	// Validate DefaultQueueMessageDelayTime must be >= 0
	if input.DefaultQueueMessageDelayTime < 0 {
		return "", fmt.Errorf("DefaultQueueMessageDelayTime cannot be negative, got: %d", input.DefaultQueueMessageDelayTime)
	}

	// Validate QueueExpires must be >= 0
	if input.QueueExpires < 0 {
		return "", fmt.Errorf("QueueExpires cannot be negative, got: %d", input.QueueExpires)
	}

	// Validate ExpireAt usage: only use if DefaultQueueMessageDelayTime > 0
	if input.ExpireAt != nil && input.DefaultQueueMessageDelayTime <= 0 {
		return "", fmt.Errorf("ExpireAt can only be set when DefaultQueueMessageDelayTime is greater than 0")
	}

	// Validate MaxAttempts
	if input.MaxAttempts <= 0 {
		return "", fmt.Errorf("MaxAttempts must be greater than 0, got: %d", input.MaxAttempts)
	}

	// Set default values if not provided
	if input.MaxAttempts == 0 {
		input.MaxAttempts = 1
	}

	input.CreatedAt = now
	input.UpdatedAt = now
	return r.Create(input, now)
}

func (r *QueueRepository) UpdateQueue(input *models.Queue, now time.Time) (bool, error) {
	// Validate QueueType
	if !isValidQueueType(input.Type) {
		return false, fmt.Errorf("invalid queue type: %s. Valid types are: standard", input.Type)
	}

	// Validate DefaultQueueMessageTTL must be >= 0
	if input.DefaultQueueMessageTTL < 0 {
		return false, fmt.Errorf("DefaultQueueMessageTTL cannot be negative, got: %d", input.DefaultQueueMessageTTL)
	}

	// Validate DefaultQueueMessageDelayTime must be >= 0
	if input.DefaultQueueMessageDelayTime < 0 {
		return false, fmt.Errorf("DefaultQueueMessageDelayTime cannot be negative, got: %d", input.DefaultQueueMessageDelayTime)
	}

	// Validate QueueExpires must be >= 0
	if input.QueueExpires < 0 {
		return false, fmt.Errorf("QueueExpires cannot be negative, got: %d", input.QueueExpires)
	}

	// Validate ExpireAt usage: only use if DefaultQueueMessageDelayTime > 0
	if input.ExpireAt != nil && input.DefaultQueueMessageDelayTime <= 0 {
		return false, fmt.Errorf("ExpireAt can only be set when DefaultQueueMessageDelayTime is greater than 0")
	}

	// Validate MaxAttempts
	if input.MaxAttempts <= 0 {
		return false, fmt.Errorf("MaxAttempts must be greater than 0, got: %d", input.MaxAttempts)
	}

	input.UpdatedAt = now
	return r.Update(input, now)
}

func (r *QueueRepository) GetQueueByCode(code string, vnamespace string, now time.Time) (*models.Queue, error) {
	query := "Code = " + code + " & VNamespace = " + vnamespace
	result, err := r.Find(query, 1, "", now)
	if err != nil {
		return nil, err
	}
	if len(result.Entities) == 0 {
		return nil, nil
	}
	return &result.Entities[0], nil
}

func (r *QueueRepository) GetQueueById(id string, now time.Time) (*models.Queue, error) {
	return r.FindByField("ID", id, now)
}

func (r *QueueRepository) Paginate(q string, pageSize int, cursor string, vNamespace string, now time.Time) (*FindResult[models.Queue], error) {
	return r.paginate(q, "", pageSize, cursor, vNamespace, now)
}

func (r *QueueRepository) PaginateBySupervisionState(q string, supervisionState models.QueueSupervisionState, pageSize int, cursor string, vNamespace string, now time.Time) (*FindResult[models.Queue], error) {
	return r.paginate(q, supervisionState, pageSize, cursor, vNamespace, now)
}

func (r *QueueRepository) paginate(q string, supervisionState models.QueueSupervisionState, pageSize int, cursor string, vNamespace string, now time.Time) (*FindResult[models.Queue], error) {
	var query string

	if q == "" && vNamespace == "" && supervisionState == "" {
		query = "ID != 0" // ID != 0 Workaround
	} else {
		var conditions []string

		// Add name search condition if q is provided
		if q != "" {
			conditions = append(conditions, "Name LIKE *"+q+"*")
		}

		// Add vNamespace filter condition if vNamespace is provided
		if vNamespace != "" {
			conditions = append(conditions, "VNamespace = "+vNamespace)
		}

		// Add supervisionState filter condition if supervisionState is provided
		if supervisionState != "" {
			conditions = append(conditions, "NodeSchedulerQueueSupervisionState = "+string(supervisionState))
		}

		// If no conditions but we got here, use the workaround
		if len(conditions) == 0 {
			query = "ID != 0"
		} else {
			query = strings.Join(conditions, " & ")
		}
	}

	return r.Find(query, pageSize, cursor, now)
}

func (r *QueueRepository) DeleteQueueById(id string, now time.Time) (bool, error) {
	return r.Delete(id, now)
}

// PaginateWithClaimWorkFilter paginates queues applying the DB-level rules from the ClaimWorkFilter.
// Only queues with MessagesCount > 0 are returned. Inclusion lists, exact exclusions, and NOT LIKE
// pattern exclusions are all pushed to the DB query.
func (r *QueueRepository) PaginateWithClaimWorkFilter(f models.ClaimWorkFilter, pageSize int, cursor string, now time.Time) (*FindResult[models.Queue], error) {
	fq := BuildQueueFilterQuery(f)

	return r.Find(fq.DBQuery, pageSize, cursor, now)
}

// isValidQueueType validates if the queue type is one of the allowed types
func isValidQueueType(queueType models.QueueType) bool {
	switch queueType {
	case models.StandardQueue:
		return true
	default:
		return false
	}
}
