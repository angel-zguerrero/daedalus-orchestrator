package db

import (
	"bytes"
	"deadalus-orch/server/internal/pkg/utils"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/cockroachdb/pebble"
)

// TTL-related key prefixes (local definitions for this subtask)
const (
	TTLDefaultSeconds = 3600 // Default TTL: 1 hour
)

// PebbleStore implements the KVStore interface using Pebble as the backend.
// It simulates column families by prefixing keys.
type PebbleStore struct {
	db            *pebble.DB
	cfPrefixes    map[string][]byte // Maps column family name to its key prefix
	ttlCfPrefixes map[string][]byte // Maps TTL column family name to its key prefix
}

// CreatePebbleStore creates and returns a new PebbleStore instance.
// (Retaining previous implementation of CreatePebbleStore as it's not part of this subtask's changes,
// but it should be compatible with the new getPrefixedKey logic if DefaultFC is handled correctly in it)
func CreatePebbleStore(dbPath string, columnFamilyNames []string, ttlColumnFamilyNames []string) (*PebbleStore, error) {
	opts := pebble.Options{}
	dbPath = filepath.Join(dbPath, "pebble")
	err := utils.EnsureDirExists(dbPath)
	if err != nil {
		return nil, err
	}
	db, err := pebble.Open(dbPath, &opts)
	if err != nil {
		return nil, fmt.Errorf("failed to open pebble database at %s: %w", dbPath, err)
	}

	cfPrefixes := make(map[string][]byte)
	ttlCfPrefixes := make(map[string][]byte)
	allCfNames := make(map[string]struct{})

	addCf := func(name string, isTTL bool) error {
		if _, exists := allCfNames[name]; exists {
			return fmt.Errorf("duplicate column family name: %s", name)
		}
		allCfNames[name] = struct{}{}
		// Using simple name as prefix, e.g., "default:", "meta:"
		// This matches the new getPrefixedKey which expects prefix + key string
		prefix := []byte(name + ":")
		if isTTL {
			ttlCfPrefixes[name] = prefix
		} else {
			cfPrefixes[name] = prefix
		}
		return nil
	}

	if err := addCf(DefaultFC, false); err != nil {
		db.Close()
		return nil, err
	}
	if err := addCf(MetaFC, false); err != nil {
		db.Close()
		return nil, err
	}

	for _, cfName := range columnFamilyNames {
		if cfName == DefaultFC || cfName == MetaFC {
			continue
		}
		if err := addCf(cfName, false); err != nil {
			db.Close()
			return nil, err
		}
	}

	for _, cfName := range ttlColumnFamilyNames {
		if cfName == DefaultFC || cfName == MetaFC {
			db.Close()
			return nil, fmt.Errorf("column family %s cannot be a TTL column family as it's a reserved name", cfName)
		}
		if err := addCf(cfName, true); err != nil {
			db.Close()
			return nil, err
		}
	}

	return &PebbleStore{
		db:            db,
		cfPrefixes:    cfPrefixes,
		ttlCfPrefixes: ttlCfPrefixes,
	}, nil
}

// getPrefixedKey constructs the actual key to be stored/retrieved in Pebble.
// It prepends the appropriate column family prefix to the given key string.
// For TTL column families, it targets the actual data entry (e.g., by appending PrefixData).
func (ps *PebbleStore) getPrefixedKey(cfName string, key string) (rawKey []byte, isTTLResolved bool, cfPrefixBytes []byte, err error) {
	var prefix []byte
	var exists bool
	isTTL := false

	resolvedCfName := cfName
	if resolvedCfName == "" {
		resolvedCfName = DefaultFC
	}

	// Check TTL CFs first
	prefix, exists = ps.ttlCfPrefixes[resolvedCfName]
	if exists {
		isTTL = true
	} else {
		// Then normal CFs
		prefix, exists = ps.cfPrefixes[resolvedCfName]
	}

	if !exists {
		// If still not found, and original cfName was empty or DefaultFC,
		// it implies DefaultFC itself is not in cfPrefixes map (problem from CreatePebbleStore)
		if resolvedCfName == DefaultFC {
			return nil, false, nil, fmt.Errorf("default column family '%s' prefix not found - store misconfiguration", DefaultFC)
		}
		// Otherwise, the specified cfName is unknown
		return nil, false, nil, fmt.Errorf("column family '%s' not found", resolvedCfName)
	}

	// For TTL, the "data key" path includes PrefixData.
	// For normal CFs, it's just prefix + key.
	// The returned cfPrefixBytes is the raw prefix for the CF (e.g., "mycf:")
	// The returned rawKey is the fully constructed key for data access.
	cfPrefixBytes = prefix
	if isTTL {
		// e.g., "myTTLCF:" + "_ttldata:" + "actualKey"
		combinedDataPrefix := bytes.Join([][]byte{prefix, []byte(PrefixData)}, nil)
		return append(combinedDataPrefix, []byte(key)...), true, cfPrefixBytes, nil
	}

	// Normal CF: "myCF:" + "actualKey"
	return append(prefix, []byte(key)...), false, cfPrefixBytes, nil
}

