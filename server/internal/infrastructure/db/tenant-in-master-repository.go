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
	repo, err := GetRepository[models.TenantInMaster](uow, AdminFC, AdminFCSector, "admin_schema", factory)
	if err != nil {
		return nil, err
	}
	return &TenantInMasterRepository{repo: repo}, nil
}

func (r *TenantInMasterRepository) CreateTenantInMaster(input *models.TenantInMaster, now time.Time) (string, error) {
	input.Status = models.PendingForAssign
	input.CreatedAt = now
	input.UpdatedAt = now
	return r.repo.Create(input, now)
}

func (r *TenantInMasterRepository) UpdateTenantInMaster(input *models.TenantInMaster, now time.Time) (bool, error) {
	input.UpdatedAt = now
	return r.repo.Update(input, now)
}

func (r *TenantInMasterRepository) GetTenantInMasterByTenantCode(code string, now time.Time) (*models.TenantInMaster, error) {
	return r.repo.FindByField("Code", code, now)
}
func (r *TenantInMasterRepository) GetTenantInMasterByTenantId(id string, now time.Time) (*models.TenantInMaster, error) {
	return r.repo.FindByField("ID", id, now)
}

func (r *TenantInMasterRepository) PaginateTenant(q string, pageSize int, cursor string, now time.Time) (*FindResult[models.TenantInMaster], error) {
	if q == "" {
		return r.repo.Find("ID != 0", pageSize, cursor, now) // ID != 0 Workaround
	} else {
		return r.repo.Find("Code LIKE *"+q+"* | Name LIKE *"+q+"*", pageSize, cursor, now) // ID != 0 Workaround
	}
}

// PaginateWithClaimWorkFilter paginates tenants applying the DB-level rules derived from the
// ClaimWorkFilter (inclusion lists, exact exclusions). Patterns in ExcludeTenantPatterns that
// cannot be expressed as DB queries are applied as an in-memory post-filter.
func (r *TenantInMasterRepository) PaginateWithClaimWorkFilter(f models.ClaimWorkFilter, pageSize int, cursor string, now time.Time) (*FindResult[models.TenantInMaster], error) {
	fq := BuildTenantFilterQuery(f)

	result, err := r.repo.Find(fq.DBQuery, pageSize, cursor, now)
	if err != nil {
		return nil, err
	}

	// Post-filter: remove records matching ExcludeTenantPatterns (no NOT LIKE at DB level)
	if len(fq.ExcludePatterns) > 0 {
		filtered := result.Entities[:0]
		for _, t := range result.Entities {
			if !MatchesExcludePatterns(t.Code, fq.ExcludePatterns) {
				filtered = append(filtered, t)
			}
		}
		result.Entities = filtered
	}

	return result, nil
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

func (r *TenantInMasterRepository) DeleteTenantInMasterById(id string, now time.Time) (bool, error) {
	return r.repo.Delete(id, now)
}
