package business_logic

import (
	"fmt"
	"sync"
	"testing"
)

// ─────────────────────────────────────────────────────────────────────────────
// claimCursorRegistry unit tests
// ─────────────────────────────────────────────────────────────────────────────

func TestCursorRegistry_GetReturnsEmptyStringByDefault(t *testing.T) {
	r := newClaimCursorRegistry()
	key := claimCursorKey{workerID: "w1", policyCode: "policy-0"}

	got := r.get(key)
	if got != "" {
		t.Errorf("expected empty string for unknown key, got %q", got)
	}
}

func TestCursorRegistry_SetAndGet(t *testing.T) {
	r := newClaimCursorRegistry()
	key := claimCursorKey{workerID: "w1", policyCode: "policy-0"}

	r.set(key, "cursor-abc")

	got := r.get(key)
	if got != "cursor-abc" {
		t.Errorf("expected %q, got %q", "cursor-abc", got)
	}
}

func TestCursorRegistry_SetEmptyCursorDeletesEntry(t *testing.T) {
	r := newClaimCursorRegistry()
	key := claimCursorKey{workerID: "w1", policyCode: "policy-0"}

	// Store a cursor then mark it as exhausted.
	r.set(key, "cursor-abc")
	r.set(key, "") // exhausted → should delete

	got := r.get(key)
	if got != "" {
		t.Errorf("expected empty string after set(\"\"), got %q", got)
	}

	// Confirm the key is truly gone from the map (not just zero-valued).
	r.mu.Lock()
	_, exists := r.cursors[key]
	r.mu.Unlock()
	if exists {
		t.Error("key should have been deleted from the map on set(\"\")")
	}
}

// TestCursorRegistry_CursorAdvancement verifies the round-robin wraparound:
// the cursor advances page by page and wraps around to "" when exhausted.
func TestCursorRegistry_CursorAdvancement(t *testing.T) {
	r := newClaimCursorRegistry()
	key := claimCursorKey{workerID: "w1", policyCode: "policy-0"}

	pages := []string{"page-1", "page-2", "page-3"}

	// Simulate three cycles where each cycle starts from the saved cursor.
	for i, page := range pages {
		// Before setting: cursor should be either "" (first) or previous page.
		prev := ""
		if i > 0 {
			prev = pages[i-1]
		}
		if got := r.get(key); got != prev {
			t.Errorf("cycle %d: expected cursor %q before set, got %q", i, prev, got)
		}
		r.set(key, page)
	}

	// Exhaust — simulates reaching the last page with no next cursor.
	r.set(key, "")

	if got := r.get(key); got != "" {
		t.Errorf("after exhaustion, expected \"\" but got %q", got)
	}
}

// TestCursorRegistry_IsolatesWorkersAndPolicies ensures different workers and
// policies do not share cursor state.
func TestCursorRegistry_IsolatesWorkersAndPolicies(t *testing.T) {
	r := newClaimCursorRegistry()

	keyA0 := claimCursorKey{workerID: "worker-A", policyCode: "policy-0"}
	keyA1 := claimCursorKey{workerID: "worker-A", policyCode: "policy-1"}
	keyB0 := claimCursorKey{workerID: "worker-B", policyCode: "policy-0"}

	r.set(keyA0, "cursor-A0")
	r.set(keyA1, "cursor-A1")
	r.set(keyB0, "cursor-B0")

	if got := r.get(keyA0); got != "cursor-A0" {
		t.Errorf("keyA0: expected %q, got %q", "cursor-A0", got)
	}
	if got := r.get(keyA1); got != "cursor-A1" {
		t.Errorf("keyA1: expected %q, got %q", "cursor-A1", got)
	}
	if got := r.get(keyB0); got != "cursor-B0" {
		t.Errorf("keyB0: expected %q, got %q", "cursor-B0", got)
	}
}

// TestCursorRegistry_TenantAndVNamespaceLevelAreIndependent verifies that the
// tenant-level cursor (tenantID="") and vnamespace-level cursors (tenantID set)
// are stored independently under the same workerID+policyCode.
func TestCursorRegistry_TenantAndVNamespaceLevelAreIndependent(t *testing.T) {
	r := newClaimCursorRegistry()

	tenantKey := claimCursorKey{workerID: "w1", policyCode: "policy-0", tenantID: ""}
	vnsKey := claimCursorKey{workerID: "w1", policyCode: "policy-0", tenantID: "tenant-123"}

	r.set(tenantKey, "tenant-page-2")
	r.set(vnsKey, "vns-page-1")

	if got := r.get(tenantKey); got != "tenant-page-2" {
		t.Errorf("tenant cursor: expected %q, got %q", "tenant-page-2", got)
	}
	if got := r.get(vnsKey); got != "vns-page-1" {
		t.Errorf("vns cursor: expected %q, got %q", "vns-page-1", got)
	}

	// Exhausting one should not affect the other.
	r.set(tenantKey, "")
	if got := r.get(vnsKey); got != "vns-page-1" {
		t.Errorf("vns cursor should be unaffected after tenant cursor reset, got %q", got)
	}
}

