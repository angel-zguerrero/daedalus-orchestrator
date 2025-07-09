package db

import (
	"deadalus-orch/server/internal/pkg/config"

	"github.com/rs/zerolog/log"
)

const (
	AdminFC       = "admin"
	MasterEventFC = "master-events"
)

func OpenMasterDB(dbPath string) (KVStore, error) {
	log.Info().Str("master-db-path", dbPath).Msg("Opening master DB in path")
	engine := config.GlobalConfiguration.MasterDBEngine
	switch engine {
	case "rocksdb":
		log.Info().Str("engine", "rocksdb").Msg("Opening master DB with RocksDB engine")
		return CreateRocksdbStore(dbPath, []string{AdminFC}, []string{MasterEventFC})
	case "pebble":
		log.Info().Str("engine", "pebble").Msg("Opening master DB with Pebble engine")
		return CreatePebbleStore(dbPath, []string{AdminFC}, []string{MasterEventFC})
	default:
		log.Warn().Str("engine_name", engine).Msg("Unrecognized master DB engine specified, defaulting to pebble.")
		return CreatePebbleStore(dbPath, []string{AdminFC}, []string{MasterEventFC})
	}
}
