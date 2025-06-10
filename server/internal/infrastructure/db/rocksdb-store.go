package db

import (
	"deadalus-orch/server/internal/pkg/utils"
	"fmt"
	"path/filepath"
	"strconv"
	"time"

	"strings"

	"github.com/linxGnu/grocksdb"
	"github.com/rs/zerolog/log"
)

/*

tmpDir := t.TempDir()

	opts := grocksdb.NewDefaultOptions()
	opts.SetCreateIfMissing(true)
	opts.SetCreateIfMissingColumnFamilies(true)

	cfOpts := grocksdb.NewDefaultOptions()

	rocks, cfHs, err := grocksdb.OpenDbColumnFamilies(opts, tmpDir, []string{DefaultFC, TestFC}, []*grocksdb.Options{cfOpts, cfOpts})
*/

func CreateRocksdbStore(dbPath string,
	columnFamilyNames []string,
	ttlColumnFamilyNames []string) (*RocksdbStore, error) {
	dbPath = filepath.Join(dbPath, "rocksdb")
	err := utils.EnsureDirExists(dbPath)
	if err != nil {
		return nil, err
	}
	db, cfH, ttCfH, err := OpenRocksDB(dbPath, columnFamilyNames, ttlColumnFamilyNames)
	if err != nil {
		return nil, err
	}
	currentCFs := make(map[string]bool)
	for cfName := range cfH {
		currentCFs[cfName] = true
	}
	for cfName := range ttCfH {
		currentCFs[cfName] = true
	}
	return &RocksdbStore{
		DB:                     db,
		ColumnFamilyHandles:    cfH,
		TTLColumnFamilyHandles: ttCfH,
		currentCFs:             currentCFs,
	}, nil
}

// OpenRocksDB opens a RocksDB database at the specified dbPath, creating and/or opening the specified column families.
// It handles the logic for listing existing column families, merging them with the requested ones,
// and ensuring default ("default", "meta") column families exist.
// It also separates handles for normal and TTL column families.
//
// Parameters:
//   - dbPath: The full file system path where the RocksDB database is located or will be created.
//   - columnFamilyNames: A list of names for regular column families to be opened/created.
//   - ttlColumnFamilyNames: A list of names for column families that should be treated as TTL column families.
//     These names must not overlap with columnFamilyNames.
//
// Returns:
//   - A pointer to the opened grocksdb.DB instance.
//   - A map of normal column family names to their grocksdb.ColumnFamilyHandle.
//   - A map of TTL column family names to their grocksdb.ColumnFamilyHandle.
//   - An error if the database cannot be opened, if column family names are duplicated,
//     or if there's an issue listing or creating column families.
func OpenRocksDB(
	dbPath string,
	columnFamilyNames []string,
	ttlColumnFamilyNames []string,
) (*grocksdb.DB, map[string]*grocksdb.ColumnFamilyHandle, map[string]*grocksdb.ColumnFamilyHandle, error) {

	log.Info().
		Str("dbPath", dbPath).
		Msg("🗄️  Opening index db")

	// Validar duplicados dentro de cada lista
	if utils.HasDuplicates(columnFamilyNames) {
		return nil, nil, nil, fmt.Errorf("duplicated names in columnFamilyNames")
	}
	if utils.HasDuplicates(ttlColumnFamilyNames) {
		return nil, nil, nil, fmt.Errorf("duplicated names in ttlColumnFamilyNames")
	}

	nameSet := make(map[string]struct{})
	for _, name := range columnFamilyNames {
		nameSet[name] = struct{}{}
	}
	for _, name := range ttlColumnFamilyNames {
		if _, exists := nameSet[name]; exists {
			return nil, nil, nil, fmt.Errorf("column family name '%s' exists in both normal and TTL sets", name)
		}
	}

	opts := grocksdb.NewDefaultOptions()
	opts.SetCreateIfMissing(true)
	opts.SetInfoLogLevel(grocksdb.WarnInfoLogLevel)
	opts.SetCreateIfMissingColumnFamilies(true)

	var err error
	var currentColumnFamilies []string
	uniqueCF := make(map[string]struct{})

	if exists, _ := utils.DirExists(filepath.Join(dbPath, "CURRENT")); exists {
		currentColumnFamilies, err = grocksdb.ListColumnFamilies(opts, dbPath)
		if err != nil {
			return nil, nil, nil, err
		}
		for _, cf := range currentColumnFamilies {
			uniqueCF[cf] = struct{}{}
		}
	}

	for _, cf := range columnFamilyNames {
		uniqueCF[cf] = struct{}{}
	}
	for _, cf := range ttlColumnFamilyNames {
		uniqueCF[cf] = struct{}{}
	}

	var allCFs []string
	for cf := range uniqueCF {
		allCFs = append(allCFs, cf)
	}

	cfSet := make(map[string]struct{}, len(allCFs))
	for _, name := range allCFs {
		cfSet[name] = struct{}{}
	}

	if _, ok := cfSet[DefaultFC]; !ok {
		allCFs = append(allCFs, DefaultFC)
	}
	if _, ok := cfSet[MetaFC]; !ok {
		allCFs = append(allCFs, MetaFC)
	}

	cfOpts := make([]*grocksdb.Options, len(allCFs))
	for i := range allCFs {
		cfOpts[i] = grocksdb.NewDefaultOptions()
		defer cfOpts[i].Destroy()
	}
	db, cfHs, err := grocksdb.OpenDbColumnFamilies(opts, dbPath, allCFs, cfOpts)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error opening database: %v", err)
	}

	normalCFHandles := make(map[string]*grocksdb.ColumnFamilyHandle)
	ttlCFHandles := make(map[string]*grocksdb.ColumnFamilyHandle)

	for i, name := range allCFs {
		handle := cfHs[i]
		if utils.Contains(ttlColumnFamilyNames, name) {
			ttlCFHandles[name] = handle
		} else {
			normalCFHandles[name] = handle
		}
	}

	return db, normalCFHandles, ttlCFHandles, nil
}