// getPrefixedKey constructs the actual key to be stored/retrieved in Pebble.
// It prepends the appropriate column family prefix to the given key string.
// For TTL column families, it targets the actual data entry (e.g., by appending PrefixData).
func (ps *PebbleStore) getPrefixedKeyOld(cfName string, key string) (rawKey []byte, isTTLResolved bool, cfPrefixBytes []byte, err error) {
	var prefix []byte
	var exists bool
	isTTL := false

	resolvedCfName := cfName
	if resolvedCfName == "" {
		resolvedCfName = DefaultFC
	}

	// Check TTL CFs first
	prefix, exists = ps.ttlCfPrefixes[resolvedCfName]
	if exists {
		isTTL = true
	} else {
		// Then normal CFs
		prefix, exists = ps.cfPrefixes[resolvedCfName]
	}

	if !exists {
		// If still not found, and original cfName was empty or DefaultFC,
		// it implies DefaultFC itself is not in cfPrefixes map (problem from CreatePebbleStore)
		if resolvedCfName == DefaultFC {
			return nil, false, nil, fmt.Errorf("default column family '%s' prefix not found - store misconfiguration", DefaultFC)
		}
		// Otherwise, the specified cfName is unknown
		return nil, false, nil, fmt.Errorf("column family '%s' not found", resolvedCfName)
	}

	// For TTL, the "data key" path includes PrefixData.
	// For normal CFs, it's just prefix + key.
	// The returned cfPrefixBytes is the raw prefix for the CF (e.g., "mycf:")
	// The returned rawKey is the fully constructed key for data access.
	cfPrefixBytes = prefix
	if isTTL {
		// e.g., "myTTLCF:" + "_ttldata:" + "actualKey"
		combinedDataPrefix := bytes.Join([][]byte{prefix, []byte("")}, nil)
		return append(combinedDataPrefix, []byte(key)...), true, cfPrefixBytes, nil
	}

	// Normal CF: "myCF:" + "actualKey"
	return append(prefix, []byte(key)...), false, cfPrefixBytes, nil
}

// Put stores the key-value pair in the specified column family.
// If columnFamily is empty, it defaults to DefaultFC.
// Handles TTL logic for TTL-enabled column families.
func (ps *PebbleStore) Put(columnFamily, key string, value []byte, ttl int) error {
	dataKey, isTTL, cfPrefix, err := ps.getPrefixedKey(columnFamily, key)
	if err != nil {
		return fmt.Errorf("Put: %w", err)
	}
	if !isTTL {
		// No TTL, escritura directa
		err := ps.db.Set(dataKey, value, pebble.Sync)
		if err != nil {
			return fmt.Errorf("Put: failed to set key '%s' in CF '%s': %w", key, columnFamily, err)
		}
		return nil
	}

	// TTL: borrar índice antiguo si existe y escribir batch completo
	ttlExpireKey := append(append([]byte(nil), cfPrefix...), []byte(PrefixTTLExpire)...)
	ttlExpireKey = append(ttlExpireKey, []byte(key)...)

	oldTTLBytes, closer, err := ps.db.Get(ttlExpireKey)
	if err != nil && !errors.Is(err, pebble.ErrNotFound) {
		return fmt.Errorf("Put: failed to get old TTL expiry for key '%s': %w", key, err)
	}
	var oldTTLMillis int64
	if err == nil {
		defer closer.Close()
		oldTTLMillis, _ = strconv.ParseInt(string(oldTTLBytes), 10, 64)
	}

	newTTLMillis := time.Now().Add(time.Duration(ttl) * time.Second).UnixMilli()

	ttlIndexKeyOld := []byte{}
	if oldTTLMillis > 0 {
		ttlIndexKeyOld = []byte(fmt.Sprintf("%s%s%020d:%s", string(cfPrefix), PrefixTTLIndex, oldTTLMillis, key))
	}
	ttlIndexKeyNew := []byte(fmt.Sprintf("%s%s%020d:%s", string(cfPrefix), PrefixTTLIndex, newTTLMillis, key))

	batch := ps.db.NewBatch()
	defer batch.Close()

	if oldTTLMillis > 0 {
		_ = batch.Delete(ttlIndexKeyOld, nil) // ignorar error si no existía
	}

	// Claves:
	// - Data: cfPrefix + _ttldata: + key
	// - Expire: cfPrefix + _ttlexpire: + key
	// - Index: cfPrefix + _ttlidx: + expiryTimestamp + ":" + key
	if err := batch.Set(dataKey, value, nil); err != nil {
		return fmt.Errorf("Put: failed to set data key: %w", err)
	}
	if err := batch.Set(ttlExpireKey, []byte(strconv.FormatInt(newTTLMillis, 10)), nil); err != nil {
		return fmt.Errorf("Put: failed to set expire key: %w", err)
	}
	if err := batch.Set(ttlIndexKeyNew, nil, nil); err != nil {
		return fmt.Errorf("Put: failed to set ttl index key: %w", err)
	}

	if err := batch.Commit(pebble.Sync); err != nil {
		return fmt.Errorf("Put: failed to commit TTL batch: %w", err)
	}

	return nil
}

