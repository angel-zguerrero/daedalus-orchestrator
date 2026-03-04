package queue_test

// Integration tests for DequeueCommand backed by PebbleDB.
// Tests verify: persistence + fair-priority ordering (strict-priority drain).

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"deadalus-orch/server/internal/infrastructure/db"
	queueCommand "deadalus-orch/server/internal/usecase/command/queue"
	"deadalus-orch/shared/models"
)

const (
	dqCF  = "dq_test_cf"
	dqCFS = "dq-test-sector"
)

func newDequeueTestStore(t *testing.T) db.KVStore {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "dequeue_pebble_*")
	require.NoError(t, err)
	store, err := db.CreatePebbleStore(tmpDir, []string{dqCF, db.AdminFC}, []string{})
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close(); _ = os.RemoveAll(tmpDir) })
	return store
}

func createQueueDQ(t *testing.T, store db.KVStore, id, code string, state models.QueueState, thresholds map[int]int, now time.Time) *models.Queue {
	t.Helper()
	uow := db.NewUnitOfWork(store, nil)
	repo, err := db.NewQueueRepository(uow, &db.DeterministicIDGeneratorFactory{}, dqCF, dqCFS)
	require.NoError(t, err)
	q := &models.Queue{
		ID: id, Code: code, Name: code, VNamespace: "test-ns",
		State: state, Type: models.StandardQueue,
		AllowDuplicated: true, MaxAttempts: 3,
		DesiredPriorityThresholds: thresholds, PriorityThresholds: thresholds,
		CreatedAt: now, UpdatedAt: now,
	}
	_, err = repo.CreateQueue(q, now)
	require.NoError(t, err)
	require.NoError(t, uow.Commit())
	return q
}

func enqueueMsgDQ(t *testing.T, store db.KVStore, msgID string, priority int, queueID string, now time.Time) {
	t.Helper()
	uow := db.NewUnitOfWork(store, nil)
	cmd := &queueCommand.EnqueueCommand{
		Messages: []models.QueueMessage{{ID: msgID, MessageID: msgID, QueueID: queueID, Priority: priority, Content: []byte("body")}},
		CF:       dqCF,
		CFS:      dqCFS,
	}
	res := cmd.Execute(uow, now)
	require.Empty(t, res.Error, "enqueue %s: %s", msgID, res.Error)
	require.NoError(t, uow.Commit())
}

func dequeueDQ(t *testing.T, store db.KVStore, queueID, workerID string, now time.Time) queueCommand.DequeueResult {
	t.Helper()
	uow := db.NewUnitOfWork(store, nil)
	cmd := &queueCommand.DequeueCommand{
		QueueID:       queueID,
		JobWorkerID:   workerID,
		LeaseDuration: 30 * time.Second,
		CF:            dqCF,
		CFS:           dqCFS,
	}
	res := cmd.Execute(uow, now)
	require.Empty(t, res.Error, "dequeue: %s", res.Error)
	require.NoError(t, uow.Commit())
	dr, ok := res.Result.(queueCommand.DequeueResult)
	require.True(t, ok)
	return dr
}

func dequeueExpectErrorDQ(t *testing.T, store db.KVStore, queueID, workerID string, now time.Time, wantSubstr string) {
	t.Helper()
	uow := db.NewUnitOfWork(store, nil)
	cmd := &queueCommand.DequeueCommand{
		QueueID:       queueID,
		JobWorkerID:   workerID,
		LeaseDuration: 30 * time.Second,
		CF:            dqCF,
		CFS:           dqCFS,
	}
	res := cmd.Execute(uow, now)
	require.Contains(t, res.Error, wantSubstr)
}