// RocksdbStore is an implementation of the KVStore interface using RocksDB as the underlying storage engine.
// It holds a reference to the RocksDB database instance and maps of column family handles
// for both regular and TTL (Time-To-Live) column families.
type RocksdbStore struct {
	*grocksdb.DB                                                   // Embedded RocksDB database instance.
	ColumnFamilyHandles    map[string]*grocksdb.ColumnFamilyHandle // Map of regular column family names to their handles.
	TTLColumnFamilyHandles map[string]*grocksdb.ColumnFamilyHandle // Map of TTL column family names to their handles.
	currentCFs             map[string]bool
}

// Get retrieves the value associated with a key from a specific column family.
// It first checks regular column families, then TTL column families.
//
// Parameters:
//   - columnFamily: The name of the column family to search within.
//   - key: The key whose value is to be retrieved.
//
// Returns:
//   - A byte slice containing the value if the key is found.
//   - nil if the key is not found.
//   - An error if the specified column family does not exist or if any other RocksDB error occurs.
func (r *RocksdbStore) Get(columnFamily, key string) ([]byte, error) {
	cf, isTTL, err := r.resolveColumnFamily(columnFamily)
	if err != nil {
		return nil, err
	}

	ro := grocksdb.NewDefaultReadOptions()
	defer ro.Destroy()

	if !isTTL {
		return r.getValue(cf, ro, key)
	}

	// TTL logic
	if expired, err := r.isTTLKeyExpired(cf, ro, key); err != nil || expired {
		return nil, err
	}

	return r.getValue(cf, ro, fmt.Sprintf("%s%s", PrefixData, key))
}