func (ps *PebbleStore) PutRaw(columnFamily string, key string, value []byte) error {
	// getPrefixedKey now returns more info, but for non-TTL Put, we only need the final key for standard set.
	// However, Put needs to differentiate TTL CFs to implement the multi-key write logic.

	resolvedCfName := columnFamily
	if resolvedCfName == "" {
		resolvedCfName = DefaultFC
	}

	actualCfPrefix, _ := ps.ttlCfPrefixes[resolvedCfName]

	actualCfPrefix, _ = ps.cfPrefixes[resolvedCfName]

	// Standard Put for non-TTL CFs
	// actualCfPrefix is like "mycf:", key is "mykey" -> prefixedKey is "mycf:mykey"
	prefixedKey := append(actualCfPrefix, []byte(key)...)
	err := ps.db.Set(prefixedKey, value, pebble.Sync)
	if err != nil {
		return fmt.Errorf("Put: failed to set key '%s' in non-TTL cf '%s': %w", key, resolvedCfName, err)
	}
	return nil
}

// CleanExpiredKeys iterates through TTL-enabled column families and removes expired keys.
// Assumes keys in TTL CFs are structured with specific prefixes for data, index, and expiry.
func (ps *PebbleStore) CleanExpiredKeys() error {
	nowMillis := time.Now().UnixMilli()

	for cfName, actualCfPrefix := range ps.ttlCfPrefixes {
		// Construct the specific prefix for scanning TTL index entries within this CF
		// actualCfPrefix is like "myTTLCF:", so ttlIndexScanPrefix becomes "myTTLCF:_ttlidx:"
		ttlIndexScanPrefixBytes := append(actualCfPrefix, []byte(PrefixTTLIndex)...)

		iterOpts := &pebble.IterOptions{
			LowerBound: ttlIndexScanPrefixBytes,
			UpperBound: prefixRangeEnd(ttlIndexScanPrefixBytes),
		}
		iter := ps.db.NewIter(iterOpts)

		// Batch deletions for this CF
		b := ps.db.NewBatch()
		var operationsInBatch int

		for iter.First(); iter.Valid(); iter.Next() {
			fullIndexKey := iter.Key() // e.g., "myTTLCF:_ttlidx:1678886400000:actualUserKey"

			// Strip actualCfPrefix + PrefixTTLIndex to get "timestamp:actualUserKey"
			keyPart := bytes.TrimPrefix(fullIndexKey, ttlIndexScanPrefixBytes)

			parts := bytes.SplitN(keyPart, []byte{':'}, 2)
			if len(parts) < 2 {
				// Malformed key, log it or handle as an error. For now, skip.
				// Consider logging: fmt.Printf("CleanExpiredKeys: malformed TTL index key in cf %s: %s\n", cfName, string(fullIndexKey))
				continue
			}

			expireAtTimestamp, err := strconv.ParseInt(string(parts[0]), 10, 64)
			if err != nil {
				// Malformed timestamp, log or handle. For now, skip.
				// Consider logging: fmt.Printf("CleanExpiredKeys: malformed timestamp in TTL index key in cf %s: %s\n", cfName, string(fullIndexKey))
				continue
			}

			if expireAtTimestamp <= nowMillis {
				originalKeyStr := string(parts[1])

				// Construct the full prefixed keys for data, expire-ref, and the index itself
				// Data key: actualCfPrefix + PrefixData + originalKeyStr
				prefixedDataKey := append(actualCfPrefix, []byte(PrefixData)...)
				prefixedDataKey = append(prefixedDataKey, []byte(originalKeyStr)...)

				// Expire ref key: actualCfPrefix + PrefixTTLExpire + originalKeyStr
				// (This key's purpose is usually to allow quick lookup of expiry time if you only have the original key)
				// Depending on the full TTL strategy, this might or might not exist.
				// For cleaning, we assume it exists if the index key exists and is expired.
				prefixedExpireRefKey := append(actualCfPrefix, []byte(PrefixTTLExpire)...)
				prefixedExpireRefKey = append(prefixedExpireRefKey, []byte(originalKeyStr)...)

				if err := b.Delete(prefixedDataKey, nil); err != nil { // pebble.NoSync implied
					iter.Close()
					b.Close()
					return fmt.Errorf("CleanExpiredKeys: failed to add data key deletion to batch for cf %s, key %s: %w", cfName, originalKeyStr, err)
				}
				if err := b.Delete(prefixedExpireRefKey, nil); err != nil {
					iter.Close()
					b.Close()
					return fmt.Errorf("CleanExpiredKeys: failed to add expire ref key deletion to batch for cf %s, key %s: %w", cfName, originalKeyStr, err)
				}
				if err := b.Delete(fullIndexKey, nil); err != nil { // Delete the index entry itself
					iter.Close()
					b.Close()
					return fmt.Errorf("CleanExpiredKeys: failed to add index key deletion to batch for cf %s, key %s: %w", cfName, string(fullIndexKey), err)
				}
				operationsInBatch++
			} else {
				// Keys in the TTL index are ordered by expiry time.
				// If we find a key that's not expired, subsequent keys also won't be.
				break
			}
		}

		iterErr := iter.Error()
		iter.Close() // Always close iterator
		if iterErr != nil {
			b.Close() // Close batch if iterator errored
			return fmt.Errorf("CleanExpiredKeys: iterator error for cf %s: %w", cfName, iterErr)
		}

		if operationsInBatch > 0 {
			if err := b.Commit(pebble.Sync); err != nil {
				b.Close() // Ensure close, though Commit usually does.
				return fmt.Errorf("CleanExpiredKeys: failed to commit batch for cf %s: %w", cfName, err)
			}
		}
		b.Close() // Close batch if no operations or after successful commit.
	}
	return nil
}

