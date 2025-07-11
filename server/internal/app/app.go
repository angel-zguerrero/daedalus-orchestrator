package app

import (
	"context"
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	grpc_server "deadalus-orch/server/internal/infrastructure/server/grpc"
	rest_server "deadalus-orch/server/internal/infrastructure/server/rest"
	"deadalus-orch/server/internal/pkg/config"
	"deadalus-orch/server/internal/telemetry"
	"deadalus-orch/shared/constants"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	dragonboatV4 "github.com/lni/dragonboat/v4"
	dragonboatV4Config "github.com/lni/dragonboat/v4/config"
	dblog "github.com/lni/dragonboat/v4/logger"
	"github.com/lni/goutils/syncutil"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Run initializes and starts the main application components.
//
// This function performs the following key initializations:
//  1. Logging: Configures zerolog for application-wide logging.
//     - Sets the time field format to Unix timestamp.
//     - Enables console-friendly output (pretty print) if LOGGER_FORMAT is "pretty" or not set.
//     - Sets the global log level based on the ENV environment variable (ErrorLevel for "production", DebugLevel otherwise).
//     - Sets a custom logger factory for Dragonboat internal logging. // This refers to the external library
//  2. Telemetry: Initializes OpenTelemetry for tracing and metrics.
//     - Configuration is driven by environment variables:
//     - ENV: Sets the environment (e.g., production, development).
//     - OTEL_ACTIVED: Enables or disables OpenTelemetry ("true" to activate).
//     - OTEL_ENDPOINT: Specifies the OpenTelemetry collector endpoint.
//     - OTEL_TRACER_SERVICE_NAME: Defines the service name for traces.
//     - A tracer provider is initialized and its shutdown is deferred.
//  3. Dragonboat Node: Sets up the local Dragonboat node for distributed consensus.
//     - Constructs the current node's address from SelfMemberHost and ClusterBasePort (from global configuration).
//     - Parses the list of initial members for the cluster (InitialMembers from global configuration).
//     - Parses the roles assigned to this node (Roles from global configuration).
//     - Performs validation checks based on whether the node is joining an existing cluster or creating a new one,
//     ensuring that necessary flags like --replica and --initial-members are provided.
//     - Initializes the Dragonboat MasterNode.
//     - Starts a goroutine to monitor and log the node's readiness for consensus.
//
// Additionally, the function contains commented-out sections for:
//   - Database Initialization: Code for initializing a RocksDB instance (commented out).
//   - gRPC Server: Code for starting a gRPC server (commented out).
//
// These components might be integrated in future versions of the application.

type Application struct {
	MasterNodeIsReady       bool
	MasterNode              *dragonboat.RaftNode
	TenantNodes             []*dragonboat.RaftNode
	TenantNodesDictionary   map[string]*dragonboat.RaftNode
	RestAPI                 *rest_server.RestServer
	GrpcAPI                 *grpc_server.GrpcServer
	NodeReadyWatcherStopper *syncutil.Stopper
	ApiLock                 sync.Mutex
	GrpcLock                sync.Mutex
	NH                      *dragonboatV4.NodeHost
}

func (app *Application) Run() {

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
	dblog.SetLoggerFactory(dragonboat.CreateZerologger) // This is from the external dragonboat lib, not the one we moved

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

	selfMemberAddr := fmt.Sprintf("%s:r%d", config.GlobalConfiguration.SelfMemberHost, config.GlobalConfiguration.ReplicaID)
	selfMember, err := dragonboat.ParseMember(selfMemberAddr, int(config.GlobalConfiguration.ClusterBasePort))

	if err != nil {
		log.Fatal().
			Err(err).
			Str("package", "app").
			Str("func", "Run").
			Msgf("❌ Failed parsing self member")
	}

	initialMembers, err := dragonboat.ParseMembersFlag(&config.GlobalConfiguration.InitialMembers, config.GlobalConfiguration.ClusterBasePort)
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

	base_path, err := db.DefaultPathProvider{}.GetDatabasePath()
	if err != nil {
		log.Fatal().
			Err(err).
			Str("package", "app").
			Str("func", "Run").
			Msgf("❌ Getting database path")
	}

	RTTMillisecond := RecommendRTTMillisecond()
	NH, err := dragonboatV4.NewNodeHost(dragonboatV4Config.NodeHostConfig{
		WALDir:         base_path + "/wal/" + strconv.FormatUint(config.GlobalConfiguration.ReplicaID, 10) + "/" + selfMember.IP + "-" + strconv.Itoa(selfMember.Port),
		NodeHostDir:    base_path + "/node/" + strconv.FormatUint(config.GlobalConfiguration.ReplicaID, 10) + "/" + selfMember.IP + "-" + strconv.Itoa(selfMember.Port),
		RTTMillisecond: RTTMillisecond,
		RaftAddress:    dragonboat.MemmberToAddr(selfMember),
	})
	app.NH = NH
	if err != nil {
		log.Fatal().
			Err(err).
			Str("package", "app").
			Str("func", "Run").
			Msgf("❌ Staring node host")
	}

	masterNode, err := dragonboat.InitMasterNode(config.GlobalConfiguration.ReplicaID, selfMember, initialMembers, config.GlobalConfiguration.Join, roles, db.DefaultPathProvider{}, NH)
	app.MasterNode = masterNode
	if err != nil {
		log.Fatal().
			Err(err).
			Str("package", "app").
			Str("func", "Run").
			Msgf("❌ Failed Init raft Master node")
	}

	tenantNodes, err := dragonboat.StartTentantNodes(config.GlobalConfiguration.ReplicaID, selfMember, config.GlobalConfiguration.Join, roles, db.DefaultPathProvider{}, initialMembers, NH)
	if err != nil {
		log.Fatal().
			Err(err).
			Str("package", "app").
			Str("func", "Run").
			Msgf("❌ Failed Init raft Tenat nodes")
	}
	app.TenantNodes = tenantNodes

	log.Info().Interface("", roles).Msg("This node has these roles")

	app.StartNodeReadyWatcherWorker(3 * time.Second)

}

func (app *Application) Stop() {
	var wg sync.WaitGroup
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Stop Master Node
	wg.Add(1)
	go func() {
		defer wg.Done()
		if app.MasterNode != nil {
			log.Info().Msg("🛑 Stopping Master Node...")
			app.MasterNode.Stop()
			log.Info().Msg("✅ Master Node stopped.")
		} else {
			log.Warn().Msg("⚠ No Master Node to stop.")
		}
	}()

	// Stop Tenant Nodes in parallel
	for i, tenantNode := range app.TenantNodes {
		if tenantNode != nil {
			wg.Add(1)
			go func(i int, node *dragonboat.RaftNode) {
				defer wg.Done()
				defer func() {
					if r := recover(); r != nil {
						log.Error().
							Interface("recover", r).
							Int("tenantIndex", i).
							Msg("❌ Panic while stopping tenant node")
					}
				}()
				log.Info().Int("tenantIndex", i).Msg("🛑 Stopping Tenant Node...")
				node.Stop()
				log.Info().Int("tenantIndex", i).Msg("✅ Tenant Node stopped.")
			}(i, tenantNode)
		}
	}

	// Stop Admin API
	wg.Add(1)
	go func() {
		defer wg.Done()
		if app.RestAPI != nil {
			app.CloseAdminAPI()
		} else {
			log.Warn().Msg("⚠ No Admin API to stop.")
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if app.GrpcAPI != nil {
			app.CloseGrpcAPI()
		} else {
			log.Warn().Msg("⚠ No Grpc API to stop.")
		}
	}()

	// Stop Watcher
	wg.Add(1)
	go func() {
		defer wg.Done()
		app.NodeReadyWatcherStopper.Stop()
		log.Info().Msg("⛔ NodeReadyWatcher stopped.")
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if app.MasterNode != nil {
			log.Info().Msg("🛑 Stopping Node Host...")
			app.NH.Close()
			log.Info().Msg("✅ Node Host stopped.")
		} else {
			log.Warn().Msg("⚠ No Node Host to stop.")
		}
	}()

	// Wait with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Info().Msg("✅ All components stopped gracefully.")
	case <-ctx.Done():
		log.Warn().Msg("⚠ Stop operation timed out. Some components may not have stopped.")
	}
}

func NewApplication() *Application {
	return &Application{
		MasterNodeIsReady:       false,
		MasterNode:              nil,
		RestAPI:                 nil,
		NodeReadyWatcherStopper: syncutil.NewStopper(),
		TenantNodes:             make([]*dragonboat.RaftNode, 0),
		TenantNodesDictionary:   make(map[string]*dragonboat.RaftNode),
	}
}
