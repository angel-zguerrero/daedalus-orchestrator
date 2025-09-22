//go:build rocksdb
// +build rocksdb

package queue_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/linxGnu/grocksdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"deadalus-orch/server/internal/infrastructure/db"
	queueCommand "deadalus-orch/server/internal/usecase/command/queue"
	"deadalus-orch/shared/models"
)

const (
	EnqueueRocksDefaultFC  = "default"
	EnqueueRocksTestFC     = "test_fc"
	EnqueueRocksTemporalFC = "temporal_fc"
	EnqueueRocksTestCFS    = "test-sector"
)

// Helper function to create test RocksDB store
func newTestRocksDBStoreForEnqueue(t *testing.T) *db.RocksdbStore {
	tmpDir := t.TempDir()
	opts := grocksdb.NewDefaultOptions()
	opts.SetCreateIfMissing(true)
	opts.SetCreateIfMissingColumnFamilies(true)
	goOp := grocksdb.NewDefaultOptions()

	rocks, cfHs, err := grocksdb.OpenDbColumnFamilies(opts, tmpDir, []string{EnqueueRocksDefaultFC, EnqueueRocksTestFC, EnqueueRocksTemporalFC}, []*grocksdb.Options{goOp, goOp, goOp})
	require.NoError(t, err)
	t.Cleanup(func() { rocks.Close() })

	columnFamilyNames, err := grocksdb.ListColumnFamilies(opts, tmpDir)
	require.NoError(t, err)

	cfMap := make(map[string]*grocksdb.ColumnFamilyHandle, len(columnFamilyNames)-1)
	for i, name := range columnFamilyNames {
		if name != EnqueueRocksTemporalFC {
			cfMap[name] = cfHs[i]
		}
	}

	ttlCFMap := make(map[string]*grocksdb.ColumnFamilyHandle, len(columnFamilyNames)-2)
	for i, name := range columnFamilyNames {
		if name == EnqueueRocksTemporalFC {
			ttlCFMap[name] = cfHs[i]
		}
	}

	return &db.RocksdbStore{
		DB:                     rocks,
		ColumnFamilyHandles:    cfMap,
		TTLColumnFamilyHandles: ttlCFMap,
	}
}

