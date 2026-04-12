package db

import (
	"fmt"
	"time"

	models "deadalus-orch/shared/models"
)

type VNamespaceRepository struct {
	*Repository[models.VNamespace]
}

func NewVNamespaceRepository(uow *UnitOfWork, factory IDGeneratorFactory, cf, cfs string) (*VNamespaceRepository, error) {
	if uow == nil {
		return nil, fmt.Errorf("UnitOfWork is required")
	}
	repo, err := GetRepository[models.VNamespace](uow, cf, cfs, "admin_schema", factory)
	if err != nil {
		return nil, err
	}
	return &VNamespaceRepository{Repository: repo}, nil
}

func (r *VNamespaceRepository) CreateVNamespace(input *models.VNamespace, now time.Time) (string, error) {
	input.CreatedAt = now
	input.UpdatedAt = now
	return r.Create(input, now)
}

func (r *VNamespaceRepository) UpdateVNamespace(input *models.VNamespace, now time.Time) (bool, error) {
	input.UpdatedAt = now
	return r.Update(input, now)
}

func (r *VNamespaceRepository) GetVNamespaceByName(name string, now time.Time) (*models.VNamespace, error) {
	return r.FindByField("Name", name, now)
}
func (r *VNamespaceRepository) GetVNamespaceById(id string, now time.Time) (*models.VNamespace, error) {
	return r.FindByField("ID", id, now)
}

func (r *VNamespaceRepository) Paginate(q string, pageSize int, cursor string, now time.Time) (*FindResult[models.VNamespace], error) {
	if q == "" {
		return r.Find("ID != 0", pageSize, cursor, now) // ID != 0 Workaround
	} else {
		return r.Find("Name LIKE *"+q+"*", pageSize, cursor, now) // ID != 0 Workaround
	}
}

// PaginateWithClaimWorkFilter paginates vnamespaces applying the DB-level rules derived from the
// ClaimWorkFilter. Inclusion lists, exact exclusions, and NOT LIKE pattern exclusions are all
// pushed to the DB query.
func (r *VNamespaceRepository) PaginateWithClaimWorkFilter(f models.ClaimWorkFilter, pageSize int, cursor string, now time.Time) (*FindResult[models.VNamespace], error) {
	fq := BuildVNamespaceFilterQuery(f)

	return r.Find(fq.DBQuery, pageSize, cursor, now)
}

func (r *VNamespaceRepository) DeleteVNamespaceById(id string, now time.Time) (bool, error) {
	return r.Delete(id, now)
}
