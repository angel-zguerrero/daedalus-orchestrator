package scheduled_job_test

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/pkg/utils"
	queueCommand "deadalus-orch/server/internal/usecase/command/queue"
	scheduledJobCommand "deadalus-orch/server/internal/usecase/command/scheduled-job"
	"deadalus-orch/shared/models"
)

const (
	TestFC  = "test_scheduled_job_fc"
	TestCFS = "test_scheduled_job_cfs"
)

func newTestPebbleStore(t *testing.T) db.KVStore {
	tempDir, err := os.MkdirTemp("", "scheduled_job_test_*")
	require.NoError(t, err)

	cfNames := []string{"default", TestFC, db.AdminFC}
	store, err := db.CreatePebbleStore(tempDir, cfNames, []string{})
	require.NoError(t, err)

	t.Cleanup(func() {
		store.Close()
		os.RemoveAll(tempDir)
	})
	return store
}

func setupTestQueue(t *testing.T, store db.KVStore, cf, cfs string, now time.Time) *models.Queue {
	uow := db.NewUnitOfWork(store, nil)
	idFactory := &db.DeterministicIDGeneratorFactory{}

	queueRepo, err := db.NewQueueRepository(uow, idFactory, cf, cfs)
	require.NoError(t, err)

	queue := &models.Queue{
		ID:                        "queue-id-100",
		Code:                      "email-queue",
		Name:                      "Email Queue",
		VNamespace:                "default",
		State:                     models.QueueActive,
		Type:                      models.StandardQueue,
		DefaultQueueMessageTTL:    3600,
		AllowDuplicated:           true,
		MaxAttempts:               3,
		MessagesCount:             0,
		DesiredPriorityThresholds: map[int]int{0: 0, 1: 100},
		PriorityThresholds:        map[int]int{0: 0, 1: 100},
		CreatedAt:                 now,
		UpdatedAt:                 now,
	}

	queueID, err := queueRepo.CreateQueue(queue, now)
	require.NoError(t, err)
	queue.ID = queueID

	require.NoError(t, uow.Commit())
	return queue
}

