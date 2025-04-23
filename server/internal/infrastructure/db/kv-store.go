package db

type Slice interface {
	Data() []byte
	Free()
	Exists() bool
}

type WriteBatch interface {
}

type KVStore interface {
	Get(key []byte) (Slice, error)
	Put(key, value []byte) error
	Write(batch interface{}) error
	DumpAll() (interface{}, error)
	Iterate(fn func(key, value []byte) error) error
	ClearAll() error
}
