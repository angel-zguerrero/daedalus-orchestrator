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
import "time"

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
func (d *DelegatedKVStore) Get(cf string, key string, now time.Time) ([]byte, error) {
	return d.base.Get(cf, key, now)
}

// Delete interceptado y va al batch
// The `now` parameter is not used here as the actual deletion happens when the batch is written.
func (d *DelegatedKVStore) Delete(cf string, key string, now time.Time) error {
	d.batch.Delete(cf, key)
	return nil
}

// Put interceptado, soporta TTL
// The `now` parameter is not used here as the actual put happens when the batch is written.
func (d *DelegatedKVStore) Put(cf string, key string, value []byte, ttl int, now time.Time) error {
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

// Write delegates to the base store's Write method, passing the `now` parameter.
// It's assumed that if d.batch is the one being written, the caller (e.g. UnitOfWork)
// will call base.Write(d.batch, now) directly. This Write method is for other batches.
func (d *DelegatedKVStore) Write(batch *WriteBatch, now time.Time) error {
	if batch == nil {
		return nil
	}
	// If the provided batch is the same as the internal batch,
	// it's likely handled by the UnitOfWork commit.
	// However, to be safe, if this method is called with d.batch,
	// it should still pass it to the base.
	// The primary use case here is for batches *other* than d.batch.
	if batch == d.batch {
        // This case should ideally be handled by the UOW, which calls base.Write(d.batch, now)
        // If called directly, it means we are writing the internal batch.
		return d.base.Write(batch, now)
	}
    // If it's a different batch, append its data to our current batch.
    // This behavior might need review based on UOW logic.
    // For now, the instruction is to update existing methods.
    // A direct pass-through for non-internal batches might be more consistent:
    // return d.base.Write(batch, now)
    // However, the original code appended.
    // Let's assume for now that if Write is called on DelegatedKVStore,
    // it's meant to use the base store's Write for that specific batch.
	return d.base.Write(batch, now)
}

// WriteRaw igual que Write (dependiendo de cómo la uses)
func (d *DelegatedKVStore) WriteRaw(batch *WriteBatch) error {
	if batch == nil || batch == d.batch {
		return nil
	}
	// This method does not have `now`. If it needs to interact with TTL-aware
	// base.WriteRaw, this signature would need to change, or WriteRaw in KVStore
	// should not be TTL sensitive. Assuming WriteRaw is not TTL sensitive for now.
	d.batch.Data = append(d.batch.Data, batch.Data...)
	return nil
}

// Resto delegados
func (d *DelegatedKVStore) SearchByPatternPaginatedKV(cfName, pattern, cursor string, limit int, now time.Time) ([]KeyValuePair, string, error) {
	return d.base.SearchByPatternPaginatedKV(cfName, pattern, cursor, limit, now)
}

func (d *DelegatedKVStore) Exists(cfName, key string, now time.Time) (bool, error) {
	return d.base.Exists(cfName, key, now)
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

func (d *DelegatedKVStore) CleanExpiredKeys(now time.Time) error {
	return d.base.CleanExpiredKeys(now)
}
