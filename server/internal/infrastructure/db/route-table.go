package db

import (
	"encoding/json"
	"fmt"
	"time"
)

// RouteTable manages a dedicated KV routing table for exchanges.
// It provides O(1) routing for Direct, Fanout exchanges and lazy-cached routing for Topic exchanges.
// Headers routing still requires per-binding evaluation, but uses a lightweight binding summary.
//
// Key formats:
//
//	Direct:  {schema}:route:d:{exchangeID}:{routingKey}         → JSON []string (queueIDs)
//	Fanout:  {schema}:route:f:{exchangeID}                      → JSON []string (queueIDs)
//	Topic:   {schema}:route:t:patterns:{exchangeID}             → JSON []TopicPatternEntry
//	         {schema}:route:t:cache:{exchangeID}:{routingKey}   → JSON []string (queueIDs, lazy cache)
//	Headers: {schema}:route:h:bindings:{exchangeID}             → JSON []HeadersBindingEntry
type RouteTable struct {
	kvStore    KVStore
	cf         string // column family
	cfs        string // column family sector
	schema     string
}

// TopicPatternEntry is a lightweight record of a topic pattern and its target queue.
type TopicPatternEntry struct {
	Pattern  string `json:"p"`
	QueueID  string `json:"q"`
	BindingID string `json:"b"` // for clean removal on delete
}

// HeadersBindingEntry is a lightweight record for headers-based routing.
type HeadersBindingEntry struct {
	BindingID string            `json:"b"`
	Headers   map[string]string `json:"h"`
	XMatch    string            `json:"x"` // "all" or "any"
	QueueID   string            `json:"q"` // empty for dynamic bindings
}

// NewRouteTable creates a new RouteTable backed by the provided KVStore.
func NewRouteTable(kvStore KVStore, cf, cfs, schema string) *RouteTable {
	return &RouteTable{kvStore: kvStore, cf: cf, cfs: cfs, schema: schema}
}

// ── Key helpers ──────────────────────────────────────────────────────────────

func (rt *RouteTable) directKey(exchangeID, routingKey string) string {
	return fmt.Sprintf("%s:route:d:%s:%s", rt.schema, exchangeID, routingKey)
}

func (rt *RouteTable) fanoutKey(exchangeID string) string {
	return fmt.Sprintf("%s:route:f:%s", rt.schema, exchangeID)
}

func (rt *RouteTable) topicPatternsKey(exchangeID string) string {
	return fmt.Sprintf("%s:route:t:patterns:%s", rt.schema, exchangeID)
}

func (rt *RouteTable) topicCacheKey(exchangeID, routingKey string) string {
	return fmt.Sprintf("%s:route:t:cache:%s:%s", rt.schema, exchangeID, routingKey)
}

func (rt *RouteTable) headersBindingsKey(exchangeID string) string {
	return fmt.Sprintf("%s:route:h:bindings:%s", rt.schema, exchangeID)
}

// ── Direct Exchange ──────────────────────────────────────────────────────────

// AddDirectRoute registers queueID for (exchangeID, routingKey).
func (rt *RouteTable) AddDirectRoute(batch *WriteBatch, exchangeID, routingKey, queueID string, now time.Time) error {
	key := rt.directKey(exchangeID, routingKey)
	ids, err := rt.readStringSlice(key, now)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if id == queueID {
			return nil // already present
		}
	}
	ids = append(ids, queueID)
	return rt.writeStringSlice(batch, key, ids, now)
}

// RemoveDirectRoute removes queueID from (exchangeID, routingKey).
func (rt *RouteTable) RemoveDirectRoute(batch *WriteBatch, exchangeID, routingKey, queueID string, now time.Time) error {
	key := rt.directKey(exchangeID, routingKey)
	ids, err := rt.readStringSlice(key, now)
	if err != nil {
		return err
	}
	filtered := ids[:0]
	for _, id := range ids {
		if id != queueID {
			filtered = append(filtered, id)
		}
	}
	if len(filtered) == 0 {
		batch.Delete(rt.cf, rt.cfs, key, now)
		return nil
	}
	return rt.writeStringSlice(batch, key, filtered, now)
}

// GetDirectRoutes returns all queueIDs registered for (exchangeID, routingKey).
func (rt *RouteTable) GetDirectRoutes(exchangeID, routingKey string, now time.Time) ([]string, error) {
	return rt.readStringSlice(rt.directKey(exchangeID, routingKey), now)
}

// ── Fanout Exchange ──────────────────────────────────────────────────────────

// AddFanoutRoute registers queueID for a fanout exchange.
func (rt *RouteTable) AddFanoutRoute(batch *WriteBatch, exchangeID, queueID string, now time.Time) error {
	key := rt.fanoutKey(exchangeID)
	ids, err := rt.readStringSlice(key, now)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if id == queueID {
			return nil
		}
	}
	ids = append(ids, queueID)
	return rt.writeStringSlice(batch, key, ids, now)
}

