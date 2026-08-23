package db_test

import (
	"bytes"
	"deadalus-orch/server/internal/infrastructure/db"
	general_command "deadalus-orch/server/internal/usecase/command/general"
	queue_command "deadalus-orch/server/internal/usecase/command/queue"
	"deadalus-orch/shared/models"
	"encoding/gob"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/lni/dragonboat/v4/statemachine"
)

// ============================================================================
// Benchmark 4: FSM / EnqueueCommand Overhead
// Goal: Measure the cost of the EnqueueCommand executing inside an FSM Update,
//       WITHOUT any Raft or network overhead. This isolates the pure
//       business logic + ORM cost.
// ============================================================================

// setupBenchEnqueueEnvironment creates a Pebble store with a queue pre-created,
// ready for EnqueueCommand benchmarks.
func setupBenchEnqueueEnvironment(b *testing.B) (db.KVStore, func()) {
	tempDir, err := os.MkdirTemp("", "enqueue_bench_*")
	if err != nil {
		b.Fatal(err)
	}

	cf := "cf-n-0"
	store, err := db.CreatePebbleStore(tempDir, []string{cf, "admin"}, nil)
	if err != nil {
		os.RemoveAll(tempDir)
		b.Fatal(err)
	}

	cfs := "test-tenant-id"
	now := time.Now()

	// Create a queue so EnqueueCommand can find it
	idFactory := &db.DeterministicIDGeneratorFactory{}
	uow := db.NewUnitOfWork(store, nil)
	queueRepo, err := db.NewQueueRepository(uow, idFactory, cf, cfs)
	if err != nil {
		store.Close()
		os.RemoveAll(tempDir)
		b.Fatal(err)
	}

	queue := &models.Queue{
		ID:                        "queue-001",
		Code:                      "bench-queue",
		Name:                      "Benchmark Queue",
		Type:                      models.StandardQueue,
		State:                     models.QueueActive,
		VNamespace:                "default",
		MaxAttempts:               1,
		DesiredPriorityThresholds: map[int]int{0: 0},
	}
	_, err = queueRepo.CreateQueue(queue, now)
	if err != nil {
		store.Close()
		os.RemoveAll(tempDir)
		b.Fatal(err)
	}

	if err := uow.Commit(); err != nil {
		store.Close()
		os.RemoveAll(tempDir)
		b.Fatal(err)
	}

	cleanup := func() {
		store.Close()
		os.RemoveAll(tempDir)
	}
	return store, cleanup
}

