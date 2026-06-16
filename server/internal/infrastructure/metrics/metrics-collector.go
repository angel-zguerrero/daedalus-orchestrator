// Package metrics provides an in-memory metrics collector that accumulates
// per-queue counters using atomic operations for lock-free, high-throughput
// metric recording. Counters are grouped into time-bucketed windows whose
// resolution is configurable (default 5 seconds).
//
// The collector is designed to sit in the hot path of message processing
// (Enqueue, Dequeue, Ack, etc.) without introducing contention. A background
// worker periodically calls FlushCompleted to harvest closed buckets and
// persist them to the KV store.
package metrics

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// InMemoryBucket holds atomic counters for a single time window.
// All fields are updated via atomic operations so no mutex is needed on the
// fast path.
type InMemoryBucket struct {
	Published    atomic.Uint64
	Delivered    atomic.Uint64
	Acked        atomic.Uint64
	Failed       atomic.Uint64
	Pending      atomic.Uint64
	InProcess    atomic.Uint64
	LatencySumMs atomic.Uint64
	LatencyCount atomic.Uint64
	LatencyMaxMs atomic.Uint64
}

// updateMax atomically updates target to hold the maximum of its current
// value and candidate using a CAS loop.
func updateMax(target *atomic.Uint64, candidate uint64) {
	for {
		current := target.Load()
		if candidate <= current {
			return
		}
		if target.CompareAndSwap(current, candidate) {
			return
		}
	}
}

// FlushedBucket is a snapshot of an InMemoryBucket at the time it was flushed.
type FlushedBucket struct {
	TenantCode string
	QueueCode  string
	VNamespace string
	Timestamp  int64 // bucket start, Unix seconds

	Published    uint64
	Delivered    uint64
	Acked        uint64
	Failed       uint64
	Pending      uint64
	InProcess    uint64
	LatencySumMs uint64
	LatencyCount uint64
	LatencyMaxMs uint64
}

// bucketKey uniquely identifies a bucket in the collector map.
func bucketKey(tenantCode, queueCode, vnamespace string, timestamp int64) string {
	return fmt.Sprintf("%s:%s:%s:%d", tenantCode, queueCode, vnamespace, timestamp)
}

// MetricsCollector accumulates per-queue metric counters in memory.
// It is safe for concurrent use.
type MetricsCollector struct {
	mu               sync.RWMutex
	resolution       int64 // bucket width in seconds
	buckets          map[string]*bucketEntry
	persistentGauges map[string]*persistentGauge
}

// persistentGauge stores the exact known DB state of a queue
type persistentGauge struct {
	tenantCode string
	queueCode  string
	vnamespace string
	pending    uint64
	inProcess  uint64
}

// bucketEntry associates dimensional metadata with its atomic counters.
type bucketEntry struct {
	tenantCode string
	queueCode  string
	vnamespace string
	timestamp  int64
	data       *InMemoryBucket
}

// NewMetricsCollector creates a new collector with the given resolution (in seconds).
// A resolution of 0 or negative defaults to 5.
func NewMetricsCollector(resolutionSeconds int) *MetricsCollector {
	if resolutionSeconds <= 0 {
		resolutionSeconds = 5
	}
	return &MetricsCollector{
		resolution:       int64(resolutionSeconds),
		buckets:          make(map[string]*bucketEntry),
		persistentGauges: make(map[string]*persistentGauge),
	}
}

// Resolution returns the configured bucket resolution in seconds.
func (mc *MetricsCollector) Resolution() int {
	return int(mc.resolution)
}

// truncateToResolution rounds a Unix-second timestamp down to the nearest
// multiple of the configured resolution.
func (mc *MetricsCollector) truncateToResolution(unixSec int64) int64 {
	return unixSec - (unixSec % mc.resolution)
}

// getOrCreateBucket returns the in-memory bucket for the given dimensions and
// current time, creating one if it doesn't exist.
func (mc *MetricsCollector) getOrCreateBucket(tenantCode, queueCode, vnamespace string) *bucketEntry {
	ts := mc.truncateToResolution(time.Now().Unix())
	key := bucketKey(tenantCode, queueCode, vnamespace, ts)

	// Fast path: read lock
	mc.mu.RLock()
	entry, ok := mc.buckets[key]
	mc.mu.RUnlock()
	if ok {
		return entry
	}

	// Slow path: write lock
	mc.mu.Lock()
	defer mc.mu.Unlock()

	// Double check after acquiring write lock.
	entry, ok = mc.buckets[key]
	if ok {
		return entry
	}

	entry = &bucketEntry{
		tenantCode: tenantCode,
		queueCode:  queueCode,
		vnamespace: vnamespace,
		timestamp:  ts,
		data:       &InMemoryBucket{},
	}
	mc.buckets[key] = entry
	return entry
}

// ── Recording methods ─────────────────────────────────────────────────────

// RecordPublish records N published (enqueued) messages for a queue.
func (mc *MetricsCollector) RecordPublish(tenantCode, queueCode, vnamespace string, count uint64) {
	entry := mc.getOrCreateBucket(tenantCode, queueCode, vnamespace)
	entry.data.Published.Add(count)
}