// resolveColumnFamily returns the column family handle and whether it's TTL
func (r *RocksdbStore) resolveColumnFamily(columnFamily string) (*grocksdb.ColumnFamilyHandle, bool, error) {
	if cf, ok := r.ColumnFamilyHandles[columnFamily]; ok {
		return cf, false, nil
	}
	if cf, ok := r.TTLColumnFamilyHandles[columnFamily]; ok {
		return cf, true, nil
	}
	return nil, false, fmt.Errorf("column family %s not found", columnFamily)
}

// getValue safely retrieves and copies data for a given key
func (r *RocksdbStore) getValue(cf *grocksdb.ColumnFamilyHandle, ro *grocksdb.ReadOptions, key string) ([]byte, error) {
	slice, err := r.DB.GetCF(ro, cf, []byte(key))
	if err != nil {
		return nil, err
	}
	defer slice.Free()

	if !slice.Exists() {
		return nil, nil
	}
	return append([]byte(nil), slice.Data()...), nil
}

// isTTLKeyExpired checks if the TTL key is expired
func (r *RocksdbStore) isTTLKeyExpired(cf *grocksdb.ColumnFamilyHandle, ro *grocksdb.ReadOptions, key string) (bool, error) {
	expireKey := fmt.Sprintf("%s%s", PrefixTTLExpire, key)
	slice, err := r.DB.GetCF(ro, cf, []byte(expireKey))
	if err != nil {
		return false, err
	}
	defer slice.Free()

	if !slice.Exists() {
		return true, nil
	}

	expireAt, err := strconv.ParseInt(string(slice.Data()), 10, 64)
	if err != nil {
		return false, fmt.Errorf("invalid expire timestamp: %w", err)
	}

	return time.Now().UnixMilli() > expireAt, nil
}

func (r *RocksdbStore) Delete(columnFamily, key string) error {
	cf, isTTL, err := r.resolveColumnFamily(columnFamily)
	if err != nil {
		return err
	}
	if !isTTL {
		// old
		wo := grocksdb.NewDefaultWriteOptions()
		defer wo.Destroy()
		return r.DB.DeleteCF(wo, cf, []byte(key))
	} else {
		ttlExpireIndexKey := fmt.Sprintf("%s%s", PrefixTTLExpire, key)
		ro := grocksdb.NewDefaultReadOptions()
		defer ro.Destroy()
		oldTTLBytes, err := r.getValue(cf, ro, ttlExpireIndexKey)
		if err != nil {
			return err
		}

		rocksBatch := grocksdb.NewWriteBatch()
		defer rocksBatch.Destroy()
		if oldTTLBytes != nil {
			oldTTLMillis, err := strconv.ParseInt(string(oldTTLBytes), 10, 64)
			if err == nil {
				oldTTLIndexKey := fmt.Sprintf("%s%020d:%s", PrefixTTLIndex, oldTTLMillis, key)
				fmt.Println("oldTTLIndexKey")
				fmt.Println(oldTTLIndexKey)
				rocksBatch.DeleteCF(cf, []byte(oldTTLIndexKey))
			}
		}

		ttlRealKey := fmt.Sprintf("%s%s", PrefixData, key)
		fmt.Println("ttlRealKey")
		fmt.Println(ttlRealKey)
		rocksBatch.DeleteCF(cf, []byte(ttlRealKey))
		fmt.Println("ttlExpireIndexKey")
		fmt.Println(ttlExpireIndexKey)
		rocksBatch.DeleteCF(cf, []byte(ttlExpireIndexKey))
		wo := grocksdb.NewDefaultWriteOptions()
		defer wo.Destroy()

		return r.DB.Write(wo, rocksBatch)
	}
}

func (r *RocksdbStore) Exists(columnFamily, key string) (bool, error) {
	val, err := r.Get(columnFamily, key)
	if err != nil {
		return false, err
	}
	return val != nil, nil
}

