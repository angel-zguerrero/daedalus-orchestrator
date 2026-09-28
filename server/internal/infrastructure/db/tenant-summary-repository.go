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

func (r *TenantSummaryRepository) PaginateTenantUpdatedAtFrom(lastUpdatedAt time.Time, pageSize int, cursor string, now time.Time) (*FindResult[models.TenantSummary], error) {
	filter := fmt.Sprintf("UpdatedAt >= '%s'", lastUpdatedAt)
	return r.Find(filter, pageSize, cursor, now)
}

// UpdateCounters allows updating multiple counters in a single operation
// Positive values increase counters, negative values decrease them
func (r *TenantSummaryRepository) UpdateCounters(tenantId string, messagesChange, queuesChange int, now time.Time, workflowsChange ...int) error {
	wfChange := 0
	if len(workflowsChange) > 0 {
		wfChange = workflowsChange[0]
	}

	summary, err := r.GetTenantSummaryById(tenantId, now)
	if err != nil {
		return err
	}

	if summary == nil {
		// If tenant doesn't exist, create new record
		newSummary := &models.TenantSummary{
			ID:             tenantId,
			WorkflowsCount: max(0, wfChange),
			MessagesCount:  max(0, messagesChange),
			QueuesCount:    max(0, queuesChange),
		}
		_, err = r.CreateTenantSummary(newSummary, now)
		return err
	}

	// Update existing counts (ensure they don't go below 0)
	summary.WorkflowsCount = max(0, summary.WorkflowsCount+wfChange)
	summary.MessagesCount = max(0, summary.MessagesCount+messagesChange)
	summary.QueuesCount = max(0, summary.QueuesCount+queuesChange)

	_, err = r.UpdateTenantSummary(summary, now)
	return err
}

// UpdateWorkflowCounter updates only the workflow counter for a tenant
func (r *TenantSummaryRepository) UpdateWorkflowCounter(tenantId string, workflowsChange int, now time.Time) error {
	return r.UpdateCounters(tenantId, 0, 0, now, workflowsChange)
}

// Helper function to get the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
