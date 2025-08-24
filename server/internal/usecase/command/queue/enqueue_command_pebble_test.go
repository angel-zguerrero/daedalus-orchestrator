package queue_test

import (
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
	EnqueueDefaultFC  = "default"
	EnqueueTestFC     = "test_fc"
	EnqueueTemporalFC = "temporal_fc"
	EnqueueTestCFS    = "test-sector"
)

// Helper function to create test Pebble store
func newTestPebbleStoreForEnqueue(t *testing.T, cfNames []string, ttlCfNames []string) db.KVStore {
	tempDir, err := os.MkdirTemp("", "enqueue_pebble_test_*")
	require.NoError(t, err)
	t.Logf("Creating Pebble store in: %s", tempDir)

	store, err := db.CreatePebbleStore(tempDir, cfNames, ttlCfNames)
	require.NoError(t, err)
	require.NotNil(t, store)

	t.Cleanup(func() {
		t.Logf("Closing and removing Pebble store from: %s", tempDir)
		store.Close()
		os.RemoveAll(tempDir)
	})
	return store
}

// Helper function to setup a test queue
func setupTestQueueForEnqueue(t *testing.T, store db.KVStore, cf, cfs string, now time.Time) *models.Queue {
	uow := db.NewUnitOfWork(store, nil)
	idFactory := &db.DeterministicIDGeneratorFactory{}

	queueRepo, err := db.NewQueueRepository(uow, idFactory, cf, cfs)
	require.NoError(t, err)

	queue := &models.Queue{
		ID:                        "test-queue-id-001",
		Code:                      "TEST_QUEUE",
		VNamespace:                "test-namespace",
		Name:                      "Test Queue",
		State:                     models.QueueActive,
		Type:                      models.StandardQueue,
		TTLQueue:                  3600,
		AllowDuplicated:           true,
		MaxAttempts:               3,
		MessagesCount:             0,
		DesiredPriorityThresholds: map[int]int{1: 100, 2: 50, 3: 30, 10: 10},
		PriorityThresholds:        map[int]int{1: 100, 2: 50, 3: 30, 10: 10},
		CreatedAt:                 now,
		UpdatedAt:                 now,
	}

	queueID, err := queueRepo.CreateQueue(queue, now)
	require.NoError(t, err)
	queue.ID = queueID

	err = uow.Commit()
	require.NoError(t, err)

	return queue
}

// Test with PebbleDB
func TestEnqueueCommand_Pebble(t *testing.T) {
	testCases := []struct {
		name string
		test func(t *testing.T)
	}{
		{"CreateNewPartitionWithFirstMessage", testCreateNewPartitionWithFirstMessage_Pebble},
		{"AddMessageToExistingPartition", testAddMessageToExistingPartition_Pebble},
		{"ValidatePriorityThresholds", testValidatePriorityThresholds_Pebble},
		{"MessageChaining", testMessageChaining_Pebble},
		{"InvalidPriority", testInvalidPriority_Pebble},
		{"QueueNotFound", testQueueNotFound_Pebble},
		{"InactiveQueue", testInactiveQueue_Pebble},
		{"MultipleMessagesOrdering", testMultipleMessagesOrdering_Pebble},
	}

	for _, tc := range testCases {
		t.Run(tc.name, tc.test)
	}
}

func testCreateNewPartitionWithFirstMessage_Pebble(t *testing.T) {
	store := newTestPebbleStoreForEnqueue(t, []string{EnqueueDefaultFC, EnqueueTestFC}, []string{EnqueueTemporalFC})
	now := time.Now()

	// Setup queue
	queue := setupTestQueueForEnqueue(t, store, EnqueueTestFC, EnqueueTestCFS, now)

	// Execute command
	uow := db.NewUnitOfWork(store, nil)
	cmd := &queueCommand.EnqueueCommand{
		Message:     models.QueueMessage{ID: "msg-001", MessageID: "msg1", Priority: 1},
		MessageCode: queue.Code,
		VNamespace:  queue.VNamespace,
		CF:          EnqueueTestFC,
		CFS:         EnqueueTestCFS,
	}
	result := cmd.Execute(uow, now)
	require.Empty(t, result.Error)
	err := uow.Commit()
	require.NoError(t, err)

	// Verify results
	verifyEnqueueResults_Pebble(t, store, EnqueueTestFC, EnqueueTestCFS, queue.ID, 1, 1, 1, now)
}

