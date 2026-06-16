package metrics_command

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"deadalus-orch/shared/models"
	"encoding/gob"
	"encoding/json"
	"time"
)

func init() {
	gob.Register(SaveMetricsBucketsCommand{})
	gob.Register(QueryMetricsRangeCommand{})
	gob.Register(DeleteExpiredMetricsCommand{})
	gob.Register(DownsampleMetricsCommand{})
}

// ── Write Commands ────────────────────────────────────────────────────────

// SaveMetricsBucketsCommand persists a batch of MetricsBuckets to the KV store.
// It is executed as a Raft FSM write command so the data is replicated.
type SaveMetricsBucketsCommand struct {
	Buckets []models.MetricsBucket
	CF      string // Column family (for tenant shard CFs)
	CFS     string // Column family sector
	IsRelay bool   // If true, this is executed by the relay worker on the Master Node and should not create an OutboxEvent
}

func (cmd *SaveMetricsBucketsCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	result := &command.CommandResult{}

	var repo *db.MetricsRepository
	if cmd.CF != "" && cmd.CFS != "" {
		repo = db.NewMetricsRepositoryWithCF(uow.KVStore, cmd.CF, cmd.CFS)
	} else {
		repo = db.NewMetricsRepository(uow.KVStore)
	}

	err := repo.SaveBuckets(cmd.Buckets, now)
	if err != nil {
		result.Error = err.Error()
		return *result
	}

	// Create an OutboxEvent for relaying these metrics to the Master Node.
	// We serialize the buckets into the Payload field.
	// Only do this if it's not a relay command, to avoid infinite loops on the Master Node!
	if !cmd.IsRelay {
		payload, err := json.Marshal(cmd.Buckets)
		if err == nil {
			idFactory := &db.DeterministicIDGeneratorFactory{}
			outboxRepo, err := db.NewOutboxEventRepository(uow, idFactory)
			if err == nil {
				tenantCode := "system"
				if len(cmd.Buckets) > 0 {
					tenantCode = cmd.Buckets[0].TenantCode
				}
				outboxEvent := models.OutboxEvent{
					ID:        idFactory.GenerateID(),
					EventType: models.EventTypeMetricsRelay,
					TenantID:  tenantCode, // Useful for tracing, though payload contains all
					Payload:   payload,
					CreatedAt: now,
				}
				outboxRepo.CreateEvent(&outboxEvent, now)
			}
		}
	}

	result.Result = len(cmd.Buckets)
	return *result
}

// DeleteExpiredMetricsCommand removes metric buckets older than the specified
// cutoff time at a given resolution. Used for data retention.
type DeleteExpiredMetricsCommand struct {
	Resolution int
	CutoffUnix int64
	CF         string
	CFS        string
}

func (cmd *DeleteExpiredMetricsCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	result := &command.CommandResult{}

	var repo *db.MetricsRepository
	if cmd.CF != "" && cmd.CFS != "" {
		repo = db.NewMetricsRepositoryWithCF(uow.KVStore, cmd.CF, cmd.CFS)
	} else {
		repo = db.NewMetricsRepository(uow.KVStore)
	}

	deleted, err := repo.DeleteOlderThan(cmd.Resolution, cmd.CutoffUnix, now)
	if err != nil {
		result.Error = err.Error()
		return *result
	}

	result.Result = deleted
	return *result
}

// DownsampleMetricsCommand reads fine-grained buckets for a time range,
// downsamples them to a coarser resolution, and persists the results.
type DownsampleMetricsCommand struct {
	SourceResolution int
	TargetResolution int
	FromUnix         int64
	ToUnix           int64
	TenantCode       string
	CF               string
	CFS              string
}

func (cmd *DownsampleMetricsCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	result := &command.CommandResult{}

	var repo *db.MetricsRepository
	if cmd.CF != "" && cmd.CFS != "" {
		repo = db.NewMetricsRepositoryWithCF(uow.KVStore, cmd.CF, cmd.CFS)
	} else {
		repo = db.NewMetricsRepository(uow.KVStore)
	}

	// Read source-resolution buckets for the given tenant.
	sourceBuckets, err := repo.QueryRangeByTenant(
		cmd.TenantCode, cmd.SourceResolution,
		cmd.FromUnix, cmd.ToUnix, 10000, now,
	)
	if err != nil {
		result.Error = err.Error()
		return *result
	}

	if len(sourceBuckets) == 0 {
		result.Result = 0
		return *result
	}

	// Downsample.
	downsampled := db.DownsampleBuckets(sourceBuckets, cmd.TargetResolution)

	// Persist downsampled buckets.
	err = repo.SaveBuckets(downsampled, now)
	if err != nil {
		result.Error = err.Error()
		return *result
	}

	result.Result = len(downsampled)
	return *result
}

// ── Read Commands ─────────────────────────────────────────────────────────

// QueryMetricsRangeCommand reads metric buckets for a queue/tenant/global
// within a time range.
type QueryMetricsRangeCommand struct {
	TenantCode string
	QueueCode  string
	VNamespace string
	Resolution int
	FromUnix   int64
	ToUnix     int64
	Limit      int
	Scope      string // "queue", "tenant", or "global"
	CF         string
	CFS        string
}

func (cmd *QueryMetricsRangeCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	result := &command.CommandResult{}

	var repo *db.MetricsRepository
	if cmd.CF != "" && cmd.CFS != "" {
		repo = db.NewMetricsRepositoryWithCF(uow.KVStore, cmd.CF, cmd.CFS)
	} else {
		repo = db.NewMetricsRepository(uow.KVStore)
	}

	var buckets []models.MetricsBucket
	var err error

	switch cmd.Scope {
	case "queue":
		buckets, err = repo.QueryRange(
			cmd.TenantCode, cmd.QueueCode, cmd.VNamespace,
			cmd.Resolution, cmd.FromUnix, cmd.ToUnix, cmd.Limit, now,
		)
	case "tenant":
		buckets, err = repo.QueryRangeByTenant(
			cmd.TenantCode, cmd.Resolution,
			cmd.FromUnix, cmd.ToUnix, cmd.Limit, now,
		)
	case "global":
		buckets, err = repo.QueryGlobal(
			cmd.Resolution, cmd.FromUnix, cmd.ToUnix, cmd.Limit, now,
		)
	default:
		result.Error = "invalid scope: must be 'queue', 'tenant', or 'global'"
		return *result
	}

	if err != nil {
		result.Error = err.Error()
		return *result
	}

	// Convert to API datapoints.
	datapoints := db.BucketsToDatapoints(buckets)

	queryResult := models.MetricsQueryResult{
		Resolution: cmd.Resolution,
		Datapoints: datapoints,
	}

	result.Result = queryResult
	return *result
}
