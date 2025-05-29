package db

import (
	"fmt"

	"strings"

	"github.com/linxGnu/grocksdb"
)

// RocksdbStore is an implementation of the KVStore interface using RocksDB as the underlying storage engine.
// It holds a reference to the RocksDB database instance and maps of column family handles
// for both regular and TTL (Time-To-Live) column families.
type RocksdbStore struct {
	*grocksdb.DB                                          // Embedded RocksDB database instance.
	ColumnFamilyHandles    map[string]*grocksdb.ColumnFamilyHandle // Map of regular column family names to their handles.
	TTLColumnFamilyHandles map[string]*grocksdb.ColumnFamilyHandle // Map of TTL column family names to their handles.
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
	cf, ok := r.ColumnFamilyHandles[columnFamily]
	if !ok {
		cf, ok = r.TTLColumnFamilyHandles[columnFamily]
		if !ok {
			return nil, fmt.Errorf("column family %s not found", columnFamily)
		}
	}
	r.DB.GetColumnFamilyMetadata()
	ro := grocksdb.NewDefaultReadOptions()
	defer ro.Destroy()
	slice, err := r.DB.GetCF(ro, cf, []byte(key))
	if err != nil {
		return nil, err
	}
	defer slice.Free()
	if slice.Exists() {
		data := append([]byte(nil), slice.Data()...)
		return data, nil
	}
	return nil, nil
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
func (r *RocksdbStore) Write(batch interface{}) error {

	wo := grocksdb.NewDefaultWriteOptions()
	defer wo.Destroy()
	batch_, ok := batch.(*grocksdb.WriteBatch)
	if !ok {
		return fmt.Errorf("invalid batch type")
	}
	return r.DB.Write(wo, batch_)
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
