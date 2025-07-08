package db

import (
	"fmt"
	"time"

	models "deadalus-orch/shared/models"
)

type TenantInMasterRepository struct {
	repo *Repository[models.TenantInMaster]
}

func NewTenantInMasterRepository(uow *UnitOfWork, factory IDGeneratorFactory) (*TenantInMasterRepository, error) {
	if uow == nil {
		return nil, fmt.Errorf("UnitOfWork is required")
	}
	repo, err := GetRepository[models.TenantInMaster](uow, AdminFC, "admin_schema", factory)
	if err != nil {
		return nil, err
	}
	return &TenantInMasterRepository{repo: repo}, nil
}

func (r *TenantInMasterRepository) CreateTenantInMaster(input models.TenantInMaster, now time.Time) (string, error) {
	input.IsAssignedToShard = false
	return r.repo.Create(&input, now)
}

func (r *TenantInMasterRepository) GetTenantInMasterByTenantCode(code string) (*models.TenantInMaster, error) {
	return r.repo.FindByField("Code", code, time.Now())
}

func (r *TenantInMasterRepository) DeleteTenantInMasterByCode(code string, now time.Time) (bool, error) {
	rootTenantInMaster, err := r.repo.FindByField("Code", code, now)
	if err != nil || rootTenantInMaster == nil {
		return false, err
	}
	if err != nil {
		return false, err
	}

	return r.repo.Delete(rootTenantInMaster.ID, now)
}
