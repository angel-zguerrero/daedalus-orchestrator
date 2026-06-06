package db

import (
	"fmt"
	"time"

	models "deadalus-orch/shared/models"
)

// DashboardSummaryRepository provides access to the single global DashboardSummary record
// stored in the master node's admin column family.
type DashboardSummaryRepository struct {
	repo *Repository[models.DashboardSummary]
}

func NewDashboardSummaryRepository(uow *UnitOfWork, factory IDGeneratorFactory) (*DashboardSummaryRepository, error) {
	if uow == nil {
		return nil, fmt.Errorf("UnitOfWork is required")
	}
	repo, err := GetRepository[models.DashboardSummary](uow, AdminFC, AdminFCSector, "admin_schema", factory)
	if err != nil {
		return nil, err
	}
	return &DashboardSummaryRepository{repo: repo}, nil
}

// GetDashboardSummary retrieves the single global dashboard summary record.
func (r *DashboardSummaryRepository) GetDashboardSummary(now time.Time) (*models.DashboardSummary, error) {
	return r.repo.FindByField("ID", models.DashboardSummaryID, now)
}

// UpsertDashboardSummary creates or updates the global dashboard summary record.
func (r *DashboardSummaryRepository) UpsertDashboardSummary(input *models.DashboardSummary, now time.Time) error {
	input.ID = models.DashboardSummaryID
	input.UpdatedAt = now

	existing, err := r.GetDashboardSummary(now)
	if err != nil {
		return err
	}

	if existing == nil {
		_, err = r.repo.Create(input, now)
		return err
	}

	_, err = r.repo.Update(input, now)
	return err
}