// RemoveFanoutRoute removes queueID from a fanout exchange.
func (rt *RouteTable) RemoveFanoutRoute(batch *WriteBatch, exchangeID, queueID string, now time.Time) error {
	key := rt.fanoutKey(exchangeID)
	ids, err := rt.readStringSlice(key, now)
	if err != nil {
		return err
	}
	filtered := ids[:0]
	for _, id := range ids {
		if id != queueID {
			filtered = append(filtered, id)
		}
	}
	if len(filtered) == 0 {
		batch.Delete(rt.cf, rt.cfs, key, now)
		return nil
	}
	return rt.writeStringSlice(batch, key, filtered, now)
}

// GetFanoutRoutes returns all queueIDs for a fanout exchange.
func (rt *RouteTable) GetFanoutRoutes(exchangeID string, now time.Time) ([]string, error) {
	return rt.readStringSlice(rt.fanoutKey(exchangeID), now)
}

// ── Topic Exchange ────────────────────────────────────────────────────────────

// AddTopicPattern registers a pattern → queueID entry for a topic exchange.
func (rt *RouteTable) AddTopicPattern(batch *WriteBatch, exchangeID, pattern, queueID, bindingID string, now time.Time) error {
	key := rt.topicPatternsKey(exchangeID)
	entries, err := rt.readTopicPatterns(key, now)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.BindingID == bindingID {
			return nil // already present
		}
	}
	entries = append(entries, TopicPatternEntry{Pattern: pattern, QueueID: queueID, BindingID: bindingID})
	return rt.writeTopicPatterns(batch, key, entries, now)
}

// RemoveTopicPattern removes all entries for bindingID from a topic exchange.
func (rt *RouteTable) RemoveTopicPattern(batch *WriteBatch, exchangeID, bindingID string, now time.Time) error {
	key := rt.topicPatternsKey(exchangeID)
	entries, err := rt.readTopicPatterns(key, now)
	if err != nil {
		return err
	}
	filtered := entries[:0]
	for _, e := range entries {
		if e.BindingID != bindingID {
			filtered = append(filtered, e)
		}
	}
	if len(filtered) == 0 {
		batch.Delete(rt.cf, rt.cfs, key, now)
		return nil
	}
	return rt.writeTopicPatterns(batch, key, filtered, now)
}

// GetTopicPatterns returns all pattern entries for a topic exchange.
func (rt *RouteTable) GetTopicPatterns(exchangeID string, now time.Time) ([]TopicPatternEntry, error) {
	return rt.readTopicPatterns(rt.topicPatternsKey(exchangeID), now)
}

// SetTopicRouteCache stores a lazy-computed routing result for (exchangeID, routingKey).
func (rt *RouteTable) SetTopicRouteCache(batch *WriteBatch, exchangeID, routingKey string, queueIDs []string, now time.Time) error {
	return rt.writeStringSlice(batch, rt.topicCacheKey(exchangeID, routingKey), queueIDs, now)
}

// GetTopicRouteCache retrieves a cached routing result. Returns nil if not cached.
func (rt *RouteTable) GetTopicRouteCache(exchangeID, routingKey string, now time.Time) ([]string, bool, error) {
	key := rt.topicCacheKey(exchangeID, routingKey)
	data, err := rt.kvStore.Get(rt.cf, rt.cfs, key, now)
	if err != nil {
		return nil, false, err
	}
	if len(data) == 0 {
		return nil, false, nil
	}
	var ids []string
	if err := json.Unmarshal(data, &ids); err != nil {
		return nil, false, err
	}
	return ids, true, nil
}

// InvalidateTopicCache writes a tombstone version bump so existing caches are logically stale.
// Since we can't do prefix deletes cheaply, we use a version counter approach:
// callers should check the version before using a cached result.
// For simplicity in this implementation, we clear all cache by writing a special invalidation marker.
// A full implementation would use a version key; here we just delete the specific cache entry
// when a pattern is modified. The batch-level approach handles most cases.
func (rt *RouteTable) InvalidateTopicCacheForRoutingKey(batch *WriteBatch, exchangeID, routingKey string, now time.Time) {
	batch.Delete(rt.cf, rt.cfs, rt.topicCacheKey(exchangeID, routingKey), now)
}

// ── Headers Exchange ──────────────────────────────────────────────────────────

// AddHeadersBinding registers a headers binding entry for a headers exchange.
func (rt *RouteTable) AddHeadersBinding(batch *WriteBatch, exchangeID, bindingID string, headers map[string]string, xmatch, queueID string, now time.Time) error {
	key := rt.headersBindingsKey(exchangeID)
	entries, err := rt.readHeadersBindings(key, now)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.BindingID == bindingID {
			return nil
		}
	}
	entries = append(entries, HeadersBindingEntry{
		BindingID: bindingID,
		Headers:   headers,
		XMatch:    xmatch,
		QueueID:   queueID,
	})
	return rt.writeHeadersBindings(batch, key, entries, now)
}