// SearchByPatternPaginatedKV searches for key-value pairs in a specified column family
// where the key matches a given pattern. It supports pagination using a cursor and limit.
//
// The pattern matching supports:
//   - Exact match: "key"
//   - Prefix match: "prefix*"
//   - Suffix match: "*suffix" (iterates all keys, less efficient)
//   - Contains match: "*substring*" (iterates all keys, less efficient)
//
// Parameters:
//   - cfName: The name of the column family to search within.
//   - pattern: The pattern to match keys against.
//   - cursor: The key to start searching from (exclusive). If empty, starts from the beginning (or prefix for prefix match).
//   - limit: The maximum number of results to return.
//
// Returns:
//   - A slice of KeyValuePair structs matching the pattern.
//   - A string representing the next cursor (the key of the last item returned), or an empty string if no more results.
//   - An error if the column family is not found or if an iterator error occurs.
func (r *RocksdbStore) SearchByPatternPaginatedKV(cfName, pattern, cursor string, limit int) ([]KeyValuePair, string, error) {
	var results []KeyValuePair
	var nextCursor string

	cf, ok := r.ColumnFamilyHandles[cfName]
	if !ok {
		cf, ok = r.TTLColumnFamilyHandles[cfName]
		if !ok {
			return nil, "", fmt.Errorf("column family %s not found", cfName)
		}
	}

	readOpts := grocksdb.NewDefaultReadOptions()
	defer readOpts.Destroy()

	iter := r.DB.NewIteratorCF(readOpts, cf)
	defer iter.Close()

	usePrefixMatch := strings.HasSuffix(pattern, "*") && !strings.HasPrefix(pattern, "*")
	prefix := strings.TrimSuffix(pattern, "*")

	if cursor == "" {
		if usePrefixMatch {
			iter.Seek([]byte(prefix))
		} else {
			iter.SeekToFirst()
		}
	} else {
		iter.Seek([]byte(cursor))
		if iter.Valid() && string(iter.Key().Data()) == cursor {
			iter.Next()
		}
	}

	count := 0
	for ; iter.Valid(); iter.Next() {
		key := iter.Key()
		keyStr := string(key.Data())
		key.Free()

		if usePrefixMatch {
			if !strings.HasPrefix(keyStr, prefix) {
				break
			}
		} else {
			// fallback to contains/endswith if not prefix pattern
			switch {
			case strings.HasPrefix(pattern, "*") && strings.HasSuffix(pattern, "*"):
				if !strings.Contains(keyStr, strings.Trim(pattern, "*")) {
					continue
				}
			case strings.HasPrefix(pattern, "*"):
				if !strings.HasSuffix(keyStr, strings.TrimPrefix(pattern, "*")) {
					continue
				}
			case strings.HasSuffix(pattern, "*"):
				if !strings.HasPrefix(keyStr, strings.TrimSuffix(pattern, "*")) {
					continue
				}
			default:
				if keyStr != pattern {
					continue
				}
			}
		}

		val := iter.Value()
		results = append(results, KeyValuePair{
			Key:   keyStr,
			Value: append([]byte(nil), val.Data()...),
		})
		val.Free()

		count++
		if count == limit {
			nextCursor = keyStr
			break
		}
	}

	if err := iter.Err(); err != nil {
		return nil, "", fmt.Errorf("iterator error: %w", err)
	}

	return results, nextCursor, nil
}

