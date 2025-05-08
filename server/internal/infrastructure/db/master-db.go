package db

import "github.com/linxGnu/grocksdb"

const (
	AdminFC = "admin"
	MetaFC  = "meta"
)

func OpenMasterDB(dbPath string) (*grocksdb.DB, map[string]*grocksdb.ColumnFamilyHandle, error) {
	return OpenDB(dbPath, []string{AdminFC, MetaFC})
}
