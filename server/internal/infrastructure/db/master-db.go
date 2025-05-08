package db

import "github.com/linxGnu/grocksdb"

const (
	AdminFC       = "admin"
	MasterEventFC = "master-events"
)

func OpenMasterDB(dbPath string) (*grocksdb.DB, map[string]*grocksdb.ColumnFamilyHandle, error) {
	return OpenDB(dbPath, []string{AdminFC})
}
