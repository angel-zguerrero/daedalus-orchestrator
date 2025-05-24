package app

import (
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	"deadalus-orch/server/internal/pkg/config"
	"deadalus-orch/server/internal/pkg/utils"
	"deadalus-orch/server/internal/telemetry"
	"deadalus-orch/shared/constants"
	"os"
	"time"

	dblog "github.com/lni/dragonboat/v4/logger"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func Run(replicaID int, roles []dragonboat.NodeRole, selfMember dragonboat.Member, initialMembers []dragonboat.Member, join bool) {

	err := utils.ValidateEnvVar()
	if err != nil {
		log.Fatal().
			Err(err).
			Str("package", "app").
			Str("func", "Run").
			Msgf("❌ Failed validation of ENV var")
	}
	err = config.LoadOrDefault("")
	if err != nil {
		log.Fatal().
			Err(err).
			Str("package", "app").
			Str("func", "Run").
			Msgf("❌ Failed loading configuration")
	}

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	if os.Getenv("LOGGER_FORMAT") == "pretty" || os.Getenv("LOGGER_FORMAT") == "" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	if os.Getenv(constants.EnvVarEnvKey) == string(constants.PRODUCTION) {
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)

	} else {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
	dblog.SetLoggerFactory(dragonboat.CreateZerologger)

	log.Info().Interface("", roles).Msg("This node has these roles")

	ctx, tp, err := telemetry.Init(
		constants.Env(os.Getenv(constants.EnvVarEnvKey)),
		os.Getenv(constants.EnvVarOtelActived) == constants.OTEL_ACTIVE_TRUE,
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

	masterNode, err := dragonboat.InitMasterNode(uint64(replicaID), selfMember, initialMembers, join, roles)
	if err != nil {
		log.Fatal().
			Err(err).
			Str("package", "app").
			Str("func", "Run").
			Msgf("❌ Failed Init raft Master node")
	}

	go func() {
		interval := 3 * time.Second
		readyUpdates := masterNode.StartNodeReadyWatcher(interval)
		for isReady := range readyUpdates {
			if isReady {
				log.Info().Msg("✅ Node is ready for consensus.")
			} else {
				log.Info().Msg("⚠️ Node is NOT ready for consensus.")
			}
		}
	}()

	/*
		dbConn, columnFamilyHandles, err := db.InitDB(configMap.DBname, db.DefaultPathProvider{})
		if err != nil {

			log.Fatal().
				Err(err).
				Str("package", "app").
				Str("func", "Run").
				Msgf("❌ Failed to init DB")
		}

		defer dbConn.Close()

		rocksdbStore := &db.RocksdbStore{DB: dbConn, ColumnFamilyHandles: columnFamilyHandles}
		if err := db.BootstrapRootUser(rocksdbStore, *configMap); err != nil {
			log.Fatal().
				Err(err).
				Str("package", "app").
				Str("func", "Run").
				Msgf("❌ Bootstrap failed")
		}
	*/
	/*
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
	*/
}
