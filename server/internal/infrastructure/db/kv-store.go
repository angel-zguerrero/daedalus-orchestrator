package db

import "github.com/linxGnu/grocksdb"

type Slice interface {
	Data() []byte
	Free()
	Exists() bool
}

type KVStore interface {
	Get(ro *grocksdb.ReadOptions, key []byte) (Slice, error)
	Put(wo *grocksdb.WriteOptions, key, value []byte) error
	Write(wo *grocksdb.WriteOptions, batch *grocksdb.WriteBatch) error
}
