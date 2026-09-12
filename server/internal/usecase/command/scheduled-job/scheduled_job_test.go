package scheduled_job_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/pkg/utils"
	bindingCommand "deadalus-orch/server/internal/usecase/command/binding"
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

func TestScheduledJob_ExchangeFanOutScenario(t *testing.T) {
	store := newTestPebbleStore(t)
	now := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
	idFactory := &db.DeterministicIDGeneratorFactory{}

	// 0. Create default VNamespace
	uowVNS := db.NewUnitOfWork(store, nil)
	vnsRepo, err := db.NewVNamespaceRepository(uowVNS, idFactory, TestFC, TestCFS)
	require.NoError(t, err)
	_, err = vnsRepo.CreateVNamespace(&models.VNamespace{
		ID:   "vns-default-id",
		Name: "default",
	}, now)
	require.NoError(t, err)
	require.NoError(t, uowVNS.Commit())

	// 1. Create Exchange
	uowEx := db.NewUnitOfWork(store, nil)
	exRepo, err := db.NewExchangeRepository(uowEx, idFactory, TestFC, TestCFS)
	require.NoError(t, err)

	exchange := &models.Exchange{
		ID:         "exchange-fanout-1",
		Code:       "events-exchange",
		Name:       "Events Exchange",
		Type:       models.Fanout,
		VNamespace: "default",
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	exID, err := exRepo.CreateExchange(exchange, now)
	require.NoError(t, err)
	exchange.ID = exID
	require.NoError(t, uowEx.Commit())

	// 2. Create 2 Queues: Queue A & Queue B
	setupQueue := func(id, code, name string) *models.Queue {
		uow := db.NewUnitOfWork(store, nil)
		qRepo, _ := db.NewQueueRepository(uow, idFactory, TestFC, TestCFS)
		q := &models.Queue{
			ID:                        id,
			Code:                      code,
			Name:                      name,
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
		qID, _ := qRepo.CreateQueue(q, now)
		q.ID = qID
		_ = uow.Commit()
		return q
	}

	queueA := setupQueue("queue-fanout-a", "queue-a", "Queue A")
	queueB := setupQueue("queue-fanout-b", "queue-b", "Queue B")

	// 3. Bind Queue A and Queue B to Exchange
	bindQueue := func(bindingID, bindingCode, queueCode string) {
		uow := db.NewUnitOfWork(store, nil)
		cmd := &bindingCommand.AssertBindingCommand{
			NewBindingID: bindingID,
			Code:         bindingCode,
			QueueCode:    queueCode,
			ExchangeCode: exchange.Code,
			VNamespace:   "default",
			BindingType:  models.BindingTypeClassic,
			CF:           TestFC,
			CFS:          TestCFS,
		}
		res := cmd.Execute(uow, now)
		require.Empty(t, res.Error)
		require.NoError(t, uow.Commit())
	}

	bindQueue("binding-a", "BIND_A", queueA.Code)
	bindQueue("binding-b", "BIND_B", queueB.Code)

	t.Run("OneOff_FanOut_RequiresAllQueuesAckBeforeDeletion", func(t *testing.T) {
		// Create OneOff scheduled job targeting the fan-out exchange
		uowCreate := db.NewUnitOfWork(store, nil)
		jobRepo, _ := db.NewScheduledJobRepository(uowCreate, idFactory, TestFC, TestCFS)

		oneOffJob := &models.ScheduledJob{
			ID:         "oneoff-fanout-job-1",
			TenantID:   TestCFS,
			TargetType: "exchange",
			TargetID:   exchange.ID,
			TargetCode: exchange.Code,
			VNamespace: "default",
			Content:    "FanOut Event Payload",
			Type:       models.ScheduledJobOneOff,
			RunAfter:   "1m",
			NextRunAt:  now,
			State:      models.ScheduledJobIdle,
		}
		_, err := jobRepo.CreateScheduledJob(oneOffJob, now)
		require.NoError(t, err)
		require.NoError(t, uowCreate.Commit())

		// Dispatch via poller
		uowPoller := db.NewUnitOfWork(store, nil)
		poller := &scheduledJobCommand.ProcessDueScheduledJobsCommand{BatchSize: 100, CF: TestFC, CFS: TestCFS}
		resPoller := poller.Execute(uowPoller, now)
		require.Empty(t, resPoller.Error)
		require.NoError(t, uowPoller.Commit())

		pRes := resPoller.Result.(*scheduledJobCommand.ProcessDueScheduledJobsResult)
		assert.Equal(t, 1, pRes.ProcessedJobs)
		assert.Equal(t, 2, pRes.DispatchedMsgs, "Should dispatch messages to both Queue A and Queue B")

		// Verify ScheduledJob state transitioned to delivered
		uowCheck := db.NewUnitOfWork(store, nil)
		jobRepoCheck, _ := db.NewScheduledJobRepository(uowCheck, idFactory, TestFC, TestCFS)
		jobCheck, err := jobRepoCheck.GetScheduledJobByID("oneoff-fanout-job-1", now)
		require.NoError(t, err)
		require.NotNil(t, jobCheck)
		assert.Equal(t, models.ScheduledJobDelivered, jobCheck.State)

		// Verify 2 trackers exist in DB
		expectedExecutionID := fmt.Sprintf("exec_%s_%d", oneOffJob.ID, oneOffJob.NextRunAt.Unix())
		trackerRepo, _ := db.NewScheduledJobTrackerRepository(uowCheck, idFactory, TestFC, TestCFS)
		trackers, err := trackerRepo.GetTrackersByExecution(expectedExecutionID, now)
		require.NoError(t, err)
		assert.Len(t, trackers, 2)

		// Worker A dequeues from Queue A and ACKs message
		uowDeqA := db.NewUnitOfWork(store, nil)
		deqCmdA := &queueCommand.DequeueCommand{
			QueueID:       queueA.ID,
			JobWorkerID:   "worker-a",
			LeaseDuration: 30 * time.Second,
			CF:            TestFC,
			CFS:           TestCFS,
		}
		deqResA := deqCmdA.Execute(uowDeqA, now)
		require.Empty(t, deqResA.Error)
		require.NoError(t, uowDeqA.Commit())

		claimedMsgA := deqResA.Result.(queueCommand.DequeueResult)
		require.NotNil(t, claimedMsgA.Lease)
		assert.Equal(t, "oneoff-fanout-job-1", claimedMsgA.Message.ScheduledJobID)

		ackTime1 := now.Add(1 * time.Second)
		uowAckA := db.NewUnitOfWork(store, nil)
		ackCmdA := &queueCommand.AckMessageCommand{
			LeaseID: claimedMsgA.Lease.ID,
			CF:      TestFC,
			CFS:     TestCFS,
		}
		ackResA := ackCmdA.Execute(uowAckA, ackTime1)
		require.Empty(t, ackResA.Error)
		require.NoError(t, uowAckA.Commit())

		// Job must STILL exist because Queue B has not ACKed yet!
		uowMidCheck := db.NewUnitOfWork(store, nil)
		jobRepoMid, _ := db.NewScheduledJobRepository(uowMidCheck, idFactory, TestFC, TestCFS)
		jobMid, err := jobRepoMid.GetScheduledJobByID("oneoff-fanout-job-1", ackTime1)
		require.NoError(t, err)
		require.NotNil(t, jobMid, "Job must NOT be deleted while Queue B is still pending ACK")
		assert.Equal(t, models.ScheduledJobDelivered, jobMid.State)

		// Worker B dequeues from Queue B and ACKs message
		uowDeqB := db.NewUnitOfWork(store, nil)
		deqCmdB := &queueCommand.DequeueCommand{
			QueueID:       queueB.ID,
			JobWorkerID:   "worker-b",
			LeaseDuration: 30 * time.Second,
			CF:            TestFC,
			CFS:           TestCFS,
		}
		deqResB := deqCmdB.Execute(uowDeqB, ackTime1)
		require.Empty(t, deqResB.Error)
		require.NoError(t, uowDeqB.Commit())

		claimedMsgB := deqResB.Result.(queueCommand.DequeueResult)
		require.NotNil(t, claimedMsgB.Lease)

		ackTime2 := ackTime1.Add(1 * time.Second)
		uowAckB := db.NewUnitOfWork(store, nil)
		ackCmdB := &queueCommand.AckMessageCommand{
			LeaseID: claimedMsgB.Lease.ID,
			CF:      TestFC,
			CFS:     TestCFS,
		}
		ackResB := ackCmdB.Execute(uowAckB, ackTime2)
		require.Empty(t, ackResB.Error)
		require.NoError(t, uowAckB.Commit())

		// Now that ALL queues ACKed, OneOff job MUST be completely DELETED
		uowFinal := db.NewUnitOfWork(store, nil)
		jobRepoFinal, _ := db.NewScheduledJobRepository(uowFinal, idFactory, TestFC, TestCFS)
		jobFinal, err := jobRepoFinal.GetScheduledJobByID("oneoff-fanout-job-1", ackTime2)
		require.NoError(t, err)
		assert.Nil(t, jobFinal, "OneOff fan-out job MUST be deleted after ALL target queues have ACKed")

		// Trackers should also be cleaned up
		trackerRepoFinal, _ := db.NewScheduledJobTrackerRepository(uowFinal, idFactory, TestFC, TestCFS)
		trackersFinal, err := trackerRepoFinal.GetTrackersByExecution(expectedExecutionID, ackTime2)
		require.NoError(t, err)
		assert.Empty(t, trackersFinal, "Trackers MUST be cleaned up after completion")
	})

	t.Run("Recurring_FanOut_ResetsToIdleOnlyAfterAllQueuesAck", func(t *testing.T) {
		uowCreate := db.NewUnitOfWork(store, nil)
		jobRepo, _ := db.NewScheduledJobRepository(uowCreate, idFactory, TestFC, TestCFS)

		recurringJob := &models.ScheduledJob{
			ID:         "recurring-fanout-job-1",
			TenantID:   TestCFS,
			TargetType: "exchange",
			TargetID:   exchange.ID,
			TargetCode: exchange.Code,
			VNamespace: "default",
			Content:    "Recurring FanOut Event Payload",
			Type:       models.ScheduledJobRecurring,
			Every:      "5m",
			NextRunAt:  now,
			State:      models.ScheduledJobIdle,
		}
		_, err := jobRepo.CreateScheduledJob(recurringJob, now)
		require.NoError(t, err)
		require.NoError(t, uowCreate.Commit())

		// Dispatch via poller
		uowPoller := db.NewUnitOfWork(store, nil)
		poller := &scheduledJobCommand.ProcessDueScheduledJobsCommand{BatchSize: 100, CF: TestFC, CFS: TestCFS}
		resPoller := poller.Execute(uowPoller, now)
		require.Empty(t, resPoller.Error)
		require.NoError(t, uowPoller.Commit())

		// ACK Queue A
		uowDeqA := db.NewUnitOfWork(store, nil)
		deqCmdA := &queueCommand.DequeueCommand{QueueID: queueA.ID, JobWorkerID: "w-a", LeaseDuration: 30 * time.Second, CF: TestFC, CFS: TestCFS}
		deqResA := deqCmdA.Execute(uowDeqA, now)
		claimedMsgA := deqResA.Result.(queueCommand.DequeueResult)
		_ = uowDeqA.Commit()

		ackTime1 := now.Add(1 * time.Second)
		uowAckA := db.NewUnitOfWork(store, nil)
		ackCmdA := &queueCommand.AckMessageCommand{LeaseID: claimedMsgA.Lease.ID, CF: TestFC, CFS: TestCFS}
		_ = ackCmdA.Execute(uowAckA, ackTime1)
		_ = uowAckA.Commit()

		// Verify state is still delivered (not reset to idle yet)
		uowMidCheck := db.NewUnitOfWork(store, nil)
		jobRepoMid, _ := db.NewScheduledJobRepository(uowMidCheck, idFactory, TestFC, TestCFS)
		jobMid, _ := jobRepoMid.GetScheduledJobByID("recurring-fanout-job-1", ackTime1)
		assert.Equal(t, models.ScheduledJobDelivered, jobMid.State)

		// ACK Queue B
		uowDeqB := db.NewUnitOfWork(store, nil)
		deqCmdB := &queueCommand.DequeueCommand{QueueID: queueB.ID, JobWorkerID: "w-b", LeaseDuration: 30 * time.Second, CF: TestFC, CFS: TestCFS}
		deqResB := deqCmdB.Execute(uowDeqB, ackTime1)
		claimedMsgB := deqResB.Result.(queueCommand.DequeueResult)
		_ = uowDeqB.Commit()

		ackTime2 := ackTime1.Add(1 * time.Second)
		uowAckB := db.NewUnitOfWork(store, nil)
		ackCmdB := &queueCommand.AckMessageCommand{LeaseID: claimedMsgB.Lease.ID, CF: TestFC, CFS: TestCFS}
		_ = ackCmdB.Execute(uowAckB, ackTime2)
		_ = uowAckB.Commit()

		// Verify state reset to idle and NextRunAt recalculated
		uowFinal := db.NewUnitOfWork(store, nil)
		jobRepoFinal, _ := db.NewScheduledJobRepository(uowFinal, idFactory, TestFC, TestCFS)
		jobFinal, err := jobRepoFinal.GetScheduledJobByID("recurring-fanout-job-1", ackTime2)
		require.NoError(t, err)
		require.NotNil(t, jobFinal)
		assert.Equal(t, models.ScheduledJobIdle, jobFinal.State, "Recurring fan-out job MUST return to idle after ALL queues ACK")
		assert.Equal(t, ackTime2.Add(5*time.Minute), jobFinal.NextRunAt)
	})
}