func testAddMessageToExistingPartition_Pebble(t *testing.T) {
	store := newTestPebbleStoreForEnqueue(t, []string{EnqueueDefaultFC, EnqueueTestFC}, []string{EnqueueTemporalFC})
	now := time.Now()

	// Setup queue
	queue := setupTestQueueForEnqueue(t, store, EnqueueTestFC, EnqueueTestCFS, now)

	// Add first message
	uow1 := db.NewUnitOfWork(store, nil)
	cmd1 := &queueCommand.EnqueueCommand{
		Message:     models.QueueMessage{ID: "msg-001", MessageID: "msg1", Priority: 1},
		MessageCode: queue.Code,
		VNamespace:  queue.VNamespace,
		CF:          EnqueueTestFC,
		CFS:         EnqueueTestCFS,
	}
	result1 := cmd1.Execute(uow1, now)
	require.Empty(t, result1.Error)
	err := uow1.Commit()
	require.NoError(t, err)

	// Verify first message
	verifyEnqueueResults_Pebble(t, store, EnqueueTestFC, EnqueueTestCFS, queue.ID, 1, 1, 1, now)

	// Add second message to same partition
	uow2 := db.NewUnitOfWork(store, nil)
	cmd2 := &queueCommand.EnqueueCommand{
		Message:     models.QueueMessage{ID: "msg-002", MessageID: "msg2", Priority: 1},
		MessageCode: queue.Code,
		VNamespace:  queue.VNamespace,
		CF:          EnqueueTestFC,
		CFS:         EnqueueTestCFS,
	}
	result2 := cmd2.Execute(uow2, now.Add(time.Second))
	require.Empty(t, result2.Error)
	err = uow2.Commit()
	require.NoError(t, err)

	// Verify results - 2 messages in partition, 2 in queue
	verifyEnqueueResults_Pebble(t, store, EnqueueTestFC, EnqueueTestCFS, queue.ID, 1, 2, 2, now)

	// Verify message chaining
	verifyMessageChaining_Pebble(t, store, EnqueueTestFC, EnqueueTestCFS, queue.ID, 1, []string{"msg1", "msg2"}, now)
}

func testValidatePriorityThresholds_Pebble(t *testing.T) {
	store := newTestPebbleStoreForEnqueue(t, []string{EnqueueDefaultFC, EnqueueTestFC}, []string{EnqueueTemporalFC})
	now := time.Now()

	// Setup queue with priority range 1-10
	queue := setupTestQueueForEnqueue(t, store, EnqueueTestFC, EnqueueTestCFS, now)

	// Test minimum priority
	uow1 := db.NewUnitOfWork(store, nil)
	cmd1 := &queueCommand.EnqueueCommand{
		Message:     models.QueueMessage{ID: "msg-min-001", MessageID: "msg_min", Priority: 1},
		MessageCode: queue.Code,
		VNamespace:  queue.VNamespace,
		CF:          EnqueueTestFC,
		CFS:         EnqueueTestCFS,
	}
	result1 := cmd1.Execute(uow1, now)
	require.Empty(t, result1.Error)
	err := uow1.Commit()
	require.NoError(t, err)

	// Test maximum priority
	uow2 := db.NewUnitOfWork(store, nil)
	cmd2 := &queueCommand.EnqueueCommand{
		Message:     models.QueueMessage{ID: "msg-max-001", MessageID: "msg_max", Priority: 10},
		MessageCode: queue.Code,
		VNamespace:  queue.VNamespace,
		CF:          EnqueueTestFC,
		CFS:         EnqueueTestCFS,
	}
	result2 := cmd2.Execute(uow2, now.Add(time.Second))
	require.Empty(t, result2.Error)
	err = uow2.Commit()
	require.NoError(t, err)

	// Verify both partitions exist
	verifyEnqueueResults_Pebble(t, store, EnqueueTestFC, EnqueueTestCFS, queue.ID, 1, 1, 2, now)
	verifyEnqueueResults_Pebble(t, store, EnqueueTestFC, EnqueueTestCFS, queue.ID, 10, 1, 2, now)
}