// TestDequeueCommand_Pebble runs all sub-tests, each in its own isolated Pebble store.
func TestDequeueCommand_Pebble(t *testing.T) {
	tests := []struct {
		name string
		fn   func(t *testing.T)
	}{
		{"BasicDequeue_SingleMessage", testDQ_BasicDequeue},
		{"LeaseIsCreated", testDQ_LeaseIsCreated},
		{"QueueCountersDecrement", testDQ_QueueCountersDecrement},
		{"EmptyQueue_ReturnsError", testDQ_EmptyQueue},
		{"QueueNotFound_ReturnsError", testDQ_QueueNotFound},
		{"QueuePaused_ReturnsError", testDQ_QueuePaused},
		{"MissingQueueID_ReturnsError", testDQ_MissingQueueID},
		{"MissingWorkerID_ReturnsError", testDQ_MissingWorkerID},
		{"ZeroLeaseDuration_ReturnsError", testDQ_ZeroLeaseDuration},
		{"MaxDeliveringMessages_Enforced", testDQ_MaxDeliveringMessages},
		{"FairPriority_HigherPriorityFirst", testDQ_FairPriority_HigherPriorityFirst},
		{"FairPriority_DrainHighestBeforeLower", testDQ_FairPriority_DrainHighestBeforeLower},
		{"FairPriority_ThreeTiers_FullDrain", testDQ_FairPriority_ThreeTiers_FullDrain},
		{"FairPriority_LowerAddedAfterHigherDrained", testDQ_FairPriority_LowerAddedAfterHigherDrained},
		{"FairPriority_SinglePriority_FIFOOrder", testDQ_FairPriority_SinglePriority_FIFOOrder},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) { tc.fn(t) })
	}
}

func testDQ_BasicDequeue(t *testing.T) {
	store := newDequeueTestStore(t)
	now := time.Now()
	q := createQueueDQ(t, store, "q-basic", "BASIC", models.QueueActive, map[int]int{1: 0}, now)
	enqueueMsgDQ(t, store, "msg-001", 1, q.ID, now)
	dr := dequeueDQ(t, store, q.ID, "worker-1", now)
	assert.Equal(t, "msg-001", dr.Message.MessageID)
	assert.Equal(t, q.ID, dr.Message.QueueID)
}

func testDQ_LeaseIsCreated(t *testing.T) {
	store := newDequeueTestStore(t)
	now := time.Now()
	q := createQueueDQ(t, store, "q-lease", "LEASE", models.QueueActive, map[int]int{1: 0}, now)
	enqueueMsgDQ(t, store, "msg-lease", 1, q.ID, now)
	dr := dequeueDQ(t, store, q.ID, "worker-lease", now)
	assert.NotEmpty(t, dr.Lease.ID)
	assert.Equal(t, "msg-lease", dr.Message.MessageID)
	assert.Equal(t, "worker-lease", dr.Lease.WorkerID)
	assert.Equal(t, models.QueueMessageLeaseStatusActive, dr.Lease.LeaseStatus)
	assert.True(t, dr.Lease.LeaseUntil.After(now))
}

func testDQ_QueueCountersDecrement(t *testing.T) {
	store := newDequeueTestStore(t)
	now := time.Now()
	q := createQueueDQ(t, store, "q-ctr", "CTR", models.QueueActive, map[int]int{1: 0}, now)
	enqueueMsgDQ(t, store, "msg-c1", 1, q.ID, now)
	enqueueMsgDQ(t, store, "msg-c2", 1, q.ID, now.Add(time.Second))
	_ = dequeueDQ(t, store, q.ID, "worker-ctr", now.Add(2*time.Second))
	uow := db.NewUnitOfWork(store, nil)
	repo, err := db.NewQueueRepository(uow, &db.DeterministicIDGeneratorFactory{}, dqCF, dqCFS)
	require.NoError(t, err)
	updated, err := repo.GetQueueById(q.ID, now)
	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, 1, updated.MessagesCount, "MessagesCount must decrease to 1")
	assert.Equal(t, 1, updated.CurrentDeliveringMessages, "CurrentDeliveringMessages must be 1")
}