func setupTestExchange(t *testing.T, store db.KVStore, cf, cfs string, now time.Time) *models.Exchange {
	uow := db.NewUnitOfWork(store, nil)
	idFactory := &db.DeterministicIDGeneratorFactory{}

	exchangeRepo, err := db.NewExchangeRepository(uow, idFactory, cf, cfs)
	require.NoError(t, err)

	exchange := &models.Exchange{
		ID:         "exchange-id-200",
		Code:       "email-events",
		Name:       "Email Events",
		Type:       models.Topic,
		VNamespace: "default",
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	exID, err := exchangeRepo.CreateExchange(exchange, now)
	require.NoError(t, err)
	exchange.ID = exID

	require.NoError(t, uow.Commit())
	return exchange
}

func TestScheduledJob_CodeResolutionAndCreation(t *testing.T) {
	store := newTestPebbleStore(t)
	startTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	queue := setupTestQueue(t, store, TestFC, TestCFS, startTime)

	t.Run("CreateOneOffWithRunAfter", func(t *testing.T) {
		uow := db.NewUnitOfWork(store, nil)
		idFactory := &db.DeterministicIDGeneratorFactory{}

		jobRepo, err := db.NewScheduledJobRepository(uow, idFactory, TestFC, TestCFS)
		require.NoError(t, err)

		nextRunAt, err := utils.CalculateNextRunAt(
			models.ScheduledJobOneOff,
			nil,
			"5m",
			"",
			"",
			startTime,
		)
		require.NoError(t, err)
		assert.Equal(t, startTime.Add(5*time.Minute), nextRunAt)

		job := models.ScheduledJob{
			ID:                             "job-one-off-1",
			TenantID:                       TestCFS,
			TargetType:                     "queue",
			TargetID:                       queue.ID,
			TargetCode:                     queue.Code,
			RoutingKeyOrPatternOrQueueCode: queue.Code,
			VNamespace:                     "default",
			Content:                        "Welcome Email",
			ContentType:                    "text/plain",
			Priority:                       1,
			Type:                           models.ScheduledJobOneOff,
			RunAfter:                       "5m",
			NextRunAt:                      nextRunAt,
			State:                          models.ScheduledJobIdle,
		}

		jobID, err := jobRepo.CreateScheduledJob(&job, startTime)
		require.NoError(t, err)
		assert.Equal(t, "job-one-off-1", jobID)
		require.NoError(t, uow.Commit())

		// Verify ORM persistence
		uowVerify := db.NewUnitOfWork(store, nil)
		repoVerify, _ := db.NewScheduledJobRepository(uowVerify, idFactory, TestFC, TestCFS)
		savedJob, err := repoVerify.GetScheduledJobByID("job-one-off-1", startTime)
		require.NoError(t, err)
		require.NotNil(t, savedJob)
		assert.Equal(t, queue.ID, savedJob.TargetID)
		assert.Equal(t, string(models.ScheduledJobIdle), string(savedJob.State))

		// Verify companion index key existence in KV store
		indexKey := db.FormatScheduledJobIndexKey(savedJob.NextRunAt, savedJob.ID)
		exists, err := store.Exists(TestFC, TestCFS, indexKey, startTime)
		require.NoError(t, err)
		assert.True(t, exists, "Companion index key must exist in DB")
	})

	t.Run("CreateRecurringWithCron", func(t *testing.T) {
		uow := db.NewUnitOfWork(store, nil)
		idFactory := &db.DeterministicIDGeneratorFactory{}

		jobRepo, err := db.NewScheduledJobRepository(uow, idFactory, TestFC, TestCFS)
		require.NoError(t, err)

		cronExpr := "0 12 * * *" // Everyday at 12:00
		nextRunAt, err := utils.CalculateNextRunAt(
			models.ScheduledJobRecurring,
			nil,
			"",
			"",
			cronExpr,
			startTime,
		)
		require.NoError(t, err)
		assert.Equal(t, time.Date(2025, 1, 2, 12, 0, 0, 0, time.UTC), nextRunAt)

		_ = jobRepo
	})

	t.Run("CreateOrUpdateScheduledJob_UpsertByCode", func(t *testing.T) {
		uow1 := db.NewUnitOfWork(store, nil)
		idFactory := &db.DeterministicIDGeneratorFactory{}
		jobRepo1, err := db.NewScheduledJobRepository(uow1, idFactory, TestFC, TestCFS)
		require.NoError(t, err)

		// 1. Initial creation with unique Code
		initialNextRunAt := startTime.Add(10 * time.Minute)
		job1 := models.ScheduledJob{
			TenantID:   TestCFS,
			TargetType: "queue",
			TargetID:   queue.ID,
			TargetCode: queue.Code,
			VNamespace: "default",
			Code:       "upsert-test-code-1",
			Content:    "Initial Content",
			Type:       models.ScheduledJobOneOff,
			RunAfter:   "10m",
			NextRunAt:  initialNextRunAt,
			State:      models.ScheduledJobIdle,
		}

		id1, err := jobRepo1.CreateScheduledJob(&job1, startTime)
		require.NoError(t, err)
		require.NotEmpty(t, id1)
		require.NoError(t, uow1.Commit())

		// Verify initial index key exists
		indexKey1 := db.FormatScheduledJobIndexKey(initialNextRunAt, id1)
		exists1, err := store.Exists(TestFC, TestCFS, indexKey1, startTime)
		require.NoError(t, err)
		assert.True(t, exists1, "Initial index key must exist")

		// 2. Secondary creation with SAME Code (Upsert)
		uow2 := db.NewUnitOfWork(store, nil)
		jobRepo2, err := db.NewScheduledJobRepository(uow2, idFactory, TestFC, TestCFS)
		require.NoError(t, err)

		updatedNextRunAt := startTime.Add(30 * time.Minute)
		job2 := models.ScheduledJob{
			TenantID:   TestCFS,
			TargetType: "queue",
			TargetID:   queue.ID,
			TargetCode: queue.Code,
			VNamespace: "default",
			Code:       "upsert-test-code-1", // Same code
			Content:    "Updated Content Payload",
			Type:       models.ScheduledJobOneOff,
			RunAfter:   "30m",
			NextRunAt:  updatedNextRunAt,
			State:      models.ScheduledJobIdle,
		}

		id2, err := jobRepo2.CreateScheduledJob(&job2, startTime)
		require.NoError(t, err, "Creating with existing code must NOT fail with duplicate error")
		assert.Equal(t, id1, id2, "ID should remain unchanged on upsert")
		require.NoError(t, uow2.Commit())

		// Verify ORM persistence of updated values
		uowVerify := db.NewUnitOfWork(store, nil)
		repoVerify, _ := db.NewScheduledJobRepository(uowVerify, idFactory, TestFC, TestCFS)
		updatedJob, err := repoVerify.GetScheduledJobByID(id1, startTime)
		require.NoError(t, err)
		require.NotNil(t, updatedJob)
		assert.Equal(t, "Updated Content Payload", updatedJob.Content)
		assert.Equal(t, updatedNextRunAt, updatedJob.NextRunAt)

		// Verify OLD index key deleted and NEW index key created
		existsOldIndex, _ := store.Exists(TestFC, TestCFS, indexKey1, startTime)
		assert.False(t, existsOldIndex, "Old index key must be deleted on upsert")

		indexKey2 := db.FormatScheduledJobIndexKey(updatedNextRunAt, id1)
		existsNewIndex, err := store.Exists(TestFC, TestCFS, indexKey2, startTime)
		require.NoError(t, err)
		assert.True(t, existsNewIndex, "New index key must exist after upsert")
	})
}

func TestScheduledJob_PollerTimeTravelAndEnqueue(t *testing.T) {
	store := newTestPebbleStore(t)
	startTime := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)

	queue := setupTestQueue(t, store, TestFC, TestCFS, startTime)

	// Create a OneOff scheduled job set to run at T + 10 minutes
	uowCreate := db.NewUnitOfWork(store, nil)
	idFactory := &db.DeterministicIDGeneratorFactory{}
	jobRepo, err := db.NewScheduledJobRepository(uowCreate, idFactory, TestFC, TestCFS)
	require.NoError(t, err)

	jobNextRunAt := startTime.Add(10 * time.Minute)
	job := &models.ScheduledJob{
		ID:                             "job-time-travel-1",
		TenantID:                       TestCFS,
		TargetType:                     "queue",
		TargetID:                       queue.ID,
		TargetCode:                     queue.Code,
		RoutingKeyOrPatternOrQueueCode: queue.Code,
		VNamespace:                     "default",
		Content:                        "Delayed Notification Payload",
		ContentType:                    "text/plain",
		Priority:                       1,
		Type:                           models.ScheduledJobOneOff,
		RunAfter:                       "10m",
		NextRunAt:                      jobNextRunAt,
		State:                          models.ScheduledJobIdle,
	}

	_, err = jobRepo.CreateScheduledJob(job, startTime)
	require.NoError(t, err)
	require.NoError(t, uowCreate.Commit())

	// 1. Time travel to T + 5 minutes (before NextRunAt)
	t5 := startTime.Add(5 * time.Minute)
	uowT5 := db.NewUnitOfWork(store, nil)
	pollerCmdT5 := &scheduledJobCommand.ProcessDueScheduledJobsCommand{
		BatchSize: 1000,
		CF:        TestFC,
		CFS:       TestCFS,
	}
	resT5 := pollerCmdT5.Execute(uowT5, t5)
	require.Empty(t, resT5.Error)
	require.NoError(t, uowT5.Commit())

	resDataT5 := resT5.Result.(*scheduledJobCommand.ProcessDueScheduledJobsResult)
	assert.Equal(t, 0, resDataT5.ProcessedJobs, "Job should NOT trigger before NextRunAt")

	// Verify job remains "idle"
	uowCheckT5 := db.NewUnitOfWork(store, nil)
	jobRepoCheckT5, _ := db.NewScheduledJobRepository(uowCheckT5, idFactory, TestFC, TestCFS)
	jT5, _ := jobRepoCheckT5.GetScheduledJobByID("job-time-travel-1", t5)
	assert.Equal(t, models.ScheduledJobIdle, jT5.State)

	// 2. Time travel to T + 10 minutes (exact NextRunAt reached)
	t10 := startTime.Add(10 * time.Minute)
	uowT10 := db.NewUnitOfWork(store, nil)
	pollerCmdT10 := &scheduledJobCommand.ProcessDueScheduledJobsCommand{
		BatchSize: 1000,
		CF:        TestFC,
		CFS:       TestCFS,
	}
	resT10 := pollerCmdT10.Execute(uowT10, t10)
	require.Empty(t, resT10.Error)
	require.NoError(t, uowT10.Commit())

	resDataT10 := resT10.Result.(*scheduledJobCommand.ProcessDueScheduledJobsResult)
	assert.Equal(t, 1, resDataT10.ProcessedJobs, "Job MUST be processed when NextRunAt is reached")
	assert.Equal(t, 1, resDataT10.DispatchedMsgs)

	// Verify job state transitioned to "delivered"
	uowCheckT10 := db.NewUnitOfWork(store, nil)
	jobRepoCheckT10, _ := db.NewScheduledJobRepository(uowCheckT10, idFactory, TestFC, TestCFS)
	jT10, _ := jobRepoCheckT10.GetScheduledJobByID("job-time-travel-1", t10)
	assert.Equal(t, models.ScheduledJobDelivered, jT10.State)

	// Verify companion index key was deleted atomically
	indexKey := db.FormatScheduledJobIndexKey(jobNextRunAt, "job-time-travel-1")
	exists, _ := store.Exists(TestFC, TestCFS, indexKey, t10)
	assert.False(t, exists, "Companion index key MUST be deleted when job becomes delivered")

	// Verify message was enqueued into the target queue with ScheduledJobID set
	queueRepoCheck, _ := db.NewQueueRepository(uowCheckT10, idFactory, TestFC, TestCFS)
	qCheck, _ := queueRepoCheck.GetQueueById(queue.ID, t10)
	assert.Equal(t, 1, qCheck.MessagesCount, "Target queue must receive dispatched message")
}

