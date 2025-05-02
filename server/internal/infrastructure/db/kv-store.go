package db

type WriteBatch interface {
}

type KVStore interface {
	Get(key []byte) ([]byte, error)
	Put(key, value []byte) error
	Write(batch interface{}) error
	DumpAll() (interface{}, error)
	Iterate(fn func(key, value []byte) error) error
	ClearAll() error
	Flush() error
}
