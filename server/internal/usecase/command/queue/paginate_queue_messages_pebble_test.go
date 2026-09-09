package queue_test

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command/queue"
	"deadalus-orch/shared/models"
)

const (
	pmCF  = "pm_test_cf"
	pmCFS = "pm-test-sector"
)

func newPaginateMessagesTestStore(t *testing.T) db.KVStore {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "pm_pebble_*")
	require.NoError(t, err)
	store, err := db.CreatePebbleStore(tmpDir, []string{pmCF, db.AdminFC}, []string{})
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close(); _ = os.RemoveAll(tmpDir) })
	return store
}

func TestPaginateQueueMessagesCommand_WithLeases(t *testing.T) {
	store := newPaginateMessagesTestStore(t)
	now := time.Date(2026, 4, 11, 12, 0, 0, 0, time.UTC)
	idFactory := &db.DeterministicIDGeneratorFactory{}

	// Setup Queue and QueueMessages and QueueMessageLease
	uow := db.NewUnitOfWork(store, nil)
	queueRepo, err := db.NewQueueRepository(uow, idFactory, pmCF, pmCFS)
	require.NoError(t, err)

	q := &models.Queue{
		ID:          "queue-1",
		Code:        "Q1",
		State:       models.QueueActive,
		Type:        models.StandardQueue,
		MaxAttempts: 3,
	}
	_, err = queueRepo.CreateQueue(q, now)
	require.NoError(t, err)

	msgRepo, err := db.NewQueueMessageRepository(uow, idFactory, pmCF, pmCFS)
	require.NoError(t, err)

	msg1 := &models.QueueMessage{
		ID:        "msg-uuid-1",
		MessageID: "msg-1",
		QueueID:   "queue-1",
	}
	msg2 := &models.QueueMessage{
		ID:        "msg-uuid-2",
		MessageID: "msg-2",
		QueueID:   "queue-1",
	}
	_, err = msgRepo.CreateQueueMessage(msg1, now)
	require.NoError(t, err)
	_, err = msgRepo.CreateQueueMessage(msg2, now)
	require.NoError(t, err)

	leaseRepo, err := db.NewQueueMessageLeaseRepository(uow, idFactory, pmCF, pmCFS)
	require.NoError(t, err)

	lease1 := &models.QueueMessageLease{
		ID:             "lease-1",
		QueueMessageID: "msg-uuid-1",
		WorkerID:       "worker-1",
		LeaseStatus:    models.QueueMessageLeaseStatusActive,
		LeaseUntil:     now.Add(1 * time.Hour),
	}
	_, err = leaseRepo.CreateQueueMessageLease(lease1, now)
	require.NoError(t, err)

	require.NoError(t, uow.Commit())

	// Execute PaginateQueueMessagesCommand
	execUow := db.NewUnitOfWork(store, nil)
	cmd := &queue.PaginateQueueMessagesCommand{
		QueueID:  "queue-1",
		PageSize: 10,
		CF:       pmCF,
		CFS:      pmCFS,
	}

	cmdResult := cmd.Execute(execUow, now)
	require.Empty(t, cmdResult.Error)

	findResult, ok := cmdResult.Result.(db.FindResult[queue.QueueMessageWithLease])
	require.True(t, ok)

	assert.Len(t, findResult.Entities, 2)

	// Verify msg1 has its lease and msg2 has nil lease
	var msg1Found, msg2Found bool
	for _, item := range findResult.Entities {
		if item.Message.ID == "msg-uuid-1" {
			msg1Found = true
			require.NotNil(t, item.Lease)
			assert.Equal(t, "lease-1", item.Lease.ID)
			assert.Equal(t, "worker-1", item.Lease.WorkerID)
		} else if item.Message.ID == "msg-uuid-2" {
			msg2Found = true
			assert.Nil(t, item.Lease)
		}
	}

	assert.True(t, msg1Found)
	assert.True(t, msg2Found)
}
