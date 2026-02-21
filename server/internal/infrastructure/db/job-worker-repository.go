package db

import (
	"fmt"
	"strings"
	"time"

	models "deadalus-orch/shared/models"
)

type JobWorkerRepository struct {
	*Repository[models.JobWorker]
}

func NewJobWorkerRepository(uow *UnitOfWork, factory IDGeneratorFactory) (*JobWorkerRepository, error) {
	if uow == nil {
		return nil, fmt.Errorf("UnitOfWork is required")
	}
	repo, err := GetRepository[models.JobWorker](uow, MasterEventFC, MasterEventFCSector, "admin_event_schema", factory)
	if err != nil {
		return nil, err
	}
	return &JobWorkerRepository{Repository: repo}, nil
}

func (r *JobWorkerRepository) CreateJobWorker(input *models.JobWorker, now time.Time) (string, error) {
	input.CreatedAt = now
	input.UpdatedAt = now
	return r.Create(input, now)
}

func (r *JobWorkerRepository) UpdateJobWorker(input *models.JobWorker, now time.Time) (bool, error) {
	input.UpdatedAt = now
	return r.Update(input, now)
}

func (r *JobWorkerRepository) GetJobWorkerByName(name string, now time.Time) (*models.JobWorker, error) {
	return r.FindByField("Name", name, now)
}

func (r *JobWorkerRepository) GetJobWorkerById(id string, now time.Time) (*models.JobWorker, error) {
	return r.FindByField("ID", id, now)
}

func (r *JobWorkerRepository) Paginate(q string, status models.JobWorkerConnectionStatus, pageSize int, cursor string, now time.Time) (*FindResult[models.JobWorker], error) {
	var conditions []string

	// Add name search condition if q is provided
	if q != "" {
		conditions = append(conditions, "Name LIKE *"+q+"*")
	}

	if status != "" {
		conditions = append(conditions, "ConnectionStatus = "+string(status))
	}

	// If no conditions but we got here, use the workaround
	if len(conditions) == 0 {
		return r.Find("ID != 0", pageSize, cursor, now) // ID != 0 Workaround
	} else {
		return r.Find(strings.Join(conditions, " & "), pageSize, cursor, now)
	}
}