// Flush manually triggers a DB flush.
func (ps *PebbleStore) Flush() error {
	if err := ps.db.Flush(); err != nil {
		return fmt.Errorf("Flush: failed to flush database: %w", err)
	}
	return nil
}

// ClearAll removes all data from all known column families.
func (ps *PebbleStore) ClearAll() error {
	allPrefixes := make(map[string][]byte)
	for cfName, cfPrefix := range ps.cfPrefixes {
		allPrefixes[cfName] = cfPrefix
	}
	for cfName, cfPrefix := range ps.ttlCfPrefixes {
		if _, exists := allPrefixes[cfName]; exists {
			// This should ideally be caught by CreatePebbleStore.
			return fmt.Errorf("ClearAll: duplicate column family name %s found", cfName)
		}
		allPrefixes[cfName] = cfPrefix
	}

	for cfName, cfPrefix := range allPrefixes {
		// DeleteRange deletes all keys in the range [start, end), including start, excluding end.
		err := ps.db.DeleteRange(cfPrefix, prefixRangeEnd(cfPrefix), pebble.Sync)
		if err != nil {
			return fmt.Errorf("ClearAll: failed to delete range for cf %s (prefix %s): %w", cfName, string(cfPrefix), err)
		}
	}
	return nil
}

// SearchByPatternPaginatedKV searches for keys matching a pattern within a column family,
// supporting pagination.
// Pattern matching rules:
// - "exactKey": exact match
// - "prefix*": prefix match
// - "*suffix": suffix match
// - "*contains*": contains match
func (ps *PebbleStore) SearchByPatternPaginatedKV(cfName, pattern, cursor string, limit int) ([]KeyValuePair, string, error) {
	var cfPrefix []byte
	var isTTL bool

	resolvedCfName := cfName
	if resolvedCfName == "" {
		resolvedCfName = DefaultFC
	}

	if prefix, ok := ps.cfPrefixes[resolvedCfName]; ok {
		cfPrefix = prefix
		isTTL = false
	} else if prefix, ok := ps.ttlCfPrefixes[resolvedCfName]; ok {
		cfPrefix = prefix
		isTTL = true
	} else {
		return nil, "", fmt.Errorf("column family %s not found", resolvedCfName)
	}

	iterOpts := &pebble.IterOptions{
		LowerBound: cfPrefix,
		UpperBound: prefixRangeEnd(cfPrefix),
	}

	iter := ps.db.NewIter(iterOpts)
	defer iter.Close()

	var results []KeyValuePair
	var nextCursor string
	count := 0

	// Manejo de cursor
	if cursor != "" {
		cursorBytes := []byte(cursor)
		startKey := append(cfPrefix, cursorBytes...)

		if iter.SeekGE(startKey) {
			if bytes.Equal(iter.Key(), startKey) {
				iter.Next()
			}
		}
	} else {
		iter.First()
	}

	for ; iter.Valid(); iter.Next() {
		if count >= limit {
			break
		}

		rawKey := iter.Key()
		keyWithoutPrefix := bytes.TrimPrefix(rawKey, cfPrefix)
		keyStr := string(keyWithoutPrefix)

		// Si es TTL, validar expiración
		if isTTL {
			// Solo considerar claves con prefijo de datos reales
			if !bytes.HasPrefix(rawKey, append(cfPrefix, []byte(PrefixData)...)) {
				continue
			}
			actualKey := string(bytes.TrimPrefix(rawKey, append(cfPrefix, []byte(PrefixData)...)))
			expireKey := append(append([]byte(nil), cfPrefix...), []byte(PrefixTTLExpire)...)
			expireKey = append(expireKey, []byte(actualKey)...)

			expired, err := ps.isTTLKeyExpired(expireKey)
			if err != nil {
				return nil, "", fmt.Errorf("SearchByPatternPaginatedKV TTL check error: %w", err)
			}
			if expired {
				continue
			}
			keyStr = actualKey // mostrar sin el prefijo "_ttldata:"
		}

		// Filtrado por patrón
		matches := false
		switch {
		case strings.HasPrefix(pattern, "*") && strings.HasSuffix(pattern, "*"):
			if len(pattern) == 1 {
				matches = true
			} else {
				matches = strings.Contains(keyStr, strings.Trim(pattern, "*"))
			}
		case strings.HasSuffix(pattern, "*"):
			matches = strings.HasPrefix(keyStr, strings.TrimSuffix(pattern, "*"))
		case strings.HasPrefix(pattern, "*"):
			matches = strings.HasSuffix(keyStr, strings.TrimPrefix(pattern, "*"))
		default:
			matches = (keyStr == pattern)
		}

		if matches {
			val := make([]byte, len(iter.Value()))
			copy(val, iter.Value())

			results = append(results, KeyValuePair{Key: keyStr, Value: val})
			nextCursor = keyStr
			count++
		}
	}

	if err := iter.Error(); err != nil {
		return nil, "", fmt.Errorf("iterator error in SearchByPatternPaginatedKV: %w", err)
	}

	if count < limit {
		nextCursor = ""
	}

	return results, nextCursor, nil
}