// TestCursorRegistry_EvictWorkerRemovesAllItsEntries verifies that evictWorker
// removes all tenant-level and vnamespace-level cursors for that worker only.
func TestCursorRegistry_EvictWorkerRemovesAllItsEntries(t *testing.T) {
	r := newClaimCursorRegistry()

	// Populate multiple cursors for worker-A (tenant and vns levels, two policies).
	keysA := []claimCursorKey{
		{workerID: "worker-A", policyCode: "policy-0"},
		{workerID: "worker-A", policyCode: "policy-0", tenantID: "t1"},
		{workerID: "worker-A", policyCode: "policy-0", tenantID: "t2"},
		{workerID: "worker-A", policyCode: "policy-1"},
		{workerID: "worker-A", policyCode: "policy-1", tenantID: "t1"},
	}
	for i, k := range keysA {
		r.set(k, fmt.Sprintf("cursor-%d", i))
	}

	// Also populate worker-B so we can verify it is untouched.
	keyB := claimCursorKey{workerID: "worker-B", policyCode: "policy-0"}
	r.set(keyB, "cursor-B")

	// Evict worker-A.
	r.evictWorker("worker-A")

	// All worker-A entries must be gone.
	for _, k := range keysA {
		if got := r.get(k); got != "" {
			t.Errorf("after evict, key %+v should be empty, got %q", k, got)
		}
	}

	// worker-B must be untouched.
	if got := r.get(keyB); got != "cursor-B" {
		t.Errorf("worker-B cursor should survive eviction of worker-A, got %q", got)
	}
}

// TestCursorRegistry_EvictNonExistentWorkerIsNoOp ensures evicting a worker
// that has no cursors registered does not panic or corrupt the registry.
func TestCursorRegistry_EvictNonExistentWorkerIsNoOp(t *testing.T) {
	r := newClaimCursorRegistry()
	key := claimCursorKey{workerID: "worker-A", policyCode: "policy-0"}
	r.set(key, "cursor-A")

	// Evicting an unknown worker should be a no-op.
	r.evictWorker("worker-UNKNOWN")

	if got := r.get(key); got != "cursor-A" {
		t.Errorf("worker-A cursor should be untouched, got %q", got)
	}
}

// TestCursorRegistry_ConcurrentSafety exercises get, set and evictWorker from
// multiple goroutines simultaneously to surface data races (run with -race).
func TestCursorRegistry_ConcurrentSafety(t *testing.T) {
	r := newClaimCursorRegistry()

	const numWorkers = 10
	const opsPerWorker = 200

	var wg sync.WaitGroup

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		workerID := fmt.Sprintf("worker-%d", w)
		go func(wid string) {
			defer wg.Done()
			for i := 0; i < opsPerWorker; i++ {
				key := claimCursorKey{
					workerID:   wid,
					policyCode: fmt.Sprintf("policy-%d", i%3),
					tenantID:   fmt.Sprintf("tenant-%d", i%5),
				}
				switch i % 4 {
				case 0:
					r.set(key, fmt.Sprintf("cursor-%d", i))
				case 1:
					_ = r.get(key)
				case 2:
					r.set(key, "") // simulate exhaustion
				case 3:
					r.evictWorker(wid)
				}
			}
		}(workerID)
	}

	wg.Wait()
	// No assertions needed — the race detector validates concurrent correctness.
}

// TestCursorRegistry_EvictWorkerPublicMethod verifies that JobWorkerBO.EvictWorkerCursors
// properly delegates to the internal registry.
func TestCursorRegistry_EvictWorkerPublicMethod(t *testing.T) {
	bo := &JobWorkerBO{
		stoppers:       make(map[string]bool),
		cursorRegistry: newClaimCursorRegistry(),
	}

	key := claimCursorKey{workerID: "w1", policyCode: "policy-0", tenantID: "t1"}
	bo.cursorRegistry.set(key, "some-cursor")

	// Call the public method that the gRPC service uses on disconnect.
	bo.EvictWorkerCursors("w1")

	if got := bo.cursorRegistry.get(key); got != "" {
		t.Errorf("EvictWorkerCursors should have cleared the cursor, got %q", got)
	}
}
