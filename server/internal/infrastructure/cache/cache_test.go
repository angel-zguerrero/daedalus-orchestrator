package cache

import (
	"deadalus-orch/shared/models"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRouteCache_Eviction(t *testing.T) {
	shortTTL := 50 * time.Millisecond
	cache := NewRouteCache(shortTTL)

	queues := []models.Queue{
		{Code: "q1", VNamespace: "default"},
	}

	cache.Set("ex1", "rk1", "default", queues)
	assert.Equal(t, 1, cache.Len())

	// Immediate Get should hit
	got, ok := cache.Get("ex1", "rk1", "default")
	assert.True(t, ok)
	assert.Equal(t, 1, len(got))

	// Wait for TTL expiry
	time.Sleep(100 * time.Millisecond)

	// Get should fail and evict the entry
	got, ok = cache.Get("ex1", "rk1", "default")
	assert.False(t, ok)
	assert.Nil(t, got)
	assert.Equal(t, 0, cache.Len())
}

func TestRouteCache_PurgeExpired(t *testing.T) {
	shortTTL := 50 * time.Millisecond
	cache := NewRouteCache(shortTTL)

	cache.Set("ex1", "rk1", "default", []models.Queue{{Code: "q1"}})
	cache.Set("ex1", "rk2", "default", []models.Queue{{Code: "q2"}})
	assert.Equal(t, 2, cache.Len())

	time.Sleep(100 * time.Millisecond)

	purged := cache.PurgeExpired()
	assert.Equal(t, 2, purged)
	assert.Equal(t, 0, cache.Len())
}

func TestExchangeCache_Eviction(t *testing.T) {
	cache := &ExchangeCache{
		cache: make(map[string]*exchangeCacheEntry),
		ttl:   50 * time.Millisecond,
	}

	info := ExchangeInfo{
		ID:   "id1",
		Code: "ex1",
		Type: models.Direct,
	}

	cache.Set("ex1", "default", info)
	assert.Equal(t, 1, cache.Len())

	got, ok := cache.Get("ex1", "default")
	assert.True(t, ok)
	assert.Equal(t, "ex1", got.Code)

	time.Sleep(100 * time.Millisecond)

	got, ok = cache.Get("ex1", "default")
	assert.False(t, ok)
	assert.Equal(t, 0, cache.Len())
}

func TestExchangeCache_PurgeExpired(t *testing.T) {
	cache := &ExchangeCache{
		cache: make(map[string]*exchangeCacheEntry),
		ttl:   50 * time.Millisecond,
	}

	cache.Set("ex1", "default", ExchangeInfo{Code: "ex1"})
	cache.Set("ex2", "default", ExchangeInfo{Code: "ex2"})
	assert.Equal(t, 2, cache.Len())

	time.Sleep(100 * time.Millisecond)

	purged := cache.PurgeExpired()
	assert.Equal(t, 2, purged)
	assert.Equal(t, 0, cache.Len())
}