// DumpAll retrieves all data from the database, organized by column family.
func (ps *PebbleStore) DumpAll() (interface{}, error) {
	result := make(map[string]map[string][]byte)
	allPrefixes := make(map[string][]byte)

	for cfName, cfPrefix := range ps.cfPrefixes {
		allPrefixes[cfName] = cfPrefix
	}
	for cfName, cfPrefix := range ps.ttlCfPrefixes {
		// Check for name collision, though CreatePebbleStore should prevent this.
		if _, exists := allPrefixes[cfName]; exists {
			return nil, fmt.Errorf("DumpAll: duplicate column family name %s found between normal and TTL CFs", cfName)
		}
		allPrefixes[cfName] = cfPrefix
	}

	for cfName, cfPrefix := range allPrefixes {
		cfResult := make(map[string][]byte)
		iterOpts := &pebble.IterOptions{
			LowerBound: cfPrefix,
			UpperBound: prefixRangeEnd(cfPrefix),
		}
		iter := ps.db.NewIter(iterOpts)

		for iter.First(); iter.Valid(); iter.Next() {
			rawKey := iter.Key()
			// The prefix stored in cfPrefixes includes the separator (e.g., "cfName:")
			// So, we need to ensure TrimPrefix works correctly.
			// If cfPrefix is "cfName:", and rawKey is "cfName:actualKey", TrimPrefix yields "actualKey".
			keyWithoutPrefix := bytes.TrimPrefix(rawKey, cfPrefix)
			keyStr := string(keyWithoutPrefix)

			// Copy value as Pebble's buffer might be reused
			val := make([]byte, len(iter.Value()))
			copy(val, iter.Value())
			cfResult[keyStr] = val
		}

		err := iter.Error()
		iter.Close() // Close must be called after checking error
		if err != nil {
			return nil, fmt.Errorf("DumpAll: iterator error for cf %s: %w", cfName, err)
		}
		result[cfName] = cfResult
	}

	return result, nil
}

