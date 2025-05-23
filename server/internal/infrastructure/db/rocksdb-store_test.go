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

	batch := grocksdb.NewWriteBatch()
	defer batch.Destroy()

	batch.PutCF(ColumnFamilyHandles[TestFC], []byte("a"), []byte("valueA"))
	batch.PutCF(ColumnFamilyHandles[TestFC], []byte("b"), []byte("valueB"))

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
