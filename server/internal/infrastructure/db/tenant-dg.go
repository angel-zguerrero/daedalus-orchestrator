package db

const (
	TenantEventFC = "tenant-events"
)

func OpenTenantDB(dbPath string) (KVStore, error) {
	return CreateRocksdbStore(dbPath, []string{}, []string{TenantEventFC})
}
