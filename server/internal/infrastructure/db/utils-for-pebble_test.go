package db_test

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func newTestPebbleStore(t *testing.T, cfNames []string, ttlCfNames []string) db.KVStore {
	tempDir, err := os.MkdirTemp("", "pebble_test_*")
	require.NoError(t, err)
	t.Logf("Creating Pebble store in: %s", tempDir)

	store, err := db.CreatePebbleStore(tempDir, cfNames, ttlCfNames)
	require.NoError(t, err)
	require.NotNil(t, store)

	t.Cleanup(func() {
		t.Logf("Closing and removing Pebble store from: %s", tempDir)
		store.Close()
		os.RemoveAll(tempDir)
	})
	return store
}