// RemoveHeadersBinding removes a headers binding entry by bindingID.
func (rt *RouteTable) RemoveHeadersBinding(batch *WriteBatch, exchangeID, bindingID string, now time.Time) error {
	key := rt.headersBindingsKey(exchangeID)
	entries, err := rt.readHeadersBindings(key, now)
	if err != nil {
		return err
	}
	filtered := entries[:0]
	for _, e := range entries {
		if e.BindingID != bindingID {
			filtered = append(filtered, e)
		}
	}
	if len(filtered) == 0 {
		batch.Delete(rt.cf, rt.cfs, key, now)
		return nil
	}
	return rt.writeHeadersBindings(batch, key, filtered, now)
}

// GetHeadersBindings returns all headers binding entries for a headers exchange.
func (rt *RouteTable) GetHeadersBindings(exchangeID string, now time.Time) ([]HeadersBindingEntry, error) {
	return rt.readHeadersBindings(rt.headersBindingsKey(exchangeID), now)
}

// ── Dynamic Exchange Bindings ──────────────────────────────────────────────────

// AddDynamicRoute registers that an exchange has at least one dynamic binding.
func (rt *RouteTable) AddDynamicRoute(batch *WriteBatch, exchangeID string, now time.Time) error {
	key := fmt.Sprintf("%s:route:dyn:%s", rt.schema, exchangeID)
	// We just need a counter or existence flag. Let's increment a counter.
	count, err := rt.getDynamicCount(key, now)
	if err != nil {
		return err
	}
	count++
	return rt.writeDynamicCount(batch, key, count, now)
}

// RemoveDynamicRoute decrements the dynamic binding counter.
func (rt *RouteTable) RemoveDynamicRoute(batch *WriteBatch, exchangeID string, now time.Time) error {
	key := fmt.Sprintf("%s:route:dyn:%s", rt.schema, exchangeID)
	count, err := rt.getDynamicCount(key, now)
	if err != nil {
		return err
	}
	if count > 0 {
		count--
	}
	if count == 0 {
		batch.Delete(rt.cf, rt.cfs, key, now)
		return nil
	}
	return rt.writeDynamicCount(batch, key, count, now)
}

// HasDynamicRoutes returns true if the exchange has any dynamic bindings.
func (rt *RouteTable) HasDynamicRoutes(exchangeID string, now time.Time) (bool, error) {
	key := fmt.Sprintf("%s:route:dyn:%s", rt.schema, exchangeID)
	count, err := rt.getDynamicCount(key, now)
	return count > 0, err
}

func (rt *RouteTable) getDynamicCount(key string, now time.Time) (int, error) {
	data, err := rt.kvStore.Get(rt.cf, rt.cfs, key, now)
	if err != nil {
		return 0, err
	}
	if len(data) == 0 {
		return 0, nil
	}
	var count int
	if err := json.Unmarshal(data, &count); err != nil {
		return 0, err
	}
	return count, nil
}

func (rt *RouteTable) writeDynamicCount(batch *WriteBatch, key string, count int, now time.Time) error {
	data, err := json.Marshal(count)
	if err != nil {
		return err
	}
	batch.Put(rt.cf, rt.cfs, key, data, now)
	return nil
}

// ── Internal helpers ─────────────────────────────────────────────────────────


func (rt *RouteTable) readStringSlice(key string, now time.Time) ([]string, error) {
	data, err := rt.kvStore.Get(rt.cf, rt.cfs, key, now)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return []string{}, nil
	}
	var ids []string
	if err := json.Unmarshal(data, &ids); err != nil {
		return nil, fmt.Errorf("corrupted route table key '%s': %w", key, err)
	}
	return ids, nil
}

func (rt *RouteTable) writeStringSlice(batch *WriteBatch, key string, ids []string, now time.Time) error {
	data, err := json.Marshal(ids)
	if err != nil {
		return err
	}
	batch.Put(rt.cf, rt.cfs, key, data, now)
	return nil
}

func (rt *RouteTable) readTopicPatterns(key string, now time.Time) ([]TopicPatternEntry, error) {
	data, err := rt.kvStore.Get(rt.cf, rt.cfs, key, now)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return []TopicPatternEntry{}, nil
	}
	var entries []TopicPatternEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("corrupted topic patterns key '%s': %w", key, err)
	}
	return entries, nil
}

func (rt *RouteTable) writeTopicPatterns(batch *WriteBatch, key string, entries []TopicPatternEntry, now time.Time) error {
	data, err := json.Marshal(entries)
	if err != nil {
		return err
	}
	batch.Put(rt.cf, rt.cfs, key, data, now)
	return nil
}

func (rt *RouteTable) readHeadersBindings(key string, now time.Time) ([]HeadersBindingEntry, error) {
	data, err := rt.kvStore.Get(rt.cf, rt.cfs, key, now)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return []HeadersBindingEntry{}, nil
	}
	var entries []HeadersBindingEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("corrupted headers bindings key '%s': %w", key, err)
	}
	return entries, nil
}

func (rt *RouteTable) writeHeadersBindings(batch *WriteBatch, key string, entries []HeadersBindingEntry, now time.Time) error {
	data, err := json.Marshal(entries)
	if err != nil {
		return err
	}
	batch.Put(rt.cf, rt.cfs, key, data, now)
	return nil
}
