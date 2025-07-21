//go:build rocksdb
// +build rocksdb

package db_test

// Ensure time is here, remove separate import "time" later
import (
	"deadalus-orch/server/internal/infrastructure/db"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Placeholder for TTLDefaultSeconds if not already available globally for tests
// Note: The actual TTLDefaultSeconds is in the pebble-store.go and not exported.
// For testing the TTL mechanism, we might need to either:
// 1. Expose TTLDefaultSeconds from the main package (not ideal for encapsulation).
// 2. Define a test-specific TTL duration if the goal is to test the *mechanism*
//    rather than the exact default duration. The `testTTLSeconds` constant is for this.
// 3. For now, we will rely on the `testTTLSeconds` for our TTL logic.

func runBackupRestoreTest(t *testing.T, srcStoreType, targetStoreType string) {
	t.Logf("Running test: Backup from %s, Restore to %s", srcStoreType, targetStoreType)

	allTestCfNames := []string{normalCF}
	allTestTtlCfNames := []string{ttlCF}

	var srcStore, targetStore db.KVStore
	var err error

	// Initialize source store
	if srcStoreType == "pebble" {
		srcStore = newTestPebbleStore(t, allTestCfNames, allTestTtlCfNames)
	} else if srcStoreType == "rocksdb" {
		srcStore = newTestRocksdbStore(t, allTestCfNames, allTestTtlCfNames)
	} else {
		t.Fatalf("Unknown source store type: %s", srcStoreType)
	}
	require.NoError(t, srcStore.ClearAll(), "Failed to clear source store") // ClearAll does not take `now`

	// Initialize target store
	if targetStoreType == "pebble" {
		targetStore = newTestPebbleStore(t, allTestCfNames, allTestTtlCfNames)
	} else if targetStoreType == "rocksdb" {
		targetStore = newTestRocksdbStore(t, allTestCfNames, allTestTtlCfNames)
	} else {
		t.Fatalf("Unknown target store type: %s", targetStoreType)
	}
	require.NoError(t, targetStore.ClearAll(), "Failed to clear target store") // ClearAll does not take `now`

	generatedKeys := make(map[string]testKeyData)

	// Data Generation and Insertion (Source Store)
	t.Log("Generating and inserting data into source store...")
	batch := db.NewWriteBatch()
	for i := 0; i < numNormalKeys; i++ {
		key := fmt.Sprintf("normalKey_%d", i)
		value := fmt.Sprintf("normalValue_%d", i)
		item := testKeyData{ColumnFamily: normalCF, Key: key, Value: value, IsTTL: false}
		generatedKeys[item.ColumnFamily+":"+item.Key] = item
		batch.Put(item.ColumnFamily, testColumnFamilySelector, item.Key, []byte(item.Value), time.Now())
	}

	// Reset batch for TTL keys
	// batch = NewWriteBatch() // Not batching TTL Puts as per current db.KVStore design for TTLs
	t.Logf("Note: TTL keys will be inserted using srcStore.Put(). This test will wait for `testTTLSeconds` (%ds) and expects CleanExpiredKeys to remove items that should have expired within this test-defined period.", testTTLSeconds)
	for i := 0; i < numTTLKeys; i++ {
		key := fmt.Sprintf("ttlKey_%d", i)
		value := fmt.Sprintf("ttlValue_%d", i)
		// For TTL keys, Put handles the multi-key logic internally
		// We rely on the store's Put method to correctly set up TTL.
		// The TTL duration used by the store's Put method for TTL CFs is
		// controlled by the db.KVStore implementation (e.g. pebble.TTLDefaultSeconds or a fixed value for RocksDB).
		// Our test will use `testTTLSeconds` for *expected* expiry checks.
		insertTime := time.Now()
		// This Put should use the system's default TTL for the ttlCF
		batch.PutTTl(ttlCF, testColumnFamilySelector, key, []byte(value), testTTLSeconds, time.Now())

		item := testKeyData{
			ColumnFamily: ttlCF,
			Key:          key,
			Value:        value,
			IsTTL:        true,
			InsertTime:   insertTime,
			ExpectedTTL:  time.Duration(testTTLSeconds) * time.Second, // This is what we expect for validation
		}
		generatedKeys[item.ColumnFamily+":"+item.Key] = item
	}

	err = srcStore.Write(batch)
	require.NoError(t, err, "Failed to write normal keys to source store")

	t.Log("Data generation complete.")

	// Backup (Source Store)
	t.Log("Performing backup from source store...")
	dumpedData, err := srcStore.DumpAll()
	require.NoError(t, err, "Failed to dump data from source store")
	require.NotNil(t, dumpedData, "Dumped data is nil")

	dumpedMap, ok := dumpedData.(map[string]map[string][]byte)
	require.True(t, ok, "Dumped data is not of expected type map[string]map[string][]byte")

	// Restore (Target Store)
	t.Log("Restoring data to target store...")
	restoreBatch := db.NewWriteBatch()
	for fullCFKey, cfData := range dumpedMap {
		parts := strings.SplitN(fullCFKey, ":", 2)
		require.Equal(t, 2, len(parts), "Invalid CF key format: %s", fullCFKey)

		cfName := parts[0]
		cfSelector := parts[1]

		for key, value := range cfData {
			restoreBatch.Put(cfName, cfSelector, key, value, time.Now())
		}
	}
	err = targetStore.WriteRaw(restoreBatch)
	require.NoError(t, err, "Failed to write restored data to target store")

	queryTime := time.Now()
	for _, originalKeyData := range generatedKeys {
		retrievedValue, err := targetStore.Get(originalKeyData.ColumnFamily, testColumnFamilySelector, originalKeyData.Key, queryTime)
		require.NoError(t, err, "Failed to get key %s from target store", originalKeyData.Key)
		require.Equal(t, originalKeyData.Value, string(retrievedValue), "Value mismatch for key %s", originalKeyData.Key)

		if originalKeyData.IsTTL {
			expireRefKeyName := originalKeyData.Key
			// For TTL data that was just restored, we expect it to exist when queried with current time.
			// The expiry check will happen later.
			retrievedExpireRefVal, err := targetStore.Get(originalKeyData.ColumnFamily, testColumnFamilySelector, expireRefKeyName, queryTime)
			require.NoError(t, err, "Failed to get expire reference key %s for %s in CF %s", expireRefKeyName, originalKeyData.Key, originalKeyData.ColumnFamily)
			require.NotNil(t, retrievedExpireRefVal, "Expire reference value for key %s (key %s, CF %s) should exist", originalKeyData.Key, expireRefKeyName, originalKeyData.ColumnFamily)
		}
	}
	t.Log("Initial data validation complete.")

	// TTL Expiry Logic Validation
	t.Log("Validating TTL expiry logic...")
	keysToTestExpiry := []testKeyData{}
	for _, kd := range generatedKeys {
		if kd.IsTTL && len(keysToTestExpiry) < 5 { // Test a few TTL keys for expiry
			keysToTestExpiry = append(keysToTestExpiry, kd)
		}
	}

	if len(keysToTestExpiry) > 0 {
		// Wait for TTL keys to expire
		// The ExpectedTTL for these keys is testTTLSeconds (e.g., 3 seconds)
		// We need to ensure we wait long enough for the CleanExpiredKeys to work.
		// The TTL is from InsertTime.
		longestWait := time.Duration(0)
		for _, kd := range keysToTestExpiry {
			// The TTL used at insertion by the store might be the store's default (e.g. 3600s for Pebble)
			// or our testTTLSeconds if the Put method was adapted to use it.
			// For this test, we assume the store's Put for a TTL CF uses a TTL that is at least testTTLSeconds,
			// or that testTTLSeconds is the actual TTL applied.
			// The critical part is that CleanExpiredKeys respects the *actual* TTL metadata written.
			// Our kd.ExpectedTTL is testTTLSeconds. We wait for this duration.
			waitUntil := kd.InsertTime.Add(kd.ExpectedTTL)
			currentWait := time.Until(waitUntil)
			if currentWait > longestWait {
				longestWait = currentWait
			}
		}

		if longestWait > 0 {
			// Add a small buffer to ensure expiry
			waitDuration := longestWait + (2 * time.Second) // Buffer for processing
			t.Logf("Waiting for %s for TTL keys (based on ExpectedTTL of %v) to expire...", waitDuration, testTTLSeconds*time.Second)
			time.Sleep(waitDuration)
		} else {
			// If longestWait is not positive, it means keys should have already expired based on ExpectedTTL.
			// This can happen if test execution was slow. Give a minimal wait for safety.
			t.Logf("TTL keys (based on ExpectedTTL of %v) should have expired. Waiting for 2s buffer just in case.", testTTLSeconds*time.Second)
			time.Sleep(2 * time.Second)
		}
		expiryCheckTime := time.Now() // The 'now' for checking expiry *after* waiting

		t.Log("Running CleanExpiredKeys on target store...")
		err = targetStore.CleanExpiredKeys(expiryCheckTime)
		require.NoError(t, err, "Failed to run CleanExpiredKeys on target store")

		for _, kd := range keysToTestExpiry {
			retrievedValue, errGet := targetStore.Get(kd.ColumnFamily, testColumnFamilySelector, kd.Key, expiryCheckTime)
			// It's okay if errGet is pebble.ErrNotFound or similar. The main thing is retrievedValue is nil.
			if errGet != nil {
				// Log the error but don't fail if it's a 'not found' type error.
				// A more robust check would be to use errors.Is(errGet, expectedNotFoundError)
				t.Logf("Got error getting key %s post-TTL: %v (this may be expected if key is properly deleted)", kd.Key, errGet)
			}
			require.Nil(t, retrievedValue, "Key %s (CF: %s) should be nil (expired and cleaned) but got value: %s", kd.Key, kd.ColumnFamily, string(retrievedValue))

			// kd is the testKeyData for an expired key
			expireRefKeyName := db.PrefixTTLExpire + kd.Key
			retrievedExpireRefVal, err := targetStore.Get(kd.ColumnFamily, testColumnFamilySelector, expireRefKeyName, expiryCheckTime)
			require.NoError(t, err, "Error getting expire reference key %s for supposedly expired key %s (post-clean) in CF %s", expireRefKeyName, kd.Key, kd.ColumnFamily)
			require.Nil(t, retrievedExpireRefVal, "Expire reference key %s for %s (CF %s) should be nil (expired and cleaned)", expireRefKeyName, kd.Key, kd.ColumnFamily)

			// For RocksDB, data might have been stored with PrefixData.
			// If CleanExpiredKeys for RocksDB targets "PrefixData + key", and data was stored as "key", data won't be cleaned by that specific path.
			// The check for `targetStore.Get(kd.ColumnFamily, kd.Key)` already covers if the main application key is gone.
			// If RocksDB's CleanExpiredKeys is supposed to remove "PrefixData + key", we should check that too if it was ever created.
			// However, our current Put logic for both stores writes the data under `kd.Key` (application key) directly within the CF.
			// The RocksDB `cleanExpiredKeys` function *does* attempt to delete `PrefixData + originalKey`.
			// So, if data was stored as `kd.Key`, this specific delete in `cleanExpiredKeys` for `PrefixData+kd.Key` won't do anything to `kd.Key`.
			// The test implicitly verifies if `kd.Key` is cleaned. If it's not, that's a failure against expectation.
			// No extra check for "PrefixData+kd.Key" needed here unless we specifically store TTL data with that prefix.
		}
		t.Logf("%d TTL keys correctly expired and cleaned up.", len(keysToTestExpiry))
	} else {
		t.Log("No TTL keys marked for expiry test (numTTLKeys might be 0 or selection logic failed).")
	}
	t.Log("TTL expiry validation complete.")
	t.Logf("Test %s to %s finished successfully.", srcStoreType, targetStoreType)
}

func TestBackupRestore_PebbleToPebble(t *testing.T) {
	runBackupRestoreTest(t, "pebble", "pebble")
}

func TestBackupRestore_RocksDBToRocksDB(t *testing.T) {
	runBackupRestoreTest(t, "rocksdb", "rocksdb")
}

func TestBackupRestore_PebbleToRocksDB(t *testing.T) {
	runBackupRestoreTest(t, "pebble", "rocksdb")
}

func TestBackupRestore_RocksDBToPebble(t *testing.T) {
	runBackupRestoreTest(t, "rocksdb", "pebble")
}
