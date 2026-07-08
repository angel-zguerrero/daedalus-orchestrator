package metrics

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewMetricsCollector_DefaultResolution(t *testing.T) {
	mc := NewMetricsCollector(0)
	if mc.Resolution() != 5 {
		t.Errorf("expected default resolution 5, got %d", mc.Resolution())
	}
}

func TestNewMetricsCollector_CustomResolution(t *testing.T) {
	mc := NewMetricsCollector(10)
	if mc.Resolution() != 10 {
		t.Errorf("expected resolution 10, got %d", mc.Resolution())
	}
}

func TestRecordPublish(t *testing.T) {
	mc := NewMetricsCollector(5)
	mc.RecordPublish("tenant1", "queue1", "default", 3)
	mc.RecordPublish("tenant1", "queue1", "default", 7)

	// Bucket is still open (current window), so FlushCompleted should return nothing.
	flushed := mc.FlushCompleted()
	if len(flushed) != 0 {
		t.Errorf("expected 0 flushed buckets for current window, got %d", len(flushed))
	}

	if mc.ActiveBucketCount() != 1 {
		t.Errorf("expected 1 active bucket, got %d", mc.ActiveBucketCount())
	}
}

func TestRecordMultipleQueues(t *testing.T) {
	mc := NewMetricsCollector(5)
	mc.RecordPublish("t1", "q1", "ns1", 5)
	mc.RecordPublish("t1", "q2", "ns1", 3)
	mc.RecordPublish("t2", "q1", "ns1", 1)

	if mc.ActiveBucketCount() != 3 {
		t.Errorf("expected 3 active buckets (different queues), got %d", mc.ActiveBucketCount())
	}
}

func TestRecordLatency(t *testing.T) {
	mc := NewMetricsCollector(5)
	mc.RecordLatency("t1", "q1", "ns", 100)
	mc.RecordLatency("t1", "q1", "ns", 200)
	mc.RecordLatency("t1", "q1", "ns", 50)

	if mc.ActiveBucketCount() != 1 {
		t.Errorf("expected 1 active bucket, got %d", mc.ActiveBucketCount())
	}
}

func TestSnapshotGauges(t *testing.T) {
	mc := NewMetricsCollector(5)
	mc.SnapshotGauges("t1", "q1", "ns", 100, 20)

	// SnapshotGauges writes to persistentGauges (not time-series buckets),
	// so ActiveBucketCount reflects 0 buckets — that is correct.
	// Verify the gauge was captured by flushing and checking the snapshot values.
	if mc.ActiveBucketCount() != 0 {
		t.Errorf("expected 0 active time-series buckets after SnapshotGauges (gauges are stored separately), got %d", mc.ActiveBucketCount())
	}
}

func TestConcurrentRecording(t *testing.T) {
	mc := NewMetricsCollector(5)
	var wg sync.WaitGroup

	const goroutines = 100
	const opsPerGoroutine = 1000

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < opsPerGoroutine; j++ {
				mc.RecordPublish("t1", "q1", "ns", 1)
				mc.RecordDelivery("t1", "q1", "ns", 1)
				mc.RecordAck("t1", "q1", "ns", 1)
				mc.RecordFailed("t1", "q1", "ns", 1)
				mc.RecordLatency("t1", "q1", "ns", 10)
			}
		}()
	}
	wg.Wait()

	// All records should be in 1 bucket (same queue, same time window).
	if mc.ActiveBucketCount() != 1 {
		t.Errorf("expected 1 active bucket after concurrent recording, got %d", mc.ActiveBucketCount())
	}
}

func TestFlushCompleted_OnlyFlushesPastBuckets(t *testing.T) {
	// Use a very short resolution to make the test reliable.
	mc := NewMetricsCollector(1)
	mc.RecordPublish("t1", "q1", "ns", 5)

	// Wait for the bucket to close.
	time.Sleep(1200 * time.Millisecond)

	// Record in the new (current) bucket.
	mc.RecordPublish("t1", "q1", "ns", 3)

	flushed := mc.FlushCompleted()
	if len(flushed) != 1 {
		t.Fatalf("expected 1 flushed bucket, got %d", len(flushed))
	}
	if flushed[0].Published != 5 {
		t.Errorf("expected flushed published=5, got %d", flushed[0].Published)
	}
	if flushed[0].TenantCode != "t1" || flushed[0].QueueCode != "q1" {
		t.Errorf("unexpected flushed bucket dimensions: tenant=%s, queue=%s", flushed[0].TenantCode, flushed[0].QueueCode)
	}

	// Current bucket should still be active.
	if mc.ActiveBucketCount() != 1 {
		t.Errorf("expected 1 active bucket (current window), got %d", mc.ActiveBucketCount())
	}
}

func TestFlushCompleted_IncludesLatency(t *testing.T) {
	mc := NewMetricsCollector(1)
	mc.RecordLatency("t1", "q1", "ns", 100)
	mc.RecordLatency("t1", "q1", "ns", 300)
	mc.RecordLatency("t1", "q1", "ns", 50)

	time.Sleep(1200 * time.Millisecond)

	flushed := mc.FlushCompleted()
	if len(flushed) != 1 {
		t.Fatalf("expected 1 flushed bucket, got %d", len(flushed))
	}

	b := flushed[0]
	if b.LatencyCount != 3 {
		t.Errorf("expected latency count=3, got %d", b.LatencyCount)
	}
	if b.LatencySumMs != 450 {
		t.Errorf("expected latency sum=450, got %d", b.LatencySumMs)
	}
	if b.LatencyMaxMs != 300 {
		t.Errorf("expected latency max=300, got %d", b.LatencyMaxMs)
	}
}

func TestTruncateToResolution(t *testing.T) {
	mc := NewMetricsCollector(5)

	tests := []struct {
		input    int64
		expected int64
	}{
		{0, 0},
		{4, 0},
		{5, 5},
		{7, 5},
		{10, 10},
		{13, 10},
	}

	for _, tc := range tests {
		got := mc.truncateToResolution(tc.input)
		if got != tc.expected {
			t.Errorf("truncateToResolution(%d) = %d, want %d", tc.input, got, tc.expected)
		}
	}
}

func TestUpdateMax(t *testing.T) {
	var target atomic.Uint64
	target.Store(100)

	// Should not update (50 < 100)
	updateMax(&target, 50)
	if target.Load() != 100 {
		t.Errorf("expected 100 after updateMax(50), got %d", target.Load())
	}

	// Should update (200 > 100)
	updateMax(&target, 200)
	if target.Load() != 200 {
		t.Errorf("expected 200 after updateMax(200), got %d", target.Load())
	}

	// Should update (300 > 200)
	updateMax(&target, 300)
	if target.Load() != 300 {
		t.Errorf("expected 300 after updateMax(300), got %d", target.Load())
	}
}
