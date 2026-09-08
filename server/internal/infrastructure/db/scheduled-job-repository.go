package db

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"deadalus-orch/server/internal/pkg/utils"
	models "deadalus-orch/shared/models"
)

const ScheduledJobIndexPrefix = "scheduled-jobs:"

type ScheduledJobRepository struct {
	*Repository[models.ScheduledJob]
}

func NewScheduledJobRepository(uow *UnitOfWork, factory IDGeneratorFactory, cf, cfs string) (*ScheduledJobRepository, error) {
	if uow == nil {
		return nil, fmt.Errorf("UnitOfWork is required")
	}
	repo, err := GetRepository[models.ScheduledJob](uow, cf, cfs, "admin_schema", factory)
	if err != nil {
		return nil, err
	}
	return &ScheduledJobRepository{Repository: repo}, nil
}

func FormatScheduledJobIndexKey(nextRunAt time.Time, jobID string) string {
	return fmt.Sprintf("%s%010d:%s", ScheduledJobIndexPrefix, nextRunAt.Unix(), jobID)
}

func ParseScheduledJobIndexKey(key string) (int64, string, error) {
	if !strings.HasPrefix(key, ScheduledJobIndexPrefix) {
		return 0, "", fmt.Errorf("key does not have scheduled job index prefix")
	}
	rest := strings.TrimPrefix(key, ScheduledJobIndexPrefix)
	parts := strings.SplitN(rest, ":", 2)
	if len(parts) != 2 {
		return 0, "", fmt.Errorf("invalid scheduled job index key format: %s", key)
	}
	ts, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, "", fmt.Errorf("invalid timestamp in scheduled job index key: %w", err)
	}
	return ts, parts[1], nil
}

func (r *ScheduledJobRepository) CreateScheduledJob(job *models.ScheduledJob, now time.Time) (string, error) {
	job.CreatedAt = now
	job.UpdatedAt = now

	id, err := r.Create(job, now)
	if err != nil {
		return "", err
	}

	if job.State == models.ScheduledJobIdle {
		indexKey := FormatScheduledJobIndexKey(job.NextRunAt, id)
		err = r.kvStore.Put(r.definition.ColumnFamily, r.definition.ColumnFamilySector, indexKey, []byte{}, 0, now)
		if err != nil {
			return "", fmt.Errorf("failed to write companion index key: %w", err)
		}
	}

	return id, nil
}

func (r *ScheduledJobRepository) UpdateScheduledJobStateAndRunAt(
	job *models.ScheduledJob,
	oldNextRunAt *time.Time,
	now time.Time,
) (bool, error) {
	job.UpdatedAt = now

	if oldNextRunAt != nil {
		oldIndexKey := FormatScheduledJobIndexKey(*oldNextRunAt, job.ID)
		_ = r.kvStore.Delete(r.definition.ColumnFamily, r.definition.ColumnFamilySector, oldIndexKey, now)
	}

	currentIndexKey := FormatScheduledJobIndexKey(job.NextRunAt, job.ID)
	if job.State == models.ScheduledJobDelivered {
		_ = r.kvStore.Delete(r.definition.ColumnFamily, r.definition.ColumnFamilySector, currentIndexKey, now)
	} else if job.State == models.ScheduledJobIdle {
		err := r.kvStore.Put(r.definition.ColumnFamily, r.definition.ColumnFamilySector, currentIndexKey, []byte{}, 0, now)
		if err != nil {
			return false, fmt.Errorf("failed to write companion index key on update: %w", err)
		}
	}

	return r.Update(job, now)
}

func (r *ScheduledJobRepository) DeleteScheduledJob(job *models.ScheduledJob, now time.Time) (bool, error) {
	if job == nil {
		return false, nil
	}

	indexKey := FormatScheduledJobIndexKey(job.NextRunAt, job.ID)
	_ = r.kvStore.Delete(r.definition.ColumnFamily, r.definition.ColumnFamilySector, indexKey, now)

	return r.Delete(job.ID, now)
}

func (r *ScheduledJobRepository) GetScheduledJobByID(id string, now time.Time) (*models.ScheduledJob, error) {
	return r.FindByField("ID", id, now)
}

func (r *ScheduledJobRepository) HandleCompletion(scheduledJobID string, now time.Time) error {
	if scheduledJobID == "" {
		return nil
	}

	job, err := r.GetScheduledJobByID(scheduledJobID, now)
	if err != nil || job == nil {
		return err
	}

	if job.Type == models.ScheduledJobOneOff {
		_, err := r.DeleteScheduledJob(job, now)
		return err
	}

	if job.Type == models.ScheduledJobRecurring {
		oldNextRunAt := job.NextRunAt
		nextRunAt, err := utils.CalculateNextRunAt(
			job.Type,
			job.RunAt,
			job.RunAfter,
			job.Every,
			job.CronExpression,
			now,
		)
		if err != nil {
			return err
		}

		job.NextRunAt = nextRunAt
		job.State = models.ScheduledJobIdle

		_, err = r.UpdateScheduledJobStateAndRunAt(job, &oldNextRunAt, now)
		return err
	}

	return nil
}

func (r *ScheduledJobRepository) PaginateScheduledJobs(
	pageSize int,
	cursor string,
	vNamespace string,
	now time.Time,
) (*FindResult[models.ScheduledJob], error) {
	var query string
	if vNamespace != "" {
		query = "VNamespace = " + vNamespace
	} else {
		query = "ID != 0"
	}
	return r.Find(query, pageSize, cursor, now)
}

func (r *ScheduledJobRepository) FindDueScheduledJobs(
	now time.Time,
	limit int,
	cursor string,
) ([]*models.ScheduledJob, string, error) {
	if limit <= 0 {
		limit = 1000
	}

	nowUnix := now.Unix()
	pattern := ScheduledJobIndexPrefix + "*"

	kvPairs, nextCursor, err := r.kvStore.SearchByPatternPaginatedKV(
		r.definition.ColumnFamily,
		r.definition.ColumnFamilySector,
		pattern,
		cursor,
		limit,
		now,
	)
	if err != nil {
		return nil, "", fmt.Errorf("failed to scan scheduled job index keys: %w", err)
	}

	var dueJobs []*models.ScheduledJob
	for _, kv := range kvPairs {
		ts, jobID, parseErr := ParseScheduledJobIndexKey(kv.Key)
		if parseErr != nil {
			continue
		}

		if ts > nowUnix {
			break
		}

		job, err := r.GetScheduledJobByID(jobID, now)
		if err != nil {
			continue
		}
		if job != nil && job.State == models.ScheduledJobIdle {
			dueJobs = append(dueJobs, job)
		}
	}

	return dueJobs, nextCursor, nil
}
