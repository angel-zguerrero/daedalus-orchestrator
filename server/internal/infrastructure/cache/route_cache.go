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
func (c *RouteCache) Get(exchangeCode, routingKey, vnamespace string) ([]models.Queue, bool) {
	key := cacheKey(exchangeCode, routingKey, vnamespace)

	c.mu.RLock()
	entry, ok := c.cache[key]
	c.mu.RUnlock()

	if !ok || time.Now().After(entry.expiresAt) {
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
	c.mu.Unlock()
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
