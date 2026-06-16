// Package db — MetricsRepository provides TSDB storage over the KVStore interface.
//
// It stores MetricsBuckets with keys designed for efficient range queries:
//
//	metrics:{tenantCode}:{queueCode}:{vnamespace}:{resolution}:{invertedTimestamp}
//
// The inverted timestamp (math.MaxInt64 - ts) ensures that the most recent data
// appears first when scanning lexicographically, which is ideal for dashboard queries.
//
// This repository operates directly on the KVStore interface (not the ORM) so it
// remains portable across Pebble, RocksDB, or any other KV backend.
package db

import (
	"encoding/json"
	"fmt"
	"math"
	"time"

	models "deadalus-orch/shared/models"
)

const (
	// MetricsKeyPrefix is the prefix for all TSDB metric keys.
	MetricsKeyPrefix = "metrics"
	// MetricsFC is the column family used for metrics storage.
	// We reuse the AdminFC so no new column family is needed.
	MetricsFC = AdminFC
	// MetricsFCSector is the column family sector for metrics.
	MetricsFCSector = AdminFCSector
)

// MetricsRepository provides TSDB storage operations via the KVStore interface.
type MetricsRepository struct {
	kvStore            KVStore
	columnFamily       string
	columnFamilySector string
}

// NewMetricsRepository creates a new MetricsRepository using the given KVStore.
func NewMetricsRepository(kvStore KVStore) *MetricsRepository {
	return &MetricsRepository{
		kvStore:            kvStore,
		columnFamily:       MetricsFC,
		columnFamilySector: MetricsFCSector,
	}
}

// NewMetricsRepositoryWithCF creates a MetricsRepository targeting a specific
// column family and sector (e.g. for tenant shards that use dynamic CFs).
func NewMetricsRepositoryWithCF(kvStore KVStore, cf, cfs string) *MetricsRepository {
	return &MetricsRepository{
		kvStore:            kvStore,
		columnFamily:       cf,
		columnFamilySector: cfs,
	}
}

// ── Key construction ──────────────────────────────────────────────────────

// invertTimestamp returns math.MaxInt64 - ts so that recent timestamps
// sort first lexicographically.
func invertTimestamp(ts int64) int64 {
	return math.MaxInt64 - ts
}

// restoreTimestamp is the inverse of invertTimestamp.
func restoreTimestamp(inverted int64) int64 {
	return math.MaxInt64 - inverted
}

// metricsKey builds the full KV key for a metric bucket.
func metricsKey(tenantCode, queueCode, vnamespace string, resolution int, timestamp int64) string {
	invTs := invertTimestamp(timestamp)
	return fmt.Sprintf("%s:%s:%s:%s:%d:%019d",
		MetricsKeyPrefix, tenantCode, queueCode, vnamespace, resolution, invTs)
}

// metricsRangePrefix builds a prefix for range queries matching a specific
// queue at a given resolution.
func metricsRangePrefix(tenantCode, queueCode, vnamespace string, resolution int) string {
	return fmt.Sprintf("%s:%s:%s:%s:%d:",
		MetricsKeyPrefix, tenantCode, queueCode, vnamespace, resolution)
}

// metricsTenantPrefix builds a prefix matching all queues for a tenant at a
// given resolution.
func metricsTenantPrefix(tenantCode string, resolution int) string {
	return fmt.Sprintf("%s:%s:", MetricsKeyPrefix, tenantCode)
}

// metricsGlobalPrefix builds a prefix matching all global metrics at a given resolution.
func metricsGlobalPrefix(resolution int) string {
	return fmt.Sprintf("%s:__global__:__all__:__all__:%d:", MetricsKeyPrefix, resolution)
}

// ── Write operations ──────────────────────────────────────────────────────

// SaveBucket persists a single MetricsBucket to the KV store.
func (r *MetricsRepository) SaveBucket(bucket *models.MetricsBucket, now time.Time) error {
	key := metricsKey(bucket.TenantCode, bucket.QueueCode, bucket.VNamespace, bucket.Resolution, bucket.Timestamp)

	bucket.CreatedAt = now
	data, err := json.Marshal(bucket)
	if err != nil {
		return fmt.Errorf("failed to marshal metrics bucket: %w", err)
	}

	return r.kvStore.Put(r.columnFamily, r.columnFamilySector, key, data, 0, now)
}