// Get retrieves the value for a key from the specified column family.
// If columnFamily is empty, it defaults to DefaultFC.
// For TTL CFs, it retrieves the actual data, not metadata keys.
// Returns nil, nil if the key is not found.
func (ps *PebbleStore) Get(columnFamily, key string) ([]byte, error) {
	dataKey, isTTL, cfPrefix, err := ps.getPrefixedKey(columnFamily, key)

	if err != nil {
		return nil, fmt.Errorf("Get: %w", err)
	}

	if isTTL {
		// Construye la clave de expiración: cfPrefix + PrefixTTLExpire + key
		expireKey := append(append([]byte(nil), cfPrefix...), []byte(PrefixTTLExpire)...)
		expireKey = append(expireKey, []byte(key)...)
		expired, err := ps.isTTLKeyExpired(expireKey)

		if err != nil || expired {
			return nil, err
		}
	}

	value, closer, err := ps.db.Get(dataKey)
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("Get: failed to get key %s in cf %s: %w", key, columnFamily, err)
	}
	defer closer.Close()

	// Copia defensiva del valor
	valueCopy := make([]byte, len(value))
	copy(valueCopy, value)
	return valueCopy, nil
}

func (ps *PebbleStore) isTTLKeyExpired(expireKey []byte) (bool, error) {
	value, closer, err := ps.db.Get(expireKey)
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return true, nil
		}
		return false, fmt.Errorf("isTTLKeyExpired: error getting expireKey: %w", err)
	}
	defer closer.Close()

	expireAt, err := strconv.ParseInt(string(value), 10, 64)
	if err != nil {
		return false, fmt.Errorf("isTTLKeyExpired: invalid timestamp: %w", err)
	}

	return time.Now().UnixMilli() > expireAt, nil
}

// Delete removes a key-value pair from the specified column family.
// If columnFamily is empty, it defaults to DefaultFC.
// Handles TTL logic for TTL-enabled column families.
func (ps *PebbleStore) Delete(columnFamily, key string) error {
	resolvedCfName := columnFamily
	if resolvedCfName == "" {
		resolvedCfName = DefaultFC
	}

	actualCfPrefix, isTTL := ps.ttlCfPrefixes[resolvedCfName]
	if !isTTL {
		var isNormal bool
		actualCfPrefix, isNormal = ps.cfPrefixes[resolvedCfName]
		if !isNormal {
			return fmt.Errorf("Delete: column family %s not found and DefaultFC misconfigured", resolvedCfName)
		}
		// Standard Delete for non-TTL CFs
		prefixedKey := append(actualCfPrefix, []byte(key)...)
		err := ps.db.Delete(prefixedKey, pebble.Sync)
		if err != nil {
			return fmt.Errorf("Delete: failed to delete key '%s' in non-TTL cf '%s': %w", key, resolvedCfName, err)
		}
		return nil
	}

	// Handle TTL Column Family
	b := ps.db.NewBatch()
	defer b.Close()

	// To find the indexKey, we need the expiry time from expireRefKey
	// actualCfPrefix is like "myTTLCF:"
	expireRefKeyBytes := append(actualCfPrefix, []byte(PrefixTTLExpire)...)
	expireRefKeyBytes = append(expireRefKeyBytes, []byte(key)...)

	expiryBytes, closer, errGetExpiry := ps.db.Get(expireRefKeyBytes)
	// It's okay if expireRefKey is not found (e.g., inconsistent data, or already partially deleted)
	// We will still attempt to delete other known key forms.
	if errGetExpiry != nil && !errors.Is(errGetExpiry, pebble.ErrNotFound) {
		// Actual error reading the expiry key
		return fmt.Errorf("Delete: error reading expiry ref key for TTL cf '%s', key '%s': %w", resolvedCfName, key, errGetExpiry)
	}
	if errGetExpiry == nil {
		defer closer.Close() // Important: call closer only if err is nil
	}

	// Data key: actualCfPrefix + key
	dataKey := append(actualCfPrefix, []byte(PrefixData)...)
	dataKey = append(dataKey, []byte(key)...)
	if err := b.Delete(dataKey, nil); err != nil {
		return fmt.Errorf("Delete: failed to add data key to batch for TTL cf '%s', key '%s': %w", resolvedCfName, key, err)
	}

	// Delete the expireRefKey itself (if it existed or not, Delete is idempotent)
	if err := b.Delete(expireRefKeyBytes, nil); err != nil {
		return fmt.Errorf("Delete: failed to add expire ref key to batch for TTL cf '%s', key '%s': %w", resolvedCfName, key, err)
	}

	// If expiryBytes were successfully read, construct and delete the indexKey
	if errGetExpiry == nil && expiryBytes != nil {
		expiryMillisStr := string(expiryBytes)
		// Index key: actualCfPrefix + PrefixTTLIndex + expiryMillisStr + ":" + key
		indexKeyStr := fmt.Sprintf("%s%s%s:%s", string(actualCfPrefix), PrefixTTLIndex, expiryMillisStr, key)
		if err := b.Delete([]byte(indexKeyStr), nil); err != nil {
			return fmt.Errorf("Delete: failed to add index key to batch for TTL cf '%s', key '%s': %w", resolvedCfName, key, err)
		}
	}
	// If expireRefKey was not found, we can't reconstruct the exact indexKey.
	// CleanExpiredKeys will eventually clean up orphaned index entries if any.

	if err := b.Commit(pebble.Sync); err != nil {
		return fmt.Errorf("Delete: failed to commit batch for TTL cf '%s', key '%s': %w", resolvedCfName, key, err)
	}
	return nil
}