func testMessageChaining_Pebble(t *testing.T) {
	store := newTestPebbleStoreForEnqueue(t, []string{EnqueueDefaultFC, EnqueueTestFC}, []string{EnqueueTemporalFC})
	now := time.Now()

	// Setup queue
	queue := setupTestQueueForEnqueue(t, store, EnqueueTestFC, EnqueueTestCFS, now)

	// Add three messages to same partition
	messageIDs := []string{"msg1", "msg2", "msg3"}
	messageDBIDs := []string{"msg-chain-001", "msg-chain-002", "msg-chain-003"}
	for i, msgID := range messageIDs {
		uow := db.NewUnitOfWork(store, nil)
		cmd := &queueCommand.EnqueueCommand{
			Message:     models.QueueMessage{ID: messageDBIDs[i], MessageID: msgID, Priority: 1},
			MessageCode: queue.Code,
			VNamespace:  queue.VNamespace,
			CF:          EnqueueTestFC,
			CFS:         EnqueueTestCFS,
		}
		result := cmd.Execute(uow, now.Add(time.Duration(i)*time.Second))
		require.Empty(t, result.Error)
		err := uow.Commit()
		require.NoError(t, err)
	}

	// Verify final state
	verifyEnqueueResults_Pebble(t, store, EnqueueTestFC, EnqueueTestCFS, queue.ID, 1, 3, 3, now)

	// Verify message chaining order
	verifyMessageChaining_Pebble(t, store, EnqueueTestFC, EnqueueTestCFS, queue.ID, 1, messageIDs, now)
}

func testInvalidPriority_Pebble(t *testing.T) {
	store := newTestPebbleStoreForEnqueue(t, []string{EnqueueDefaultFC, EnqueueTestFC}, []string{EnqueueTemporalFC})
	now := time.Now()

	// Setup queue with priority range 1-10
	queue := setupTestQueueForEnqueue(t, store, EnqueueTestFC, EnqueueTestCFS, now)

	// Test priority too low
	uow1 := db.NewUnitOfWork(store, nil)
	cmd1 := &queueCommand.EnqueueCommand{
		Message:     models.QueueMessage{ID: "msg-low-001", MessageID: "msg_low", Priority: 0},
		MessageCode: queue.Code,
		VNamespace:  queue.VNamespace,
		CF:          EnqueueTestFC,
		CFS:         EnqueueTestCFS,
	}
	result1 := cmd1.Execute(uow1, now)
	assert.NotEmpty(t, result1.Error)
	assert.Contains(t, result1.Error, "Priority")

	// Test priority too high
	uow2 := db.NewUnitOfWork(store, nil)
	cmd2 := &queueCommand.EnqueueCommand{
		Message:     models.QueueMessage{ID: "msg-high-001", MessageID: "msg_high", Priority: 11},
		MessageCode: queue.Code,
		VNamespace:  queue.VNamespace,
		CF:          EnqueueTestFC,
		CFS:         EnqueueTestCFS,
	}
	result2 := cmd2.Execute(uow2, now)
	assert.NotEmpty(t, result2.Error)
	assert.Contains(t, result2.Error, "Priority")
}

