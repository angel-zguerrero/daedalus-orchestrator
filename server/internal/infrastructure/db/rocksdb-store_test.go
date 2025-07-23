//go:build rocksdb
// +build rocksdb

package db_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"deadalus-orch/server/internal/infrastructure/db"
)

func TestRocksdbStore_PutAndGet(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := db.CreateRocksdbStore(tmpDir, []string{DefaultFC, TestFC}, nil)
	require.NoError(t, err)
	defer store.Close()

	key := "key"
	value := []byte("value")
	now := time.Now()

	err = store.Put(TestFC, testColumnFamilySector, key, value, 0, now)
	require.NoError(t, err)

	result, err := store.Get(TestFC, testColumnFamilySector, key, now)
	require.NoError(t, err)
	assert.Equal(t, value, result)
}

func TestRocksdbStore_Get_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := db.CreateRocksdbStore(tmpDir, []string{DefaultFC, TestFC}, nil)
	require.NoError(t, err)
	defer store.Close()
	now := time.Now()

	result, err := store.Get(TestFC, testColumnFamilySector, "nonexistent", now)
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestRocksdbStore_WriteBatch(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := db.CreateRocksdbStore(tmpDir, []string{DefaultFC, TestFC}, nil)
	require.NoError(t, err)
	defer store.Close()
	now := time.Now()

	batch := db.NewWriteBatch()
	batch.Put(TestFC, testColumnFamilySector, "a", []byte("valueA"), now)
	batch.Put(TestFC, testColumnFamilySector, "b", []byte("valueB"), now)

	err = store.Write(batch)
	require.NoError(t, err)

	resultA, err := store.Get(TestFC, testColumnFamilySector, "a", now)
	require.NoError(t, err)
	assert.Equal(t, []byte("valueA"), resultA)

	resultB, err := store.Get(TestFC, testColumnFamilySector, "b", now)
	require.NoError(t, err)
	assert.Equal(t, []byte("valueB"), resultB)
}

func TestRocksdbStore_SearchByPatternPaginatedKV_MatchSingle(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := db.CreateRocksdbStore(tmpDir, []string{DefaultFC, TestFC}, nil)
	require.NoError(t, err)
	defer store.Close()
	now := time.Now()

	require.NoError(t, store.Put(TestFC, testColumnFamilySector, "user:123:name", []byte("Alice"), 0, now))

	results, next, err := store.SearchByPatternPaginatedKV(TestFC, testColumnFamilySector, "user:123:*", "", 10, now)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "user:123:name", results[0].Key)
	assert.Equal(t, []byte("Alice"), results[0].Value)
	assert.Equal(t, "", next)
}

func TestRocksdbStore_SearchByPatternPaginatedKV_MatchMultiplePages(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := db.CreateRocksdbStore(tmpDir, []string{DefaultFC, TestFC}, nil)
	require.NoError(t, err)
	defer store.Close()
	now := time.Now()

	require.NoError(t, store.Put(TestFC, testColumnFamilySector, "user:1", []byte("a"), 0, now))
	require.NoError(t, store.Put(TestFC, testColumnFamilySector, "user:2", []byte("b"), 0, now))
	require.NoError(t, store.Put(TestFC, testColumnFamilySector, "user:3", []byte("c"), 0, now))

	var all []db.KeyValuePair
	cursor := ""
	for {
		page, next, err := store.SearchByPatternPaginatedKV(TestFC, testColumnFamilySector, "user:*", cursor, 2, now)
		require.NoError(t, err)
		all = append(all, page...)
		if next == "" {
			break
		}
		cursor = next
	}
	require.Len(t, all, 3)
}

func TestRocksdbStore_SearchByPatternPaginatedKV_NoMatch(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := db.CreateRocksdbStore(tmpDir, []string{DefaultFC, TestFC}, nil)
	require.NoError(t, err)
	defer store.Close()
	now := time.Now()

	require.NoError(t, store.Put(TestFC, testColumnFamilySector, "product:1", []byte("item"), 0, now))

	results, next, err := store.SearchByPatternPaginatedKV(TestFC, testColumnFamilySector, "user:*", "", 10, now)
	require.NoError(t, err)
	require.Empty(t, results)
	require.Equal(t, "", next)
}

