package db

import (
	"fmt"
	"time"

	models "deadalus-orch/shared/models"
)

type ScheduledJobTrackerRepository struct {
	*Repository[models.ScheduledJobTracker]
}

func NewScheduledJobTrackerRepository(uow *UnitOfWork, factory IDGeneratorFactory, cf, cfs string) (*ScheduledJobTrackerRepository, error) {
	if uow == nil {
		return nil, fmt.Errorf("UnitOfWork is required")
	}
	repo, err := GetRepository[models.ScheduledJobTracker](uow, cf, cfs, "admin_schema", factory)
	if err != nil {
		return nil, err
	}
	return &ScheduledJobTrackerRepository{Repository: repo}, nil
}

func (r *ScheduledJobTrackerRepository) CreateTracker(tracker *models.ScheduledJobTracker, now time.Time) (string, error) {
	if tracker.ID == "" {
		generatedID := r.idGeneratorFactory.GenerateID()
		if generatedID != "" {
			tracker.ID = generatedID
		} else {
			defaultFactory := &DefaultIDGeneratorFactory{}
			tracker.ID = defaultFactory.GenerateID()
		}
	}
	tracker.CreatedAt = now
	tracker.UpdatedAt = now
	return r.Create(tracker, now)
}

func (r *ScheduledJobTrackerRepository) BulkCreateTrackers(trackers []*models.ScheduledJobTracker, now time.Time) ([]string, error) {
	for _, tracker := range trackers {
		if tracker.ID == "" {
			generatedID := r.idGeneratorFactory.GenerateID()
			if generatedID != "" {
				tracker.ID = generatedID
			} else {
				defaultFactory := &DefaultIDGeneratorFactory{}
				tracker.ID = defaultFactory.GenerateID()
			}
		}
		tracker.CreatedAt = now
		tracker.UpdatedAt = now
	}
	return r.BulkCreate(trackers, now)
}

func (r *ScheduledJobTrackerRepository) GetTracker(scheduledJobID, executionID, queueID string, now time.Time) (*models.ScheduledJobTracker, error) {
	filter := fmt.Sprintf("ScheduledJobID = '%s' & ExecutionID = '%s' & QueueID = '%s'", scheduledJobID, executionID, queueID)
	res, err := r.Find(filter, 1, "", now)
	if err != nil {
		return nil, err
	}
	if len(res.Entities) == 0 {
		return nil, nil
	}
	return &res.Entities[0], nil
}

func (r *ScheduledJobTrackerRepository) GetTrackersByExecution(executionID string, now time.Time) ([]models.ScheduledJobTracker, error) {
	if executionID == "" {
		return []models.ScheduledJobTracker{}, nil
	}
	filter := fmt.Sprintf("ExecutionID = '%s'", executionID)
	res, err := r.Find(filter, 1000, "", now)
	if err != nil {
		return nil, err
	}
	return res.Entities, nil
}

func (r *ScheduledJobTrackerRepository) DeleteTrackersForExecution(executionID string, now time.Time) error {
	trackers, err := r.GetTrackersByExecution(executionID, now)
	if err != nil {
		return err
	}
	if len(trackers) == 0 {
		return nil
	}

	ids := make([]string, len(trackers))
	for i, t := range trackers {
		ids[i] = t.ID
	}

	_, err = r.BulkDelete(ids, now)
	return err
}