func testQueueNotFound_Pebble(t *testing.T) {
	store := newTestPebbleStoreForEnqueue(t, []string{EnqueueDefaultFC, EnqueueTestFC}, []string{EnqueueTemporalFC})
	now := time.Now()

	// Try to enqueue to non-existent queue
	uow := db.NewUnitOfWork(store, nil)
	cmd := &queueCommand.EnqueueCommand{
		Message:     models.QueueMessage{ID: "msg-notfound-001", MessageID: "msg1", Priority: 1},
		MessageCode: "NON_EXISTENT",
		VNamespace:  "test-namespace",
		CF:          EnqueueTestFC,
		CFS:         EnqueueTestCFS,
	}
	result := cmd.Execute(uow, now)
	assert.NotEmpty(t, result.Error)
	assert.Contains(t, result.Error, "Queue")
}

func testInactiveQueue_Pebble(t *testing.T) {
	store := newTestPebbleStoreForEnqueue(t, []string{EnqueueDefaultFC, EnqueueTestFC}, []string{EnqueueTemporalFC})
	now := time.Now()

	// Setup inactive queue
	uow := db.NewUnitOfWork(store, nil)
	idFactory := &db.DeterministicIDGeneratorFactory{}

	queueRepo, err := db.NewQueueRepository(uow, idFactory, EnqueueTestFC, EnqueueTestCFS)
	require.NoError(t, err)

	queue := &models.Queue{
		ID:                        "test-queue-id-inactive-001",
		Code:                      "INACTIVE_QUEUE",
		VNamespace:                "test-namespace",
		Name:                      "Inactive Queue",
		State:                     models.QueueStopped, // Inactive
		Type:                      models.StandardQueue,
		TTLQueue:                  3600,
		AllowDuplicated:           true,
		MaxAttempts:               3,
		MessagesCount:             0,
		DesiredPriorityThresholds: map[int]int{1: 100, 2: 50, 3: 30, 10: 10},
		PriorityThresholds:        map[int]int{1: 100, 2: 50, 3: 30, 10: 10},
		CreatedAt:                 now,
		UpdatedAt:                 now,
	}

	queueID, err := queueRepo.CreateQueue(queue, now)
	require.NoError(t, err)
	queue.ID = queueID
	err = uow.Commit()
	require.NoError(t, err)

	// Try to enqueue to inactive queue
	uow2 := db.NewUnitOfWork(store, nil)
	cmd := &queueCommand.EnqueueCommand{
		Message:     models.QueueMessage{ID: "msg-inactive-001", MessageID: "msg1", Priority: 1},
		MessageCode: queue.Code,
		VNamespace:  queue.VNamespace,
		CF:          EnqueueTestFC,
		CFS:         EnqueueTestCFS,
	}
	result := cmd.Execute(uow2, now)
	assert.NotEmpty(t, result.Error)
	assert.Contains(t, result.Error, "active")
}

