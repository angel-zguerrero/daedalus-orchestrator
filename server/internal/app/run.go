package app

import (
	"deadalus-orch/server/internal/infrastructure/db"
	server "deadalus-orch/server/internal/infrastructure/server/grpc"
	"deadalus-orch/server/internal/pkg/config"
	"deadalus-orch/server/internal/pkg/utils"
	"deadalus-orch/server/internal/telemetry"
	"deadalus-orch/shared/constants"
	"flag"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func Run() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	if os.Getenv("LOGGER_FORMAT") == "pretty" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}

	err := utils.ValidateEnvVar()
	if err != nil {
		log.Fatal().
			Err(err).
			Str("package", "app").
			Str("func", "Run").
			Msgf("❌ Failed validation of ENV var")
	}

	ctx, tp, err := telemetry.Init(
		constants.Env(os.Getenv(constants.EnvVarEnvKey)),
		os.Getenv(constants.EnvVarOtelActived) == string(constants.OTEL_ACTIVE_TRUE),
		os.Getenv(constants.EnvVarOtelEndpoint),
		os.Getenv(constants.EnvVarOtelTracerServiceName),
	)

	if err != nil {
		log.Fatal().
			Err(err).
			Str("package", "app").
			Str("func", "Run").
			Msgf("❌ Failed Init Telemetry")
	}

	defer func() {
		_ = tp.Shutdown(ctx)
	}()

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	if os.Getenv(constants.EnvVarEnvKey) == string(constants.PRODUCTION) {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)

	}
	flagConfig := flag.String("config", "", "Path to the daedalus.conf configuration file (optional)")
	flag.Parse()

	configMap, err := config.LoadOrDefault(*flagConfig)
	if err != nil {
		log.Fatal().
			Err(err).
			Str("package", "app").
			Str("func", "Run").
			Msgf("❌ Failed loading configuration")
	}

	// Database run
	dbConn, err := db.InitDB(configMap.DBname, db.DefaultPathProvider{})
	if err != nil {

		log.Fatal().
			Err(err).
			Str("package", "app").
			Str("func", "Run").
			Msgf("❌ Failed to init DB")
	}

	defer dbConn.Close()

	rocksdbStore := &db.RocksdbStore{DB: dbConn}
	if err := db.BootstrapRootUser(rocksdbStore, *configMap); err != nil {
		log.Fatal().
			Err(err).
			Str("package", "app").
			Str("func", "Run").
			Msgf("❌ Bootstrap failed")
	}

	err = server.StartGRPC(
		*configMap,
		server.DefaultListener,
		server.DefaultGRPCServerFactory,
	)
	if err != nil {

		log.Fatal().
			Err(err).
			Str("package", "app").
			Str("func", "Run").
			Msgf("❌Failed to start gRPC server")
	}
}