func TestRocksdbStore_SearchByPatternPaginatedKV_InvalidColumnFamily(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := db.CreateRocksdbStore(tmpDir, []string{DefaultFC}, nil)
	require.NoError(t, err)
	defer store.Close()
	now := time.Now()

	_, _, err = store.SearchByPatternPaginatedKV("nonexistent", testColumnFamilySector, "pattern:*", "", 10, now)
	require.Error(t, err)
	require.Contains(t, err.Error(), "column family")
}

func TestRocksdbStore_Delete_ExistingKey(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := db.CreateRocksdbStore(tmpDir, []string{DefaultFC, TestFC}, nil)
	require.NoError(t, err)
	defer store.Close()
	now := time.Now()

	key := "delete-key"
	value := []byte("to-delete")

	require.NoError(t, store.Put(TestFC, testColumnFamilySector, key, value, 0, now))
	require.NoError(t, store.Delete(TestFC, testColumnFamilySector, key, now))

	result, err := store.Get(TestFC, testColumnFamilySector, key, now)
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestRocksdbStore_Delete_NonExistentKey(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := db.CreateRocksdbStore(tmpDir, []string{DefaultFC, TestFC}, nil)
	require.NoError(t, err)
	defer store.Close()
	now := time.Now()

	err = store.Delete(TestFC, testColumnFamilySector, "nonexistent", now)
	assert.NoError(t, err)
}

func TestRocksdbStore_Delete_InvalidColumnFamily(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := db.CreateRocksdbStore(tmpDir, []string{DefaultFC}, nil)
	require.NoError(t, err)
	defer store.Close()
	now := time.Now()

	err = store.Delete("nonexistent_cf", testColumnFamilySector, "key", now)
	require.Error(t, err)
	require.Contains(t, err.Error(), "column family")
}

func TestRocksdbStore_Delete_TTLColumnFamily(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := db.CreateRocksdbStore(tmpDir, []string{}, []string{TestFC})
	require.NoError(t, err)
	defer store.Close()
	now := time.Now()

	key := "ttl-key"
	value := []byte("ttl-value")

	require.NoError(t, store.Put(TestFC, testColumnFamilySector, key, value, 3, now))
	require.NoError(t, store.Delete(TestFC, testColumnFamilySector, key, now))

	result, err := store.Get(TestFC, testColumnFamilySector, key, now)
	require.NoError(t, err)
	assert.Nil(t, result)

	dumpX, err := store.DumpAll()
	require.NoError(t, err)

	// The dump should be empty or not contain the test column family
	assert.Empty(t, dumpX)
	dump := dumpX.(map[string]map[string][]byte)
	for cf, kvs := range dump {
		for key := range kvs {
			assert.NotContains(t, key, key, fmt.Sprintf("Key %s in CF %s should not contain deleted TTL key", key, cf))
		}
	}
}

func TestRocksdbStore_Delete_TTLColumnFamilyWaitForTTL(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := db.CreateRocksdbStore(tmpDir, []string{}, []string{TestFC})
	require.NoError(t, err)
	defer store.Close()
	now := time.Now()

	key := "ttl-key"
	value := []byte("ttl-value")

	require.NoError(t, store.Put(TestFC, testColumnFamilySector, key, value, 3, now))

	time.Sleep(3 * time.Second)
	afterSleepNow := time.Now()

	require.NoError(t, store.Delete(TestFC, testColumnFamilySector, key, afterSleepNow))

	result, err := store.Get(TestFC, testColumnFamilySector, key, afterSleepNow)
	require.NoError(t, err)
	assert.Nil(t, result)

	dumpX, err := store.DumpAll()
	require.NoError(t, err)

	// The dump should be empty or not contain the test column family
	assert.Empty(t, dumpX)
	dump := dumpX.(map[string]map[string][]byte)
	for cf, kvs := range dump {
		for key := range kvs {
			assert.NotContains(t, key, key, fmt.Sprintf("Key %s in CF %s should not contain deleted TTL key", key, cf))
		}
	}
}

func TestRocksdbStore_ColumnFamilyOperations(t *testing.T) {
	tmpDir := t.TempDir()
	// Start with only DefaultFC, TestFC will be created dynamically by tests if needed by Put/Get
	// For CF operations, we want to explicitly create them.
	store, err := db.CreateRocksdbStore(tmpDir, []string{DefaultFC}, nil)
	require.NoError(t, err)
	defer store.Close()

	cfName := "new_cf_rocks"
	cfNameTTL := "new_cf_ttl_rocks"

	// 1. ExistsColumnFamily - initially not found
	exists, isTTL, err := store.ExistsColumnFamily(cfName)
	require.NoError(t, err)
	assert.False(t, exists)
	assert.False(t, isTTL)

	// 2. CreateColumnFamily - non-TTL
	err = store.CreateColumnFamily(cfName, false)
	require.NoError(t, err)

	// 3. ExistsColumnFamily - non-TTL should exist
	exists, isTTL, err = store.ExistsColumnFamily(cfName)
	require.NoError(t, err)
	assert.True(t, exists)
	assert.False(t, isTTL)

	// 4. CreateColumnFamily - TTL
	err = store.CreateColumnFamily(cfNameTTL, true)
	require.NoError(t, err)

	// 5. ExistsColumnFamily - TTL should exist
	exists, isTTL, err = store.ExistsColumnFamily(cfNameTTL)
	require.NoError(t, err)
	assert.True(t, exists)
	assert.True(t, isTTL)

	// 6. CreateColumnFamily - already exists
	err = store.CreateColumnFamily(cfName, false)
	require.Error(t, err) // Should error as it already exists

	// 7. Put/Get in new non-TTL CF
	key1, val1 := "key1_rocks", []byte("val1_rocks")
	err = store.Put(cfName, testColumnFamilySector, key1, val1, 0, time.Now())
	require.NoError(t, err)
	retVal1, err := store.Get(cfName, testColumnFamilySector, key1, time.Now())
	require.NoError(t, err)
	assert.Equal(t, val1, retVal1)

	// 8. Put/Get in new TTL CF
	key2, val2 := "key2_rocks", []byte("val2_rocks")
	// Using a short TTL for testing if needed, but basic Put/Get doesn't rely on TTL expiry for this test
	err = store.Put(cfNameTTL, testColumnFamilySector, key2, val2, 3600, time.Now())
	require.NoError(t, err)
	retVal2, err := store.Get(cfNameTTL, testColumnFamilySector, key2, time.Now())
	require.NoError(t, err)
	assert.Equal(t, val2, retVal2)

	// 9. DeleteColumnFamily - non-TTL
	err = store.DeleteColumnFamily(cfName)
	require.NoError(t, err)

	// 10. ExistsColumnFamily - non-TTL should not exist
	exists, isTTL, err = store.ExistsColumnFamily(cfName)
	require.NoError(t, err)
	assert.False(t, exists)
	assert.False(t, isTTL)

	// 11. Get from deleted non-TTL CF should fail
	_, err = store.Get(cfName, testColumnFamilySector, key1, time.Now())
	require.Error(t, err) // Expect an error as CF is gone

	// 12. DeleteColumnFamily - TTL
	err = store.DeleteColumnFamily(cfNameTTL)
	require.NoError(t, err)

	// 13. ExistsColumnFamily - TTL should not exist
	exists, isTTL, err = store.ExistsColumnFamily(cfNameTTL)
	require.NoError(t, err)
	assert.False(t, exists)
	assert.False(t, isTTL)

	// 14. DeleteColumnFamily - already deleted / not found
	err = store.DeleteColumnFamily(cfName)
	require.Error(t, err) // Should error as it's already deleted
}

func TestRocksdbStore_CleanExpiredKeys(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := db.CreateRocksdbStore(tmpDir, []string{}, []string{TestFC})
	require.NoError(t, err)
	defer store.Close()

	key := "my-ttl-key"
	value := []byte("some data")
	now := time.Now()

	// Put a key with a 1-second TTL
	err = store.Put(TestFC, testColumnFamilySector, key, value, 1, now)
	require.NoError(t, err)
	// Wait for the key to expire
	time.Sleep(2 * time.Second)

	// Clean expired keys
	err = store.CleanExpiredKeys(time.Now())
	require.NoError(t, err)

	// Dump all data and verify that the keys are gone
	dumpX, err := store.DumpAll()
	require.NoError(t, err)

	// The dump should be empty
	assert.Empty(t, dumpX)

	dump := dumpX.(map[string]map[string][]byte)
	for cf, kvs := range dump {
		for key := range kvs {
			assert.NotContains(t, key, key, fmt.Sprintf("Key %s in CF %s should not contain deleted TTL key", key, cf))
		}
	}
}

func TestRocksdbStore_CleanExpiredKeysWithBatch(t *testing.T) {
	tmpDir := t.TempDir()
	//store, err := db.CreatePebbleStore(tmpDir, []string{"non_ttl_cf"}, []string{TestFC})
	store, err := db.CreateRocksdbStore(tmpDir, []string{"non_ttl_cf"}, []string{TestFC})
	require.NoError(t, err)
	defer store.Close()

	now := time.Now()
	expiredKey := "expired-key"
	validTTLKey := "valid-ttl-key"
	nonTTLKey := "non-ttl-key"

	batch := db.NewWriteBatch()
	batch.PutTTl(TestFC, testColumnFamilySector, expiredKey, []byte("will expire"), 1, now)     // 1-second TTL
	batch.PutTTl(TestFC, testColumnFamilySector, validTTLKey, []byte("should remain"), 30, now) // 30-second TTL
	batch.Put("non_ttl_cf", testColumnFamilySector, nonTTLKey, []byte("no ttl"), now)           // No TTL
	err = store.Write(batch)
	require.NoError(t, err)

	// Wait for the short TTL to expire
	time.Sleep(2 * time.Second)

	// Clean expired keys
	err = store.CleanExpiredKeys(time.Now())
	require.NoError(t, err)

	// Dump all data and verify
	dumpX, err := store.DumpAll()
	require.NoError(t, err)
	dump := dumpX.(map[string]map[string][]byte)
	// Check that the expired key is gone
	fullColumnFamily := TestFC + ":test-sector"
	_, cfExists := dump[fullColumnFamily]

	if cfExists {
		_, keyExists := dump[fullColumnFamily][expiredKey]
		assert.False(t, keyExists, "Expired key should have been removed")
	}

	// Check that the valid TTL key is still present
	assert.NotNil(t, dump[fullColumnFamily][validTTLKey], "Valid TTL key should be present")

	// Check that the non-TTL key is still present
	assert.NotNil(t, dump["non_ttl_cf:test-sector"][nonTTLKey], "Non-TTL key should be present")

	for cf, kvs := range dump {
		for key := range kvs {
			assert.NotContains(t, key, expiredKey, fmt.Sprintf("Key %s in CF %s should not contain deleted TTL key", key, cf))
		}
	}
}

func TestRocksdbStore_DeleteWithBatch(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := db.CreateRocksdbStore(tmpDir, []string{"non_ttl_cf"}, []string{TestFC})
	require.NoError(t, err)
	defer store.Close()

	now := time.Now()
	ttlKeyToDelete := "ttl-key-to-delete"
	ttlKeyToKeep := "ttl-key-to-keep"
	nonTTLKey := "non-ttl-key"

	// Batch write TTL and non-TTL entries
	batch := db.NewWriteBatch()
	batch.PutTTl(TestFC, testColumnFamilySector, ttlKeyToDelete, []byte("delete-me"), 30, now)
	batch.PutTTl(TestFC, testColumnFamilySector, ttlKeyToKeep, []byte("keep-me"), 30, now)
	batch.Put("non_ttl_cf", testColumnFamilySector, nonTTLKey, []byte("no-ttl"), now)
	err = store.Write(batch)
	require.NoError(t, err)

	// Delete one of the TTL keys
	err = store.Delete(TestFC, testColumnFamilySector, ttlKeyToDelete, now)
	require.NoError(t, err)

	// Dump all data and verify the state
	dumpX, err := store.DumpAll()
	require.NoError(t, err)
	dump := dumpX.(map[string]map[string][]byte)
	fullColumnFamily := TestFC + ":test-sector"

	// Check that the deleted TTL key is gone
	_, cfExists := dump[fullColumnFamily]
	assert.True(t, cfExists, "Column family should exist")
	if cfExists {
		_, keyExists := dump[fullColumnFamily][ttlKeyToDelete]
		assert.False(t, keyExists, "Deleted TTL key should not be present")
	}

	// Check that the other TTL key is still present
	assert.NotNil(t, dump[fullColumnFamily][ttlKeyToKeep], "Kept TTL key should be present")

	// Check that the non-TTL key is still present
	assert.NotNil(t, dump["non_ttl_cf:test-sector"][nonTTLKey], "Non-TTL key should be present")

	for cf, kvs := range dump {
		for key := range kvs {
			assert.NotContains(t, key, ttlKeyToDelete, fmt.Sprintf("Key %s in CF %s should not contain deleted TTL key", key, cf))
		}
	}
}

func TestRocksdbStore_BatchDeletion(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := db.CreateRocksdbStore(tmpDir, []string{"non_ttl_cf"}, []string{TestFC})
	require.NoError(t, err)
	defer store.Close()

	now := time.Now()
	ttlKeyToDelete := "ttl-key-to-delete"
	ttlKeyToKeep := "ttl-key-to-keep"
	nonTTLKey := "non-ttl-key"

	// Batch write TTL and non-TTL entries
	batch := db.NewWriteBatch()
	batch.PutTTl(TestFC, testColumnFamilySector, ttlKeyToDelete, []byte("delete-me"), 30, now)
	batch.PutTTl(TestFC, testColumnFamilySector, ttlKeyToKeep, []byte("keep-me"), 30, now)
	batch.Put("non_ttl_cf", testColumnFamilySector, nonTTLKey, []byte("no-ttl"), now)
	err = store.Write(batch)
	require.NoError(t, err)

	// Delete one of the TTL keys

	batchDeletion := db.NewWriteBatch()
	batchDeletion.Delete(TestFC, testColumnFamilySector, ttlKeyToDelete, now)
	err = store.Write(batchDeletion)
	require.NoError(t, err)

	// Dump all data and verify the state
	dumpX, err := store.DumpAll()
	require.NoError(t, err)
	dump := dumpX.(map[string]map[string][]byte)

	fullColumnFamily := TestFC + ":test-sector"

	_, cfExists := dump[fullColumnFamily]
	assert.True(t, cfExists, "Column family should exist")
	if cfExists {
		_, keyExists := dump[fullColumnFamily][ttlKeyToDelete]
		assert.False(t, keyExists, "Deleted TTL key should not be present")
	}

	// Check that the other TTL key is still present
	assert.NotNil(t, dump[fullColumnFamily][ttlKeyToKeep], "Kept TTL key should be present")

	// Check that the non-TTL key is still present
	assert.NotNil(t, dump["non_ttl_cf:test-sector"][nonTTLKey], "Non-TTL key should be present")

	// Ensure no key in the dump contains "ttl-key-to-delete"
	for cf, kvs := range dump {
		for key := range kvs {
			assert.NotContains(t, key, ttlKeyToDelete, fmt.Sprintf("Key %s in CF %s should not contain deleted TTL key", key, cf))
		}
	}
}

func TestRocksdbStore_DeleteWithBatchWaitForTTL(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := db.CreateRocksdbStore(tmpDir, []string{"non_ttl_cf"}, []string{TestFC})
	require.NoError(t, err)
	defer store.Close()

	now := time.Now()
	ttlKeyToDelete := "ttl-key-to-delete"
	ttlKeyToKeep := "ttl-key-to-keep"
	nonTTLKey := "non-ttl-key"

	// Batch write TTL and non-TTL entries
	batch := db.NewWriteBatch()
	batch.PutTTl(TestFC, testColumnFamilySector, ttlKeyToDelete, []byte("delete-me"), 3, now)
	batch.PutTTl(TestFC, testColumnFamilySector, ttlKeyToKeep, []byte("keep-me"), 30, now)
	batch.Put("non_ttl_cf", testColumnFamilySector, nonTTLKey, []byte("no-ttl"), now)
	err = store.Write(batch)
	require.NoError(t, err)

	time.Sleep(4 * time.Second) // Wait for the TTL of the key to delete to expire
	afterSleepNow := time.Now()

	// Delete one of the TTL keys
	err = store.Delete(TestFC, testColumnFamilySector, ttlKeyToDelete, afterSleepNow)
	require.NoError(t, err)

	// Dump all data and verify the state
	dumpX, err := store.DumpAll()
	require.NoError(t, err)
	dump := dumpX.(map[string]map[string][]byte)

	fullColumnFamily := TestFC + ":test-sector"

	_, cfExists := dump[fullColumnFamily]
	assert.True(t, cfExists, "Column family should exist")
	if cfExists {
		_, keyExists := dump[fullColumnFamily][ttlKeyToDelete]
		assert.False(t, keyExists, "Deleted TTL key should not be present")
	}

	// Check that the other TTL key is still present
	assert.NotNil(t, dump[fullColumnFamily][ttlKeyToKeep], "Kept TTL key should be present")

	// Check that the non-TTL key is still present
	assert.NotNil(t, dump["non_ttl_cf:test-sector"][nonTTLKey], "Non-TTL key should be present")

	// Ensure no key in the dump contains "ttl-key-to-delete"
	for cf, kvs := range dump {
		for key := range kvs {
			assert.NotContains(t, key, ttlKeyToDelete, fmt.Sprintf("Key %s in CF %s should not contain deleted TTL key", key, cf))
		}
	}
}

func TestRocksdbStore_BatchDeletionWaitForTTL(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := db.CreateRocksdbStore(tmpDir, []string{"non_ttl_cf"}, []string{TestFC})
	require.NoError(t, err)
	defer store.Close()

	now := time.Now()
	ttlKeyToDelete := "ttl-key-to-delete"
	ttlKeyToKeep := "ttl-key-to-keep"
	nonTTLKey := "non-ttl-key"

	// Batch write TTL and non-TTL entries
	batch := db.NewWriteBatch()
	batch.PutTTl(TestFC, testColumnFamilySector, ttlKeyToDelete, []byte("delete-me"), 3, now)
	batch.PutTTl(TestFC, testColumnFamilySector, ttlKeyToKeep, []byte("keep-me"), 30, now)
	batch.Put("non_ttl_cf", testColumnFamilySector, nonTTLKey, []byte("no-ttl"), now)
	err = store.Write(batch)
	require.NoError(t, err)

	time.Sleep(4 * time.Second) // Wait for the TTL of the key to delete to expire
	afterSleepNow := time.Now()

	// Delete one of the TTL keys

	batchDeletion := db.NewWriteBatch()
	batchDeletion.Delete(TestFC, testColumnFamilySector, ttlKeyToDelete, afterSleepNow)
	err = store.Write(batchDeletion)
	require.NoError(t, err)

	// Dump all data and verify the state
	dumpX, err := store.DumpAll()
	require.NoError(t, err)
	dump := dumpX.(map[string]map[string][]byte)

	fmt.Println(dump)

	fullColumnFamily := TestFC + ":test-sector"

	_, cfExists := dump[fullColumnFamily]
	assert.True(t, cfExists, "Column family should exist")
	if cfExists {
		_, keyExists := dump[fullColumnFamily][ttlKeyToDelete]
		assert.False(t, keyExists, "Deleted TTL key should not be present")
	}

	// Check that the other TTL key is still present
	assert.NotNil(t, dump[fullColumnFamily][ttlKeyToKeep], "Kept TTL key should be present")

	// Check that the non-TTL key is still present
	assert.NotNil(t, dump["non_ttl_cf:test-sector"][nonTTLKey], "Non-TTL key should be present")

	// Ensure no key in the dump contains "ttl-key-to-delete"
	for cf, kvs := range dump {
		for key := range kvs {
			assert.NotContains(t, key, ttlKeyToDelete, fmt.Sprintf("Key %s in CF %s should not contain deleted TTL key", key, cf))
		}
	}
}