// Put stores a key-value pair into the specified column family.
// If the key already exists, its value will be overwritten.
// It first checks regular column families, then TTL column families.
//
// Parameters:
//   - columnFamily: The name of the column family where the key-value pair will be stored.
//   - key: The key to store.
//   - value: The value to store (as a byte slice).
//
// Returns:
//   - An error if the specified column family does not exist or if any other RocksDB error occurs during the put operation.
func (r *RocksdbStore) Put(columnFamily, key string, value []byte) error {

	// Ensure column family exists in the new DB before putting data
	if _, ok := r.currentCFs[columnFamily]; !ok {
		// Attempt to create the column family.
		// This assumes default options. Specific options would require more info from snapshot or config.
		// Also, it doesn't distinguish between normal and TTL CFs based on snapshot data alone.
		// This is a simplification; a full solution might need DDL commands in the snapshot
		// or a pre-defined schema.
		opts := grocksdb.NewDefaultOptions()
		// Note: opts should be destroyed, but its lifecycle here is tricky.
		// Ideally, CF creation is less dynamic or handled by the stateMachineImpl.
		newCfHandle, createErr := r.DB.CreateColumnFamily(opts, columnFamily)
		opts.Destroy()
		if createErr != nil {
			return createErr
		}
		// Assuming it's a normal CF. If it needs to be TTL, that info isn't in this basic entry.
		r.ColumnFamilyHandles[columnFamily] = newCfHandle
		r.currentCFs[columnFamily] = true
	}

	cf, ok := r.ColumnFamilyHandles[columnFamily]
	if !ok {
		cf, ok = r.TTLColumnFamilyHandles[columnFamily]
		if !ok {
			return fmt.Errorf("column family %s not found", columnFamily)
		}
	}
	wo := grocksdb.NewDefaultWriteOptions()
	defer wo.Destroy()
	return r.DB.PutCF(wo, cf, []byte(key), value)
}

// Write applies a batch of operations to the database atomically.
// The provided batch must be of type *grocksdb.WriteBatch.
//
// Parameters:
//   - batch: A *grocksdb.WriteBatch containing the operations to be written.
//
// Returns:
//   - An error if the provided batch is not of the correct type or if any RocksDB error occurs during the write operation.
func (r *RocksdbStore) Write(batch *WriteBatch) error {
	rocksBatch := grocksdb.NewWriteBatch()
	defer rocksBatch.Destroy()

	for _, op := range batch.Data {
		cf, ok := r.ColumnFamilyHandles[op.CF]
		if !ok {
			cf, ok = r.TTLColumnFamilyHandles[op.CF]
			if !ok {
				return fmt.Errorf("column family %s not found", op.CF)
			}
		}

		switch op.Type {
		case "put":
			rocksBatch.PutCF(cf, []byte(op.Key), op.Value)
		case "delete":
			rocksBatch.DeleteCF(cf, []byte(op.Key))
		default:
			return fmt.Errorf("unsupported operation type: %s", op.Type)
		}
	}

	wo := grocksdb.NewDefaultWriteOptions()
	defer wo.Destroy()

	return r.DB.Write(wo, rocksBatch)
}

// DumpAll retrieves all key-value pairs from all column families (both regular and TTL) in the database.
// The result is returned as a map where keys are column family names and values are maps of key-value pairs
// within that column family.
//
// This method can be memory-intensive for large databases.
//
// Returns:
//   - A map[string]map[string][]byte representing all data in the database.
//   - An error if any RocksDB iterator error occurs.
func (r *RocksdbStore) DumpAll() (interface{}, error) {
	ro := grocksdb.NewDefaultReadOptions()
	defer ro.Destroy()

	result := make(map[string]map[string][]byte)

	allCFs := map[string]*grocksdb.ColumnFamilyHandle{}

	for name, handle := range r.ColumnFamilyHandles {
		allCFs[name] = handle
	}
	for name, handle := range r.TTLColumnFamilyHandles {
		if _, exists := allCFs[name]; !exists {
			allCFs[name] = handle
		}
	}

	for cfName, cfHandle := range allCFs {
		cfResult := make(map[string][]byte)
		it := r.DB.NewIteratorCF(ro, cfHandle)
		defer it.Close()

		for it.SeekToFirst(); it.Valid(); it.Next() {
			key := it.Key()
			value := it.Value()

			cfResult[string(key.Data())] = append([]byte(nil), value.Data()...)

			key.Free()
			value.Free()
		}

		if err := it.Err(); err != nil {
			return nil, fmt.Errorf("iterator error in CF %s: %w", cfName, err)
		}

		result[cfName] = cfResult
	}

	return result, nil
}

