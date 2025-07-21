//go:build !rocksdb

package db

import (
	"fmt"
	"time"
)

// RocksdbStore is a placeholder for when RocksDB is not compiled.
type RocksdbStore struct{}

// CreateRocksdbStore returns an error as RocksDB is not supported in this build.
// The return type is KVStore, which this placeholder must conform to if it were to be successfully returned.
func CreateRocksdbStore(dbPath string, columnFamilyNames []string, ttlColumnFamilyNames []string) (KVStore, error) {
	return nil, fmt.Errorf("RocksDB support is not compiled into this build. Please build with '-tags rocksdb'")
}

// Get is a placeholder method for RocksdbStore.
func (s *RocksdbStore) Get(columnFamily, columnFamilySector, key string, now time.Time) ([]byte, error) {
	return nil, fmt.Errorf("RocksDB not supported in this build")
}

// Delete is a placeholder method for RocksdbStore.
func (s *RocksdbStore) Delete(columnFamily, columnFamilySector, key string, now time.Time) error {
	return fmt.Errorf("RocksDB not supported in this build")
}

// Put is a placeholder method for RocksdbStore.
func (s *RocksdbStore) Put(columnFamily, columnFamilySector, key string, value []byte, ttl int, now time.Time) error {
	return fmt.Errorf("RocksDB not supported in this build")
}

// PutRaw is a placeholder method for RocksdbStore.
func (s *RocksdbStore) PutRaw(columnFamily, columnFamilySector, key string, value []byte) error {
	return fmt.Errorf("RocksDB not supported in this build")
}

// Write is a placeholder method for RocksdbStore.
func (s *RocksdbStore) Write(batch *WriteBatch) error {
	return fmt.Errorf("RocksDB not supported in this build")
}

// WriteRaw is a placeholder method for RocksdbStore.
func (s *RocksdbStore) WriteRaw(batch *WriteBatch) error {
	return fmt.Errorf("RocksDB not supported in this build")
}

// SearchByPatternPaginatedKV is a placeholder method for RocksdbStore.
func (s *RocksdbStore) SearchByPatternPaginatedKV(cfName, cfSector, pattern, cursor string, limit int, now time.Time) ([]KeyValuePair, string, error) {
	return nil, "", fmt.Errorf("RocksDB not supported in this build")
}

// Exists is a placeholder method for RocksdbStore.
func (s *RocksdbStore) Exists(cfName, cfSector, key string, now time.Time) (bool, error) {
	return false, fmt.Errorf("RocksDB not supported in this build")
}

// DumpAll is a placeholder method for RocksdbStore.
func (s *RocksdbStore) DumpAll() (interface{}, error) {
	return nil, fmt.Errorf("RocksDB not supported in this build")
}

// Iterate is a placeholder method for RocksdbStore.
func (s *RocksdbStore) Iterate(fn func(cfName string, cfSector string, key, value []byte) error) error {
	return fmt.Errorf("RocksDB not supported in this build")
}

// ClearAll is a placeholder method for RocksdbStore.
func (s *RocksdbStore) ClearAll() error {
	return fmt.Errorf("RocksDB not supported in this build")
}

// Flush is a placeholder method for RocksdbStore.
func (s *RocksdbStore) Flush() error {
	return fmt.Errorf("RocksDB not supported in this build")
}

// Close is a placeholder method for RocksdbStore.
func (s *RocksdbStore) Close() error {
	return fmt.Errorf("RocksDB not supported in this build")
}

// CleanExpiredKeys is a placeholder method for RocksdbStore.
func (s *RocksdbStore) CleanExpiredKeys(now time.Time) error {
	return fmt.Errorf("RocksDB not supported in this build")
}

// CreateColumnFamily is a placeholder method for RocksdbStore.
func (s *RocksdbStore) CreateColumnFamily(columnFamilyName string, isTtl bool) error {
	return fmt.Errorf("RocksDB not supported in this build")
}

// DeleteColumnFamily is a placeholder method for RocksdbStore.
func (s *RocksdbStore) DeleteColumnFamily(columnFamilyName string) error {
	return fmt.Errorf("RocksDB not supported in this build")
}

// ExistsColumnFamily is a placeholder method for RocksdbStore.
func (s *RocksdbStore) ExistsColumnFamily(columnFamilyName string) (bool, bool, error) {
	return false, false, fmt.Errorf("RocksDB not supported in this build")
}