func testDQ_EmptyQueue(t *testing.T) {
	store := newDequeueTestStore(t)
	now := time.Now()
	q := createQueueDQ(t, store, "q-empty", "EMPTY", models.QueueActive, map[int]int{1: 0}, now)
	dequeueExpectErrorDQ(t, store, q.ID, "w1", now, "empty")
}

func testDQ_QueueNotFound(t *testing.T) {
	store := newDequeueTestStore(t)
	dequeueExpectErrorDQ(t, store, "nonexistent-id", "w1", time.Now(), "not found")
}

func testDQ_QueuePaused(t *testing.T) {
	store := newDequeueTestStore(t)
	now := time.Now()
	q := createQueueDQ(t, store, "q-paused", "PAUSED", models.QueuePaused, map[int]int{1: 0}, now)
	dequeueExpectErrorDQ(t, store, q.ID, "w1", now, "not available")
}

func testDQ_MissingQueueID(t *testing.T) {
	store := newDequeueTestStore(t)
	uow := db.NewUnitOfWork(store, nil)
	cmd := &queueCommand.DequeueCommand{QueueID: "", JobWorkerID: "w1", LeaseDuration: time.Second, CF: dqCF, CFS: dqCFS}
	r := cmd.Execute(uow, time.Now())
	assert.Contains(t, r.Error, "QueueID")
}

func testDQ_MissingWorkerID(t *testing.T) {
	store := newDequeueTestStore(t)
	uow := db.NewUnitOfWork(store, nil)
	cmd := &queueCommand.DequeueCommand{QueueID: "some-id", JobWorkerID: "", LeaseDuration: time.Second, CF: dqCF, CFS: dqCFS}
	r := cmd.Execute(uow, time.Now())
	assert.Contains(t, r.Error, "JobWorkerID")
}

func testDQ_ZeroLeaseDuration(t *testing.T) {
	store := newDequeueTestStore(t)
	uow := db.NewUnitOfWork(store, nil)
	cmd := &queueCommand.DequeueCommand{QueueID: "some-id", JobWorkerID: "w1", LeaseDuration: 0, CF: dqCF, CFS: dqCFS}
	r := cmd.Execute(uow, time.Now())
	assert.Contains(t, r.Error, "LeaseDuration")
}

func testDQ_MaxDeliveringMessages(t *testing.T) {
	store := newDequeueTestStore(t)
	now := time.Now()
	uow0 := db.NewUnitOfWork(store, nil)
	repo0, err := db.NewQueueRepository(uow0, &db.DeterministicIDGeneratorFactory{}, dqCF, dqCFS)
	require.NoError(t, err)
	q := &models.Queue{
		ID: "q-maxdlv", Code: "MAX_DLV", Name: "Max Delivering",
		VNamespace: "test-ns", State: models.QueueActive, Type: models.StandardQueue,
		MaxDeliveringMessages:     1,
		AllowDuplicated:           true,
		MaxAttempts:               3,
		DesiredPriorityThresholds: map[int]int{1: 0},
		PriorityThresholds:        map[int]int{1: 0},
		CreatedAt:                 now,
		UpdatedAt:                 now,
	}
	_, err = repo0.CreateQueue(q, now)
	require.NoError(t, err)
	require.NoError(t, uow0.Commit())
	enqueueMsgDQ(t, store, "mdlv-1", 1, q.ID, now)
	enqueueMsgDQ(t, store, "mdlv-2", 1, q.ID, now.Add(time.Second))
	_ = dequeueDQ(t, store, q.ID, "worker-a", now.Add(2*time.Second))
	dequeueExpectErrorDQ(t, store, q.ID, "worker-b", now.Add(3*time.Second), "maximum number of in-flight messages")
}