// Iterate iterates over all key-value pairs in all column families (both regular and TTL)
// and executes the provided function `fn` for each pair.
// If `fn` returns an error, the iteration stops and the error is returned.
//
// Parameters:
//   - fn: A function that takes the column family name (string), key (byte slice), and value (byte slice)
//     and returns an error.
//
// Returns:
//   - An error if `fn` returns an error or if any RocksDB iterator error occurs.
func (r *RocksdbStore) Iterate(fn func(cfName string, key, value []byte) error) error {
	ro := grocksdb.NewDefaultReadOptions()
	defer ro.Destroy()

	allCFs := map[string]*grocksdb.ColumnFamilyHandle{}

	for name, handle := range r.ColumnFamilyHandles {
		allCFs[name] = handle
	}
	for name, handle := range r.TTLColumnFamilyHandles {
		if _, exists := allCFs[name]; !exists {
			allCFs[name] = handle
		}
	}

	for cfName, cfHandle := range allCFs {
		it := r.DB.NewIteratorCF(ro, cfHandle)
		defer it.Close()

		for it.SeekToFirst(); it.Valid(); it.Next() {
			key := it.Key()
			value := it.Value()

			err := fn(cfName, key.Data(), value.Data())

			key.Free()
			value.Free()

			if err != nil {
				return err
			}
		}

		if err := it.Err(); err != nil {
			return fmt.Errorf("iterator error in CF %s: %w", cfName, err)
		}
	}

	return nil
}

// ClearAll removes all key-value pairs from all column families in the database.
// This is a destructive operation. It iterates through each key in each column family and deletes it.
//
// Returns:
//   - An error if any RocksDB error occurs during deletion or iteration.
func (r *RocksdbStore) ClearAll() error {
	// Combinamos ColumnFamilyHandles y TTLColumnFamilyHandles en un solo mapa
	allCFs := map[string]*grocksdb.ColumnFamilyHandle{}

	// Añadimos las ColumnFamilyHandles
	for name, handle := range r.ColumnFamilyHandles {
		allCFs[name] = handle
	}

	// Añadimos las TTLColumnFamilyHandles, sin duplicados
	for name, handle := range r.TTLColumnFamilyHandles {
		if _, exists := allCFs[name]; !exists {
			allCFs[name] = handle
		}
	}

	// Iteramos sobre todas las ColumnFamilyHandles combinadas
	for _, cf := range allCFs {
		ro := grocksdb.NewDefaultReadOptions()
		defer ro.Destroy()

		wo := grocksdb.NewDefaultWriteOptions()
		defer wo.Destroy()

		it := r.DB.NewIteratorCF(ro, cf)
		defer it.Close()

		// Eliminamos las claves de esta columna
		for it.SeekToFirst(); it.Valid(); it.Next() {
			key := it.Key()
			err := r.DB.DeleteCF(wo, cf, key.Data())
			key.Free()
			if err != nil {
				return fmt.Errorf("failed to delete key from column family: %w", err)
			}
		}

		// Verificamos si hubo algún error durante la iteración
		if err := it.Err(); err != nil {
			return fmt.Errorf("iterator error: %w", err)
		}
	}

	return nil
}

// Flush flushes all memtable data to disk for all column families.
// It first flushes each column family individually and then performs a general database flush and WAL flush.
// The flush operations wait for completion.
//
// Returns:
//   - An error if any RocksDB error occurs during the flush operations.
func (r *RocksdbStore) Flush() error {
	fo := grocksdb.NewDefaultFlushOptions()
	defer fo.Destroy()
	fo.SetWait(true)

	allCFs := map[string]*grocksdb.ColumnFamilyHandle{}

	for name, handle := range r.ColumnFamilyHandles {
		allCFs[name] = handle
	}

	for name, handle := range r.TTLColumnFamilyHandles {
		if _, exists := allCFs[name]; !exists {
			allCFs[name] = handle
		}
	}

	for _, cf := range allCFs {
		err := r.DB.FlushCF(cf, fo)
		if err != nil {
			return fmt.Errorf("failed to flush column family: %w", err)
		}
	}

	err := r.DB.Flush(fo)
	if err != nil {
		return fmt.Errorf("failed to flush: %w", err)
	}

	return r.DB.FlushWAL(true)
}

