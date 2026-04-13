package db

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

// Get checks the pending batch first (last write wins), then falls through to base.
// This ensures that multiple commands executing within the same Dragonboat Update batch
// see each other's pending writes, avoiding lost-update anomalies on shared counters
// (e.g. CurrentDeliveringMessages).
func (d *DelegatedKVStore) Get(cf, cfs, key string, now time.Time) ([]byte, error) {
	// Walk backwards so the most-recent pending operation wins.
	for i := len(d.batch.Data) - 1; i >= 0; i-- {
		entry := d.batch.Data[i]
		if entry.CF == cf && entry.CFS == cfs && entry.Key == key {
			if entry.Type == "delete" {
				return nil, nil // key was deleted in this batch
			}
			// "put" — return the pending value
			return entry.Value, nil
		}
	}
	return d.base.Get(cf, cfs, key, now)
}

// Delete interceptado y va al batch
// The `now` parameter is not used here as the actual deletion happens when the batch is written.
func (d *DelegatedKVStore) Delete(cf, cfs, key string, now time.Time) error {
	d.batch.Delete(cf, cfs, key, now)
	return nil
}

// Put interceptado, soporta TTL
// The `now` parameter is not used here as the actual put happens when the batch is written.
func (d *DelegatedKVStore) Put(cf, cfs, key string, value []byte, ttl int, now time.Time) error {
	if ttl > 0 {
		d.batch.PutTTl(cf, cfs, key, value, ttl, now)
	} else {
		d.batch.Put(cf, cfs, key, value, now)
	}
	return nil
}

// PutRaw = sin TTL
func (d *DelegatedKVStore) PutRaw(cf, cfs, key string, value []byte) error {
	d.batch.Put(cf, cfs, key, value, time.Now())
	return nil
}

// Write delegates to the base store's Write method, passing the `now` parameter.
// It's assumed that if d.batch is the one being written, the caller (e.g. UnitOfWork)
// will call base.Write(d.batch, now) directly. This Write method is for other batches.
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
	// This method does not have `now`. If it needs to interact with TTL-aware
	// base.WriteRaw, this signature would need to change, or WriteRaw in KVStore
	// should not be TTL sensitive. Assuming WriteRaw is not TTL sensitive for now.
	d.batch.Data = append(d.batch.Data, batch.Data...)
	return nil
}

// Resto delegados
func (d *DelegatedKVStore) SearchByPatternPaginatedKV(cfName, cfs, pattern, cursor string, limit int, now time.Time) ([]KeyValuePair, string, error) {
	return d.base.SearchByPatternPaginatedKV(cfName, cfs, pattern, cursor, limit, now)
}

func (d *DelegatedKVStore) Exists(cfName, cfs, key string, now time.Time) (bool, error) {
	// Check pending batch first.
	for i := len(d.batch.Data) - 1; i >= 0; i-- {
		entry := d.batch.Data[i]
		if entry.CF == cfName && entry.CFS == cfs && entry.Key == key {
			return entry.Type != "delete", nil
		}
	}
	return d.base.Exists(cfName, cfs, key, now)
}

func (d *DelegatedKVStore) DumpAll() (interface{}, error) {
	return d.base.DumpAll()
}

func (d *DelegatedKVStore) Iterate(fn func(cfName string, cfSector string, key, value []byte) error) error {
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

func (d *DelegatedKVStore) CreateColumnFamily(columnFamilyName string, isTtl bool) error {
	return d.base.CreateColumnFamily(columnFamilyName, isTtl)
}

func (d *DelegatedKVStore) DeleteColumnFamily(columnFamilyName string) error {
	return d.base.DeleteColumnFamily(columnFamilyName)
}

func (d *DelegatedKVStore) ExistsColumnFamily(columnFamilyName string) (bool, bool, error) {
	return d.base.ExistsColumnFamily(columnFamilyName)
}
