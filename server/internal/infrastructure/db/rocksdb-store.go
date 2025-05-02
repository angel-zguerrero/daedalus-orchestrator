package db

import (
	"fmt"

	"github.com/linxGnu/grocksdb"
)

type RocksdbStore struct {
	*grocksdb.DB
}

func (r *RocksdbStore) Get(key string) ([]byte, error) {

	ro := grocksdb.NewDefaultReadOptions()
	defer ro.Destroy()
	slice, err := r.DB.Get(ro, []byte(key))
	if err != nil {
		return nil, err
	}
	defer slice.Free()
	if slice.Exists() {
		data := append([]byte(nil), slice.Data()...)
		return data, nil
	}
	return nil, nil
}

func (r *RocksdbStore) Put(key string, value []byte) error {

	wo := grocksdb.NewDefaultWriteOptions()
	defer wo.Destroy()
	return r.DB.Put(wo, []byte(key), value)
}

func (r *RocksdbStore) Write(batch interface{}) error {

	wo := grocksdb.NewDefaultWriteOptions()
	defer wo.Destroy()
	batch_, ok := batch.(*grocksdb.WriteBatch)
	if !ok {
		return fmt.Errorf("invalid batch type")
	}
	return r.DB.Write(wo, batch_)
}

func (r *RocksdbStore) DumpAll() (interface{}, error) {

	ro := grocksdb.NewDefaultReadOptions()
	defer ro.Destroy()
	result := make(map[string][]byte)

	it := r.DB.NewIterator(ro)
	defer it.Close()

	for it.SeekToFirst(); it.Valid(); it.Next() {
		key := it.Key()
		value := it.Value()

		result[string(key.Data())] = append([]byte(nil), value.Data()...)

		key.Free()
		value.Free()
	}

	if err := it.Err(); err != nil {
		return nil, fmt.Errorf("iterator error: %w", err)
	}

	return result, nil
}

func (r *RocksdbStore) Iterate(fn func(key, value []byte) error) error {

	ro := grocksdb.NewDefaultReadOptions()
	defer ro.Destroy()

	it := r.DB.NewIterator(ro)
	defer it.Close()

	for it.SeekToFirst(); it.Valid(); it.Next() {
		key := it.Key()
		value := it.Value()

		err := fn(key.Data(), value.Data())

		key.Free()
		value.Free()

		if err != nil {
			return err
		}
	}

	if err := it.Err(); err != nil {
		return fmt.Errorf("iterator error: %w", err)
	}

	return nil
}

func (r *RocksdbStore) ClearAll() error {

	ro := grocksdb.NewDefaultReadOptions()
	defer ro.Destroy()

	wo := grocksdb.NewDefaultWriteOptions()
	defer wo.Destroy()

	it := r.DB.NewIterator(ro)
	defer it.Close()

	for it.SeekToFirst(); it.Valid(); it.Next() {
		key := it.Key()
		err := r.DB.Delete(wo, key.Data())
		key.Free()
		if err != nil {
			return fmt.Errorf("failed to delete key: %w", err)
		}
	}

	if err := it.Err(); err != nil {
		return fmt.Errorf("iterator error: %w", err)
	}

	return nil
}

func (r *RocksdbStore) Flush() error {

	fo := grocksdb.NewDefaultFlushOptions()
	defer fo.Destroy()
	fo.SetWait(true)

	err := r.DB.Flush(fo)
	if err != nil {
		return err
	}
	return r.DB.FlushWAL(true)
}
