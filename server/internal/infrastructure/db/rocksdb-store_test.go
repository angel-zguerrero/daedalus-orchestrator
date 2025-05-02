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
