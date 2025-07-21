package db

import (
	"deadalus-orch/server/internal/pkg/config"

	"github.com/rs/zerolog/log"
)

const (
	TenantEventFC         = "tenant-events"
	TenantEventFCSelector = "tenant-events-selector"
)

func OpenTenantDB(dbPath string) (KVStore, error) {
	engine := config.GlobalConfiguration.TenantDBEngine
	switch engine {
	case "rocksdb":
		log.Info().Str("engine", "rocksdb").Msg("Opening tenant DB with RocksDB engine")
		return CreateRocksdbStore(dbPath, []string{}, []string{TenantEventFC})
	case "pebble":
		log.Info().Str("engine", "pebble").Msg("Opening tenant DB with Pebble engine")
		return CreatePebbleStore(dbPath, []string{}, []string{TenantEventFC})
	default:
		log.Warn().Str("engine_name", engine).Msg("Unrecognized tenant DB engine specified, defaulting to pebble.")
		return CreatePebbleStore(dbPath, []string{}, []string{TenantEventFC})
	}
}