// Exists checks if a key exists in the specified column family.
// If columnFamily is empty, it defaults to DefaultFC.
func (ps *PebbleStore) Exists(columnFamily, key string) (bool, error) {
	value, err := ps.Get(columnFamily, key)
	if err != nil {
		// An error occurred during Get (e.g., invalid column family, DB error)
		return false, err // Error is already context-rich from Get or getPrefixedKey
	}
	// If err is nil, then Get succeeded.
	// If value is not nil, key exists. If value is nil, key was not found (as per Get's contract).
	return value != nil, nil
}

// Close closes the Pebble database.
func (ps *PebbleStore) Close() error {
	if ps.db == nil {
		return nil // Or an error indicating it's already closed or was never opened
	}
	return ps.db.Close()
}

// TODO: Implement remaining KVStore interface methods (e.g., iterators, batch operations).
// Note: The previous Set/Get/Delete methods with []byte keys have been removed/replaced.

// Write performs a batch of operations (Put/Delete) atomically.
// Assumes WriteBatch and X are defined in the same package (e.g. kv-store.go)
func (ps *PebbleStore) Write(batch *WriteBatch) error {
	if batch == nil || len(batch.Data) == 0 {
		return nil
	}

	b := ps.db.NewBatch()
	defer b.Close()

	nowMillis := time.Now().UnixMilli()

	for _, op := range batch.Data {
		dataKey, isTTL, cfPrefix, err := ps.getPrefixedKey(op.CF, op.Key)
		if err != nil {
			return fmt.Errorf("Write: error getting prefixed key for cf '%s', key '%s': %w", op.CF, op.Key, err)
		}

		switch op.Type {
		case "put":
			if !isTTL {
				if err := b.Set(dataKey, op.Value, nil); err != nil {
					return fmt.Errorf("Write: failed to add put for key '%s': %w", op.Key, err)
				}
			} else {
				newTTLMillis := nowMillis + int64(op.TTL)*1000

				ttlExpireKey := append(append([]byte(nil), cfPrefix...), []byte(PrefixTTLExpire)...)
				ttlExpireKey = append(ttlExpireKey, []byte(op.Key)...)

				oldTTLBytes, closer, err := ps.db.Get(ttlExpireKey)
				if err != nil && !errors.Is(err, pebble.ErrNotFound) {
					return fmt.Errorf("Write: failed to get old TTL for key '%s': %w", op.Key, err)
				}
				var oldTTLMillis int64
				if err == nil {
					defer closer.Close()
					oldTTLMillis, _ = strconv.ParseInt(string(oldTTLBytes), 10, 64)
				}

				if oldTTLMillis > 0 {
					oldIndexKey := fmt.Sprintf("%s%s%020d:%s", string(cfPrefix), PrefixTTLIndex, oldTTLMillis, op.Key)
					if err := b.Delete([]byte(oldIndexKey), nil); err != nil {
						return fmt.Errorf("Write: failed to delete old TTL index key: %w", err)
					}
				}

				newIndexKey := fmt.Sprintf("%s%s%020d:%s", string(cfPrefix), PrefixTTLIndex, newTTLMillis, op.Key)

				if err := b.Set(dataKey, op.Value, nil); err != nil {
					return fmt.Errorf("Write: failed to set data key: %w", err)
				}
				if err := b.Set(ttlExpireKey, []byte(strconv.FormatInt(newTTLMillis, 10)), nil); err != nil {
					return fmt.Errorf("Write: failed to set TTL expire key: %w", err)
				}
				if err := b.Set([]byte(newIndexKey), nil, nil); err != nil {
					return fmt.Errorf("Write: failed to set TTL index key: %w", err)
				}
			}

		case "delete":
			if !isTTL {
				if err := b.Delete(dataKey, nil); err != nil {
					return fmt.Errorf("Write: failed to delete key '%s': %w", op.Key, err)
				}
			} else {
				ttlExpireKey := append(append([]byte(nil), cfPrefix...), []byte(PrefixTTLExpire)...)
				ttlExpireKey = append(ttlExpireKey, []byte(op.Key)...)

				oldTTLBytes, closer, err := ps.db.Get(ttlExpireKey)
				if err != nil && !errors.Is(err, pebble.ErrNotFound) {
					return fmt.Errorf("Write: failed to get TTL expire key for delete: %w", err)
				}
				if err == nil {
					defer closer.Close()
					oldTTLMillis, err := strconv.ParseInt(string(oldTTLBytes), 10, 64)
					if err == nil {
						oldIndexKey := fmt.Sprintf("%s%s%020d:%s", string(cfPrefix), PrefixTTLIndex, oldTTLMillis, op.Key)
						if err := b.Delete([]byte(oldIndexKey), nil); err != nil {
							return fmt.Errorf("Write: failed to delete TTL index key: %w", err)
						}
					}
				}

				if err := b.Delete(dataKey, nil); err != nil {
					return fmt.Errorf("Write: failed to delete data key: %w", err)
				}
				if err := b.Delete(ttlExpireKey, nil); err != nil {
					return fmt.Errorf("Write: failed to delete expire key: %w", err)
				}
			}

		default:
			return fmt.Errorf("Write: unsupported operation type '%s'", op.Type)
		}
	}

	if err := b.Commit(pebble.Sync); err != nil {
		return fmt.Errorf("Write: failed to commit batch: %w", err)
	}

	return nil
}

