package db

import "github.com/linxGnu/grocksdb"

const (
	TenantEventFC = "tenant-events"
)

// OpenTenantDB opens the tenant database at the specified path.
// It is a convenience function that calls OpenDB with predefined column families
// specific to a tenant database:
//   - Regular Column Families: None predefined (empty list).
//   - TTL Column Families: TenantEventFC ("tenant-events")
//
// Parameters:
//   - dbPath: The file system path where the tenant RocksDB database is located or will be created.
//
// Returns:
//   - A pointer to the opened grocksdb.DB instance.
//   - A map of normal column family names to their grocksdb.ColumnFamilyHandle (will be empty unless "default" or "meta" are created by OpenDB).
//   - A map of TTL column family names to their grocksdb.ColumnFamilyHandle (containing "tenant-events").
//   - An error if the database cannot be opened.
func OpenTenantDB(dbPath string) (*grocksdb.DB, map[string]*grocksdb.ColumnFamilyHandle, map[string]*grocksdb.ColumnFamilyHandle, error) {
	return OpenDB(dbPath, []string{}, []string{TenantEventFC})
}
