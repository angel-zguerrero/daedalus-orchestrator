//go:build rocksdb
// +build rocksdb

package db

import (
	"deadalus-orch/server/internal/pkg/utils"
	"fmt"
	"path/filepath"
	"slices"
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
	extraNormalCF []string,
	extraTTLCF []string,
) (*grocksdb.DB, map[string]*grocksdb.ColumnFamilyHandle, map[string]*grocksdb.ColumnFamilyHandle, error) {

	log.Info().
		Str("dbPath", dbPath).
		Msg("🗄️  Opening RocksDB")

	opts := grocksdb.NewDefaultOptions()
	opts.SetCreateIfMissing(true)
	opts.SetCreateIfMissingColumnFamilies(true)
	opts.SetInfoLogLevel(grocksdb.WarnInfoLogLevel)

	existingCFs := make(map[string]struct{})
	allCFs := make(map[string]struct{})
	explicitNormalSet := make(map[string]struct{})
	explicitTTLSet := make(map[string]struct{})

	// Paso 1: Detectar si la base ya existe
	dbExists, _ := utils.FileExists(filepath.Join(dbPath, "CURRENT"))
	if dbExists {
		names, err := grocksdb.ListColumnFamilies(opts, dbPath)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("error listing column families: %w", err)
		}
		for _, name := range names {
			existingCFs[name] = struct{}{}
			allCFs[name] = struct{}{}
		}
	}

	// Paso 2: Agregar CFs explícitas por parámetro
	for _, name := range extraNormalCF {
		allCFs[name] = struct{}{}
		explicitNormalSet[name] = struct{}{}
	}
	for _, name := range extraTTLCF {
		allCFs[name] = struct{}{}
		explicitTTLSet[name] = struct{}{}
	}

	// Paso 3: Asegurar que default y meta estén siempre
	if _, ok := allCFs[DefaultFC]; !ok {
		allCFs[DefaultFC] = struct{}{}
	}
	if _, ok := allCFs[MetaFC]; !ok {
		allCFs[MetaFC] = struct{}{}
	}

	// Paso 4: Ordenar los CFs para consistencia
	var cfNames []string
	for name := range allCFs {
		cfNames = append(cfNames, name)
	}
	slices.Sort(cfNames) // Usa sort.Strings(cfNames) si prefieres compatibilidad total

	// Paso 5: Crear opciones por CF
	cfOpts := make([]*grocksdb.Options, len(cfNames))
	for i := range cfNames {
		cfOpts[i] = grocksdb.NewDefaultOptions()
		defer cfOpts[i].Destroy()
	}

	// Paso 6: Abrir DB
	db, cfHandles, err := grocksdb.OpenDbColumnFamilies(opts, dbPath, cfNames, cfOpts)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error opening RocksDB: %w", err)
	}

	// Paso 7: Clasificar handles
	normalCFHandles := make(map[string]*grocksdb.ColumnFamilyHandle)
	ttlCFHandles := make(map[string]*grocksdb.ColumnFamilyHandle)

	for i, name := range cfNames {
		handle := cfHandles[i]

		if _, ok := explicitTTLSet[name]; ok {
			ttlCFHandles[name] = handle
		} else if _, ok := explicitNormalSet[name]; ok {
			normalCFHandles[name] = handle
		} else if strings.HasPrefix(name, "cf-ttl-") {
			ttlCFHandles[name] = handle
		} else if strings.HasPrefix(name, "cf-n-") || name == DefaultFC || name == MetaFC {
			normalCFHandles[name] = handle
		} else {
			log.Warn().Str("name", name).Msg("🔶 Column family with unknown prefix or classification — assigning as normal")
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
func (r *RocksdbStore) Get(columnFamily, columnFamilySector, key string, now time.Time) ([]byte, error) {
	if columnFamilySector == "" {
		return nil, fmt.Errorf("column family sector cannot be empty")
	}

	cf, isTTL, err := r.resolveColumnFamily(columnFamily)
	if err != nil {
		return nil, err
	}

	ro := grocksdb.NewDefaultReadOptions()
	defer ro.Destroy()

	if !isTTL {
		return r.getValue(cf, ro, columnFamilySector, key)
	}

	// TTL logic
	if expired, err := r.isTTLKeyExpired(cf, ro, columnFamilySector, key, now); err != nil || expired {
		return nil, err
	}

	return r.getValue(cf, ro, columnFamilySector, key)
}

func (r *RocksdbStore) GetRaw(columnFamily, key string) ([]byte, error) {
	cf, _, err := r.resolveColumnFamily(columnFamily)
	if err != nil {
		return nil, err
	}

	ro := grocksdb.NewDefaultReadOptions()
	defer ro.Destroy()

	return r.getValue(cf, ro, key)
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
func (r *RocksdbStore) getValue(cf *grocksdb.ColumnFamilyHandle, ro *grocksdb.ReadOptions, cfSelector, key string) ([]byte, error) {
	finalKey := fmt.Sprintf("%s:%s", cfSelector, key)
	slice, err := r.DB.GetCF(ro, cf, []byte(finalKey))
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
func (r *RocksdbStore) isTTLKeyExpired(cf *grocksdb.ColumnFamilyHandle, ro *grocksdb.ReadOptions, cfSelector, key string, now time.Time) (bool, error) {
	finalKey := fmt.Sprintf("%s:%s", cfSelector, key)
	expireKey := fmt.Sprintf("%s%s", PrefixTTLExpire, finalKey)
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

	return now.UnixMilli() > expireAt, nil
}

func (r *RocksdbStore) Delete(columnFamily, columnFamilySector, key string, now time.Time) error {
	if columnFamilySector == "" {
		return fmt.Errorf("column family sector cannot be empty")
	}

	cf, isTTL, err := r.resolveColumnFamily(columnFamily)
	if err != nil {
		return err
	}
	if !isTTL {
		// old
		wo := grocksdb.NewDefaultWriteOptions()
		defer wo.Destroy()
		finalKey := fmt.Sprintf("%s:%s", columnFamilySector, key)
		return r.DB.DeleteCF(wo, cf, []byte(finalKey))
	} else {
		finalKey := fmt.Sprintf("%s:%s", columnFamilySector, key)
		ttlExpireIndexKey := fmt.Sprintf("%s%s", PrefixTTLExpire, finalKey)
		ro := grocksdb.NewDefaultReadOptions()
		defer ro.Destroy()
		oldTTLBytes, err := r.getValue(cf, ro, columnFamilySector, ttlExpireIndexKey)
		if err != nil {
			return err
		}

		rocksBatch := grocksdb.NewWriteBatch()
		defer rocksBatch.Destroy()
		if oldTTLBytes != nil {
			oldTTLMillis, err := strconv.ParseInt(string(oldTTLBytes), 10, 64)
			if err == nil {
				oldTTLIndexKey := fmt.Sprintf("%s%020d:%s", PrefixTTLIndex, oldTTLMillis, finalKey)
				rocksBatch.DeleteCF(cf, []byte(oldTTLIndexKey))
			}
		}

		dataKey := finalKey
		rocksBatch.DeleteCF(cf, []byte(dataKey))
		rocksBatch.DeleteCF(cf, []byte(ttlExpireIndexKey))
		wo := grocksdb.NewDefaultWriteOptions()
		defer wo.Destroy()

		return r.DB.Write(wo, rocksBatch)
	}
}

func (r *RocksdbStore) Exists(columnFamily, columnFamilySector, key string, now time.Time) (bool, error) {
	val, err := r.Get(columnFamily, columnFamilySector, key, now)
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
func (r *RocksdbStore) SearchByPatternPaginatedKV(cfName, cfSelector, pattern, cursor string, limit int, now time.Time) ([]KeyValuePair, string, error) {
	if cfSelector == "" {
		return nil, "", fmt.Errorf("column family sector cannot be empty")
	}
	var results []KeyValuePair
	var nextCursor string
	cf, isTTL, err := r.resolveColumnFamily(cfName)
	if err != nil {
		return nil, "", err
	}

	readOpts := grocksdb.NewDefaultReadOptions()
	defer readOpts.Destroy()

	iter := r.DB.NewIteratorCF(readOpts, cf)
	defer iter.Close()

	usePrefixMatch := strings.HasSuffix(pattern, "*") && !strings.HasPrefix(pattern, "*")
	prefix := strings.TrimSuffix(pattern, "*")

	if cursor == "" {
		if usePrefixMatch {
			finalKey := fmt.Sprintf("%s:%s", cfSelector, prefix)
			iter.Seek([]byte(finalKey))
		} else {
			iter.SeekToFirst()
		}
	} else {
		finalKey := fmt.Sprintf("%s:%s", cfSelector, cursor)
		iter.Seek([]byte(finalKey))
		if iter.Valid() && string(iter.Key().Data()) == finalKey {
			iter.Next()
		}
	}

	for ; iter.Valid(); iter.Next() {
		key := iter.Key()
		keyStr := string(key.Data())
		key.Free()

		// Pattern check
		match := false
		if usePrefixMatch {
			match = strings.HasPrefix(keyStr, prefix)
			if !match {
				break
			}
		} else {
			switch {
			case strings.HasPrefix(pattern, "*") && strings.HasSuffix(pattern, "*"):
				match = strings.Contains(keyStr, strings.Trim(pattern, "*"))
			case strings.HasPrefix(pattern, "*"):
				match = strings.HasSuffix(keyStr, strings.TrimPrefix(pattern, "*"))
			case strings.HasSuffix(pattern, "*"):
				match = strings.HasPrefix(keyStr, strings.TrimSuffix(pattern, "*"))
			default:
				match = (keyStr == pattern)
			}
		}
		if !match {
			continue
		}

		// TTL check
		if isTTL {
			if expired, err := r.isTTLKeyExpired(cf, readOpts, cfSelector, keyStr, now); err != nil {
				return nil, "", err
			} else if expired {
				continue
			}
		}

		val := iter.Value()
		results = append(results, KeyValuePair{
			Key:   keyStr,
			Value: append([]byte(nil), val.Data()...),
		})
		val.Free()

		nextCursor = keyStr
		if len(results) >= limit {
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
func (r *RocksdbStore) Put(columnFamily, columnFamilySector, key string, value []byte, ttl int, now time.Time) error {
	if columnFamilySector == "" {
		return fmt.Errorf("column family sector cannot be empty")
	}

	cf, isTTL, err := r.resolveColumnFamily(columnFamily)
	if err != nil {
		return err
	}
	if !isTTL {
		wo := grocksdb.NewDefaultWriteOptions()
		defer wo.Destroy()
		finalKey := fmt.Sprintf("%s:%s", columnFamilySector, key)
		return r.DB.PutCF(wo, cf, []byte(finalKey), value)
	} else {
		rocksBatch := grocksdb.NewWriteBatch()
		defer rocksBatch.Destroy()

		wo := grocksdb.NewDefaultWriteOptions()
		defer wo.Destroy()

		ttlMillis := now.Add(time.Duration(ttl) * time.Second).UnixMilli()

		finalKey := fmt.Sprintf("%s:%s", columnFamilySector, key)
		dataKey := finalKey
		ttlExpireIndexKey := fmt.Sprintf("%s%s", PrefixTTLExpire, finalKey)

		ro := grocksdb.NewDefaultReadOptions()
		defer ro.Destroy()

		oldTTLBytes, err := r.getValue(cf, ro, columnFamilySector, ttlExpireIndexKey)
		if err != nil {
			return err
		}
		if oldTTLBytes != nil {
			oldTTLMillis, err := strconv.ParseInt(string(oldTTLBytes), 10, 64)
			if err == nil {
				oldTTLIndexKey := fmt.Sprintf("%s%020d:%s", PrefixTTLIndex, oldTTLMillis, finalKey)
				rocksBatch.DeleteCF(cf, []byte(oldTTLIndexKey))
			}
		}

		rocksBatch.PutCF(cf, []byte(dataKey), value)

		newTTLIndexKey := fmt.Sprintf("%s%020d:%s", PrefixTTLIndex, ttlMillis, finalKey)
		rocksBatch.PutCF(cf, []byte(newTTLIndexKey), nil)

		rocksBatch.PutCF(cf, []byte(ttlExpireIndexKey), []byte(strconv.FormatInt(ttlMillis, 10)))
		return r.DB.Write(wo, rocksBatch)
	}
}

func (r *RocksdbStore) PutRaw(columnFamily, columnFamilySector, key string, value []byte) error {
	if columnFamilySector == "" {
		return fmt.Errorf("column family sector cannot be empty")
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
	finalKey := fmt.Sprintf("%s:%s", columnFamilySector, key)
	return r.DB.PutCF(wo, cf, []byte(finalKey), value)
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
		cf, isTTL, err := r.resolveColumnFamily(op.CF)
		if err != nil {
			return err
		}

		switch op.Type {
		case "put":
			finalKey := fmt.Sprintf("%s:%s", op.CFS, op.Key)
			if !isTTL {
				rocksBatch.PutCF(cf, []byte(finalKey), op.Value)
			} else {
				ttlMillis := op.Now.Add(time.Duration(op.TTL) * time.Second).UnixMilli()

				dataKey := finalKey
				ttlExpireIndexKey := fmt.Sprintf("%s%s", PrefixTTLExpire, finalKey)

				ro := grocksdb.NewDefaultReadOptions()
				defer ro.Destroy()

				oldTTLBytes, err := r.getValue(cf, ro, op.CFS, ttlExpireIndexKey)
				if err != nil {
					return err
				}
				if oldTTLBytes != nil {
					oldTTLMillis, err := strconv.ParseInt(string(oldTTLBytes), 10, 64)
					if err == nil {
						oldTTLIndexKey := fmt.Sprintf("%s%020d:%s", PrefixTTLIndex, oldTTLMillis, finalKey)
						rocksBatch.DeleteCF(cf, []byte(oldTTLIndexKey))
					}
				}
				rocksBatch.PutCF(cf, []byte(dataKey), op.Value)
				newTTLIndexKey := fmt.Sprintf("%s%020d:%s", PrefixTTLIndex, ttlMillis, finalKey)
				rocksBatch.PutCF(cf, []byte(newTTLIndexKey), nil)
				rocksBatch.PutCF(cf, []byte(ttlExpireIndexKey), []byte(strconv.FormatInt(ttlMillis, 10)))
			}
		case "delete":
			finalKey := fmt.Sprintf("%s:%s", op.CFS, op.Key)
			if !isTTL {
				rocksBatch.DeleteCF(cf, []byte(finalKey))
			} else {
				ro := grocksdb.NewDefaultReadOptions()
				defer ro.Destroy()
				ttlExpireIndexKey := fmt.Sprintf("%s%s", PrefixTTLExpire, finalKey)
				oldTTLBytes, err := r.getValue(cf, ro, op.CFS, ttlExpireIndexKey)
				if err != nil {
					return err
				}

				if oldTTLBytes != nil {
					oldTTLMillis, err := strconv.ParseInt(string(oldTTLBytes), 10, 64)
					if err == nil {
						oldTTLIndexKey := fmt.Sprintf("%s%020d:%s", PrefixTTLIndex, oldTTLMillis, finalKey)
						rocksBatch.DeleteCF(cf, []byte(oldTTLIndexKey))
					}
				}

				dataKey := finalKey
				rocksBatch.DeleteCF(cf, []byte(dataKey))
				rocksBatch.DeleteCF(cf, []byte(ttlExpireIndexKey))
			}
		default:
			return fmt.Errorf("unsupported operation type: %s", op.Type)
		}
	}

	wo := grocksdb.NewDefaultWriteOptions()
	defer wo.Destroy()

	return r.DB.Write(wo, rocksBatch)
}

func (r *RocksdbStore) WriteRaw(batch *WriteBatch) error {
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
			finalKey := fmt.Sprintf("%s:%s", op.CFS, op.Key)
			rocksBatch.PutCF(cf, []byte(finalKey), op.Value)
		case "delete":
			finalKey := fmt.Sprintf("%s:%s", op.CFS, op.Key)
			rocksBatch.DeleteCF(cf, []byte(finalKey))
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
func (r *RocksdbStore) Iterate(fn func(cfName string, cfSelector string, key, value []byte) error) error {
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

			keyParts := strings.SplitN(string(key.Data()), ":", 2)
			if len(keyParts) != 2 {
				key.Free()
				value.Free()
				continue
			}

			err := fn(cfName, keyParts[0], []byte(keyParts[1]), value.Data())

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

// CreateColumnFamily creates a new column family in the RocksDB store.
// If the column family already exists, it returns an error.
// Parameters:
//   - columnFamilyName: The name of the column family to create.
//   - isTtl: A boolean indicating whether the new column family should be a TTL column family.
//
// Returns:
//   - An error if the column family already exists or if any RocksDB error occurs.
func (r *RocksdbStore) CreateColumnFamily(columnFamilyName string, isTtl bool) error {
	if _, exists := r.currentCFs[columnFamilyName]; exists {
		return fmt.Errorf("column family %s already exists", columnFamilyName)
	}

	opts := grocksdb.NewDefaultOptions()
	defer opts.Destroy()

	cfHandle, err := r.DB.CreateColumnFamily(opts, columnFamilyName)
	if err != nil {
		return fmt.Errorf("failed to create column family %s: %w", columnFamilyName, err)
	}

	if isTtl {
		r.TTLColumnFamilyHandles[columnFamilyName] = cfHandle
	} else {
		r.ColumnFamilyHandles[columnFamilyName] = cfHandle
	}
	r.currentCFs[columnFamilyName] = true
	return nil
}

// DeleteColumnFamily deletes a column family from the RocksDB store.
// Parameters:
//   - columnFamilyName: The name of the column family to delete.
//
// Returns:
//   - An error if the column family does not exist or if any RocksDB error occurs.
func (r *RocksdbStore) DeleteColumnFamily(columnFamilyName string) error {
	cfHandle, isTtl, err := r.resolveColumnFamily(columnFamilyName)
	if err != nil {
		return err // Column family not found
	}

	err = r.DB.DropColumnFamily(cfHandle)
	if err != nil {
		return fmt.Errorf("failed to delete column family %s: %w", columnFamilyName, err)
	}

	// Remove from internal tracking
	if isTtl {
		delete(r.TTLColumnFamilyHandles, columnFamilyName)
	} else {
		delete(r.ColumnFamilyHandles, columnFamilyName)
	}
	delete(r.currentCFs, columnFamilyName)
	cfHandle.Destroy() // Release the handle after dropping
	return nil
}

// ExistsColumnFamily checks if a column family exists in the RocksDB store.
// Parameters:
//   - columnFamilyName: The name of the column family to check.
//
// Returns:
//   - A boolean indicating whether the column family exists.
//   - A boolean indicating whether it is a TTL column family.
//   - An error if any RocksDB error occurs.
func (r *RocksdbStore) ExistsColumnFamily(columnFamilyName string) (bool, bool, error) {
	_, isTtl, err := r.resolveColumnFamily(columnFamilyName)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return false, false, nil
		}
		return false, false, err // Other error
	}
	return true, isTtl, nil
}

func (r *RocksdbStore) CleanExpiredKeys(now time.Time) error {
	for name, handle := range r.TTLColumnFamilyHandles {
		err := cleanExpiredKeys(r.DB, handle, now)
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
//   - now: The current time.
//
// Returns:
//   - An error if iterator operations or batch write operations fail.
func cleanExpiredKeys(db_instance *grocksdb.DB, cf *grocksdb.ColumnFamilyHandle, now time.Time) error {
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

	nowMillis := now.UnixMilli()
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

		dataKey := []byte(originalKey)
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
