package db

import (
	"fmt"
	"sort"
	"time"

	models "deadalus-orch/shared/models"
)

const MaxWorkflowVersionsLimit = 1000

type WorkflowDefinitionVersionRepository struct {
	*Repository[models.WorkflowDefinitionVersion]
}

func NewWorkflowDefinitionVersionRepository(uow *UnitOfWork, factory IDGeneratorFactory, cf, cfs string) (*WorkflowDefinitionVersionRepository, error) {
	if uow == nil {
		return nil, fmt.Errorf("UnitOfWork is required")
	}
	repo, err := GetRepository[models.WorkflowDefinitionVersion](uow, cf, cfs, "admin_schema", factory)
	if err != nil {
		return nil, err
	}
	return &WorkflowDefinitionVersionRepository{Repository: repo}, nil
}

func (r *WorkflowDefinitionVersionRepository) CreateVersion(input *models.WorkflowDefinitionVersion, now time.Time) (string, error) {
	if input.WorkflowDefinitionID == "" {
		return "", fmt.Errorf("WorkflowDefinitionID is required")
	}
	if input.Version <= 0 {
		return "", fmt.Errorf("Version must be greater than 0")
	}
	if input.ID == "" {
		input.ID = fmt.Sprintf("%s-v%d", input.WorkflowDefinitionID, input.Version)
	}
	if input.CreatedAt.IsZero() {
		input.CreatedAt = now
	}
	id, err := r.Create(input, now)
	if err != nil {
		return "", err
	}

	// Enforce limit of 1000 version records per workflow
	r.enforceMaxVersionsLimit(input.WorkflowDefinitionID, now)

	return id, nil
}

func (r *WorkflowDefinitionVersionRepository) enforceMaxVersionsLimit(workflowDefID string, now time.Time) {
	query := fmt.Sprintf("WorkflowDefinitionID = %s", workflowDefID)
	res, err := r.Find(query, 2000, "", now)
	if err != nil || res == nil || len(res.Entities) <= MaxWorkflowVersionsLimit {
		return
	}

	versions := res.Entities
	sort.Slice(versions, func(i, j int) bool {
		return versions[i].Version < versions[j].Version
	})

	toDeleteCount := len(versions) - MaxWorkflowVersionsLimit
	for i := 0; i < toDeleteCount; i++ {
		_, _ = r.Delete(versions[i].ID, now)
	}
}

func (r *WorkflowDefinitionVersionRepository) GetVersionByNumber(workflowDefID string, version int32, now time.Time) (*models.WorkflowDefinitionVersion, error) {
	query := fmt.Sprintf("WorkflowDefinitionID = %s & Version = %d", workflowDefID, version)
	res, err := r.Find(query, 1, "", now)
	if err != nil || res == nil || len(res.Entities) == 0 {
		return nil, err
	}
	return &res.Entities[0], nil
}

func (r *WorkflowDefinitionVersionRepository) ListVersions(workflowDefID string, pageSize int, cursor string, now time.Time) (*FindResult[models.WorkflowDefinitionVersion], error) {
	if pageSize <= 0 {
		pageSize = 50
	}
	query := fmt.Sprintf("WorkflowDefinitionID = %s", workflowDefID)
	return r.Find(query, pageSize, cursor, now)
}
