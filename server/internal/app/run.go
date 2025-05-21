package app

import (
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	"deadalus-orch/server/internal/pkg/utils"
	"deadalus-orch/server/internal/telemetry"
	"deadalus-orch/shared/constants"
	"fmt"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func Run(replicaID int, roles []dragonboat.NodeRole, selfMember dragonboat.Member, initialMembers []dragonboat.Member, join bool) {
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

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	if os.Getenv(constants.EnvVarEnvKey) == string(constants.PRODUCTION) {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)

	}

	fmt.Println("This node has these roles: ", roles)
	dragonboat.InitMasterNode(uint64(replicaID), selfMember, initialMembers, join, roles)
	/*
		configMap, err := config.LoadOrDefault(*flagConfig)
		if err != nil {
			log.Fatal().
				Err(err).
				Str("package", "app").
				Str("func", "Run").
				Msgf("❌ Failed loading configuration")
		}

		fmt.Println(configMap)
	*/
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
