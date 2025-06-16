package db

type DelegatedKVStore struct {
	base  KVStore
	batch *WriteBatch
}

func NewDelegatedKVStore(base KVStore, batch *WriteBatch) *DelegatedKVStore {
	return &DelegatedKVStore{
		base:  base,
		batch: batch,
	}
}

// Get simplemente delega
func (d *DelegatedKVStore) Get(cf string, key string) ([]byte, error) {
	return d.base.Get(cf, key)
}

// Delete interceptado y va al batch
func (d *DelegatedKVStore) Delete(cf string, key string) error {
	d.batch.Delete(cf, key)
	return nil
}

// Put interceptado, soporta TTL
func (d *DelegatedKVStore) Put(cf string, key string, value []byte, ttl int) error {
	if ttl > 0 {
		d.batch.PutTTl(cf, key, value, ttl)
	} else {
		d.batch.Put(cf, key, value)
	}
	return nil
}

// PutRaw = sin TTL
func (d *DelegatedKVStore) PutRaw(cf string, key string, value []byte) error {
	d.batch.Put(cf, key, value)
	return nil
}

// Write es ignorado (ya estamos usando el batch del UnitOfWork)
func (d *DelegatedKVStore) Write(batch *WriteBatch) error {

	if batch == nil || batch == d.batch {
		return nil
	}
	d.batch.Data = append(d.batch.Data, batch.Data...)
	return nil
}

// WriteRaw igual que Write (dependiendo de cómo la uses)
func (d *DelegatedKVStore) WriteRaw(batch *WriteBatch) error {
	if batch == nil || batch == d.batch {
		return nil
	}
	d.batch.Data = append(d.batch.Data, batch.Data...)
	return nil
}

// Resto delegados
func (d *DelegatedKVStore) SearchByPatternPaginatedKV(cfName, pattern, cursor string, limit int) ([]KeyValuePair, string, error) {
	return d.base.SearchByPatternPaginatedKV(cfName, pattern, cursor, limit)
}

func (d *DelegatedKVStore) Exists(cfName, key string) (bool, error) {
	return d.base.Exists(cfName, key)
}

func (d *DelegatedKVStore) DumpAll() (interface{}, error) {
	return d.base.DumpAll()
}

func (d *DelegatedKVStore) Iterate(fn func(cfName string, key, value []byte) error) error {
	return d.base.Iterate(fn)
}

func (d *DelegatedKVStore) ClearAll() error {
	return d.base.ClearAll()
}

func (d *DelegatedKVStore) Flush() error {
	return d.base.Flush()
}

func (d *DelegatedKVStore) Close() error {
	return d.base.Close()
}

func (d *DelegatedKVStore) CleanExpiredKeys() error {
	return d.base.CleanExpiredKeys()
}
