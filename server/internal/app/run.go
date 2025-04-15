package app

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/infrastructure/server"
	"deadalus-orch/server/internal/pkg/config"
	"flag"
	"log"
)

func Run() {
	flagConfig := flag.String("config", "", "Path to the daedalus.conf configuration file (optional)")
	flag.Parse()

	configMap, err := config.LoadOrDefault(*flagConfig)
	if err != nil {
		log.Fatalf("❌ Failed loading configuration: %v", err)
	}
	// Database run
	dbConn, err := db.InitDB(configMap["db_name"])
	if err != nil {
		log.Fatalf("❌ Failed to init DB: %v", err)
	}

	defer dbConn.Close()

	rocksdbStore := &db.RocksdbStore{DB: dbConn}
	if err := db.BootstrapRootUser(rocksdbStore, configMap); err != nil {
		log.Fatalf("❌ Bootstrap failed: %v", err)
	}

	err = server.StartGRPC(
		configMap,
		server.DefaultListener,
		server.DefaultGRPCServerFactory,
	)
	if err != nil {
		log.Fatalf("Failed to start gRPC server: %v", err)
	}
}
