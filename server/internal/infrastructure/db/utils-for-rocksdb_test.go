package db_test

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// Helper to create a RocksDB store for testing
func newTestRocksdbStore(t *testing.T, cfNames []string, ttlCfNames []string) db.KVStore {
	tempDir, err := os.MkdirTemp("", "rocksdb_test_*")
	require.NoError(t, err)
	t.Logf("Creating RocksDB store in: %s", tempDir)

	store, err := db.CreateRocksdbStore(tempDir, cfNames, ttlCfNames)
	require.NoError(t, err)
	require.NotNil(t, store)

	t.Cleanup(func() {
		t.Logf("Closing and removing RocksDB store from: %s", tempDir)
		store.Close()
		os.RemoveAll(tempDir)
	})
	return store
}
