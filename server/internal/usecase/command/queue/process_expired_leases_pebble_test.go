package queue_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/shared/models"
)

const (
	pelCF  = "pel_test_cf"
	pelCFS = "pel-test-sector"
)

func newPELTestStore(t *testing.T) db.KVStore {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "pel_pebble_*")
	require.NoError(t, err)
	store, err := db.CreatePebbleStore(tmpDir, []string{pelCF, db.AdminFC}, []string{})
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close(); _ = os.RemoveAll(tmpDir) })
	return store
}

// TestFindExpiredLeases_Debug validates that FindExpiredLeases returns leases
// whose LeaseUntil is in the past.
func TestFindExpiredLeases_Debug(t *testing.T) {
	store := newPELTestStore(t)
	now := time.Date(2026, 4, 11, 12, 0, 0, 0, time.UTC)

	idFactory := &db.DeterministicIDGeneratorFactory{}

	// 1. Directly create a lease via the repository
	{
		uow := db.NewUnitOfWork(store, nil)
		leaseRepo, err := db.NewQueueMessageLeaseRepository(uow, idFactory, pelCF, pelCFS)
		require.NoError(t, err)

		lease := &models.QueueMessageLease{
			ID:             "test-lease-1",
			QueueMessageID: "msg-1",
			WorkerID:       "worker-1",
			LeaseStatus:    models.QueueMessageLeaseStatusActive,
			LeaseUntil:     now.Add(10 * time.Second), // expires at now+10s
		}

		_, err = leaseRepo.CreateQueueMessageLease(lease, now)
		require.NoError(t, err)
		require.NoError(t, uow.Commit())

		t.Logf("Created lease: ID=%s, LeaseUntil=%v", lease.ID, lease.LeaseUntil)
	}

	// 2. Verify direct lookup works
	{
		uow := db.NewUnitOfWork(store, nil)
		leaseRepo, err := db.NewQueueMessageLeaseRepository(uow, idFactory, pelCF, pelCFS)
		require.NoError(t, err)

		found, err := leaseRepo.GetQueueMessageLeaseByID("test-lease-1", now)
		require.NoError(t, err)
		require.NotNil(t, found)
		t.Logf("Direct lookup: ID=%s, Status=%s, LeaseUntil=%v", found.ID, found.LeaseStatus, found.LeaseUntil)
	}

	// 3. Test finding by LeaseStatus only (should work since it's an indexed field)
	{
		uow := db.NewUnitOfWork(store, nil)
		leaseRepo, err := db.NewQueueMessageLeaseRepository(uow, idFactory, pelCF, pelCFS)
		require.NoError(t, err)

		query := fmt.Sprintf("LeaseStatus = '%s'", models.QueueMessageLeaseStatusActive)
		t.Logf("Query (status only): %s", query)
		result, err := leaseRepo.Find(query, 100, "", now)
		require.NoError(t, err)
		t.Logf("Status-only query returned %d results", len(result.Entities))
		assert.Equal(t, 1, len(result.Entities), "Should find 1 active lease by status")
	}

	// 4. Test finding by LeaseUntil < future (should find the lease)
	futureTime := now.Add(20 * time.Second)
	{
		uow := db.NewUnitOfWork(store, nil)
		leaseRepo, err := db.NewQueueMessageLeaseRepository(uow, idFactory, pelCF, pelCFS)
		require.NoError(t, err)

		query := fmt.Sprintf("LeaseUntil < '%s'", futureTime)
		t.Logf("Query (LeaseUntil only): %s", query)
		result, err := leaseRepo.Find(query, 100, "", now)
		require.NoError(t, err)
		t.Logf("LeaseUntil-only query returned %d results", len(result.Entities))
		assert.Equal(t, 1, len(result.Entities), "Should find 1 lease with LeaseUntil < futureTime")
	}

	// 5. Test full FindExpiredLeases (AND query)
	{
		uow := db.NewUnitOfWork(store, nil)
		leaseRepo, err := db.NewQueueMessageLeaseRepository(uow, idFactory, pelCF, pelCFS)
		require.NoError(t, err)

		result, err := leaseRepo.FindExpiredLeases(100, 0, futureTime)
		require.NoError(t, err)
		t.Logf("FindExpiredLeases returned %d results", len(result.Entities))
		assert.Equal(t, 1, len(result.Entities), "FindExpiredLeases should return 1 expired lease")
	}
}
