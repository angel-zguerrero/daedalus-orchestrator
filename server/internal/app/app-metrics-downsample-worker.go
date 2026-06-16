package app

import (
	"context"
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	"deadalus-orch/server/internal/pkg/config"
	metrics_command "deadalus-orch/server/internal/usecase/command/metrics"
	"time"

	"github.com/rs/zerolog/log"
)

// StartMetricsAggregationWorker is a placeholder for background aggregation
// across tenants if we decide to pre-compute dashboard metrics. Currently,
// dashboard metrics are queried from the master node TSDB.
func (app *Application) StartMetricsAggregationWorker(interval time.Duration) {
	app.MetricsAggregationWorkerStopper.RunWorker(func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				// Future: Aggregate queue metrics into global metrics.
			case <-app.MetricsAggregationWorkerStopper.ShouldStop():
				return
			}
		}
	})
}

// StartMetricsDownsampleWorker periodically downsamples old metric buckets
// (e.g. 5s -> 1m -> 1h) and deletes expired data.
func (app *Application) StartMetricsDownsampleWorker(interval time.Duration) {
	app.MetricsDownsampleWorkerStopper.RunWorker(func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if !app.MasterNodeIsReady {
					continue
				}

				app.runMetricsDownsampleCycle()

			case <-app.MetricsDownsampleWorkerStopper.ShouldStop():
				log.Info().Msg("ℹ️  MetricsDownsample worker stopped gracefully")
				return
			}
		}
	})
}

func (app *Application) runMetricsDownsampleCycle() {
	rawResolution := config.GlobalConfiguration.MetricsBucketResolution
	if rawResolution <= 0 {
		rawResolution = 5
	}

	retentionHours := config.GlobalConfiguration.MetricsRetentionRawHours
	if retentionHours <= 0 {
		retentionHours = 1
	}

	cutoff := time.Now().Add(-time.Duration(retentionHours) * time.Hour).Unix()

	// 1. Iterate over all tenant shards and run deletion for old raw buckets.
	// We run this as an FSM command to ensure data is deleted consistently.
	for tenantCode, tenantNode := range app.TenantNodesDictionary {
		if tenantNode == nil {
			continue
		}

		delCmd := metrics_command.DeleteExpiredMetricsCommand{
			Resolution: rawResolution,
			CutoffUnix: cutoff,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		_, err := dragonboat.ExecuteRepositoryCommand[int](
			tenantNode, ctx, &delCmd, 30*time.Second, log.Logger, "DeleteExpiredMetrics",
		)
		cancel()

		if err != nil {
			log.Err(err).Str("tenant", tenantCode).Msg("Failed to delete expired metrics")
		}
	}
}