// RecordDelivery records N delivered (dequeued/claimed) messages for a queue.
func (mc *MetricsCollector) RecordDelivery(tenantCode, queueCode, vnamespace string, count uint64) {
	entry := mc.getOrCreateBucket(tenantCode, queueCode, vnamespace)
	entry.data.Delivered.Add(count)
}

// RecordAck records N acknowledged messages for a queue.
func (mc *MetricsCollector) RecordAck(tenantCode, queueCode, vnamespace string, count uint64) {
	entry := mc.getOrCreateBucket(tenantCode, queueCode, vnamespace)
	entry.data.Acked.Add(count)
}

// RecordFailed records N failed messages for a queue.
func (mc *MetricsCollector) RecordFailed(tenantCode, queueCode, vnamespace string, count uint64) {
	entry := mc.getOrCreateBucket(tenantCode, queueCode, vnamespace)
	entry.data.Failed.Add(count)
}

// RecordLatency records a single latency observation (in milliseconds).
func (mc *MetricsCollector) RecordLatency(tenantCode, queueCode, vnamespace string, latencyMs uint64) {
	entry := mc.getOrCreateBucket(tenantCode, queueCode, vnamespace)
	entry.data.LatencySumMs.Add(latencyMs)
	entry.data.LatencyCount.Add(1)
	updateMax(&entry.data.LatencyMaxMs, latencyMs)
}

// UpdateGauges takes an exact point-in-time snapshot of gauges (Pending and InProcess)
// directly from DB operation results and remembers them.
func (mc *MetricsCollector) UpdateGauges(tenantCode, queueCode, vnamespace string, pending, inProcess uint64) {
	key := fmt.Sprintf("%s:%s:%s", tenantCode, queueCode, vnamespace)

	mc.mu.Lock()
	defer mc.mu.Unlock()

	gauge, ok := mc.persistentGauges[key]
	if !ok {
		gauge = &persistentGauge{
			tenantCode: tenantCode,
			queueCode:  queueCode,
			vnamespace: vnamespace,
		}
		mc.persistentGauges[key] = gauge
	}
	gauge.pending = pending
	gauge.inProcess = inProcess

	// If pending and inProcess are 0, it means the queue has no backlog.
	// But we keep it in persistentGauges until next FlushCompleted so it can flush one last 0-bucket.
}

// SnapshotGauges is kept for backward compatibility if needed, but it's replaced by UpdateGauges.
func (mc *MetricsCollector) SnapshotGauges(tenantCode, queueCode, vnamespace string, pending, inProcess uint64) {
	mc.UpdateGauges(tenantCode, queueCode, vnamespace, pending, inProcess)
}

// ── Flushing ──────────────────────────────────────────────────────────────

// FlushCompleted returns all buckets whose time window has fully elapsed
// (i.e. their timestamp + resolution <= now). Flushed buckets are removed
// from the collector.
func (mc *MetricsCollector) FlushCompleted() []FlushedBucket {
	now := time.Now().Unix()
	cutoff := mc.truncateToResolution(now) // current bucket start — don't flush this one

	mc.mu.Lock()
	defer mc.mu.Unlock()

	var result []FlushedBucket
	flushedKeys := make(map[string]struct{})

	for key, entry := range mc.buckets {
		if entry.timestamp < cutoff {
			fb := FlushedBucket{
				TenantCode:   entry.tenantCode,
				QueueCode:    entry.queueCode,
				VNamespace:   entry.vnamespace,
				Timestamp:    entry.timestamp,
				Published:    entry.data.Published.Load(),
				Delivered:    entry.data.Delivered.Load(),
				Acked:        entry.data.Acked.Load(),
				Failed:       entry.data.Failed.Load(),
			}

			// Add the gauges
			gKey := fmt.Sprintf("%s:%s:%s", entry.tenantCode, entry.queueCode, entry.vnamespace)
			if gauge, ok := mc.persistentGauges[gKey]; ok {
				fb.Pending = gauge.pending
				fb.InProcess = gauge.inProcess
			}

			fb.LatencySumMs = entry.data.LatencySumMs.Load()
			fb.LatencyCount = entry.data.LatencyCount.Load()
			fb.LatencyMaxMs = entry.data.LatencyMaxMs.Load()

			result = append(result, fb)
			flushedKeys[key] = struct{}{}
			delete(mc.buckets, key)
		}
	}

	// Process queues with known backlogs
	for key, gauge := range mc.persistentGauges {
		bKey := bucketKey(gauge.tenantCode, gauge.queueCode, gauge.vnamespace, cutoff-mc.resolution)
		// Only create an artificial bucket if throughput didn't already create one
		if _, exists := flushedKeys[bKey]; !exists {
			result = append(result, FlushedBucket{
				TenantCode: gauge.tenantCode,
				QueueCode:  gauge.queueCode,
				VNamespace: gauge.vnamespace,
				Timestamp:  cutoff - mc.resolution,
				Pending:    gauge.pending,
				InProcess:  gauge.inProcess,
			})
		}

		// If a queue falls completely flat, remove it from memory tracking
		if gauge.pending == 0 && gauge.inProcess == 0 {
			delete(mc.persistentGauges, key)
		}
	}

	return result
}

// ActiveBucketCount returns the number of buckets currently held in memory
// (useful for monitoring/debugging).
func (mc *MetricsCollector) ActiveBucketCount() int {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	return len(mc.buckets)
}