// BenchmarkEnqueueCommand_Execute measures EnqueueCommand.Execute() directly
// (bypassing Raft entirely). This is the core business logic benchmark.
func BenchmarkEnqueueCommand_Execute(b *testing.B) {
	messageCounts := []int{1, 10, 50, 100}
	for _, count := range messageCounts {
		b.Run(fmt.Sprintf("Messages_%d", count), func(b *testing.B) {
			store, cleanup := setupBenchEnqueueEnvironment(b)
			defer cleanup()

			cf := "cf-n-0"
			cfs := "test-tenant-id"
			now := time.Now()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				messages := make([]models.QueueMessage, count)
				for j := 0; j < count; j++ {
					messages[j] = models.QueueMessage{
						ID:               fmt.Sprintf("msg-%d-%d", i, j),
						MessageID:        fmt.Sprintf("mid-%d-%d", i, j),
						QueueID:          "queue-001",
						Priority:         0,
						ContentType:      "application/json",
						ContentLength:    256,
						Content:          generateTestValue(256),
						Handler:          "default",
						VNamespace:       "default",
					}
				}

				cmd := &queue_command.EnqueueCommand{
					Messages: messages,
					CF:       cf,
					CFS:      cfs,
				}

				uow := db.NewUnitOfWork(store, nil)
				result := cmd.Execute(uow, now)
				if result.Error != "" {
					b.Fatalf("EnqueueCommand failed: %s", result.Error)
				}

				// Commit the UOW as would happen in the real FSM
				if err := uow.Commit(); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkEnqueueCommand_GobRoundTrip measures the full gob encode→decode
// roundtrip for an FSM_Command containing an EnqueueCommand.
// This simulates what happens when a command goes through Raft.
func BenchmarkEnqueueCommand_GobRoundTrip(b *testing.B) {
	messageCounts := []int{1, 10, 50, 100}
	for _, count := range messageCounts {
		b.Run(fmt.Sprintf("Messages_%d", count), func(b *testing.B) {
			messages := make([]models.QueueMessage, count)
			for j := 0; j < count; j++ {
				messages[j] = models.QueueMessage{
					ID:            fmt.Sprintf("msg-%d", j),
					MessageID:     fmt.Sprintf("mid-%d", j),
					QueueID:       "queue-001",
					Priority:      0,
					ContentType:   "application/json",
					ContentLength: 256,
					Content:       generateTestValue(256),
					Handler:       "default",
					VNamespace:    "default",
				}
			}

			cmd := queue_command.EnqueueCommand{
				Messages: messages,
				CF:       "cf-n-0",
				CFS:      "test-tenant-id",
			}

			fsmCmd := general_command.FSM_Command{
				Now:  time.Now().UnixNano(),
				Type: general_command.REPOSITORY_COMMAND,
				CMD:  cmd,
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Encode (simulates RaftNode.Write)
				var buf bytes.Buffer
				if err := gob.NewEncoder(&buf).Encode(fsmCmd); err != nil {
					b.Fatal(err)
				}

				// Decode (simulates KVBaseStateMachine.Update)
				var decoded general_command.FSM_Command
				if err := gob.NewDecoder(bytes.NewReader(buf.Bytes())).Decode(&decoded); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkFSM_UpdateSimulation simulates what KVBaseStateMachine.Update() does:
// decode gob → dispatch → execute command → encode result.
// This is the most realistic isolated benchmark of the FSM hot path.
func BenchmarkFSM_UpdateSimulation(b *testing.B) {
	messageCounts := []int{1, 10, 50}
	for _, count := range messageCounts {
		b.Run(fmt.Sprintf("Messages_%d", count), func(b *testing.B) {
			store, cleanup := setupBenchEnqueueEnvironment(b)
			defer cleanup()

			cf := "cf-n-0"
			cfs := "test-tenant-id"

			// Pre-encode a command
			messages := make([]models.QueueMessage, count)
			for j := 0; j < count; j++ {
				messages[j] = models.QueueMessage{
					ID:            fmt.Sprintf("template-msg-%d", j),
					MessageID:     fmt.Sprintf("template-mid-%d", j),
					QueueID:       "queue-001",
					Priority:      0,
					ContentType:   "application/json",
					ContentLength: 256,
					Content:       generateTestValue(256),
					Handler:       "default",
					VNamespace:    "default",
				}
			}

			now := time.Now()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Phase 1: Create unique messages per iteration
				iterMessages := make([]models.QueueMessage, count)
				for j := 0; j < count; j++ {
					iterMessages[j] = messages[j]
					iterMessages[j].ID = fmt.Sprintf("msg-%d-%d", i, j)
					iterMessages[j].MessageID = fmt.Sprintf("mid-%d-%d", i, j)
				}

				cmd := queue_command.EnqueueCommand{
					Messages: iterMessages,
					CF:       cf,
					CFS:      cfs,
				}

				fsmCmd := general_command.FSM_Command{
					Now:  now.UnixNano(),
					Type: general_command.REPOSITORY_COMMAND,
					CMD:  cmd,
				}

				// Phase 2: GOB encode (simulates propose serialization)
				var encodeBuf bytes.Buffer
				if err := gob.NewEncoder(&encodeBuf).Encode(fsmCmd); err != nil {
					b.Fatal(err)
				}

				// Phase 3: GOB decode (simulates FSM Update entry processing)
				var decodedCmd general_command.FSM_Command
				if err := gob.NewDecoder(bytes.NewReader(encodeBuf.Bytes())).Decode(&decodedCmd); err != nil {
					b.Fatal(err)
				}

				// Phase 4: Execute command (simulates FSM dispatch)
				enqueueCmd := decodedCmd.CMD.(queue_command.EnqueueCommand)
				uow := db.NewUnitOfWork(store, nil)
				result := enqueueCmd.Execute(uow, now)
				if result.Error != "" {
					b.Fatalf("EnqueueCommand failed at iteration %d: %s", i, result.Error)
				}

				// Phase 5: GOB encode result (simulates returning result via statemachine.Result)
				var resultBuf bytes.Buffer
				if err := gob.NewEncoder(&resultBuf).Encode(result); err != nil {
					b.Fatal(err)
				}

				// Phase 6: Commit UOW (simulates batch write to Pebble)
				if err := uow.Commit(); err != nil {
					b.Fatal(err)
				}

				// Create the statemachine result (as the real code does)
				_ = statemachine.Result{
					Value: uint64(len(encodeBuf.Bytes())),
					Data:  resultBuf.Bytes(),
				}
			}
		})
	}
}

// BenchmarkFSM_UpdateSimulation_SimpleRWPut simulates a simple RW Put through
// the FSM path for comparison. This is the minimal-overhead FSM path.
func BenchmarkFSM_UpdateSimulation_SimpleRWPut(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "simple_rw_bench_*")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	cf := "cf-n-0"
	store, err := db.CreatePebbleStore(tempDir, []string{cf, "admin"}, nil)
	if err != nil {
		b.Fatal(err)
	}
	defer store.Close()

	now := time.Now()
	value := generateTestValue(256)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("bench-key-%d", i)

		// Simulate the RW command path
		fsmCmd := general_command.FSM_Command{
			Now:  now.UnixNano(),
			Type: general_command.RW,
			CMD: general_command.RWK_Command{
				Op: general_command.Write,
				CMD: general_command.WK_Command{
					Key:                key,
					Value:              value,
					ColumnFamilyName:   cf,
					ColumnFamilySector: "test-sector",
					Op:                 general_command.PutOp,
				},
			},
		}

		// Phase 1: GOB encode
		var encodeBuf bytes.Buffer
		if err := gob.NewEncoder(&encodeBuf).Encode(fsmCmd); err != nil {
			b.Fatal(err)
		}

		// Phase 2: GOB decode
		var decodedCmd general_command.FSM_Command
		if err := gob.NewDecoder(bytes.NewReader(encodeBuf.Bytes())).Decode(&decodedCmd); err != nil {
			b.Fatal(err)
		}

		// Phase 3: Execute (just a simple Put)
		rwCmd := decodedCmd.CMD.(general_command.RWK_Command)
		wCmd := rwCmd.CMD.(general_command.WK_Command)
		batch := db.NewWriteBatch()
		batch.Put(wCmd.ColumnFamilyName, wCmd.ColumnFamilySector, wCmd.Key, wCmd.Value, now)
		if err := store.Write(batch); err != nil {
			b.Fatal(err)
		}
	}
}