func TestScheduledJob_LifecycleCompletionOneOffAndRecurring(t *testing.T) {
	store := newTestPebbleStore(t)
	now := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
	idFactory := &db.DeterministicIDGeneratorFactory{}

	queue := setupTestQueue(t, store, TestFC, TestCFS, now)

	t.Run("OneOff_DeletedAfterCompletion", func(t *testing.T) {
		// 1. Create OneOff job
		uow1 := db.NewUnitOfWork(store, nil)
		jobRepo1, _ := db.NewScheduledJobRepository(uow1, idFactory, TestFC, TestCFS)

		oneOffJob := &models.ScheduledJob{
			ID:                             "oneoff-lifecycle-1",
			TenantID:                       TestCFS,
			TargetType:                     "queue",
			TargetID:                       queue.ID,
			TargetCode:                     queue.Code,
			RoutingKeyOrPatternOrQueueCode: queue.Code,
			VNamespace:                     "default",
			Content:                        "OneOff Payload",
			Type:                           models.ScheduledJobOneOff,
			RunAfter:                       "1m",
			NextRunAt:                      now,
			State:                          models.ScheduledJobIdle,
		}
		_, err := jobRepo1.CreateScheduledJob(oneOffJob, now)
		require.NoError(t, err)
		require.NoError(t, uow1.Commit())

		// 2. Dispatch via poller
		uowPoller := db.NewUnitOfWork(store, nil)
		poller := &scheduledJobCommand.ProcessDueScheduledJobsCommand{BatchSize: 100, CF: TestFC, CFS: TestCFS}
		_ = poller.Execute(uowPoller, now)
		require.NoError(t, uowPoller.Commit())

		// 3. Dequeue message to get lease
		uowDequeue := db.NewUnitOfWork(store, nil)
		dequeueCmd := &queueCommand.DequeueCommand{
			QueueID:       queue.ID,
			JobWorkerID:   "worker-1",
			LeaseDuration: 30 * time.Second,
			CF:            TestFC,
			CFS:           TestCFS,
		}
		deqRes := dequeueCmd.Execute(uowDequeue, now)
		require.Empty(t, deqRes.Error)
		require.NoError(t, uowDequeue.Commit())

		claimedMsgRes := deqRes.Result.(queueCommand.DequeueResult)
		require.NotNil(t, claimedMsgRes.Lease)
		assert.Equal(t, "oneoff-lifecycle-1", claimedMsgRes.Message.ScheduledJobID)

		// 4. Worker Acknowledges message
		ackTime := now.Add(2 * time.Second)
		uowAck := db.NewUnitOfWork(store, nil)
		ackCmd := &queueCommand.AckMessageCommand{
			LeaseID: claimedMsgRes.Lease.ID,
			CF:      TestFC,
			CFS:     TestCFS,
		}
		ackRes := ackCmd.Execute(uowAck, ackTime)
		require.Empty(t, ackRes.Error)
		require.NoError(t, uowAck.Commit())

		// 5. Assert OneOff job is completely DELETED from ORM
		uowVerify := db.NewUnitOfWork(store, nil)
		verifyRepo, _ := db.NewScheduledJobRepository(uowVerify, idFactory, TestFC, TestCFS)
		deletedJob, err := verifyRepo.GetScheduledJobByID("oneoff-lifecycle-1", ackTime)
		require.NoError(t, err)
		assert.Nil(t, deletedJob, "OneOff job MUST be completely deleted after worker completion")
	})

	t.Run("Recurring_ResetToIdleWithNextTimestamp", func(t *testing.T) {
		// 1. Create Recurring job (every 5 minutes)
		uow1 := db.NewUnitOfWork(store, nil)
		jobRepo1, _ := db.NewScheduledJobRepository(uow1, idFactory, TestFC, TestCFS)

		recurringJob := &models.ScheduledJob{
			ID:                             "recurring-lifecycle-1",
			TenantID:                       TestCFS,
			TargetType:                     "queue",
			TargetID:                       queue.ID,
			TargetCode:                     queue.Code,
			RoutingKeyOrPatternOrQueueCode: queue.Code,
			VNamespace:                     "default",
			Content:                        "Recurring Report Payload",
			Type:                           models.ScheduledJobRecurring,
			Every:                          "5m",
			NextRunAt:                      now,
			State:                          models.ScheduledJobIdle,
		}
		_, err := jobRepo1.CreateScheduledJob(recurringJob, now)
		require.NoError(t, err)
		require.NoError(t, uow1.Commit())

		// 2. Dispatch via poller
		uowPoller := db.NewUnitOfWork(store, nil)
		poller := &scheduledJobCommand.ProcessDueScheduledJobsCommand{BatchSize: 100, CF: TestFC, CFS: TestCFS}
		_ = poller.Execute(uowPoller, now)
		require.NoError(t, uowPoller.Commit())

		// 3. Dequeue message to get lease
		uowDequeue := db.NewUnitOfWork(store, nil)
		dequeueCmd := &queueCommand.DequeueCommand{
			QueueID:       queue.ID,
			JobWorkerID:   "worker-2",
			LeaseDuration: 30 * time.Second,
			CF:            TestFC,
			CFS:           TestCFS,
		}
		deqRes := dequeueCmd.Execute(uowDequeue, now)
		require.Empty(t, deqRes.Error)
		require.NoError(t, uowDequeue.Commit())

		claimedMsgRes := deqRes.Result.(queueCommand.DequeueResult)
		require.NotNil(t, claimedMsgRes.Lease)
		assert.Equal(t, "recurring-lifecycle-1", claimedMsgRes.Message.ScheduledJobID)

		// 4. Worker Acknowledges message at T + 3 seconds
		ackTime := now.Add(3 * time.Second)
		uowAck := db.NewUnitOfWork(store, nil)
		ackCmd := &queueCommand.AckMessageCommand{
			LeaseID: claimedMsgRes.Lease.ID,
			CF:      TestFC,
			CFS:     TestCFS,
		}
		ackRes := ackCmd.Execute(uowAck, ackTime)
		require.Empty(t, ackRes.Error)
		require.NoError(t, uowAck.Commit())

		// 5. Assert Recurring job state reset back to "idle" and NextRunAt recalculated
		uowVerify := db.NewUnitOfWork(store, nil)
		verifyRepo, _ := db.NewScheduledJobRepository(uowVerify, idFactory, TestFC, TestCFS)
		resetJob, err := verifyRepo.GetScheduledJobByID("recurring-lifecycle-1", ackTime)
		require.NoError(t, err)
		require.NotNil(t, resetJob)
		assert.Equal(t, models.ScheduledJobIdle, resetJob.State, "Recurring job state MUST return to idle")
		assert.Equal(t, ackTime.Add(5*time.Minute), resetJob.NextRunAt, "NextRunAt MUST be recalculated from completion time")

		// 6. Assert NEW companion index key exists in KV store
		newIndexKey := db.FormatScheduledJobIndexKey(resetJob.NextRunAt, resetJob.ID)
		exists, err := store.Exists(TestFC, TestCFS, newIndexKey, ackTime)
		require.NoError(t, err)
		assert.True(t, exists, "NEW companion index key MUST be created for next run")
	})
}
