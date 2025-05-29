package db

import "github.com/linxGnu/grocksdb"

const (
	AdminFC       = "admin"
	MasterEventFC = "master-events"
)

// OpenMasterDB opens the master database at the specified path.
// It is a convenience function that calls OpenDB with predefined column families
// specific to the master database:
//   - Regular Column Families: AdminFC ("admin")
//   - TTL Column Families: MasterEventFC ("master-events")
//
// Parameters:
//   - dbPath: The file system path where the master RocksDB database is located or will be created.
//
// Returns:
//   - A pointer to the opened grocksdb.DB instance.
//   - A map of normal column family names to their grocksdb.ColumnFamilyHandle (containing "admin").
//   - A map of TTL column family names to their grocksdb.ColumnFamilyHandle (containing "master-events").
//   - An error if the database cannot be opened.
func OpenMasterDB(dbPath string) (*grocksdb.DB, map[string]*grocksdb.ColumnFamilyHandle, map[string]*grocksdb.ColumnFamilyHandle, error) {
	return OpenDB(dbPath, []string{AdminFC}, []string{MasterEventFC})
}
