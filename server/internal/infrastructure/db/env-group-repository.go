package db

import (
	"fmt"
	"time"

	models "deadalus-orch/shared/models"
)

type EnvGroupRepository struct {
	*Repository[models.EnvGroup]
}

func NewEnvGroupRepository(uow *UnitOfWork, factory IDGeneratorFactory, cf, cfs string) (*EnvGroupRepository, error) {
	if uow == nil {
		return nil, fmt.Errorf("UnitOfWork is required")
	}
	repo, err := GetRepository[models.EnvGroup](uow, cf, cfs, "admin_schema", factory)
	if err != nil {
		return nil, err
	}
	return &EnvGroupRepository{Repository: repo}, nil
}

func (r *EnvGroupRepository) CreateEnvGroup(input *models.EnvGroup, now time.Time) (string, error) {
	if input.Code == "" {
		return "", fmt.Errorf("Code is required")
	}
	if input.Type != models.EnvGroupTypeConfig && input.Type != models.EnvGroupTypeSecret {
		return "", fmt.Errorf("invalid Type: %s. Must be 'config' or 'secret'", input.Type)
	}
	input.CreatedAt = now
	input.UpdatedAt = now
	return r.Create(input, now)
}

func (r *EnvGroupRepository) UpdateEnvGroup(input *models.EnvGroup, now time.Time) (bool, error) {
	input.UpdatedAt = now
	return r.Update(input, now)
}

func (r *EnvGroupRepository) GetEnvGroupByID(id string, now time.Time) (*models.EnvGroup, error) {
	return r.FindByField("ID", id, now)
}

func (r *EnvGroupRepository) GetEnvGroupByCode(code string, vnamespace string, now time.Time) (*models.EnvGroup, error) {
	if vnamespace == "" {
		vnamespace = "default"
	}
	query := fmt.Sprintf("Code = %s & VNamespace = %s", code, vnamespace)
	res, err := r.Find(query, 1, "", now)
	if err != nil || len(res.Entities) == 0 {
		return nil, err
	}
	return &res.Entities[0], nil
}

func (r *EnvGroupRepository) ListEnvGroups(scope string, tenantID string, pageSize int, cursor string, now time.Time) (*FindResult[models.EnvGroup], error) {
	var query string
	if scope != "" {
		query = fmt.Sprintf("Scope = %s", scope)
		if scope == string(models.EnvGroupScopeTenant) && tenantID != "" {
			query += fmt.Sprintf(" & TenantID = %s", tenantID)
		}
	} else if tenantID != "" {
		query = fmt.Sprintf("TenantID = %s", tenantID)
	}

	if query == "" {
		query = "ID != 0"
	}

	if pageSize <= 0 {
		pageSize = 50
	}

	return r.Find(query, pageSize, cursor, now)
}

func (r *EnvGroupRepository) DeleteEnvGroup(id string, now time.Time) (bool, error) {
	return r.Delete(id, now)
}