// ─── Fair-priority tests ────────────────────────────────────────────────────────────────────────────
//
// DequeueCommand creates a fresh PriorityQueue per call (all processedCounts=0,
// currentIndex=0 pointing to highest priority). One Dequeue() on that new PQ
// always returns the highest-priority non-empty partition. The observable
// behaviour across N consecutive DequeueCommand calls is therefore:
//
//   All P(n) messages before any P(n-1) message  (strict-priority drain)
//
// This is the fair-priority guarantee: lower priorities are never permanently
// starved — they are served as soon as all higher-priority messages are gone.

// testDQ_FairPriority_HigherPriorityFirst proves that the higher-priority
// message is always delivered first, regardless of insertion order.
func testDQ_FairPriority_HigherPriorityFirst(t *testing.T) {
	store := newDequeueTestStore(t)
	now := time.Now()
	q := createQueueDQ(t, store, "q-hp", "HIGH_PRIO", models.QueueActive, map[int]int{1: 0, 0: 0}, now)
	// Enqueue LOW-priority FIRST, then high-priority — order must not matter.
	enqueueMsgDQ(t, store, "low-msg", 0, q.ID, now)
	enqueueMsgDQ(t, store, "high-msg", 1, q.ID, now.Add(time.Second))
	dr1 := dequeueDQ(t, store, q.ID, "w1", now.Add(2*time.Second))
	assert.Equal(t, "high-msg", dr1.Message.MessageID, "P1 must be served before P0")
	assert.Equal(t, 1, dr1.Message.Priority)
	dr2 := dequeueDQ(t, store, q.ID, "w2", now.Add(3*time.Second))
	assert.Equal(t, "low-msg", dr2.Message.MessageID, "P0 served only after P1 is drained")
	assert.Equal(t, 0, dr2.Message.Priority)
}

// testDQ_FairPriority_DrainHighestBeforeLower verifies ALL high-priority
// messages are dequeued before ANY low-priority message is returned.
func testDQ_FairPriority_DrainHighestBeforeLower(t *testing.T) {
	store := newDequeueTestStore(t)
	now := time.Now()
	q := createQueueDQ(t, store, "q-drain", "DRAIN", models.QueueActive, map[int]int{2: 0, 1: 0}, now)
	const highCount, lowCount = 4, 3
	total := highCount + lowCount
	// Enqueue low-priority FIRST (should not influence priority ordering).
	for i := 0; i < lowCount; i++ {
		enqueueMsgDQ(t, store, fmt.Sprintf("low-%d", i), 1, q.ID, now.Add(time.Duration(i)*time.Millisecond))
	}
	for i := 0; i < highCount; i++ {
		enqueueMsgDQ(t, store, fmt.Sprintf("high-%d", i), 2, q.ID, now.Add(time.Duration(lowCount+i)*time.Millisecond))
	}
	got := make([]int, 0, total)
	for i := 0; i < total; i++ {
		dr := dequeueDQ(t, store, q.ID, fmt.Sprintf("w-%d", i), now.Add(time.Duration(total+i)*time.Millisecond))
		got = append(got, dr.Message.Priority)
	}
	for i := 0; i < highCount; i++ {
		assert.Equal(t, 2, got[i], "[pos %d] expected P2 — all P2 must be drained before any P1", i)
	}
	for i := highCount; i < total; i++ {
		assert.Equal(t, 1, got[i], "[pos %d] expected P1 — P1 follows after P2 is drained", i)
	}
}

