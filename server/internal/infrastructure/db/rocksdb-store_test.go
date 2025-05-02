package db_test

import (
	"testing"

	"github.com/linxGnu/grocksdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"deadalus-orch/server/internal/infrastructure/db"
)

func TestRocksdbStore_PutAndGet(t *testing.T) {
	tmpDir := t.TempDir()

	opts := grocksdb.NewDefaultOptions()
	opts.SetCreateIfMissing(true)

	rocks, err := grocksdb.OpenDb(opts, tmpDir)
	require.NoError(t, err)
	defer rocks.Close()

	store := &db.RocksdbStore{DB: rocks}

	key := []byte("key")
	value := []byte("value")

	err = store.Put(key, value)
	require.NoError(t, err)

	result, err := store.Get(key)
	require.NoError(t, err)
	assert.Equal(t, value, result)
}

func TestRocksdbStore_Get_NotFound(t *testing.T) {
	tmpDir := t.TempDir()

	opts := grocksdb.NewDefaultOptions()
	opts.SetCreateIfMissing(true)

	rocks, err := grocksdb.OpenDb(opts, tmpDir)
	require.NoError(t, err)
	defer rocks.Close()

	store := &db.RocksdbStore{DB: rocks}

	result, err := store.Get([]byte("nonexistent"))
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestRocksdbStore_WriteBatch(t *testing.T) {
	tmpDir := t.TempDir()

	opts := grocksdb.NewDefaultOptions()
	opts.SetCreateIfMissing(true)

	rocks, err := grocksdb.OpenDb(opts, tmpDir)
	require.NoError(t, err)
	defer rocks.Close()

	store := &db.RocksdbStore{DB: rocks}

	batch := grocksdb.NewWriteBatch()
	defer batch.Destroy()

	batch.Put([]byte("a"), []byte("valueA"))
	batch.Put([]byte("b"), []byte("valueB"))

	err = store.Write(batch)
	require.NoError(t, err)

	// Verifica clave "a"
	resultA, err := store.Get([]byte("a"))
	require.NoError(t, err)
	assert.Equal(t, []byte("valueA"), resultA)

	// Verifica clave "b"
	resultB, err := store.Get([]byte("b"))
	require.NoError(t, err)
	assert.Equal(t, []byte("valueB"), resultB)
}
