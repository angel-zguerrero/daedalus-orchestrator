package db

import (
	"fmt"
	"time"

	models "deadalus-orch/shared/models"
)

type TenantSummaryRepository struct {
	*Repository[models.TenantSummary]
}

func NewTenantSummaryRepository(uow *UnitOfWork, factory IDGeneratorFactory) (*TenantSummaryRepository, error) {
	if uow == nil {
		return nil, fmt.Errorf("UnitOfWork is required")
	}
	repo, err := GetRepository[models.TenantSummary](uow, AdminFC, AdminFCSector, "admin_schema", factory)
	if err != nil {
		return nil, err
	}
	return &TenantSummaryRepository{Repository: repo}, nil
}

func (r *TenantSummaryRepository) CreateTenantSummary(input *models.TenantSummary, now time.Time) (string, error) {
	input.CreatedAt = now
	input.UpdatedAt = now
	return r.Create(input, now)
}

func (r *TenantSummaryRepository) UpdateTenantSummary(input *models.TenantSummary, now time.Time) (bool, error) {
	input.UpdatedAt = now
	return r.Update(input, now)
}

func (r *TenantSummaryRepository) GetTenantSummaryById(id string, now time.Time) (*models.TenantSummary, error) {
	return r.FindByField("ID", id, now)
}

func (r *TenantSummaryRepository) Paginate(q string, pageSize int, cursor string, now time.Time) (*FindResult[models.TenantSummary], error) {
	if q == "" {
		return r.Find("ID != 0", pageSize, cursor, now) // ID != 0 Workaround
	} else {
		return r.Find("Name LIKE *"+q+"*", pageSize, cursor, now) // ID != 0 Workaround
	}
}

func (r *TenantSummaryRepository) DeleteTenantSummaryById(id string, now time.Time) (bool, error) {
	return r.Delete(id, now)
}

// IncreaseMessageCount increases the MessageCount for a tenant by the specified amount
// If the tenant doesn't exist, creates a new record with MessageCount = amount
func (r *TenantSummaryRepository) IncreaseMessageCount(tenantId string, amount int, now time.Time) error {
	summary, err := r.GetTenantSummaryById(tenantId, now)
	if err != nil {
		return err
	}

	if summary == nil {
		// If tenant doesn't exist, create new record
		newSummary := &models.TenantSummary{
			ID:             tenantId,
			MessagesCount:  amount,
			ExchangesCount: 0,
			QueuesCount:    0,
		}
		_, err = r.CreateTenantSummary(newSummary, now)
		return err
	}

	// Increase existing count
	summary.MessagesCount += amount
	_, err = r.UpdateTenantSummary(summary, now)
	return err
}

// DecreaseMessageCount decreases the MessageCount for a tenant by the specified amount
// If the tenant doesn't exist, creates a new record with MessageCount = 0
func (r *TenantSummaryRepository) DecreaseMessageCount(tenantId string, amount int, now time.Time) error {
	summary, err := r.GetTenantSummaryById(tenantId, now)
	if err != nil {
		return err
	}

	if summary == nil {
		// If tenant doesn't exist, create new record with 0
		newSummary := &models.TenantSummary{
			ID:             tenantId,
			MessagesCount:  0,
			ExchangesCount: 0,
			QueuesCount:    0,
		}
		_, err = r.CreateTenantSummary(newSummary, now)
		return err
	}

	// Decrease existing count (ensure it doesn't go below 0)
	summary.MessagesCount -= amount
	if summary.MessagesCount < 0 {
		summary.MessagesCount = 0
	}
	_, err = r.UpdateTenantSummary(summary, now)
	return err
}

// IncreaseExchangeCount increases the ExchangeCount for a tenant by the specified amount
// If the tenant doesn't exist, creates a new record with ExchangeCount = amount
func (r *TenantSummaryRepository) IncreaseExchangeCount(tenantId string, amount int, now time.Time) error {
	summary, err := r.GetTenantSummaryById(tenantId, now)
	if err != nil {
		return err
	}

	if summary == nil {
		// If tenant doesn't exist, create new record
		newSummary := &models.TenantSummary{
			ID:             tenantId,
			MessagesCount:  0,
			ExchangesCount: amount,
			QueuesCount:    0,
		}
		_, err = r.CreateTenantSummary(newSummary, now)
		return err
	}

	// Increase existing count
	summary.ExchangesCount += amount
	_, err = r.UpdateTenantSummary(summary, now)
	return err
}

// DecreaseExchangeCount decreases the ExchangeCount for a tenant by the specified amount
// If the tenant doesn't exist, creates a new record with ExchangeCount = 0
func (r *TenantSummaryRepository) DecreaseExchangeCount(tenantId string, amount int, now time.Time) error {
	summary, err := r.GetTenantSummaryById(tenantId, now)
	if err != nil {
		return err
	}

	if summary == nil {
		// If tenant doesn't exist, create new record with 0
		newSummary := &models.TenantSummary{
			ID:             tenantId,
			MessagesCount:  0,
			ExchangesCount: 0,
			QueuesCount:    0,
		}
		_, err = r.CreateTenantSummary(newSummary, now)
		return err
	}

	// Decrease existing count (ensure it doesn't go below 0)
	summary.ExchangesCount -= amount
	if summary.ExchangesCount < 0 {
		summary.ExchangesCount = 0
	}
	_, err = r.UpdateTenantSummary(summary, now)
	return err
}

// IncreaseQueueCount increases the QueueCount for a tenant by the specified amount
// If the tenant doesn't exist, creates a new record with QueueCount = amount
func (r *TenantSummaryRepository) IncreaseQueueCount(tenantId string, amount int, now time.Time) error {
	summary, err := r.GetTenantSummaryById(tenantId, now)
	if err != nil {
		return err
	}

	if summary == nil {
		// If tenant doesn't exist, create new record
		newSummary := &models.TenantSummary{
			ID:             tenantId,
			MessagesCount:  0,
			ExchangesCount: 0,
			QueuesCount:    amount,
		}
		_, err = r.CreateTenantSummary(newSummary, now)
		return err
	}

	// Increase existing count
	summary.QueuesCount += amount
	_, err = r.UpdateTenantSummary(summary, now)
	return err
}

// DecreaseQueueCount decreases the QueueCount for a tenant by the specified amount
// If the tenant doesn't exist, creates a new record with QueueCount = 0
func (r *TenantSummaryRepository) DecreaseQueueCount(tenantId string, amount int, now time.Time) error {
	summary, err := r.GetTenantSummaryById(tenantId, now)
	if err != nil {
		return err
	}

	if summary == nil {
		// If tenant doesn't exist, create new record with 0
		newSummary := &models.TenantSummary{
			ID:             tenantId,
			MessagesCount:  0,
			ExchangesCount: 0,
			QueuesCount:    0,
		}
		_, err = r.CreateTenantSummary(newSummary, now)
		return err
	}

	// Decrease existing count (ensure it doesn't go below 0)
	summary.QueuesCount -= amount
	if summary.QueuesCount < 0 {
		summary.QueuesCount = 0
	}
	_, err = r.UpdateTenantSummary(summary, now)
	return err
}
