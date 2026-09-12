package db

import (
	"fmt"
	"time"

	models "deadalus-orch/shared/models"
)

type EnvVarRepository struct {
	*Repository[models.EnvVar]
}

func NewEnvVarRepository(uow *UnitOfWork, factory IDGeneratorFactory, cf, cfs string) (*EnvVarRepository, error) {
	if uow == nil {
		return nil, fmt.Errorf("UnitOfWork is required")
	}
	repo, err := GetRepository[models.EnvVar](uow, cf, cfs, "admin_schema", factory)
	if err != nil {
		return nil, err
	}
	return &EnvVarRepository{Repository: repo}, nil
}

func (r *EnvVarRepository) CreateEnvVar(input *models.EnvVar, now time.Time) (string, error) {
	if input.GroupID == "" {
		return "", fmt.Errorf("GroupID is required")
	}
	if input.Key == "" {
		return "", fmt.Errorf("Key is required")
	}
	input.GroupIDComp = input.GroupID
	input.CreatedAt = now
	input.UpdatedAt = now
	return r.Create(input, now)
}

func (r *EnvVarRepository) UpdateEnvVar(input *models.EnvVar, now time.Time) (bool, error) {
	input.GroupIDComp = input.GroupID
	input.UpdatedAt = now
	return r.Update(input, now)
}

func (r *EnvVarRepository) GetEnvVarByID(id string, now time.Time) (*models.EnvVar, error) {
	return r.FindByField("ID", id, now)
}

func (r *EnvVarRepository) GetEnvVarsByGroupID(groupID string, now time.Time) ([]models.EnvVar, error) {
	if groupID == "" {
		return []models.EnvVar{}, nil
	}
	return r.FindByGroup("GroupID", groupID, now)
}

func (r *EnvVarRepository) GetEnvVarByKey(groupID string, key string, now time.Time) (*models.EnvVar, error) {
	vars, err := r.GetEnvVarsByGroupID(groupID, now)
	if err != nil {
		return nil, err
	}
	for _, v := range vars {
		if v.Key == key {
			return &v, nil
		}
	}
	return nil, nil
}

func (r *EnvVarRepository) DeleteEnvVar(id string, now time.Time) (bool, error) {
	return r.Delete(id, now)
}

func (r *EnvVarRepository) DeleteEnvVarsByGroupID(groupID string, now time.Time) error {
	vars, err := r.GetEnvVarsByGroupID(groupID, now)
	if err != nil {
		return err
	}
	for _, v := range vars {
		_, _ = r.Delete(v.ID, now)
	}
	return nil
}
