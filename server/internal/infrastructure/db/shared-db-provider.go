package db

import (
	"deadalus-orch/server/internal/pkg/config"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/rs/zerolog/log"
)

// SharedDBProvider manages a single shared KVStore instance used by all Raft shards.
// Instead of each shard opening its own database (which limits scalability due to
// file descriptors, memory, and compaction overhead), all shards share one database.
// Data isolation between tenants is maintained through Column Family prefixes.
//
// Thread-safe: all methods can be called concurrently from multiple goroutines.
type SharedDBProvider struct {
	mu    sync.Mutex
	store KVStore
	path  string
	refs  int32 // number of active references (shards using this DB)
}

// NewSharedDBProvider creates a new SharedDBProvider.
// The database is not opened until the first call to Acquire().
func NewSharedDBProvider() *SharedDBProvider {
	return &SharedDBProvider{}
}

// Acquire returns the shared KVStore instance, opening the database if needed.
// Each caller must eventually call Release() when it no longer needs the store.
// The database path is determined by the PathProvider on first open.
func (p *SharedDBProvider) Acquire(pathProvider PathProvider) (KVStore, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.store != nil {
		atomic.AddInt32(&p.refs, 1)
		return p.store, nil
	}

	dbPath, err := pathProvider.GetDatabasePath()
	if err != nil {
		return nil, fmt.Errorf("SharedDBProvider: failed to get database path: %w", err)
	}

	store, err := openSharedDB(dbPath)
	if err != nil {
		return nil, fmt.Errorf("SharedDBProvider: failed to open shared database at %s: %w", dbPath, err)
	}

	p.store = store
	p.path = dbPath
	atomic.AddInt32(&p.refs, 1)

	log.Info().
		Str("path", dbPath).
		Msg("🗄️ Shared database opened successfully")

	return p.store, nil
}

// Release decrements the reference count. When the last reference is released,
// the database is closed.
func (p *SharedDBProvider) Release() error {
	newRefs := atomic.AddInt32(&p.refs, -1)
	if newRefs > 0 {
		return nil
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.store == nil {
		return nil
	}

	log.Info().
		Str("path", p.path).
		Msg("🗄️ Closing shared database (last reference released)")

	err := p.store.Close()
	p.store = nil
	p.path = ""
	return err
}

// GetStore returns the current shared KVStore instance without incrementing the
// reference count. Returns nil if the database has not been opened yet.
// This is useful for read-only access when a reference is already held.
func (p *SharedDBProvider) GetStore() KVStore {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.store
}

// openSharedDB opens a shared database with all required column families
// for both master and tenant shards.
func openSharedDB(dbPath string) (KVStore, error) {
	log.Info().Str("shared-db-path", dbPath).Msg("Opening shared DB in path")

	// The shared DB needs column families from both master and tenant configurations.
	// Normal CFs: AdminFC (used by both master and tenant shards)
	// TTL CFs: MasterEventFC (master events), TenantEventFC (tenant events)
	// Dynamic CFs (cf-n-X) are created on-the-fly when tenants are assigned.
	columnFamilies := []string{AdminFC}
	ttlColumnFamilies := []string{MasterEventFC, TenantEventFC}

	engine := config.GlobalConfiguration.TenantDBEngine
	if config.GlobalConfiguration.MasterDBEngine != "" {
		engine = config.GlobalConfiguration.MasterDBEngine
	}

	switch engine {
	case "rocksdb":
		log.Info().Str("engine", "rocksdb").Msg("Opening shared DB with RocksDB engine")
		return CreateRocksdbStore(dbPath, columnFamilies, ttlColumnFamilies)
	case "pebble":
		log.Info().Str("engine", "pebble").Msg("Opening shared DB with Pebble engine")
		return CreatePebbleStore(dbPath, columnFamilies, ttlColumnFamilies)
	default:
		log.Warn().Str("engine_name", engine).Msg("Unrecognized DB engine specified, defaulting to pebble.")
		return CreatePebbleStore(dbPath, columnFamilies, ttlColumnFamilies)
	}
}