// SaveBuckets persists multiple MetricsBuckets atomically using a WriteBatch.
func (r *MetricsRepository) SaveBuckets(buckets []models.MetricsBucket, now time.Time) error {
	if len(buckets) == 0 {
		return nil
	}

	batch := NewWriteBatch()
	for i := range buckets {
		buckets[i].CreatedAt = now
		key := metricsKey(buckets[i].TenantCode, buckets[i].QueueCode, buckets[i].VNamespace, buckets[i].Resolution, buckets[i].Timestamp)

		data, err := json.Marshal(&buckets[i])
		if err != nil {
			return fmt.Errorf("failed to marshal metrics bucket: %w", err)
		}

		batch.Put(r.columnFamily, r.columnFamilySector, key, data, now)
	}

	return r.kvStore.Write(batch)
}

// ── Read operations ───────────────────────────────────────────────────────

// QueryRange reads metric buckets for a specific queue within a time range.
// Results are returned sorted by timestamp descending (most recent first).
func (r *MetricsRepository) QueryRange(
	tenantCode, queueCode, vnamespace string,
	resolution int,
	from, to int64,
	limit int,
	now time.Time,
) ([]models.MetricsBucket, error) {
	prefix := metricsRangePrefix(tenantCode, queueCode, vnamespace, resolution)
	return r.queryByPrefix(prefix, from, to, limit, now)
}

// QueryRangeByTenant reads all metric buckets across all queues for a tenant
// within a time range. The caller must aggregate the results by timestamp
// if a tenant-level summary is needed.
func (r *MetricsRepository) QueryRangeByTenant(
	tenantCode string,
	resolution int,
	from, to int64,
	limit int,
	now time.Time,
) ([]models.MetricsBucket, error) {
	// Use a broader prefix that matches all queues for this tenant.
	// We filter by resolution in the scan.
	prefix := metricsTenantPrefix(tenantCode, resolution)
	return r.queryByPrefixFilterResolution(prefix, resolution, from, to, limit, now)
}

// QueryGlobal reads all aggregated metric buckets across all tenants on this node.
func (r *MetricsRepository) QueryGlobal(
	resolution int,
	from, to int64,
	limit int,
	now time.Time,
) ([]models.MetricsBucket, error) {
	prefix := MetricsKeyPrefix + ":"
	return r.queryByPrefixFilterResolution(prefix, resolution, from, to, limit, now)
}

// queryByPrefix is the internal method that scans the KV store using a prefix
// and applies a time range filter.
func (r *MetricsRepository) queryByPrefix(
	prefix string,
	from, to int64,
	limit int,
	now time.Time,
) ([]models.MetricsBucket, error) {
	if limit <= 0 {
		limit = 500
	}

	pattern := prefix + "*"
	cursor := ""
	var results []models.MetricsBucket

	for {
		items, nextCursor, err := r.kvStore.SearchByPatternPaginatedKV(
			r.columnFamily, r.columnFamilySector,
			pattern, cursor, limit, now,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to query metrics: %w", err)
		}

		for _, item := range items {
			var bucket models.MetricsBucket
			if err := json.Unmarshal(item.Value, &bucket); err != nil {
				continue // skip corrupted entries
			}

			// Apply time range filter.
			if bucket.Timestamp >= from && bucket.Timestamp <= to {
				results = append(results, bucket)
			}
		}

		if nextCursor == "" || len(items) == 0 {
			break
		}
		cursor = nextCursor

		// Safety limit to prevent infinite loops.
		if len(results) >= limit {
			break
		}
	}

	return results, nil
}

// queryByPrefixFilterResolution scans a broader prefix and filters by resolution.
func (r *MetricsRepository) queryByPrefixFilterResolution(
	prefix string,
	resolution int,
	from, to int64,
	limit int,
	now time.Time,
) ([]models.MetricsBucket, error) {
	if limit <= 0 {
		limit = 500
	}

	pattern := prefix + "*"
	cursor := ""
	var results []models.MetricsBucket

	for {
		items, nextCursor, err := r.kvStore.SearchByPatternPaginatedKV(
			r.columnFamily, r.columnFamilySector,
			pattern, cursor, limit, now,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to query metrics: %w", err)
		}

		for _, item := range items {
			var bucket models.MetricsBucket
			if err := json.Unmarshal(item.Value, &bucket); err != nil {
				continue
			}

			if bucket.Resolution == resolution && bucket.Timestamp >= from && bucket.Timestamp <= to {
				results = append(results, bucket)
			}
		}

		if nextCursor == "" || len(items) == 0 {
			break
		}
		cursor = nextCursor

		if len(results) >= limit {
			break
		}
	}

	return results, nil
}

// ── Delete operations ─────────────────────────────────────────────────────