func (ps *PebbleStore) WriteRaw(batch *WriteBatch) error { // batch.Data is []X
	if batch == nil || len(batch.Data) == 0 {
		return nil // Nothing to do
	}

	b := ps.db.NewBatch()
	defer b.Close() // Ensure batch is closed even if errors occur

	for _, op := range batch.Data { // op is of type X
		prefixedKey, _, _, err := ps.getPrefixedKeyOld(op.CF, op.Key)
		if err != nil {
			return fmt.Errorf("Write: error getting prefixed key for cf '%s', key '%s': %w", op.CF, op.Key, err)
		}

		switch op.Type {
		case "put":
			if err := b.Set(prefixedKey, op.Value, nil); err != nil { // pebble.NoSync is implied for batch operations until Commit
				return fmt.Errorf("Write: failed to add Put operation to batch for key '%s': %w", op.Key, err)
			}
		case "delete":
			if err := b.Delete(prefixedKey, nil); err != nil { // pebble.NoSync is implied
				return fmt.Errorf("Write: failed to add Delete operation to batch for key '%s': %w", op.Key, err)
			}
		default:
			return fmt.Errorf("Write: unknown operation type '%s' for key '%s'", op.Type, op.Key)
		}
	}

	// Commit the batch with Sync to ensure data is written to persistent storage.
	if err := b.Commit(pebble.Sync); err != nil {
		return fmt.Errorf("Write: failed to commit batch: %w", err)
	}

	return nil
}

// Iterate calls the given function for each key-value pair in the database.
// Iteration is done per column family.
func (ps *PebbleStore) Iterate(fn func(cfName string, key, value []byte) error) error {
	allPrefixes := make(map[string][]byte)
	for cfName, cfPrefix := range ps.cfPrefixes {
		allPrefixes[cfName] = cfPrefix
	}
	for cfName, cfPrefix := range ps.ttlCfPrefixes {
		if _, exists := allPrefixes[cfName]; exists {
			// This should ideally be caught by CreatePebbleStore, but good for safety.
			return fmt.Errorf("Iterate: duplicate column family name %s found", cfName)
		}
		allPrefixes[cfName] = cfPrefix
	}

	for cfName, cfPrefix := range allPrefixes {
		iterOpts := &pebble.IterOptions{
			LowerBound: cfPrefix,
			UpperBound: prefixRangeEnd(cfPrefix),
		}
		iter := ps.db.NewIter(iterOpts)

		for iter.First(); iter.Valid(); iter.Next() {
			rawKey := iter.Key()
			keyWithoutPrefix := bytes.TrimPrefix(rawKey, cfPrefix)

			// Value is not copied here; fn is responsible if it needs to retain the slice.
			value := iter.Value()

			if err := fn(cfName, keyWithoutPrefix, value); err != nil {
				iter.Close() // Close iterator before returning error from callback
				return err   // Propagate error from callback
			}
		}

		err := iter.Error()
		iter.Close() // Close must be called after checking error
		if err != nil {
			return fmt.Errorf("Iterate: iterator error for cf %s: %w", cfName, err)
		}
	}
	return nil
}

// prefixRangeEnd devuelve el menor []byte que es mayor que todos los posibles prefijos con el mismo comienzo.
// Por ejemplo, si el prefijo es []byte("abc"), devolverá []byte("abd").
func prefixRangeEnd(prefix []byte) []byte {
	if len(prefix) == 0 {
		// Si el prefijo está vacío, no hay límite superior razonable.
		return nil
	}
	end := make([]byte, len(prefix))
	copy(end, prefix)
	for i := len(end) - 1; i >= 0; i-- {
		if end[i] < 0xFF {
			end[i]++
			return end[:i+1]
		}
	}
	// Si todos los bytes eran 0xFF, no hay un límite superior válido.
	return nil
}
