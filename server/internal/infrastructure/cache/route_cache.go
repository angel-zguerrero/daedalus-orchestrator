package cache

import (
	"deadalus-orch/shared/models"
	"strings"
	"sync"
	"time"
)

// RouteCache is a process-level, in-memory cache for resolved routing results.
// It eliminates Raft read round-trips on the publish hot path by caching the
// result of exchange-lookup + route-resolution + queue-hydration.
//
// Invalidation:
//   - Explicit: after any binding create/delete via InvalidateExchange().
//   - TTL-based: entries expire after a configurable duration as a safety net.
//
// Thread-safety: all methods are safe for concurrent use.
type RouteCache struct {
	mu    sync.RWMutex
	cache map[string]*routeCacheEntry
	ttl   time.Duration
}

type routeCacheEntry struct {
	queues    []models.Queue
	expiresAt time.Time
}

// Global singleton — shared by ExchangeBO and BindingBO (or any other layer).
var globalRouteCache *RouteCache
var once sync.Once

// GlobalRouteCache returns the process-wide RouteCache singleton.
func GlobalRouteCache() *RouteCache {
	once.Do(func() {
		globalRouteCache = NewRouteCache(5 * time.Second) // 5s TTL safety net
	})
	return globalRouteCache
}

// NewRouteCache creates a RouteCache with the given TTL.
func NewRouteCache(ttl time.Duration) *RouteCache {
	return &RouteCache{
		cache: make(map[string]*routeCacheEntry),
		ttl:   ttl,
	}
}

// cacheKey builds a deterministic key for a routing lookup.
func cacheKey(exchangeCode, routingKey, vnamespace string) string {
	return exchangeCode + "\x00" + routingKey + "\x00" + vnamespace
}

// Get returns cached queues for the given routing parameters.
// The second return value is false on cache miss or expiry.
// Expired entries are deleted immediately upon access.
func (c *RouteCache) Get(exchangeCode, routingKey, vnamespace string) ([]models.Queue, bool) {
	key := cacheKey(exchangeCode, routingKey, vnamespace)

	c.mu.RLock()
	entry, ok := c.cache[key]
	c.mu.RUnlock()

	if !ok {
		return nil, false
	}

	if time.Now().After(entry.expiresAt) {
		c.mu.Lock()
		delete(c.cache, key)
		c.mu.Unlock()
		return nil, false
	}

	// Return a shallow copy so callers can't mutate our cached slice.
	result := make([]models.Queue, len(entry.queues))
	copy(result, entry.queues)
	return result, true
}

// Set stores the resolved queues for the given routing parameters.
func (c *RouteCache) Set(exchangeCode, routingKey, vnamespace string, queues []models.Queue) {
	key := cacheKey(exchangeCode, routingKey, vnamespace)

	// Store a copy to prevent external mutation.
	stored := make([]models.Queue, len(queues))
	copy(stored, queues)

	c.mu.Lock()
	c.cache[key] = &routeCacheEntry{
		queues:    stored,
		expiresAt: time.Now().Add(c.ttl),
	}

	// Periodic inline cleanup when cache size grows
	if len(c.cache) > 500 {
		now := time.Now()
		for k, e := range c.cache {
			if now.After(e.expiresAt) {
				delete(c.cache, k)
			}
		}
	}
	c.mu.Unlock()
}

// PurgeExpired explicitly removes all expired entries from the cache.
// Returns the number of purged entries.
func (c *RouteCache) PurgeExpired() int {
	now := time.Now()
	purged := 0
	c.mu.Lock()
	for k, entry := range c.cache {
		if now.After(entry.expiresAt) {
			delete(c.cache, k)
			purged++
		}
	}
	c.mu.Unlock()
	return purged
}

// InvalidateExchange removes ALL cached entries for the given exchange code
// across all routing keys and vnamespaces. Call this after any binding
// create/update/delete that touches this exchange.
func (c *RouteCache) InvalidateExchange(exchangeCode string) {
	prefix := exchangeCode + "\x00"

	c.mu.Lock()
	for key := range c.cache {
		if strings.HasPrefix(key, prefix) {
			delete(c.cache, key)
		}
	}
	c.mu.Unlock()
}

// InvalidateAll clears the entire cache. Useful on leader changes or
// schema migrations.
func (c *RouteCache) InvalidateAll() {
	c.mu.Lock()
	c.cache = make(map[string]*routeCacheEntry)
	c.mu.Unlock()
}

// Len returns the current number of cached entries (for metrics/debugging).
func (c *RouteCache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.cache)
}