func testMultipleMessagesOrdering_Pebble(t *testing.T) {
	store := newTestPebbleStoreForEnqueue(t, []string{EnqueueDefaultFC, EnqueueTestFC}, []string{EnqueueTemporalFC})
	now := time.Now()

	// Setup queue
	queue := setupTestQueueForEnqueue(t, store, EnqueueTestFC, EnqueueTestCFS, now)

	// Add messages to different partitions
	messages := []struct {
		id       string
		dbID     string
		priority int
	}{
		{"msg1_p1", "msg-multi-001", 1},
		{"msg2_p1", "msg-multi-002", 1},
		{"msg1_p2", "msg-multi-003", 2},
		{"msg2_p2", "msg-multi-004", 2},
		{"msg1_p3", "msg-multi-005", 3},
	}

	for i, msg := range messages {
		uow := db.NewUnitOfWork(store, nil)
		cmd := &queueCommand.EnqueueCommand{
			Message:     models.QueueMessage{ID: msg.dbID, MessageID: msg.id, Priority: msg.priority},
			MessageCode: queue.Code,
			VNamespace:  queue.VNamespace,
			CF:          EnqueueTestFC,
			CFS:         EnqueueTestCFS,
		}
		result := cmd.Execute(uow, now.Add(time.Duration(i)*time.Second))
		require.Empty(t, result.Error)
		err := uow.Commit()
		require.NoError(t, err)
	}

	// Verify partitions
	verifyEnqueueResults_Pebble(t, store, EnqueueTestFC, EnqueueTestCFS, queue.ID, 1, 2, 5, now)
	verifyEnqueueResults_Pebble(t, store, EnqueueTestFC, EnqueueTestCFS, queue.ID, 2, 2, 5, now)
	verifyEnqueueResults_Pebble(t, store, EnqueueTestFC, EnqueueTestCFS, queue.ID, 3, 1, 5, now)

	// Verify message chaining for each partition
	verifyMessageChaining_Pebble(t, store, EnqueueTestFC, EnqueueTestCFS, queue.ID, 1, []string{"msg1_p1", "msg2_p1"}, now)
	verifyMessageChaining_Pebble(t, store, EnqueueTestFC, EnqueueTestCFS, queue.ID, 2, []string{"msg1_p2", "msg2_p2"}, now)
	verifyMessageChaining_Pebble(t, store, EnqueueTestFC, EnqueueTestCFS, queue.ID, 3, []string{"msg1_p3"}, now)
}

// Verification functions for Pebble
func verifyEnqueueResults_Pebble(t *testing.T, store db.KVStore, cf, cfs, queueID string, priority, expectedPartitionCount, expectedQueueCount int, now time.Time) {
	uow := db.NewUnitOfWork(store, nil)
	idFactory := &db.DeterministicIDGeneratorFactory{}

	// Verify queue message count
	queueRepo, err := db.NewQueueRepository(uow, idFactory, cf, cfs)
	require.NoError(t, err)

	queue, err := queueRepo.GetQueueById(queueID, now)
	require.NoError(t, err)
	assert.Equal(t, expectedQueueCount, queue.MessagesCount)

	// Verify partition message count
	partitionRepo, err := db.NewQueuePartitionRepository(uow, idFactory, cf, cfs)
	require.NoError(t, err)

	partition, err := partitionRepo.GetQueuePartitionByQueueIDAndPriority(queueID, priority, now)
	require.NoError(t, err)
	require.NotNil(t, partition)
	assert.Equal(t, expectedPartitionCount, partition.MessagesCount)
}

func verifyMessageChaining_Pebble(t *testing.T, store db.KVStore, cf, cfs, queueID string, priority int, expectedMessageIDs []string, now time.Time) {
	uow := db.NewUnitOfWork(store, nil)
	idFactory := &db.DeterministicIDGeneratorFactory{}

	partitionRepo, err := db.NewQueuePartitionRepository(uow, idFactory, cf, cfs)
	require.NoError(t, err)

	messageRepo, err := db.NewQueueMessageRepository(uow, idFactory, cf, cfs)
	require.NoError(t, err)

	// Get partition
	partition, err := partitionRepo.GetQueuePartitionByQueueIDAndPriority(queueID, priority, now)
	require.NoError(t, err)
	require.NotNil(t, partition, "Partition should exist for queueID=%s, priority=%d", queueID, priority)

	// Traverse message chain
	var actualMessageIDs []string
	currentMessageID := partition.FirstQueueMessageID

	for currentMessageID != "" {
		message, err := messageRepo.GetQueueMessageById(currentMessageID, now)
		require.NoError(t, err)
		require.NotNil(t, message, "Message should exist for ID=%s", currentMessageID)

		actualMessageIDs = append(actualMessageIDs, message.MessageID)
		currentMessageID = message.NextQueueMessageID
	}

	// Verify order
	assert.Equal(t, expectedMessageIDs, actualMessageIDs, "Message chain order should match")
}
