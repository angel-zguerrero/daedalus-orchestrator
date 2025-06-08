package db

const (
	AdminFC       = "admin"
	MasterEventFC = "master-events"
)

func OpenMasterDB(dbPath string) (KVStore, error) {
	return CreateRocksdbStore(dbPath, []string{AdminFC}, []string{MasterEventFC})
}
