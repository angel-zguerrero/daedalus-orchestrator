package db

import (
	"os"
	"testing"
	"time"

	models "deadalus-orch/shared/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetricsRepository_SaveAndDeleteOlderThan(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "metrics_repo_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	store, err := CreatePebbleStore(tempDir, []string{MetricsFC}, nil)
	require.NoError(t, err)
	defer store.Close()

	repo := NewMetricsRepository(store)
	now := time.Now()

	buckets := []models.MetricsBucket{
		{
			TenantCode: "t1",
			QueueCode:  "q1",
			VNamespace: "default",
			Resolution: 5,
			Timestamp:  1000,
			Published:  10,
		},
		{
			TenantCode: "t1",
			QueueCode:  "q1",
			VNamespace: "default",
			Resolution: 5,
			Timestamp:  2000,
			Published:  20,
		},
		{
			TenantCode: "t1",
			QueueCode:  "q1",
			VNamespace: "default",
			Resolution: 60,
			Timestamp:  1000,
			Published:  30,
		},
	}

	err = repo.SaveBuckets(buckets, now)
	require.NoError(t, err)

	// Verify all 3 buckets queryable
	res5, err := repo.QueryRange("t1", "q1", "default", 5, 0, 3000, 10, now)
	require.NoError(t, err)
	assert.Equal(t, 2, len(res5))

	res60, err := repo.QueryRange("t1", "q1", "default", 60, 0, 3000, 10, now)
	require.NoError(t, err)
	assert.Equal(t, 1, len(res60))

	// Delete resolution 5 buckets older than 1500 (should delete timestamp 1000)
	deleted, err := repo.DeleteOlderThan(5, 1500, now)
	require.NoError(t, err)
	assert.Equal(t, 1, deleted)

	// Query again for resolution 5
	res5After, err := repo.QueryRange("t1", "q1", "default", 5, 0, 3000, 10, now)
	require.NoError(t, err)
	assert.Equal(t, 1, len(res5After))
	assert.Equal(t, int64(2000), res5After[0].Timestamp)

	// Resolution 60 bucket should remain intact
	res60After, err := repo.QueryRange("t1", "q1", "default", 60, 0, 3000, 10, now)
	require.NoError(t, err)
	assert.Equal(t, 1, len(res60After))
}
