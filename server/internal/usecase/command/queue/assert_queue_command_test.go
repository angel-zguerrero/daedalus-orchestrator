package queue_test

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"deadalus-orch/server/internal/infrastructure/db"
	queue_command "deadalus-orch/server/internal/usecase/command/queue"
	"deadalus-orch/shared/models"
)

const (
	assertCF  = "assert_test_cf"
	assertCFS = "assert-test-sector"
)

func newAssertQueueTestStore(t *testing.T) db.KVStore {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "assert_queue_pebble_*")
	require.NoError(t, err)
	store, err := db.CreatePebbleStore(tmpDir, []string{assertCF, db.AdminFC}, []string{})
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close(); _ = os.RemoveAll(tmpDir) })
	return store
}

func TestAssertQueueCommand_TypeAndWorkflowDefinitionID_UpgradedForExistingQueue(t *testing.T) {
	store := newAssertQueueTestStore(t)
	now := time.Now()
	idFactory := &db.DeterministicIDGeneratorFactory{}

	// 1. Create an existing queue with Type = StandardQueue and empty WorkflowDefinitionID
	existingQueueID := "q_existing_123"
	queueCode := "wf-act-MYWORKFLOW"
	{
		uow := db.NewUnitOfWork(store, nil)
		queueRepo, err := db.NewQueueRepository(uow, idFactory, assertCF, assertCFS)
		require.NoError(t, err)

		existingQueue := &models.Queue{
			ID:          existingQueueID,
			Code:        queueCode,
			Name:        "Pre-existing Standard Queue",
			Type:        models.StandardQueue,
			State:       models.QueueActive,
			MaxAttempts: 5,
		}
		_, err = queueRepo.CreateQueue(existingQueue, now)
		require.NoError(t, err)
		require.NoError(t, uow.Commit())
	}

	// 2. Execute AssertQueueCommand specifying Type = WorkflowActivityQueue and WorkflowDefinitionID = "def_999"
	{
		uow := db.NewUnitOfWork(store, nil)
		assertCmd := &queue_command.AssertQueueCommand{
			Queues: []models.Queue{
				{
					Code:                 queueCode,
					Name:                 "My Workflow Activity Queue",
					Type:                 models.WorkflowActivityQueue,
					WorkflowDefinitionID: "def_999",
					MaxAttempts:          10,
				},
			},
			CF:  assertCF,
			CFS: assertCFS,
		}

		result := assertCmd.Execute(uow, now)
		require.Empty(t, result.Error)
		require.NoError(t, uow.Commit())
	}

	// 3. Verify that the updated queue in DB retains the new Type and WorkflowDefinitionID
	{
		uow := db.NewUnitOfWork(store, nil)
		queueRepo, err := db.NewQueueRepository(uow, idFactory, assertCF, assertCFS)
		require.NoError(t, err)

		q, err := queueRepo.GetQueueByCode(queueCode, "", now)
		require.NoError(t, err)
		require.NotNil(t, q)
		assert.Equal(t, existingQueueID, q.ID)
		assert.Equal(t, models.WorkflowActivityQueue, q.Type, "Queue Type must be upgraded to WorkflowActivityQueue")
		assert.Equal(t, "def_999", q.WorkflowDefinitionID, "WorkflowDefinitionID must be set")
	}
}

func TestAssertQueueCommand_NewQueue_TypeAndWorkflowDefinitionID_Set(t *testing.T) {
	store := newAssertQueueTestStore(t)
	now := time.Now()
	idFactory := &db.DeterministicIDGeneratorFactory{}

	queueCode := "wf-exec-NEWWORKFLOW"
	{
		uow := db.NewUnitOfWork(store, nil)
		assertCmd := &queue_command.AssertQueueCommand{
			Queues: []models.Queue{
				{
					Code:                 queueCode,
					Name:                 "New Execution Queue",
					Type:                 models.WorkflowExecutionQueue,
					WorkflowDefinitionID: "def_exec_123",
					MaxAttempts:          10,
				},
			},
			CF:  assertCF,
			CFS: assertCFS,
		}

		result := assertCmd.Execute(uow, now)
		require.Empty(t, result.Error)
		require.NoError(t, uow.Commit())
	}

	{
		uow := db.NewUnitOfWork(store, nil)
		queueRepo, err := db.NewQueueRepository(uow, idFactory, assertCF, assertCFS)
		require.NoError(t, err)

		q, err := queueRepo.GetQueueByCode(queueCode, "", now)
		require.NoError(t, err)
		require.NotNil(t, q)
		assert.Equal(t, models.WorkflowExecutionQueue, q.Type)
		assert.Equal(t, "def_exec_123", q.WorkflowDefinitionID)
	}
}

func TestAssertQueueCommand_PreservesExistingTypeWhenPassedTypeIsEmpty(t *testing.T) {
	store := newAssertQueueTestStore(t)
	now := time.Now()
	idFactory := &db.DeterministicIDGeneratorFactory{}

	queueCode := "wf-act-PRESERVE"
	{
		uow := db.NewUnitOfWork(store, nil)
		queueRepo, err := db.NewQueueRepository(uow, idFactory, assertCF, assertCFS)
		require.NoError(t, err)

		existingQueue := &models.Queue{
			ID:                   "q_preserve_1",
			Code:                 queueCode,
			Name:                 "Existing Activity Queue",
			Type:                 models.WorkflowActivityQueue,
			WorkflowDefinitionID: "def_preserve_456",
			MaxAttempts:          5,
		}
		_, err = queueRepo.CreateQueue(existingQueue, now)
		require.NoError(t, err)
		require.NoError(t, uow.Commit())
	}

	// Assert without specifying Type or WorkflowDefinitionID
	{
		uow := db.NewUnitOfWork(store, nil)
		assertCmd := &queue_command.AssertQueueCommand{
			Queues: []models.Queue{
				{
					Code:        queueCode,
					Name:        "Updated Queue Name Only",
					MaxAttempts: 10,
				},
			},
			CF:  assertCF,
			CFS: assertCFS,
		}

		result := assertCmd.Execute(uow, now)
		require.Empty(t, result.Error)
		require.NoError(t, uow.Commit())
	}

	{
		uow := db.NewUnitOfWork(store, nil)
		queueRepo, err := db.NewQueueRepository(uow, idFactory, assertCF, assertCFS)
		require.NoError(t, err)

		q, err := queueRepo.GetQueueByCode(queueCode, "", now)
		require.NoError(t, err)
		require.NotNil(t, q)
		assert.Equal(t, models.WorkflowActivityQueue, q.Type)
		assert.Equal(t, "def_preserve_456", q.WorkflowDefinitionID)
	}
}
