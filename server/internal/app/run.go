package app

import (
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	"deadalus-orch/server/internal/pkg/config"
	"deadalus-orch/server/internal/telemetry"
	"deadalus-orch/shared/constants"
	"os"
	"time"

	dblog "github.com/lni/dragonboat/v4/logger"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func Run() {

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

	selfMember, err := dragonboat.ParseMember(config.GlobalConfiguration.SelfMemberAddr)

	if err != nil {
		log.Fatal().
			Err(err).
			Str("package", "app").
			Str("func", "Run").
			Msgf("❌ Failed parsing self member")
	}

	initialMembers, err := dragonboat.ParseMembersFlag(&config.GlobalConfiguration.InitialMembers)
	if err != nil {
		log.Fatal().
			Err(err).
			Str("package", "app").
			Str("func", "Run").
			Msgf("❌ Failed Getting initial members")
	}

	roles, err := dragonboat.ParseRolesFlag(&config.GlobalConfiguration.Roles)

	if err != nil {
		log.Fatal().
			Err(err).
			Str("package", "app").
			Str("func", "Run").
			Msgf("❌ Failed parsing roles")
	}

	if config.GlobalConfiguration.Join {
		if config.GlobalConfiguration.ReplicaID == 0 {
			log.Fatal().
				Err(err).
				Str("package", "app").
				Str("func", "Run").
				Msgf("❌ Must specify --replica when joining a cluster.")
		}
	}

	if !config.GlobalConfiguration.Join {

		if !dragonboat.ContainsRole(roles, dragonboat.RoleConsensus) {
			log.Fatal().
				Err(err).
				Str("package", "app").
				Str("func", "Run").
				Msgf("❌ The role 'consensus' is required when creating a new cluster.")
		}
		if config.GlobalConfiguration.ReplicaID == 0 {
			log.Fatal().
				Err(err).
				Str("package", "app").
				Str("func", "Run").
				Msgf("❌ Must specify --replica when creating a new cluster.")
		}

		if len(config.GlobalConfiguration.InitialMembers) == 0 {
			log.Fatal().
				Err(err).
				Str("package", "app").
				Str("func", "Run").
				Msgf("❌ Must provide --initial-members when creating a new cluster.")
		}

		if !dragonboat.IsMemberInMemberArray(selfMember, initialMembers) {
			log.Fatal().
				Err(err).
				Str("package", "app").
				Str("func", "Run").
				Msgf("❌ This node (%s) must be present in initial-members: %v", selfMember.IP, initialMembers)
		}
	}

	masterNode, err := dragonboat.InitMasterNode(config.GlobalConfiguration.ReplicaID, selfMember, initialMembers, config.GlobalConfiguration.Join, roles)
	if err != nil {
		log.Fatal().
			Err(err).
			Str("package", "app").
			Str("func", "Run").
			Msgf("❌ Failed Init raft Master node")
	}

	log.Info().Interface("", config.GlobalConfiguration.Roles).Msg("This node has these roles")

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
