package db

import "github.com/linxGnu/grocksdb"

type sliceWrapper struct {
	s *grocksdb.Slice
}

func (w *sliceWrapper) Data() []byte {
	return w.s.Data()
}

func (w *sliceWrapper) Free() {
	w.s.Free()
}

func (w *sliceWrapper) Exists() bool {
	return w.s.Exists()
}

type RocksdbStore struct {
	*grocksdb.DB
}

func (r *RocksdbStore) Get(ro *grocksdb.ReadOptions, key []byte) (Slice, error) {
	slice, err := r.DB.Get(ro, key)
	if err != nil {
		return nil, err
	}
	return &sliceWrapper{s: slice}, nil
}

func (r *RocksdbStore) Put(wo *grocksdb.WriteOptions, key, value []byte) error {
	return r.DB.Put(wo, key, value)
}

func (r *RocksdbStore) Write(wo *grocksdb.WriteOptions, batch *grocksdb.WriteBatch) error {
	return r.DB.Write(wo, batch)
}
