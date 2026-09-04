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
// Expired entries are deleted immediately upon access.
func (c *ExchangeCache) Get(code, vnamespace string) (ExchangeInfo, bool) {
	key := exchangeKey(code, vnamespace)

	c.mu.RLock()
	entry, ok := c.cache[key]
	c.mu.RUnlock()

	if !ok {
		return ExchangeInfo{}, false
	}

	if time.Now().After(entry.expiresAt) {
		c.mu.Lock()
		delete(c.cache, key)
		c.mu.Unlock()
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
func (c *ExchangeCache) PurgeExpired() int {
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

// Len returns the current number of cached entries.
func (c *ExchangeCache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.cache)
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
