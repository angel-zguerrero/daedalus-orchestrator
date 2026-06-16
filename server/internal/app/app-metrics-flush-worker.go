package app

import (
	"context"
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	"deadalus-orch/server/internal/infrastructure/metrics"
	"deadalus-orch/server/internal/pkg/config"
	metrics_command "deadalus-orch/server/internal/usecase/command/metrics"
	"deadalus-orch/shared/models"
	"time"

	"github.com/rs/zerolog/log"
)

// StartMetricsFlushWorker starts a background worker that periodically flushes
// completed metric buckets from the in-memory collector to the KV store via
// the Raft FSM. It runs at the configured bucket resolution interval.
func (app *Application) StartMetricsFlushWorker(interval time.Duration) {
	app.MetricsFlushWorkerStopper.RunWorker(func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if !app.MasterNodeIsReady {
					log.Debug().Msg("⏳ MetricsFlush worker waiting for master node to be ready")
					continue
				}

				select {
				case <-app.MetricsFlushWorkerStopper.ShouldStop():
					log.Info().Msg("🛑 MetricsFlush worker received stop signal before execution")
					return
				default:
				}

				app.flushMetrics()

			case <-app.MetricsFlushWorkerStopper.ShouldStop():
				// Final flush before stopping.
				app.flushMetrics()
				log.Info().Msg("ℹ️  MetricsFlush worker stopped gracefully")
				return
			}
		}
	})
}

// flushMetrics harvests completed buckets from the in-memory collector and
// converts them into MetricsBucket models ready for persistence.
func (app *Application) flushMetrics() {
	if app.MetricsCollector == nil {
		return
	}

	flushed := app.MetricsCollector.FlushCompleted()
	if len(flushed) == 0 {
		return
	}

	resolution := config.GlobalConfiguration.MetricsBucketResolution
	if resolution <= 0 {
		resolution = 5
	}

	buckets := flushedBucketsToModels(flushed, resolution)

	log.Debug().
		Int("buckets", len(buckets)).
		Msg("📊 Flushing metric buckets")

	// Group buckets by tenant to write to the correct shard.
	bucketsByTenant := make(map[string][]models.MetricsBucket)
	for _, b := range buckets {
		bucketsByTenant[b.TenantCode] = append(bucketsByTenant[b.TenantCode], b)
	}

	for tenantCode, tenantBuckets := range bucketsByTenant {
		app.writeMetricsBucketsToShard(tenantCode, tenantBuckets)
	}
}

// flushedBucketsToModels converts collector FlushedBuckets to shared model MetricsBuckets.
func flushedBucketsToModels(flushed []metrics.FlushedBucket, resolution int) []models.MetricsBucket {
	result := make([]models.MetricsBucket, 0, len(flushed))
	for _, f := range flushed {
		result = append(result, models.MetricsBucket{
			TenantCode:   f.TenantCode,
			QueueCode:    f.QueueCode,
			VNamespace:   f.VNamespace,
			Timestamp:    f.Timestamp,
			Resolution:   resolution,
			Published:    f.Published,
			Delivered:    f.Delivered,
			Acked:        f.Acked,
			Failed:       f.Failed,
			Pending:      f.Pending,
			InProcess:    f.InProcess,
			LatencySumMs: f.LatencySumMs,
			LatencyCount: f.LatencyCount,
			LatencyMaxMs: f.LatencyMaxMs,
		})
	}
	return result
}

// writeMetricsBucketsToShard finds the correct tenant shard and persists the
// metric buckets via an FSM write command. If no shard is found for the tenant,
// the buckets are silently discarded (this can happen during tenant migration).
func (app *Application) writeMetricsBucketsToShard(tenantCode string, buckets []models.MetricsBucket) {
	tenantNode, ok := app.TenantNodesDictionary[tenantCode]
	if !ok || tenantNode == nil {
		log.Debug().
			Str("tenant", tenantCode).
			Int("buckets", len(buckets)).
			Msg("⚠️  No shard found for tenant metrics, discarding")
		return
	}

	cmd := metrics_command.SaveMetricsBucketsCommand{
		Buckets: buckets,
		// No dynamic CF/CFS needed as it defaults to AdminFC/AdminFCSector
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := dragonboat.ExecuteRepositoryCommand[int](
		tenantNode,
		ctx,
		&cmd,
		10*time.Second,
		log.Logger,
		"SaveMetricsBuckets",
	)

	if err != nil {
		log.Err(err).
			Str("tenant", tenantCode).
			Msg("❌ Failed to persist metric buckets")
	}
}



