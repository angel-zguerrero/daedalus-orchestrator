package db

import "github.com/linxGnu/grocksdb"

const ()

func OpenTenantDB(dbPath string) (*grocksdb.DB, map[string]*grocksdb.ColumnFamilyHandle, error) {
	return OpenDB(dbPath, []string{})
}
