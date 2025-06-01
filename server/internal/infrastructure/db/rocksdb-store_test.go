package db_test

import (
	"testing"

	"github.com/linxGnu/grocksdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"deadalus-orch/server/internal/infrastructure/db"
)

const TestFC = "test_fc"
const DefaultFC = "default"

func TestRocksdbStore_PutAndGet(t *testing.T) {
	tmpDir := t.TempDir()

	opts := grocksdb.NewDefaultOptions()
	opts.SetCreateIfMissing(true)
	opts.SetCreateIfMissingColumnFamilies(true)

	goOp := grocksdb.NewDefaultOptions()

	rocks, cfHs, err := grocksdb.OpenDbColumnFamilies(opts, tmpDir, []string{DefaultFC, TestFC}, []*grocksdb.Options{goOp, goOp})
	require.NoError(t, err)
	defer rocks.Close()

	columnFamilyNames, err := grocksdb.ListColumnFamilies(opts, tmpDir)
	require.NoError(t, err)

	ColumnFamilyHandles := make(map[string]*grocksdb.ColumnFamilyHandle, len(columnFamilyNames))
	for index, name := range columnFamilyNames {
		ColumnFamilyHandles[name] = cfHs[index]
	}

	store := &db.RocksdbStore{DB: rocks, ColumnFamilyHandles: ColumnFamilyHandles}

	key := "key"
	value := []byte("value")

	err = store.Put(TestFC, key, value)
	require.NoError(t, err)

	result, err := store.Get(TestFC, key)
	require.NoError(t, err)
	assert.Equal(t, value, result)
}