// DeleteOlderThan removes all metric buckets at the given resolution whose
// timestamp is older than cutoff. This is used for data retention.
func (r *MetricsRepository) DeleteOlderThan(resolution int, cutoff int64, now time.Time) (int, error) {
	// Scan all metrics with this resolution and delete those older than cutoff.
	pattern := fmt.Sprintf("%s:*:%d:*", MetricsKeyPrefix, resolution)
	cursor := ""
	deleted := 0

	for {
		items, nextCursor, err := r.kvStore.SearchByPatternPaginatedKV(
			r.columnFamily, r.columnFamilySector,
			pattern, cursor, 100, now,
		)
		if err != nil {
			return deleted, fmt.Errorf("failed to scan metrics for deletion: %w", err)
		}

		batch := NewWriteBatch()
		for _, item := range items {
			var bucket models.MetricsBucket
			if err := json.Unmarshal(item.Value, &bucket); err != nil {
				continue
			}

			if bucket.Timestamp < cutoff {
				batch.Delete(r.columnFamily, r.columnFamilySector, item.Key, now)
				deleted++
			}
		}

		if batch.Count() > 0 {
			if err := r.kvStore.Write(batch); err != nil {
				return deleted, fmt.Errorf("failed to delete expired metrics: %w", err)
			}
		}

		if nextCursor == "" || len(items) == 0 {
			break
		}
		cursor = nextCursor
	}

	return deleted, nil
}

// ── Downsampling ──────────────────────────────────────────────────────────

// DownsampleBuckets takes fine-grained buckets and produces coarser aggregated
// buckets. For example, 12 five-second buckets → 1 one-minute bucket.
//
// The function groups input buckets by (tenant, queue, vnamespace) and by
// the new target timestamp (truncated to targetResolution), then merges them.
func DownsampleBuckets(buckets []models.MetricsBucket, targetResolution int) []models.MetricsBucket {
	type groupKey struct {
		tenant     string
		queue      string
		vnamespace string
		timestamp  int64
	}

	groups := make(map[groupKey]*models.MetricsBucket)

	for _, b := range buckets {
		targetTs := b.Timestamp - (b.Timestamp % int64(targetResolution))
		key := groupKey{b.TenantCode, b.QueueCode, b.VNamespace, targetTs}

		agg, exists := groups[key]
		if !exists {
			agg = &models.MetricsBucket{
				TenantCode: b.TenantCode,
				QueueCode:  b.QueueCode,
				VNamespace: b.VNamespace,
				Timestamp:  targetTs,
				Resolution: targetResolution,
			}
			groups[key] = agg
		}

		// Counters are summed.
		agg.Published += b.Published
		agg.Delivered += b.Delivered
		agg.Acked += b.Acked
		agg.Failed += b.Failed

		// Gauges: use the latest value (highest timestamp).
		if b.Timestamp >= agg.Timestamp {
			agg.Pending = b.Pending
			agg.InProcess = b.InProcess
		}

		// Latency: sum totals, keep max.
		agg.LatencySumMs += b.LatencySumMs
		agg.LatencyCount += b.LatencyCount
		if b.LatencyMaxMs > agg.LatencyMaxMs {
			agg.LatencyMaxMs = b.LatencyMaxMs
		}
	}

	result := make([]models.MetricsBucket, 0, len(groups))
	for _, agg := range groups {
		result = append(result, *agg)
	}
	return result
}

// ── Helpers for API responses ─────────────────────────────────────────────

// BucketsToDatapoints converts MetricsBuckets into API-facing MetricsDatapoints,
// computing derived metrics like msg/second and average latency.
func BucketsToDatapoints(buckets []models.MetricsBucket) []models.MetricsDatapoint {
	points := make([]models.MetricsDatapoint, 0, len(buckets))

	for _, b := range buckets {
		dp := models.MetricsDatapoint{
			Timestamp:    b.Timestamp,
			Published:    b.Published,
			Delivered:    b.Delivered,
			Acked:        b.Acked,
			Failed:       b.Failed,
			Pending:      b.Pending,
			InProcess:    b.InProcess,
			MaxLatencyMs: b.LatencyMaxMs,
		}

		// Compute messages per second.
		if b.Resolution > 0 {
			dp.MsgPerSecond = float64(b.Published) / float64(b.Resolution)
		}

		// Compute average latency.
		if b.LatencyCount > 0 {
			dp.AvgLatencyMs = float64(b.LatencySumMs) / float64(b.LatencyCount)
		}

		points = append(points, dp)
	}

	return points
}
