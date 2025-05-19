package db

import (
	"fmt"

	"github.com/linxGnu/grocksdb"
)

type RocksdbStore struct {
	*grocksdb.DB
	ColumnFamilyHandles    map[string]*grocksdb.ColumnFamilyHandle
	TTLColumnFamilyHandles map[string]*grocksdb.ColumnFamilyHandle
}

func (r *RocksdbStore) Get(columnFamily, key string) ([]byte, error) {
	cf, ok := r.ColumnFamilyHandles[columnFamily]
	if !ok {
		cf, ok = r.TTLColumnFamilyHandles[columnFamily]
		if !ok {
			return nil, fmt.Errorf("column family %s not found", columnFamily)
		}
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
		cf, ok = r.TTLColumnFamilyHandles[columnFamily]
		if !ok {
			return fmt.Errorf("column family %s not found", columnFamily)
		}
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

	result := make(map[string]map[string][]byte)

	allCFs := map[string]*grocksdb.ColumnFamilyHandle{}

	for name, handle := range r.ColumnFamilyHandles {
		allCFs[name] = handle
	}
	for name, handle := range r.TTLColumnFamilyHandles {
		if _, exists := allCFs[name]; !exists {
			allCFs[name] = handle
		}
	}

	for cfName, cfHandle := range allCFs {
		cfResult := make(map[string][]byte)
		it := r.DB.NewIteratorCF(ro, cfHandle)
		defer it.Close()

		for it.SeekToFirst(); it.Valid(); it.Next() {
			key := it.Key()
			value := it.Value()

			cfResult[string(key.Data())] = append([]byte(nil), value.Data()...)

			key.Free()
			value.Free()
		}

		if err := it.Err(); err != nil {
			return nil, fmt.Errorf("iterator error in CF %s: %w", cfName, err)
		}

		result[cfName] = cfResult
	}

	return result, nil
}

func (r *RocksdbStore) Iterate(fn func(cfName string, key, value []byte) error) error {
	ro := grocksdb.NewDefaultReadOptions()
	defer ro.Destroy()

	allCFs := map[string]*grocksdb.ColumnFamilyHandle{}

	for name, handle := range r.ColumnFamilyHandles {
		allCFs[name] = handle
	}
	for name, handle := range r.TTLColumnFamilyHandles {
		if _, exists := allCFs[name]; !exists {
			allCFs[name] = handle
		}
	}

	for cfName, cfHandle := range allCFs {
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
	// Combinamos ColumnFamilyHandles y TTLColumnFamilyHandles en un solo mapa
	allCFs := map[string]*grocksdb.ColumnFamilyHandle{}

	// Añadimos las ColumnFamilyHandles
	for name, handle := range r.ColumnFamilyHandles {
		allCFs[name] = handle
	}

	// Añadimos las TTLColumnFamilyHandles, sin duplicados
	for name, handle := range r.TTLColumnFamilyHandles {
		if _, exists := allCFs[name]; !exists {
			allCFs[name] = handle
		}
	}

	// Iteramos sobre todas las ColumnFamilyHandles combinadas
	for _, cf := range allCFs {
		ro := grocksdb.NewDefaultReadOptions()
		defer ro.Destroy()

		wo := grocksdb.NewDefaultWriteOptions()
		defer wo.Destroy()

		it := r.DB.NewIteratorCF(ro, cf)
		defer it.Close()

		// Eliminamos las claves de esta columna
		for it.SeekToFirst(); it.Valid(); it.Next() {
			key := it.Key()
			err := r.DB.DeleteCF(wo, cf, key.Data())
			key.Free()
			if err != nil {
				return fmt.Errorf("failed to delete key from column family: %w", err)
			}
		}

		// Verificamos si hubo algún error durante la iteración
		if err := it.Err(); err != nil {
			return fmt.Errorf("iterator error: %w", err)
		}
	}

	return nil
}

func (r *RocksdbStore) Flush() error {
	fo := grocksdb.NewDefaultFlushOptions()
	defer fo.Destroy()
	fo.SetWait(true)

	allCFs := map[string]*grocksdb.ColumnFamilyHandle{}

	for name, handle := range r.ColumnFamilyHandles {
		allCFs[name] = handle
	}

	for name, handle := range r.TTLColumnFamilyHandles {
		if _, exists := allCFs[name]; !exists {
			allCFs[name] = handle
		}
	}

	for _, cf := range allCFs {
		err := r.DB.FlushCF(cf, fo)
		if err != nil {
			return fmt.Errorf("failed to flush column family: %w", err)
		}
	}

	err := r.DB.Flush(fo)
	if err != nil {
		return fmt.Errorf("failed to flush: %w", err)
	}

	return r.DB.FlushWAL(true)
}

func (r *RocksdbStore) Close() error {
	r.DB.Close()
	return nil
}
