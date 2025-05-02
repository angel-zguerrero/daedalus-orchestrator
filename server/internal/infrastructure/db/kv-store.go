package db

type WriteBatch interface {
}

type KVStore interface {
	Get(columnFamily string, key string) ([]byte, error)
	Put(columnFamily string, key string, value []byte) error
	Write(batch interface{}) error
	DumpAll() (interface{}, error)
	Iterate(fn func(cfName string, key, value []byte) error) error
	ClearAll() error
	Flush() error
}
