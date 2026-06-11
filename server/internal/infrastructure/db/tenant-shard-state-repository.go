package db

import (
	"fmt"
	"time"

	models "deadalus-orch/shared/models"
)

type TenantShardStateRepository struct {
	repo *Repository[models.TenantShardState]
}

func NewTenantShardStateRepository(uow *UnitOfWork, factory IDGeneratorFactory, adminFC string, adminFCSector string, schema string) (*TenantShardStateRepository, error) {
	if uow == nil {
		return nil, fmt.Errorf("UnitOfWork is required")
	}
	repo, err := GetRepository[models.TenantShardState](uow, adminFC, adminFCSector, schema, factory)
	if err != nil {
		return nil, err
	}
	return &TenantShardStateRepository{repo: repo}, nil
}

func (r *TenantShardStateRepository) GetByTenantID(tenantID string, now time.Time) (*models.TenantShardState, error) {
	return r.repo.FindByField("ID", tenantID, now)
}

func (r *TenantShardStateRepository) CreateOrUpdate(input *models.TenantShardState, now time.Time) (string, error) {
	input.UpdatedAt = now
	existing, err := r.GetByTenantID(input.ID, now)
	if err == nil && existing != nil {
		_, err = r.repo.Update(input, now)
		return input.ID, err
	}
	return r.repo.Create(input, now)
}

func (r *TenantShardStateRepository) DeleteByTenantID(tenantID string, now time.Time) (bool, error) {
	existing, err := r.GetByTenantID(tenantID, now)
	if err != nil || existing == nil {
		return false, err
	}
	return r.repo.Delete(existing.ID, now)
}