// Helper function to setup a test queue for RocksDB
func setupTestQueueForEnqueueRocksDB(t *testing.T, store db.KVStore, cf, cfs string, now time.Time) *models.Queue {
	uow := db.NewUnitOfWork(store, nil)
	idFactory := &db.DeterministicIDGeneratorFactory{}

	queueRepo, err := db.NewQueueRepository(uow, idFactory, cf, cfs)
	require.NoError(t, err)

	queue := &models.Queue{
		ID:                        "test-queue-id-rocks-001",
		Code:                      "TEST_QUEUE_ROCKS",
		Name:                      "Test Queue RocksDB",
		VNamespace:                "test-namespace",
		State:                     models.QueueActive,
		Type:                      models.StandardQueue,
		DefaultQueueMessageTTL:                  3600,
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

// Test with RocksDB
func TestEnqueueCommand_RocksDB(t *testing.T) {
	testCases := []struct {
		name string
		test func(t *testing.T)
	}{
		{"CreateNewPartitionWithFirstMessage", testCreateNewPartitionWithFirstMessage_RocksDB},
		{"AddMessageToExistingPartition", testAddMessageToExistingPartition_RocksDB},
		{"ValidatePriorityThresholds", testValidatePriorityThresholds_RocksDB},
		{"MessageChaining", testMessageChaining_RocksDB},
		{"InvalidPriority", testInvalidPriority_RocksDB},
		{"QueueNotFound", testQueueNotFound_RocksDB},
		{"InactiveQueue", testInactiveQueue_RocksDB},
		{"MultipleMessagesOrdering", testMultipleMessagesOrdering_RocksDB},
		{"EmptyMessagesArray", testEmptyMessagesArray_RocksDB},
		{"BulkMessageEnqueue", testBulkMessageEnqueue_RocksDB},
		{"MultiQueueSupport", testMultiQueueSupport_RocksDB},
	}

	for _, tc := range testCases {
		t.Run(tc.name, tc.test)
	}
}

func testCreateNewPartitionWithFirstMessage_RocksDB(t *testing.T) {
	store := newTestRocksDBStoreForEnqueue(t)
	now := time.Now()

	// Setup queue
	queue := setupTestQueueForEnqueueRocksDB(t, store, EnqueueRocksTestFC, EnqueueRocksTestCFS, now)

	// Execute command
	uow := db.NewUnitOfWork(store, nil)
	cmd := &queueCommand.EnqueueCommand{
		Messages: []models.QueueMessage{{ID: "msg-001", MessageID: "msg1", Priority: 1, QueueID: queue.ID}},
		CF:       EnqueueRocksTestFC,
		CFS:      EnqueueRocksTestCFS,
	}
	result := cmd.Execute(uow, now)
	require.Empty(t, result.Error)
	err := uow.Commit()
	require.NoError(t, err)

	// Verify results
	verifyEnqueueResults_RocksDB(t, store, EnqueueRocksTestFC, EnqueueRocksTestCFS, queue.ID, 1, 1, 1, now)
}

func testAddMessageToExistingPartition_RocksDB(t *testing.T) {
	store := newTestRocksDBStoreForEnqueue(t)
	now := time.Now()

	// Setup queue
	queue := setupTestQueueForEnqueueRocksDB(t, store, EnqueueRocksTestFC, EnqueueRocksTestCFS, now)

	// Add first message
	uow1 := db.NewUnitOfWork(store, nil)
	cmd1 := &queueCommand.EnqueueCommand{
		Messages: []models.QueueMessage{{ID: "msg-001", MessageID: "msg1", Priority: 1, QueueID: queue.ID}},
		CF:       EnqueueRocksTestFC,
		CFS:      EnqueueRocksTestCFS,
	}
	result1 := cmd1.Execute(uow1, now)
	require.Empty(t, result1.Error)
	err := uow1.Commit()
	require.NoError(t, err)

	// Verify first message
	verifyEnqueueResults_RocksDB(t, store, EnqueueRocksTestFC, EnqueueRocksTestCFS, queue.ID, 1, 1, 1, now)

	// Add second message to same partition
	uow2 := db.NewUnitOfWork(store, nil)
	cmd2 := &queueCommand.EnqueueCommand{
		Messages: []models.QueueMessage{{ID: "msg-002", MessageID: "msg2", Priority: 1, QueueID: queue.ID}},
		CF:       EnqueueRocksTestFC,
		CFS:      EnqueueRocksTestCFS,
	}
	result2 := cmd2.Execute(uow2, now.Add(time.Second))
	require.Empty(t, result2.Error)
	err = uow2.Commit()
	require.NoError(t, err)

	// Verify results - 2 messages in partition, 2 in queue
	verifyEnqueueResults_RocksDB(t, store, EnqueueRocksTestFC, EnqueueRocksTestCFS, queue.ID, 1, 2, 2, now)

	// Verify message chaining
	verifyMessageChaining_RocksDB(t, store, EnqueueRocksTestFC, EnqueueRocksTestCFS, queue.ID, 1, []string{"msg1", "msg2"}, now)
}

func testValidatePriorityThresholds_RocksDB(t *testing.T) {
	store := newTestRocksDBStoreForEnqueue(t)
	now := time.Now()

	// Setup queue with priority range 1-10
	queue := setupTestQueueForEnqueueRocksDB(t, store, EnqueueRocksTestFC, EnqueueRocksTestCFS, now)

	// Test minimum priority
	uow1 := db.NewUnitOfWork(store, nil)
	cmd1 := &queueCommand.EnqueueCommand{
		Messages: []models.QueueMessage{{ID: "msg-min-001", MessageID: "msg_min", Priority: 1, QueueID: queue.ID}},
		CF:       EnqueueRocksTestFC,
		CFS:      EnqueueRocksTestCFS,
	}
	result1 := cmd1.Execute(uow1, now)
	require.Empty(t, result1.Error)
	err := uow1.Commit()
	require.NoError(t, err)

	// Test maximum priority
	uow2 := db.NewUnitOfWork(store, nil)
	cmd2 := &queueCommand.EnqueueCommand{
		Messages: []models.QueueMessage{{ID: "msg-max-001", MessageID: "msg_max", Priority: 10, QueueID: queue.ID}},
		CF:       EnqueueRocksTestFC,
		CFS:      EnqueueRocksTestCFS,
	}
	result2 := cmd2.Execute(uow2, now.Add(time.Second))
	require.Empty(t, result2.Error)
	err = uow2.Commit()
	require.NoError(t, err)

	// Verify both partitions exist
	verifyEnqueueResults_RocksDB(t, store, EnqueueRocksTestFC, EnqueueRocksTestCFS, queue.ID, 1, 1, 2, now)
	verifyEnqueueResults_RocksDB(t, store, EnqueueRocksTestFC, EnqueueRocksTestCFS, queue.ID, 10, 1, 2, now)
}

func testMessageChaining_RocksDB(t *testing.T) {
	store := newTestRocksDBStoreForEnqueue(t)
	now := time.Now()

	// Setup queue
	queue := setupTestQueueForEnqueueRocksDB(t, store, EnqueueRocksTestFC, EnqueueRocksTestCFS, now)

	// Add three messages to same partition in a single command
	uow := db.NewUnitOfWork(store, nil)
	cmd := &queueCommand.EnqueueCommand{
		Messages: []models.QueueMessage{
			{ID: "msg-chain-001", MessageID: "msg1", Priority: 1, QueueID: queue.ID},
			{ID: "msg-chain-002", MessageID: "msg2", Priority: 1, QueueID: queue.ID},
			{ID: "msg-chain-003", MessageID: "msg3", Priority: 1, QueueID: queue.ID},
		},
		CF:  EnqueueRocksTestFC,
		CFS: EnqueueRocksTestCFS,
	}
	result := cmd.Execute(uow, now)
	require.Empty(t, result.Error)
	err := uow.Commit()
	require.NoError(t, err)

	// Verify final state
	verifyEnqueueResults_RocksDB(t, store, EnqueueRocksTestFC, EnqueueRocksTestCFS, queue.ID, 1, 3, 3, now)

	// Verify message chaining order
	verifyMessageChaining_RocksDB(t, store, EnqueueRocksTestFC, EnqueueRocksTestCFS, queue.ID, 1, []string{"msg1", "msg2", "msg3"}, now)
}

func testInvalidPriority_RocksDB(t *testing.T) {
	store := newTestRocksDBStoreForEnqueue(t)
	now := time.Now()

	// Setup queue with priority range 1-10
	queue := setupTestQueueForEnqueueRocksDB(t, store, EnqueueRocksTestFC, EnqueueRocksTestCFS, now)

	// Test priority too low
	uow1 := db.NewUnitOfWork(store, nil)
	cmd1 := &queueCommand.EnqueueCommand{
		Messages: []models.QueueMessage{{ID: "msg-low-001", MessageID: "msg_low", Priority: 0, QueueID: queue.ID}},
		CF:       EnqueueRocksTestFC,
		CFS:      EnqueueRocksTestCFS,
	}
	result1 := cmd1.Execute(uow1, now)
	assert.NotEmpty(t, result1.Error)
	assert.Contains(t, result1.Error, "Priority")

	// Test priority too high
	uow2 := db.NewUnitOfWork(store, nil)
	cmd2 := &queueCommand.EnqueueCommand{
		Messages: []models.QueueMessage{{ID: "msg-high-001", MessageID: "msg_high", Priority: 11, QueueID: queue.ID}},
		CF:       EnqueueRocksTestFC,
		CFS:      EnqueueRocksTestCFS,
	}
	result2 := cmd2.Execute(uow2, now)
	assert.NotEmpty(t, result2.Error)
	assert.Contains(t, result2.Error, "Priority")
}

func testQueueNotFound_RocksDB(t *testing.T) {
	store := newTestRocksDBStoreForEnqueue(t)
	now := time.Now()

	// Try to enqueue to non-existent queue
	uow := db.NewUnitOfWork(store, nil)
	cmd := &queueCommand.EnqueueCommand{
		Messages: []models.QueueMessage{{ID: "msg-notfound-001", MessageID: "msg1", Priority: 1, QueueID: "NON_EXISTENT_QUEUE_ID"}},
		CF:       EnqueueRocksTestFC,
		CFS:      EnqueueRocksTestCFS,
	}
	result := cmd.Execute(uow, now)
	assert.NotEmpty(t, result.Error)
	assert.Contains(t, result.Error, "Queue")
}

func testInactiveQueue_RocksDB(t *testing.T) {
	store := newTestRocksDBStoreForEnqueue(t)
	now := time.Now()

	// Setup inactive queue
	uow := db.NewUnitOfWork(store, nil)
	idFactory := &db.DeterministicIDGeneratorFactory{}

	queueRepo, err := db.NewQueueRepository(uow, idFactory, EnqueueRocksTestFC, EnqueueRocksTestCFS)
	require.NoError(t, err)

	queue := &models.Queue{
		ID:                        "test-queue-id-rocks-inactive-001",
		Code:                      "INACTIVE_QUEUE_ROCKS",
		Name:                      "Inactive Queue RocksDB",
		VNamespace:                "test-namespace",
		State:                     models.QueueStopped, // Inactive
		Type:                      models.StandardQueue,
		DefaultQueueMessageTTL:                  3600,
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
		Messages: []models.QueueMessage{{ID: "msg-inactive-001", MessageID: "msg1", Priority: 1, QueueID: queue.ID}},
		CF:       EnqueueRocksTestFC,
		CFS:      EnqueueRocksTestCFS,
	}
	result := cmd.Execute(uow2, now)
	assert.NotEmpty(t, result.Error)
	assert.Contains(t, result.Error, "active")
}

func testMultipleMessagesOrdering_RocksDB(t *testing.T) {
	store := newTestRocksDBStoreForEnqueue(t)
	now := time.Now()

	// Setup queue
	queue := setupTestQueueForEnqueueRocksDB(t, store, EnqueueRocksTestFC, EnqueueRocksTestCFS, now)

	// Add messages to different partitions in a single command
	uow := db.NewUnitOfWork(store, nil)
	cmd := &queueCommand.EnqueueCommand{
		Messages: []models.QueueMessage{
			{ID: "msg-multi-001", MessageID: "msg1_p1", Priority: 1, QueueID: queue.ID},
			{ID: "msg-multi-002", MessageID: "msg2_p1", Priority: 1, QueueID: queue.ID},
			{ID: "msg-multi-003", MessageID: "msg1_p2", Priority: 2, QueueID: queue.ID},
			{ID: "msg-multi-004", MessageID: "msg2_p2", Priority: 2, QueueID: queue.ID},
			{ID: "msg-multi-005", MessageID: "msg1_p3", Priority: 3, QueueID: queue.ID},
		},
		CF:  EnqueueRocksTestFC,
		CFS: EnqueueRocksTestCFS,
	}
	result := cmd.Execute(uow, now)
	require.Empty(t, result.Error)
	err := uow.Commit()
	require.NoError(t, err)

	// Verify partitions
	verifyEnqueueResults_RocksDB(t, store, EnqueueRocksTestFC, EnqueueRocksTestCFS, queue.ID, 1, 2, 5, now)
	verifyEnqueueResults_RocksDB(t, store, EnqueueRocksTestFC, EnqueueRocksTestCFS, queue.ID, 2, 2, 5, now)
	verifyEnqueueResults_RocksDB(t, store, EnqueueRocksTestFC, EnqueueRocksTestCFS, queue.ID, 3, 1, 5, now)

	// Verify message chaining for each partition
	verifyMessageChaining_RocksDB(t, store, EnqueueRocksTestFC, EnqueueRocksTestCFS, queue.ID, 1, []string{"msg1_p1", "msg2_p1"}, now)
	verifyMessageChaining_RocksDB(t, store, EnqueueRocksTestFC, EnqueueRocksTestCFS, queue.ID, 2, []string{"msg1_p2", "msg2_p2"}, now)
	verifyMessageChaining_RocksDB(t, store, EnqueueRocksTestFC, EnqueueRocksTestCFS, queue.ID, 3, []string{"msg1_p3"}, now)
}

// Verification functions for RocksDB
func verifyEnqueueResults_RocksDB(t *testing.T, store db.KVStore, cf, cfs, queueID string, priority, expectedPartitionCount, expectedQueueCount int, now time.Time) {
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

func verifyMessageChaining_RocksDB(t *testing.T, store db.KVStore, cf, cfs, queueID string, priority int, expectedMessageIDs []string, now time.Time) {
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

// Test empty messages array
func testEmptyMessagesArray_RocksDB(t *testing.T) {
	store := newTestRocksDBStoreForEnqueue(t)
	now := time.Now()

	// Try to enqueue empty array
	uow := db.NewUnitOfWork(store, nil)
	cmd := &queueCommand.EnqueueCommand{
		Messages: []models.QueueMessage{},
		CF:       EnqueueRocksTestFC,
		CFS:      EnqueueRocksTestCFS,
	}
	result := cmd.Execute(uow, now)
	assert.NotEmpty(t, result.Error)
	assert.Contains(t, result.Error, "No messages provided")
}

// Test bulk message enqueue with large number of messages
func testBulkMessageEnqueue_RocksDB(t *testing.T) {
	store := newTestRocksDBStoreForEnqueue(t)
	now := time.Now()

	// Setup queue
	queue := setupTestQueueForEnqueueRocksDB(t, store, EnqueueRocksTestFC, EnqueueRocksTestCFS, now)

	// Create 20 messages across different priorities
	var messages []models.QueueMessage
	priorities := []int{1, 2, 3, 10} // Use valid priorities for this queue
	for i := 0; i < 20; i++ {
		priority := priorities[i%len(priorities)] // Cycle through valid priorities
		messages = append(messages, models.QueueMessage{
			ID:        fmt.Sprintf("msg-bulk-%03d", i+1),
			MessageID: fmt.Sprintf("bulk_msg_%d", i+1),
			Priority:  priority,
			QueueID:   queue.ID,
		})
	}

	// Execute bulk enqueue
	uow := db.NewUnitOfWork(store, nil)
	cmd := &queueCommand.EnqueueCommand{
		Messages: messages,
		CF:       EnqueueRocksTestFC,
		CFS:      EnqueueRocksTestCFS,
	}
	result := cmd.Execute(uow, now)
	require.Empty(t, result.Error)
	err := uow.Commit()
	require.NoError(t, err)

	// Verify total queue message count
	verifyEnqueueResults_RocksDB(t, store, EnqueueRocksTestFC, EnqueueRocksTestCFS, queue.ID, 1, 5, 20, now)  // 5 messages in priority 1
	verifyEnqueueResults_RocksDB(t, store, EnqueueRocksTestFC, EnqueueRocksTestCFS, queue.ID, 2, 5, 20, now)  // 5 messages in priority 2
	verifyEnqueueResults_RocksDB(t, store, EnqueueRocksTestFC, EnqueueRocksTestCFS, queue.ID, 3, 5, 20, now)  // 5 messages in priority 3
	verifyEnqueueResults_RocksDB(t, store, EnqueueRocksTestFC, EnqueueRocksTestCFS, queue.ID, 10, 5, 20, now) // 5 messages in priority 10

	// Verify message chaining for priority 1
	expectedP1 := []string{"bulk_msg_1", "bulk_msg_5", "bulk_msg_9", "bulk_msg_13", "bulk_msg_17"}
	verifyMessageChaining_RocksDB(t, store, EnqueueRocksTestFC, EnqueueRocksTestCFS, queue.ID, 1, expectedP1, now)
}

// Test multi-queue support - processing messages from different queues in a single command
func testMultiQueueSupport_RocksDB(t *testing.T) {
	store := newTestRocksDBStoreForEnqueue(t)
	now := time.Now()

	// Setup two different queues
	queue1 := setupTestQueueForEnqueueRocksDB(t, store, EnqueueRocksTestFC, EnqueueRocksTestCFS, now)

	// Create second queue
	uow := db.NewUnitOfWork(store, nil)
	idFactory := &db.DeterministicIDGeneratorFactory{}
	queueRepo, err := db.NewQueueRepository(uow, idFactory, EnqueueRocksTestFC, EnqueueRocksTestCFS)
	require.NoError(t, err)

	queue2 := &models.Queue{
		ID:                        "test-queue-id-rocks-002",
		Code:                      "TEST_QUEUE_ROCKS_2",
		Name:                      "Test Queue RocksDB 2",
		VNamespace:                "test-namespace",
		State:                     models.QueueActive,
		Type:                      models.StandardQueue,
		DefaultQueueMessageTTL:                  3600,
		AllowDuplicated:           true,
		MaxAttempts:               3,
		MessagesCount:             0,
		DesiredPriorityThresholds: map[int]int{1: 100, 2: 50, 3: 30, 10: 10},
		PriorityThresholds:        map[int]int{1: 100, 2: 50, 3: 30, 10: 10},
		CreatedAt:                 now,
		UpdatedAt:                 now,
	}

	queueID2, err := queueRepo.CreateQueue(queue2, now)
	require.NoError(t, err)
	queue2.ID = queueID2
	err = uow.Commit()
	require.NoError(t, err)

	// Execute command with messages from both queues
	uow2 := db.NewUnitOfWork(store, nil)
	cmd := &queueCommand.EnqueueCommand{
		Messages: []models.QueueMessage{
			{ID: "msg-q1-001", MessageID: "msg1_queue1", Priority: 1, QueueID: queue1.ID},
			{ID: "msg-q1-002", MessageID: "msg2_queue1", Priority: 1, QueueID: queue1.ID},
			{ID: "msg-q2-001", MessageID: "msg1_queue2", Priority: 1, QueueID: queue2.ID},
			{ID: "msg-q2-002", MessageID: "msg2_queue2", Priority: 2, QueueID: queue2.ID},
			{ID: "msg-q1-003", MessageID: "msg3_queue1", Priority: 2, QueueID: queue1.ID},
		},
		CF:  EnqueueRocksTestFC,
		CFS: EnqueueRocksTestCFS,
	}
	result := cmd.Execute(uow2, now)
	require.Empty(t, result.Error)
	err = uow2.Commit()
	require.NoError(t, err)

	// Verify results for queue1: 2 messages in priority 1, 1 message in priority 2
	verifyEnqueueResults_RocksDB(t, store, EnqueueRocksTestFC, EnqueueRocksTestCFS, queue1.ID, 1, 2, 3, now)
	verifyEnqueueResults_RocksDB(t, store, EnqueueRocksTestFC, EnqueueRocksTestCFS, queue1.ID, 2, 1, 3, now)

	// Verify results for queue2: 1 message in priority 1, 1 message in priority 2
	verifyEnqueueResults_RocksDB(t, store, EnqueueRocksTestFC, EnqueueRocksTestCFS, queue2.ID, 1, 1, 2, now)
	verifyEnqueueResults_RocksDB(t, store, EnqueueRocksTestFC, EnqueueRocksTestCFS, queue2.ID, 2, 1, 2, now)

	// Verify message chaining for each queue and priority
	verifyMessageChaining_RocksDB(t, store, EnqueueRocksTestFC, EnqueueRocksTestCFS, queue1.ID, 1, []string{"msg1_queue1", "msg2_queue1"}, now)
	verifyMessageChaining_RocksDB(t, store, EnqueueRocksTestFC, EnqueueRocksTestCFS, queue1.ID, 2, []string{"msg3_queue1"}, now)
	verifyMessageChaining_RocksDB(t, store, EnqueueRocksTestFC, EnqueueRocksTestCFS, queue2.ID, 1, []string{"msg1_queue2"}, now)
	verifyMessageChaining_RocksDB(t, store, EnqueueRocksTestFC, EnqueueRocksTestCFS, queue2.ID, 2, []string{"msg2_queue2"}, now)
}
