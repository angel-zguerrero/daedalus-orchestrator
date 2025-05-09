package db

import "github.com/linxGnu/grocksdb"

const (
	TenantEventFC = "tenant-events"
)

func OpenTenantDB(dbPath string) (*grocksdb.DB, map[string]*grocksdb.ColumnFamilyHandle, map[string]*grocksdb.ColumnFamilyHandle, error) {
	return OpenDB(dbPath, []string{}, []string{TenantEventFC})
}