// testDQ_FairPriority_ThreeTiers_FullDrain exercises P2 > P1 > P0 and verifies
// the complete strict-priority sequence: 3xP2, 4xP1, 2xP0.
func testDQ_FairPriority_ThreeTiers_FullDrain(t *testing.T) {
	store := newDequeueTestStore(t)
	now := time.Now()
	q := createQueueDQ(t, store, "q-3tier", "THREE_TIER", models.QueueActive, map[int]int{2: 0, 1: 0, 0: 0}, now)
	const p2c, p1c, p0c = 3, 4, 2
	total := p2c + p1c + p0c
	for i := 0; i < p2c; i++ {
		enqueueMsgDQ(t, store, fmt.Sprintf("p2-%d", i), 2, q.ID, now.Add(time.Duration(i)*time.Millisecond))
	}
	for i := 0; i < p1c; i++ {
		enqueueMsgDQ(t, store, fmt.Sprintf("p1-%d", i), 1, q.ID, now.Add(time.Duration(p2c+i)*time.Millisecond))
	}
	for i := 0; i < p0c; i++ {
		enqueueMsgDQ(t, store, fmt.Sprintf("p0-%d", i), 0, q.ID, now.Add(time.Duration(p2c+p1c+i)*time.Millisecond))
	}
	want := make([]int, 0, total)
	for i := 0; i < p2c; i++ {
		want = append(want, 2)
	}
	for i := 0; i < p1c; i++ {
		want = append(want, 1)
	}
	for i := 0; i < p0c; i++ {
		want = append(want, 0)
	}
	got := make([]int, 0, total)
	for i := 0; i < total; i++ {
		dr := dequeueDQ(t, store, q.ID, fmt.Sprintf("w-%d", i), now.Add(time.Duration(total+i)*time.Millisecond))
		got = append(got, dr.Message.Priority)
	}
	assert.Equal(t, want, got, "strict-priority sequence: P2x%d, P1x%d, P0x%d", p2c, p1c, p0c)
}

// testDQ_FairPriority_LowerAddedAfterHigherDrained verifies that lower-priority
// messages added after all higher-priority messages have been consumed are still
// served correctly in subsequent DequeueCommand calls.
func testDQ_FairPriority_LowerAddedAfterHigherDrained(t *testing.T) {
	store := newDequeueTestStore(t)
	now := time.Now()
	q := createQueueDQ(t, store, "q-late-low", "LATE_LOW", models.QueueActive, map[int]int{2: 0, 1: 0}, now)
	enqueueMsgDQ(t, store, "p2-a", 2, q.ID, now)
	enqueueMsgDQ(t, store, "p2-b", 2, q.ID, now.Add(time.Millisecond))
	dr1 := dequeueDQ(t, store, q.ID, "w1", now.Add(2*time.Millisecond))
	assert.Equal(t, 2, dr1.Message.Priority, "first dequeue must return P2")
	dr2 := dequeueDQ(t, store, q.ID, "w2", now.Add(3*time.Millisecond))
	assert.Equal(t, 2, dr2.Message.Priority, "second dequeue must return remaining P2")
	// P2 fully drained — now add a P1 message.
	enqueueMsgDQ(t, store, "p1-late", 1, q.ID, now.Add(4*time.Millisecond))
	dr3 := dequeueDQ(t, store, q.ID, "w3", now.Add(5*time.Millisecond))
	assert.Equal(t, 1, dr3.Message.Priority, "third dequeue must return the late P1 message")
	assert.Equal(t, "p1-late", dr3.Message.MessageID)
}

// testDQ_FairPriority_SinglePriority_FIFOOrder confirms same-priority messages
// are delivered in FIFO (insertion-time) order.
func testDQ_FairPriority_SinglePriority_FIFOOrder(t *testing.T) {
	store := newDequeueTestStore(t)
	now := time.Now()
	q := createQueueDQ(t, store, "q-fifo", "FIFO", models.QueueActive, map[int]int{1: 0}, now)
	ids := []string{"first", "second", "third", "fourth"}
	for i, id := range ids {
		enqueueMsgDQ(t, store, id, 1, q.ID, now.Add(time.Duration(i)*time.Second))
	}
	for i, wantID := range ids {
		dr := dequeueDQ(t, store, q.ID, fmt.Sprintf("w-%d", i), now.Add(time.Duration(len(ids)+i)*time.Second))
		assert.Equal(t, wantID, dr.Message.MessageID, "FIFO position %d", i)
	}
}