// Close closes the RocksDB database.
// After this call, the RocksdbStore instance should not be used.
//
// Returns:
//   - nil (errors during close are typically handled internally by grocksdb or are not recoverable).
func (r *RocksdbStore) Close() error {
	r.DB.Close()
	return nil
}

func (r *RocksdbStore) CleanExpiredKeys() error {
	for name, handle := range r.TTLColumnFamilyHandles {
		err := cleanExpiredKeys(r.DB, handle)
		if err != nil {
			return fmt.Errorf("error cleaning expired keys for CF %s: %w", name, err)
		}
	}
	return nil
}

// cleanExpiredKeys iterates through a TTL-enabled column family and removes keys that have expired.
// It scans keys prefixed with `PrefixTTLIndex`, which store `expireAtTimestamp:originalKey`.
// If `expireAtTimestamp` is in the past, it deletes the `PrefixTTLIndex` key,
// the actual data key (`PrefixData:originalKey`), and the reference key (`PrefixTTLExpire:originalKey`).
// It performs deletions in batches (up to `maxDeletions`) to avoid holding locks for too long.
//
// Parameters:
//   - db: The RocksDB instance.
//   - cf: The column family handle for the TTL-enabled column family to clean.
//
// Returns:
//   - An error if iterator operations or batch write operations fail.
func cleanExpiredKeys(db_instance *grocksdb.DB, cf *grocksdb.ColumnFamilyHandle) error {
	const maxDeletions = 1000
	var deleted int64

	readOpts := grocksdb.NewDefaultReadOptions()
	defer readOpts.Destroy()

	writeOpts := grocksdb.NewDefaultWriteOptions()
	defer writeOpts.Destroy()

	batch := grocksdb.NewWriteBatch()
	defer batch.Destroy()

	it := db_instance.NewIteratorCF(readOpts, cf)
	defer it.Close()

	nowMillis := time.Now().UnixMilli()
	prefix := []byte(PrefixTTLIndex)

	for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
		key := it.Key()
		keyBytes := append([]byte(nil), key.Data()...)
		key.Free()

		keyStr := string(keyBytes)
		trimmed := strings.TrimPrefix(keyStr, PrefixTTLIndex)
		sepIdx := strings.IndexByte(trimmed, ':')
		if sepIdx <= 0 || sepIdx >= len(trimmed)-1 {
			continue
		}

		expireAtStr := trimmed[:sepIdx]
		originalKey := trimmed[sepIdx+1:]

		expireAt, err := strconv.ParseInt(expireAtStr, 10, 64)
		if err != nil {
			continue
		}

		if expireAt > nowMillis {
			break
		}

		dataKey := []byte(PrefixData + originalKey)
		expireRefKey := []byte(PrefixTTLExpire + originalKey)

		ro := grocksdb.NewDefaultReadOptions()
		defer ro.Destroy()

		batch.DeleteCF(cf, dataKey)
		batch.DeleteCF(cf, expireRefKey)
		batch.DeleteCF(cf, keyBytes)

		deleted++
		if deleted >= maxDeletions {
			break
		}
	}

	if err := it.Err(); err != nil {
		return fmt.Errorf("iterator error: %w", err)
	}

	if deleted > 0 {
		if err := db_instance.Write(writeOpts, batch); err != nil {
			return fmt.Errorf("failed to write batch for expired keys: %w", err)
		}
	}

	return nil
}