func TestRocksdbStore_Get_NotFound(t *testing.T) {
	tmpDir := t.TempDir()

	opts := grocksdb.NewDefaultOptions()
	opts.SetCreateIfMissing(true)
	opts.SetCreateIfMissingColumnFamilies(true)

	goOp := grocksdb.NewDefaultOptions()

	rocks, cfHs, err := grocksdb.OpenDbColumnFamilies(opts, tmpDir, []string{DefaultFC, TestFC}, []*grocksdb.Options{goOp, goOp})
	require.NoError(t, err)
	defer rocks.Close()

	columnFamilyNames, err := grocksdb.ListColumnFamilies(opts, tmpDir)
	require.NoError(t, err)

	ColumnFamilyHandles := make(map[string]*grocksdb.ColumnFamilyHandle, len(columnFamilyNames))
	for index, name := range columnFamilyNames {
		ColumnFamilyHandles[name] = cfHs[index]
	}

	store := &db.RocksdbStore{DB: rocks, ColumnFamilyHandles: ColumnFamilyHandles}

	result, err := store.Get(TestFC, "nonexistent")
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestRocksdbStore_WriteBatch(t *testing.T) {
	tmpDir := t.TempDir()

	opts := grocksdb.NewDefaultOptions()
	opts.SetCreateIfMissing(true)
	opts.SetCreateIfMissingColumnFamilies(true)

	cfOpts := grocksdb.NewDefaultOptions()

	rocks, cfHs, err := grocksdb.OpenDbColumnFamilies(opts, tmpDir, []string{DefaultFC, TestFC}, []*grocksdb.Options{cfOpts, cfOpts})
	require.NoError(t, err)
	defer rocks.Close()

	columnFamilyNames, err := grocksdb.ListColumnFamilies(opts, tmpDir)
	require.NoError(t, err)

	columnFamilyHandles := make(map[string]*grocksdb.ColumnFamilyHandle, len(columnFamilyNames))
	for index, name := range columnFamilyNames {
		columnFamilyHandles[name] = cfHs[index]
	}

	store := &db.RocksdbStore{
		DB:                     rocks,
		ColumnFamilyHandles:    columnFamilyHandles,
		TTLColumnFamilyHandles: make(map[string]*grocksdb.ColumnFamilyHandle),
	}

	// Usar nueva estructura WriteBatch
	batch := db.NewWriteBatch()
	batch.Put(TestFC, "a", []byte("valueA"))
	batch.Put(TestFC, "b", []byte("valueB"))

	err = store.Write(batch)
	require.NoError(t, err)

	// Verifica clave "a"
	resultA, err := store.Get(TestFC, "a")
	require.NoError(t, err)
	assert.Equal(t, []byte("valueA"), resultA)

	// Verifica clave "b"
	resultB, err := store.Get(TestFC, "b")
	require.NoError(t, err)
	assert.Equal(t, []byte("valueB"), resultB)
}

func TestRocksdbStore_SearchByPatternPaginatedKV_MatchSingle(t *testing.T) {
	tmpDir := t.TempDir()

	opts := grocksdb.NewDefaultOptions()
	opts.SetCreateIfMissing(true)
	opts.SetCreateIfMissingColumnFamilies(true)
	goOp := grocksdb.NewDefaultOptions()

	rocks, cfHs, err := grocksdb.OpenDbColumnFamilies(opts, tmpDir, []string{DefaultFC, TestFC}, []*grocksdb.Options{goOp, goOp})
	require.NoError(t, err)
	defer rocks.Close()

	columnFamilyNames, err := grocksdb.ListColumnFamilies(opts, tmpDir)
	require.NoError(t, err)

	cfMap := make(map[string]*grocksdb.ColumnFamilyHandle, len(columnFamilyNames))
	for i, name := range columnFamilyNames {
		cfMap[name] = cfHs[i]
	}

	store := &db.RocksdbStore{DB: rocks, ColumnFamilyHandles: cfMap}

	require.NoError(t, store.Put(TestFC, "user:123:name", []byte("Alice")))

	results, next, err := store.SearchByPatternPaginatedKV(TestFC, "user:123:*", "", 10)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "user:123:name", results[0].Key)
	assert.Equal(t, []byte("Alice"), results[0].Value)
	assert.Equal(t, "", next)
}
func TestRocksdbStore_SearchByPatternPaginatedKV_MatchMultiplePages(t *testing.T) {
	tmpDir := t.TempDir()

	opts := grocksdb.NewDefaultOptions()
	opts.SetCreateIfMissing(true)
	opts.SetCreateIfMissingColumnFamilies(true)
	goOp := grocksdb.NewDefaultOptions()

	rocks, cfHs, err := grocksdb.OpenDbColumnFamilies(opts, tmpDir, []string{DefaultFC, TestFC}, []*grocksdb.Options{goOp, goOp})
	require.NoError(t, err)
	defer rocks.Close()

	columnFamilyNames, err := grocksdb.ListColumnFamilies(opts, tmpDir)
	require.NoError(t, err)

	cfMap := make(map[string]*grocksdb.ColumnFamilyHandle, len(columnFamilyNames))
	for i, name := range columnFamilyNames {
		cfMap[name] = cfHs[i]
	}

	store := &db.RocksdbStore{DB: rocks, ColumnFamilyHandles: cfMap}

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
func TestRocksdbStore_SearchByPatternPaginatedKV_NoMatch(t *testing.T) {
	tmpDir := t.TempDir()

	opts := grocksdb.NewDefaultOptions()
	opts.SetCreateIfMissing(true)
	opts.SetCreateIfMissingColumnFamilies(true)
	goOp := grocksdb.NewDefaultOptions()

	rocks, cfHs, err := grocksdb.OpenDbColumnFamilies(opts, tmpDir, []string{DefaultFC, TestFC}, []*grocksdb.Options{goOp, goOp})
	require.NoError(t, err)
	defer rocks.Close()

	columnFamilyNames, err := grocksdb.ListColumnFamilies(opts, tmpDir)
	require.NoError(t, err)

	cfMap := make(map[string]*grocksdb.ColumnFamilyHandle, len(columnFamilyNames))
	for i, name := range columnFamilyNames {
		cfMap[name] = cfHs[i]
	}

	store := &db.RocksdbStore{DB: rocks, ColumnFamilyHandles: cfMap}

	require.NoError(t, store.Put(TestFC, "product:1", []byte("item")))

	results, next, err := store.SearchByPatternPaginatedKV(TestFC, "user:*", "", 10)
	require.NoError(t, err)
	require.Empty(t, results)
	require.Equal(t, "", next)
}

func TestRocksdbStore_SearchByPatternPaginatedKV_InvalidColumnFamily(t *testing.T) {
	tmpDir := t.TempDir()

	opts := grocksdb.NewDefaultOptions()
	opts.SetCreateIfMissing(true)
	opts.SetCreateIfMissingColumnFamilies(true)
	goOp := grocksdb.NewDefaultOptions()

	rocks, _, err := grocksdb.OpenDbColumnFamilies(opts, tmpDir, []string{DefaultFC}, []*grocksdb.Options{goOp})
	require.NoError(t, err)
	defer rocks.Close()

	store := &db.RocksdbStore{DB: rocks, ColumnFamilyHandles: map[string]*grocksdb.ColumnFamilyHandle{}}

	_, _, err = store.SearchByPatternPaginatedKV("nonexistent", "pattern:*", "", 10)
	require.Error(t, err)
	require.Contains(t, err.Error(), "column family")
}

func TestRocksdbStore_Delete_ExistingKey(t *testing.T) {
	tmpDir := t.TempDir()

	opts := grocksdb.NewDefaultOptions()
	opts.SetCreateIfMissing(true)
	opts.SetCreateIfMissingColumnFamilies(true)

	cfOpts := grocksdb.NewDefaultOptions()
	rocks, cfHs, err := grocksdb.OpenDbColumnFamilies(opts, tmpDir, []string{DefaultFC, TestFC}, []*grocksdb.Options{cfOpts, cfOpts})
	require.NoError(t, err)
	defer rocks.Close()

	cfMap := make(map[string]*grocksdb.ColumnFamilyHandle)
	for i, name := range []string{DefaultFC, TestFC} {
		cfMap[name] = cfHs[i]
	}

	store := &db.RocksdbStore{DB: rocks, ColumnFamilyHandles: cfMap}

	key := "delete-key"
	value := []byte("to-delete")

	require.NoError(t, store.Put(TestFC, key, value))

	// Delete the key
	require.NoError(t, store.Delete(TestFC, key))

	// Verify it's deleted
	result, err := store.Get(TestFC, key)
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestRocksdbStore_Delete_NonExistentKey(t *testing.T) {
	tmpDir := t.TempDir()

	opts := grocksdb.NewDefaultOptions()
	opts.SetCreateIfMissing(true)
	opts.SetCreateIfMissingColumnFamilies(true)

	cfOpts := grocksdb.NewDefaultOptions()
	rocks, cfHs, err := grocksdb.OpenDbColumnFamilies(opts, tmpDir, []string{DefaultFC, TestFC}, []*grocksdb.Options{cfOpts, cfOpts})
	require.NoError(t, err)
	defer rocks.Close()

	cfMap := make(map[string]*grocksdb.ColumnFamilyHandle)
	for i, name := range []string{DefaultFC, TestFC} {
		cfMap[name] = cfHs[i]
	}

	store := &db.RocksdbStore{DB: rocks, ColumnFamilyHandles: cfMap}

	// Try deleting a key that doesn't exist
	err = store.Delete(TestFC, "nonexistent")
	assert.NoError(t, err)
}

func TestRocksdbStore_Delete_InvalidColumnFamily(t *testing.T) {
	tmpDir := t.TempDir()

	opts := grocksdb.NewDefaultOptions()
	opts.SetCreateIfMissing(true)
	opts.SetCreateIfMissingColumnFamilies(true)

	cfOpts := grocksdb.NewDefaultOptions()
	rocks, cfHs, err := grocksdb.OpenDbColumnFamilies(opts, tmpDir, []string{DefaultFC}, []*grocksdb.Options{cfOpts})
	require.NoError(t, err)
	defer rocks.Close()

	store := &db.RocksdbStore{
		DB:                     rocks,
		ColumnFamilyHandles:    map[string]*grocksdb.ColumnFamilyHandle{"default": cfHs[0]},
		TTLColumnFamilyHandles: map[string]*grocksdb.ColumnFamilyHandle{}, // no test_fc
	}

	// Attempt to delete using a non-existent column family
	err = store.Delete("nonexistent_cf", "key")
	require.Error(t, err)
	require.Contains(t, err.Error(), "column family")
}

func TestRocksdbStore_Delete_TTLColumnFamily(t *testing.T) {
	tmpDir := t.TempDir()

	opts := grocksdb.NewDefaultOptions()
	opts.SetCreateIfMissing(true)
	opts.SetCreateIfMissingColumnFamilies(true)

	cfOpts := grocksdb.NewDefaultOptions()
	rocks, cfHs, err := grocksdb.OpenDbColumnFamilies(opts, tmpDir, []string{DefaultFC, TestFC}, []*grocksdb.Options{cfOpts, cfOpts})
	require.NoError(t, err)
	defer rocks.Close()

	ttlCFMap := make(map[string]*grocksdb.ColumnFamilyHandle)
	ttlCFMap[TestFC] = cfHs[1]

	store := &db.RocksdbStore{
		DB:                     rocks,
		ColumnFamilyHandles:    map[string]*grocksdb.ColumnFamilyHandle{},
		TTLColumnFamilyHandles: ttlCFMap,
	}

	key := "ttl-key"
	value := []byte("ttl-value")

	require.NoError(t, store.Put(TestFC, key, value))

	// Delete the key from TTL column family
	require.NoError(t, store.Delete(TestFC, key))

	// Ensure it's deleted
	result, err := store.Get(TestFC, key)
	require.NoError(t, err)
	assert.Nil(t, result)
}
