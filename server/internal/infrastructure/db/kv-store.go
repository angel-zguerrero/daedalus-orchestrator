package db

type WriteBatch interface {
}

type KVStore interface {
	Get(key string) ([]byte, error)
	Put(key string, value []byte) error
	Write(batch interface{}) error
	DumpAll() (interface{}, error)
	Iterate(fn func(key, value []byte) error) error
	ClearAll() error
	Flush() error
}
