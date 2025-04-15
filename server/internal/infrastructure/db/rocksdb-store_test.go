package db

import (
	"testing"

	"github.com/linxGnu/grocksdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRocksdbStore_PutAndGet(t *testing.T) {
	tmpDir := t.TempDir()

	opts := grocksdb.NewDefaultOptions()
	opts.SetCreateIfMissing(true)

	rocks, err := grocksdb.OpenDb(opts, tmpDir)
	require.NoError(t, err)
	defer rocks.Close()

	store := &RocksdbStore{DB: rocks}

	wo := grocksdb.NewDefaultWriteOptions()
	defer wo.Destroy()
	ro := grocksdb.NewDefaultReadOptions()
	defer ro.Destroy()

	key := []byte("key")
	value := []byte("value")

	err = store.Put(wo, key, value)
	require.NoError(t, err)

	result, err := store.Get(ro, key)
	require.NoError(t, err)
	defer result.Free()

	assert.True(t, result.Exists())
	assert.Equal(t, value, result.Data())
}

func TestRocksdbStore_Get_NotFound(t *testing.T) {
	tmpDir := t.TempDir()

	opts := grocksdb.NewDefaultOptions()
	opts.SetCreateIfMissing(true)

	rocks, err := grocksdb.OpenDb(opts, tmpDir)
	require.NoError(t, err)
	defer rocks.Close()

	store := &RocksdbStore{DB: rocks}

	ro := grocksdb.NewDefaultReadOptions()
	defer ro.Destroy()

	result, err := store.Get(ro, []byte("nonexistent"))
	require.NoError(t, err)
	defer result.Free()

	assert.False(t, result.Exists())
}

func TestRocksdbStore_WriteBatch(t *testing.T) {
	tmpDir := t.TempDir()

	opts := grocksdb.NewDefaultOptions()
	opts.SetCreateIfMissing(true)

	rocks, err := grocksdb.OpenDb(opts, tmpDir)
	require.NoError(t, err)
	defer rocks.Close()

	store := &RocksdbStore{DB: rocks}

	wo := grocksdb.NewDefaultWriteOptions()
	defer wo.Destroy()
	ro := grocksdb.NewDefaultReadOptions()
	defer ro.Destroy()

	batch := grocksdb.NewWriteBatch()
	defer batch.Destroy()

	batch.Put([]byte("a"), []byte("valueA"))
	batch.Put([]byte("b"), []byte("valueB"))

	err = store.Write(wo, batch)
	require.NoError(t, err)

	// Verifica clave "a"
	resultA, err := store.Get(ro, []byte("a"))
	require.NoError(t, err)
	defer resultA.Free()
	assert.True(t, resultA.Exists())
	assert.Equal(t, []byte("valueA"), resultA.Data())

	// Verifica clave "b"
	resultB, err := store.Get(ro, []byte("b"))
	require.NoError(t, err)
	defer resultB.Free()
	assert.True(t, resultB.Exists())
	assert.Equal(t, []byte("valueB"), resultB.Data())
}
