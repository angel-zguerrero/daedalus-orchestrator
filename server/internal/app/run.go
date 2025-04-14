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

	if err := db.BootstrapRootUser(dbConn, configMap); err != nil {
		log.Fatalf("❌ Bootstrap failed: %v", err)
	}

	err = server.StartGRPC(configMap, dbConn)
	if err != nil {
		log.Fatalf("❌ Server failed: %v", err)
	}
}
