package db_test

import (
	"bytes"
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/shared/models"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

// ============================================================================
// Benchmark 2: Repository ORM Overhead
// Goal: Measure the cost of the ORM layer (reflection, indexing, JSON)
//       vs direct Pebble writes for the same data
// ============================================================================

// newBenchQueueMessageRepo creates a QueueMessage repository for benchmarking.
func newBenchQueueMessageRepo(b *testing.B) (db.KVStore, *db.QueueMessageRepository, func()) {
	tempDir, err := os.MkdirTemp("", "repo_bench_*")
	if err != nil {
		b.Fatal(err)
	}

	cf := "cf-n-0"
	store, err := db.CreatePebbleStore(tempDir, []string{cf}, nil)
	if err != nil {
		os.RemoveAll(tempDir)
		b.Fatal(err)
	}

	cfs := "test-tenant-id"
	idFactory := &db.DeterministicIDGeneratorFactory{}
	uow := db.NewUnitOfWork(store, nil)
	repo, err := db.NewQueueMessageRepository(uow, idFactory, cf, cfs)
	if err != nil {
		store.Close()
		os.RemoveAll(tempDir)
		b.Fatal(err)
	}

	cleanup := func() {
		store.Close()
		os.RemoveAll(tempDir)
	}
	return store, repo, cleanup
}

// newBenchQueueRepo creates a Queue repository for benchmarking.
func newBenchQueueRepo(b *testing.B) (db.KVStore, *db.QueueRepository, func()) {
	tempDir, err := os.MkdirTemp("", "repo_bench_*")
	if err != nil {
		b.Fatal(err)
	}

	cf := "cf-n-0"
	store, err := db.CreatePebbleStore(tempDir, []string{cf}, nil)
	if err != nil {
		os.RemoveAll(tempDir)
		b.Fatal(err)
	}

	cfs := "test-tenant-id"
	idFactory := &db.DeterministicIDGeneratorFactory{}
	uow := db.NewUnitOfWork(store, nil)
	repo, err := db.NewQueueRepository(uow, idFactory, cf, cfs)
	if err != nil {
		store.Close()
		os.RemoveAll(tempDir)
		b.Fatal(err)
	}

	cleanup := func() {
		store.Close()
		os.RemoveAll(tempDir)
	}
	return store, repo, cleanup
}

// makeTestQueueMessage creates a test QueueMessage with realistic data.
func makeTestQueueMessage(id string) *models.QueueMessage {
	return &models.QueueMessage{
		ID:               id,
		MessageID:        "msg-" + id,
		QueueID:          "queue-001",
		QueuePartitionID: "queue-001-p-0",
		Priority:         0,
		Attempts:         0,
		ContentType:      "application/json",
		ContentLength:    256,
		Content:          generateTestValue(256),
		Handler:          "default-handler",
		VNamespace:       "default",
	}
}

// BenchmarkRepository_CreateSingle_QueueMessage measures the cost of creating
// a single QueueMessage through the ORM.
func BenchmarkRepository_CreateSingle_QueueMessage(b *testing.B) {
	_, repo, cleanup := newBenchQueueMessageRepo(b)
	defer cleanup()

	now := time.Now()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msg := makeTestQueueMessage(fmt.Sprintf("msg-%d", i))
		_, err := repo.CreateQueueMessage(msg, now)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkRepository_DirectPebblePut_SameData measures putting the same
// QueueMessage data directly to Pebble (bypassing ORM).
// This is the "ideal" baseline for comparison.
func BenchmarkRepository_DirectPebblePut_SameData(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "direct_put_bench_*")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	cf := "cf-n-0"
	store, err := db.CreatePebbleStore(tempDir, []string{cf}, nil)
	if err != nil {
		b.Fatal(err)
	}
	defer store.Close()

	now := time.Now()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msg := makeTestQueueMessage(fmt.Sprintf("msg-%d", i))
		dataBytes, _ := json.Marshal(msg)

		// Write only what's strictly needed: 1 data key + 1 queue index key
		batch := db.NewWriteBatch()
		dataKey := fmt.Sprintf("admin_schema:queue_messages:data:%s", msg.ID)
		queueIdxKey := fmt.Sprintf("admin_schema:queue_messages:idx:QueueID:%s:%s", msg.QueueID, msg.ID)
		batch.Put(cf, "test-tenant-id", dataKey, dataBytes, now)
		batch.Put(cf, "test-tenant-id", queueIdxKey, []byte(msg.ID), now)

		if err := store.Write(batch); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkRepository_DirectPebblePut_BatchN measures batched direct Pebble puts.
func BenchmarkRepository_DirectPebblePut_BatchN(b *testing.B) {
	batchSizes := []int{10, 50, 100, 500}
	for _, batchSize := range batchSizes {
		b.Run(fmt.Sprintf("BatchSize_%d", batchSize), func(b *testing.B) {
			tempDir, err := os.MkdirTemp("", "direct_batch_bench_*")
			if err != nil {
				b.Fatal(err)
			}
			defer os.RemoveAll(tempDir)

			cf := "cf-n-0"
			store, err := db.CreatePebbleStore(tempDir, []string{cf}, nil)
			if err != nil {
				b.Fatal(err)
			}
			defer store.Close()

			now := time.Now()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				batch := db.NewWriteBatch()
				for j := 0; j < batchSize; j++ {
					id := fmt.Sprintf("msg-%d-%d", i, j)
					msg := makeTestQueueMessage(id)
					dataBytes, _ := json.Marshal(msg)

					dataKey := fmt.Sprintf("admin_schema:queue_messages:data:%s", msg.ID)
					queueIdxKey := fmt.Sprintf("admin_schema:queue_messages:idx:QueueID:%s:%s", msg.QueueID, msg.ID)
					batch.Put(cf, "test-tenant-id", dataKey, dataBytes, now)
					batch.Put(cf, "test-tenant-id", queueIdxKey, []byte(msg.ID), now)
				}

				if err := store.Write(batch); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// ============================================================================
// Benchmark 3: Serialization Overhead (gob vs JSON)
// Goal: Measure serialization costs independently
// ============================================================================

// BenchmarkSerialization_GobEncode_QueueMessage measures gob encoding of a QueueMessage.
func BenchmarkSerialization_GobEncode_QueueMessage(b *testing.B) {
	msg := makeTestQueueMessage("msg-bench-001")
	msg.Content = generateTestValue(256)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		if err := gob.NewEncoder(&buf).Encode(msg); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkSerialization_GobDecode_QueueMessage measures gob decoding of a QueueMessage.
func BenchmarkSerialization_GobDecode_QueueMessage(b *testing.B) {
	msg := makeTestQueueMessage("msg-bench-001")
	msg.Content = generateTestValue(256)

	var encoded bytes.Buffer
	if err := gob.NewEncoder(&encoded).Encode(msg); err != nil {
		b.Fatal(err)
	}
	data := encoded.Bytes()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var decoded models.QueueMessage
		if err := gob.NewDecoder(bytes.NewReader(data)).Decode(&decoded); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkSerialization_JsonMarshal_QueueMessage measures JSON marshaling of a QueueMessage.
func BenchmarkSerialization_JsonMarshal_QueueMessage(b *testing.B) {
	msg := makeTestQueueMessage("msg-bench-001")
	msg.Content = generateTestValue(256)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := json.Marshal(msg); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkSerialization_JsonUnmarshal_QueueMessage measures JSON unmarshaling of a QueueMessage.
func BenchmarkSerialization_JsonUnmarshal_QueueMessage(b *testing.B) {
	msg := makeTestQueueMessage("msg-bench-001")
	msg.Content = generateTestValue(256)

	data, err := json.Marshal(msg)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var decoded models.QueueMessage
		if err := json.Unmarshal(data, &decoded); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkSerialization_GobEncode_FSMCommand measures gob encoding of a full FSM command
// containing an EnqueueCommand with N messages.
func BenchmarkSerialization_GobEncode_FSMCommand(b *testing.B) {
	messageCounts := []int{1, 10, 50, 100}
	for _, count := range messageCounts {
		b.Run(fmt.Sprintf("Messages_%d", count), func(b *testing.B) {
			messages := make([]models.QueueMessage, count)
			for i := 0; i < count; i++ {
				m := makeTestQueueMessage(fmt.Sprintf("msg-%d", i))
				messages[i] = *m
			}

			// We encode just the messages slice to approximate FSM_Command encoding cost.
			// The actual FSM_Command has wrapping but this captures the bulk of the data.
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				var buf bytes.Buffer
				if err := gob.NewEncoder(&buf).Encode(messages); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkSerialization_GobDecode_FSMCommand measures gob decoding of messages batch.
func BenchmarkSerialization_GobDecode_FSMCommand(b *testing.B) {
	messageCounts := []int{1, 10, 50, 100}
	for _, count := range messageCounts {
		b.Run(fmt.Sprintf("Messages_%d", count), func(b *testing.B) {
			messages := make([]models.QueueMessage, count)
			for i := 0; i < count; i++ {
				m := makeTestQueueMessage(fmt.Sprintf("msg-%d", i))
				messages[i] = *m
			}

			var encoded bytes.Buffer
			if err := gob.NewEncoder(&encoded).Encode(messages); err != nil {
				b.Fatal(err)
			}
			data := encoded.Bytes()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				var decoded []models.QueueMessage
				if err := gob.NewDecoder(bytes.NewReader(data)).Decode(&decoded); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkFmtSprintf_KeyConstruction measures the overhead of key construction via fmt.Sprintf.
func BenchmarkFmtSprintf_KeyConstruction(b *testing.B) {
	schema := "admin_schema"
	table := "queue_messages"
	field := "QueueID"
	value := "queue-001-abcdef1234567890"
	id := "msg-abcdef1234567890abcd"

	b.Run("Sprintf", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = fmt.Sprintf("%s:%s:idx:%s:%s:%s", schema, table, field, value, id)
		}
	})

	b.Run("StringsJoin", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = strings.Join([]string{schema, table, "idx", field, value, id}, ":")
		}
	})

	b.Run("StringConcat", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = schema + ":" + table + ":idx:" + field + ":" + value + ":" + id
		}
	})

	b.Run("Builder", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			var sb strings.Builder
			sb.WriteString(schema)
			sb.WriteByte(':')
			sb.WriteString(table)
			sb.WriteString(":idx:")
			sb.WriteString(field)
			sb.WriteByte(':')
			sb.WriteString(value)
			sb.WriteByte(':')
			sb.WriteString(id)
			_ = sb.String()
		}
	})
}
