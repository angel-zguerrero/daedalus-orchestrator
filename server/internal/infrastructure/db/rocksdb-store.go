package db

import (
	"fmt"

	"github.com/linxGnu/grocksdb"
)

type RocksdbStore struct {
	*grocksdb.DB
	ColumnFamilyHandles map[string]*grocksdb.ColumnFamilyHandle
}

func (r *RocksdbStore) Get(columnFamily, key string) ([]byte, error) {
	cf, ok := r.ColumnFamilyHandles[columnFamily]
	if !ok {
		return nil, fmt.Errorf("column family %s not found", columnFamily)
	}
	r.DB.GetColumnFamilyMetadata()
	ro := grocksdb.NewDefaultReadOptions()
	defer ro.Destroy()
	slice, err := r.DB.GetCF(ro, cf, []byte(key))
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

func (r *RocksdbStore) Put(columnFamily, key string, value []byte) error {
	cf, ok := r.ColumnFamilyHandles[columnFamily]
	if !ok {
		return fmt.Errorf("column family %s not found", columnFamily)
	}
	wo := grocksdb.NewDefaultWriteOptions()
	defer wo.Destroy()
	return r.DB.PutCF(wo, cf, []byte(key), value)
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

func (r *RocksdbStore) Iterate(fn func(cfName string, key, value []byte) error) error {
	ro := grocksdb.NewDefaultReadOptions()
	defer ro.Destroy()

	for cfName, cfHandle := range r.ColumnFamilyHandles {
		it := r.DB.NewIteratorCF(ro, cfHandle)
		defer it.Close()

		for it.SeekToFirst(); it.Valid(); it.Next() {
			key := it.Key()
			value := it.Value()

			err := fn(cfName, key.Data(), value.Data())

			key.Free()
			value.Free()

			if err != nil {
				return err
			}
		}

		if err := it.Err(); err != nil {
			return fmt.Errorf("iterator error in CF %s: %w", cfName, err)
		}
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
func (r *RocksdbStore) Close() error {
	r.DB.Close()
	return nil
}
