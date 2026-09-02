package cache

import (
	"deadalus-orch/shared/models"
	"sync"
	"time"
)

// ExchangeInfo holds the minimal exchange data needed for routing.
// This is much lighter than a full models.Exchange.
type ExchangeInfo struct {
	ID   string
	Code string
	Type models.ExchangeType
}

// ExchangeCache caches exchange Code → {ID, Type} mappings.
// Exchanges almost never change after creation, so this cache has a
// long TTL (60s) with explicit invalidation on create/delete.
type ExchangeCache struct {
	mu    sync.RWMutex
	cache map[string]*exchangeCacheEntry
	ttl   time.Duration
}

type exchangeCacheEntry struct {
	info      ExchangeInfo
	expiresAt time.Time
}

var globalExchangeCache *ExchangeCache
var exchangeOnce sync.Once

// GlobalExchangeCache returns the process-wide ExchangeCache singleton.
func GlobalExchangeCache() *ExchangeCache {
	exchangeOnce.Do(func() {
		globalExchangeCache = &ExchangeCache{
			cache: make(map[string]*exchangeCacheEntry),
			ttl:   60 * time.Second, // exchanges rarely change
		}
	})
	return globalExchangeCache
}

// exchangeKey builds a key for the exchange cache.
func exchangeKey(code, vnamespace string) string {
	return code + "\x00" + vnamespace
}

// Get returns cached exchange info. Second return is false on miss/expiry.
func (c *ExchangeCache) Get(code, vnamespace string) (ExchangeInfo, bool) {
	key := exchangeKey(code, vnamespace)

	c.mu.RLock()
	entry, ok := c.cache[key]
	c.mu.RUnlock()

	if !ok || time.Now().After(entry.expiresAt) {
		return ExchangeInfo{}, false
	}
	return entry.info, true
}

// Set stores exchange info.
func (c *ExchangeCache) Set(code, vnamespace string, info ExchangeInfo) {
	key := exchangeKey(code, vnamespace)

	c.mu.Lock()
	c.cache[key] = &exchangeCacheEntry{
		info:      info,
		expiresAt: time.Now().Add(c.ttl),
	}
	c.mu.Unlock()
}

// Invalidate removes a specific exchange entry.
func (c *ExchangeCache) Invalidate(code, vnamespace string) {
	key := exchangeKey(code, vnamespace)

	c.mu.Lock()
	delete(c.cache, key)
	c.mu.Unlock()
}

// InvalidateAll clears the entire exchange cache.
func (c *ExchangeCache) InvalidateAll() {
	c.mu.Lock()
	c.cache = make(map[string]*exchangeCacheEntry)
	c.mu.Unlock()
}
