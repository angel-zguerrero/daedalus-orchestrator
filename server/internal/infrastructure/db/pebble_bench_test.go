package db_test

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/cockroachdb/pebble"
)

// ============================================================================
// Benchmark 1: Pebble Raw Performance
// Goal: Measure raw Pebble throughput independent of all other layers
// ============================================================================

// newBenchPebbleStore creates a temporary Pebble store for benchmarks.
func newBenchPebbleStore(b *testing.B, cfNames []string, ttlCfNames []string) (db.KVStore, func()) {
	tempDir, err := os.MkdirTemp("", "pebble_bench_*")
	if err != nil {
		b.Fatal(err)
	}

	store, err := db.CreatePebbleStore(tempDir, cfNames, ttlCfNames)
	if err != nil {
		os.RemoveAll(tempDir)
		b.Fatal(err)
	}

	cleanup := func() {
		store.Close()
		os.RemoveAll(tempDir)
	}
	return store, cleanup
}

// newBenchPebbleRaw creates a raw Pebble DB for benchmarking with specific options.
func newBenchPebbleRaw(b *testing.B, cacheSize int64, memTableSize int) (*pebble.DB, func()) {
	tempDir, err := os.MkdirTemp("", "pebble_raw_bench_*")
	if err != nil {
		b.Fatal(err)
	}

	cache := pebble.NewCache(cacheSize)
	opts := &pebble.Options{
		MemTableSize: memTableSize,
		Cache:        cache,
	}
	pdb, err := pebble.Open(tempDir, opts)
	cache.Unref()
	if err != nil {
		os.RemoveAll(tempDir)
		b.Fatal(err)
	}

	cleanup := func() {
		pdb.Close()
		os.RemoveAll(tempDir)
	}
	return pdb, cleanup
}

// generateTestValue generates a byte slice of the specified size (simulates message content).
func generateTestValue(size int) []byte {
	val := make([]byte, size)
	for i := range val {
		val[i] = byte(i % 256)
	}
	return val
}

// BenchmarkPebbleRaw_SinglePut measures raw Pebble single-key Put performance.
// Uses the CURRENT Daedalus config (cache=0, memtable=32KB).
func BenchmarkPebbleRaw_SinglePut_CurrentConfig(b *testing.B) {
	pdb, cleanup := newBenchPebbleRaw(b, 0, 1024*32) // cache=0, memtable=32KB
	defer cleanup()

	value := generateTestValue(500)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i)
		if err := pdb.Set([]byte(key), value, pebble.NoSync); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkPebbleRaw_SinglePut_OptimizedConfig uses a reasonable config (cache=32MB, memtable=4MB).
func BenchmarkPebbleRaw_SinglePut_OptimizedConfig(b *testing.B) {
	pdb, cleanup := newBenchPebbleRaw(b, 32*1024*1024, 4*1024*1024) // cache=32MB, memtable=4MB
	defer cleanup()

	value := generateTestValue(500)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i)
		if err := pdb.Set([]byte(key), value, pebble.NoSync); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkPebbleRaw_BatchPut_CurrentConfig measures batch Put performance with current config.
func BenchmarkPebbleRaw_BatchPut_CurrentConfig(b *testing.B) {
	batchSizes := []int{10, 50, 100, 500, 1000, 2000}
	for _, batchSize := range batchSizes {
		b.Run(fmt.Sprintf("BatchSize_%d", batchSize), func(b *testing.B) {
			pdb, cleanup := newBenchPebbleRaw(b, 0, 1024*32)
			defer cleanup()

			value := generateTestValue(500)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				batch := pdb.NewBatch()
				for j := 0; j < batchSize; j++ {
					key := fmt.Sprintf("key-%d-%d", i, j)
					if err := batch.Set([]byte(key), value, nil); err != nil {
						b.Fatal(err)
					}
				}
				if err := batch.Commit(pebble.NoSync); err != nil {
					b.Fatal(err)
				}
				batch.Close()
			}
		})
	}
}

// BenchmarkPebbleRaw_BatchPut_OptimizedConfig measures batch Put performance with optimized config.
func BenchmarkPebbleRaw_BatchPut_OptimizedConfig(b *testing.B) {
	batchSizes := []int{10, 50, 100, 500, 1000, 2000}
	for _, batchSize := range batchSizes {
		b.Run(fmt.Sprintf("BatchSize_%d", batchSize), func(b *testing.B) {
			pdb, cleanup := newBenchPebbleRaw(b, 32*1024*1024, 4*1024*1024)
			defer cleanup()

			value := generateTestValue(500)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				batch := pdb.NewBatch()
				for j := 0; j < batchSize; j++ {
					key := fmt.Sprintf("key-%d-%d", i, j)
					if err := batch.Set([]byte(key), value, nil); err != nil {
						b.Fatal(err)
					}
				}
				if err := batch.Commit(pebble.NoSync); err != nil {
					b.Fatal(err)
				}
				batch.Close()
			}
		})
	}
}

