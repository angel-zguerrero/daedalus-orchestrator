package db_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"deadalus-orch/server/internal/infrastructure/db"
)

func TestPebbleStore_PutAndGet(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := db.CreatePebbleStore(tmpDir, []string{DefaultFC, TestFC}, nil)
	require.NoError(t, err)
	defer store.Close()

	key := "key"
	value := []byte("value")

	err = store.Put(TestFC, key, value)
	require.NoError(t, err)

	result, err := store.Get(TestFC, key)
	require.NoError(t, err)
	assert.Equal(t, value, result)
}

func TestPebbleStore_Get_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := db.CreatePebbleStore(tmpDir, []string{DefaultFC, TestFC}, nil)
	require.NoError(t, err)
	defer store.Close()

	result, err := store.Get(TestFC, "nonexistent")
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestPebbleStore_WriteBatch(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := db.CreatePebbleStore(tmpDir, []string{DefaultFC, TestFC}, nil)
	require.NoError(t, err)
	defer store.Close()

	batch := db.NewWriteBatch()
	batch.Put(TestFC, "a", []byte("valueA"))
	batch.Put(TestFC, "b", []byte("valueB"))

	err = store.Write(batch)
	require.NoError(t, err)

	resultA, err := store.Get(TestFC, "a")
	require.NoError(t, err)
	assert.Equal(t, []byte("valueA"), resultA)

	resultB, err := store.Get(TestFC, "b")
	require.NoError(t, err)
	assert.Equal(t, []byte("valueB"), resultB)
}

func TestPebbleStore_SearchByPatternPaginatedKV_MatchSingle(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := db.CreatePebbleStore(tmpDir, []string{DefaultFC, TestFC}, nil)
	require.NoError(t, err)
	defer store.Close()

	require.NoError(t, store.Put(TestFC, "user:123:name", []byte("Alice")))

	results, next, err := store.SearchByPatternPaginatedKV(TestFC, "user:123:*", "", 10)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "user:123:name", results[0].Key)
	assert.Equal(t, []byte("Alice"), results[0].Value)
	assert.Equal(t, "", next)
}

func TestPebbleStore_SearchByPatternPaginatedKV_MatchMultiplePages(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := db.CreatePebbleStore(tmpDir, []string{DefaultFC, TestFC}, nil)
	require.NoError(t, err)
	defer store.Close()

	require.NoError(t, store.Put(TestFC, "user:1", []byte("a")))
	require.NoError(t, store.Put(TestFC, "user:2", []byte("b")))
	require.NoError(t, store.Put(TestFC, "user:3", []byte("c")))

	var all []db.KeyValuePair
	cursor := ""
	for {
		page, next, err := store.SearchByPatternPaginatedKV(TestFC, "user:*", cursor, 2)
		require.NoError(t, err)
		all = append(all, page...)
		if next == "" {
			break
		}
		cursor = next
	}
	require.Len(t, all, 3)
}

func TestPebbleStore_SearchByPatternPaginatedKV_NoMatch(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := db.CreatePebbleStore(tmpDir, []string{DefaultFC, TestFC}, nil)
	require.NoError(t, err)
	defer store.Close()

	require.NoError(t, store.Put(TestFC, "product:1", []byte("item")))

	results, next, err := store.SearchByPatternPaginatedKV(TestFC, "user:*", "", 10)
	require.NoError(t, err)
	require.Empty(t, results)
	require.Equal(t, "", next)
}

func TestPebbleStore_SearchByPatternPaginatedKV_InvalidColumnFamily(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := db.CreatePebbleStore(tmpDir, []string{DefaultFC}, nil)
	require.NoError(t, err)
	defer store.Close()

	_, _, err = store.SearchByPatternPaginatedKV("nonexistent", "pattern:*", "", 10)
	require.Error(t, err)
	require.Contains(t, err.Error(), "column family")
}

func TestPebbleStore_Delete_ExistingKey(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := db.CreatePebbleStore(tmpDir, []string{DefaultFC, TestFC}, nil)
	require.NoError(t, err)
	defer store.Close()

	key := "delete-key"
	value := []byte("to-delete")

	require.NoError(t, store.Put(TestFC, key, value))
	require.NoError(t, store.Delete(TestFC, key))

	result, err := store.Get(TestFC, key)
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestPebbleStore_Delete_NonExistentKey(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := db.CreatePebbleStore(tmpDir, []string{DefaultFC, TestFC}, nil)
	require.NoError(t, err)
	defer store.Close()

	err = store.Delete(TestFC, "nonexistent")
	assert.NoError(t, err)
}

func TestPebbleStore_Delete_InvalidColumnFamily(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := db.CreatePebbleStore(tmpDir, []string{DefaultFC}, nil)
	require.NoError(t, err)
	defer store.Close()

	err = store.Delete("nonexistent_cf", "key")
	require.Error(t, err)
	require.Contains(t, err.Error(), "column family")
}

func TestPebbleStore_Delete_TTLColumnFamily(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := db.CreatePebbleStore(tmpDir, []string{}, []string{TestFC})
	require.NoError(t, err)
	defer store.Close()

	key := "ttl-key"
	value := []byte("ttl-value")

	require.NoError(t, store.Put(TestFC, key, value))
	require.NoError(t, store.Delete(TestFC, key))

	result, err := store.Get(TestFC, key)
	require.NoError(t, err)
	assert.Nil(t, result)
}
