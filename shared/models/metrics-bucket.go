package models

import (
	"encoding/gob"
	"time"
)

func init() {
	gob.Register(MetricsBucket{})
	gob.Register([]MetricsBucket{})
	gob.Register(MetricsQueryResult{})
}

// MetricsBucket represents a single time-series data point aggregated over a
// configurable time window (resolution). It stores counters and gauges for a
// specific queue within a tenant.
//
// Buckets are persisted in the KV store with keys designed for efficient range
// queries using lexicographic ordering:
//
//	metrics:{tenantCode}:{queueCode}:{vnamespace}:{resolution}:{invertedTimestamp}
type MetricsBucket struct {
	// Dimensions — identify what this bucket measures.
	TenantCode string
	QueueCode  string
	VNamespace string

	// Timestamp is the start of the bucket window (truncated to resolution), in Unix seconds.
	Timestamp int64
	// Resolution is the bucket width in seconds (e.g. 5, 60, 3600).
	Resolution int

	// ── Counters (accumulated during the window) ──

	// Published is the number of messages enqueued during this window.
	Published uint64
	// Delivered is the number of messages dequeued / claimed during this window.
	Delivered uint64
	// Acked is the number of messages acknowledged during this window.
	Acked uint64
	// Failed is the number of messages that failed or expired their lease during this window.
	Failed uint64

	// ── Gauges (snapshot at the moment the bucket was flushed) ──

	// Pending is the number of messages waiting in the queue at flush time.
	Pending uint64
	// InProcess is the number of messages currently being processed at flush time.
	InProcess uint64

	// ── Latency (accumulated during the window) ──

	// LatencySumMs is the total latency in milliseconds (sum of all processing times).
	// Divide by LatencyCount to get the average latency.
	LatencySumMs uint64
	// LatencyCount is the number of latency samples collected.
	LatencyCount uint64
	// LatencyMaxMs is the maximum latency observed in this window.
	LatencyMaxMs uint64

	// CreatedAt records when this bucket was persisted.
	CreatedAt time.Time
}

// MetricsDatapoint is the API-facing representation of a metrics bucket,
// including derived fields like messages-per-second and average latency.
type MetricsDatapoint struct {
	Timestamp    int64   `json:"timestamp"`
	Published    uint64  `json:"published"`
	Delivered    uint64  `json:"delivered"`
	Acked        uint64  `json:"acked"`
	Failed       uint64  `json:"failed"`
	Pending      uint64  `json:"pending"`
	InProcess    uint64  `json:"in_process"`
	MsgPerSecond float64 `json:"msg_per_second"`
	AvgLatencyMs float64 `json:"avg_latency_ms"`
	MaxLatencyMs uint64  `json:"max_latency_ms"`
}

// MetricsQueryResult is the top-level response returned by the metrics REST API.
type MetricsQueryResult struct {
	Resolution int                `json:"resolution"`
	Datapoints []MetricsDatapoint `json:"datapoints"`
}