// BenchmarkPebbleRaw_Get_AfterBulkLoad measures read performance after inserting N keys.
func BenchmarkPebbleRaw_Get_AfterBulkLoad(b *testing.B) {
	keyCounts := []int{1000, 5000, 10000}
	for _, keyCount := range keyCounts {
		b.Run(fmt.Sprintf("After_%d_keys", keyCount), func(b *testing.B) {
			pdb, cleanup := newBenchPebbleRaw(b, 32*1024*1024, 4*1024*1024)
			defer cleanup()

			value := generateTestValue(500)
			// Pre-load keys
			for i := 0; i < keyCount; i++ {
				key := fmt.Sprintf("key-%d", i)
				if err := pdb.Set([]byte(key), value, pebble.NoSync); err != nil {
					b.Fatal(err)
				}
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				key := fmt.Sprintf("key-%d", i%keyCount)
				_, closer, err := pdb.Get([]byte(key))
				if err != nil {
					b.Fatal(err)
				}
				closer.Close()
			}
		})
	}
}

// BenchmarkPebbleStore_Write_Batch tests PebbleStore.Write() through the KVStore interface.
// This includes the prefix key construction overhead.
func BenchmarkPebbleStore_Write_Batch(b *testing.B) {
	batchSizes := []int{10, 50, 100, 500, 1000}
	for _, batchSize := range batchSizes {
		b.Run(fmt.Sprintf("BatchSize_%d", batchSize), func(b *testing.B) {
			store, cleanup := newBenchPebbleStore(b, []string{"bench_cf"}, nil)
			defer cleanup()

			value := generateTestValue(500)
			now := time.Now()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				batch := db.NewWriteBatch()
				for j := 0; j < batchSize; j++ {
					key := fmt.Sprintf("key-%d-%d", i, j)
					batch.Put("bench_cf", "bench-sector", key, value, now)
				}
				if err := store.Write(batch); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkPebbleStore_Get measures PebbleStore.Get() through the KVStore interface.
func BenchmarkPebbleStore_Get(b *testing.B) {
	store, cleanup := newBenchPebbleStore(b, []string{"bench_cf"}, nil)
	defer cleanup()

	value := generateTestValue(500)
	now := time.Now()

	// Pre-load 5000 keys
	keyCount := 5000
	for i := 0; i < keyCount; i++ {
		key := fmt.Sprintf("key-%d", i)
		err := store.Put("bench_cf", "bench-sector", key, value, 0, now)
		if err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i%keyCount)
		_, err := store.Get("bench_cf", "bench-sector", key, now)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkPebbleStore_Exists measures PebbleStore.Exists() through the KVStore interface.
func BenchmarkPebbleStore_Exists(b *testing.B) {
	store, cleanup := newBenchPebbleStore(b, []string{"bench_cf"}, nil)
	defer cleanup()

	value := generateTestValue(500)
	now := time.Now()

	// Pre-load 5000 keys
	keyCount := 5000
	for i := 0; i < keyCount; i++ {
		key := fmt.Sprintf("key-%d", i)
		err := store.Put("bench_cf", "bench-sector", key, value, 0, now)
		if err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i%keyCount)
		_, err := store.Exists("bench_cf", "bench-sector", key, now)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkPebbleStore_SearchByPattern measures pattern search performance.
func BenchmarkPebbleStore_SearchByPattern(b *testing.B) {
	store, cleanup := newBenchPebbleStore(b, []string{"bench_cf"}, nil)
	defer cleanup()

	value := generateTestValue(100)
	now := time.Now()

	// Pre-load 5000 keys with a common prefix
	keyCount := 5000
	for i := 0; i < keyCount; i++ {
		key := fmt.Sprintf("queue:msg:%d", i)
		err := store.Put("bench_cf", "bench-sector", key, value, 0, now)
		if err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := store.SearchByPatternPaginatedKV("bench_cf", "bench-sector", "queue:msg:*", "", 100, now)
		if err != nil {
			b.Fatal(err)
		}
	}
}
