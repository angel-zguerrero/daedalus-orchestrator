package app

import (
	"context"
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	rest_api_admin "deadalus-orch/server/internal/infrastructure/server/rest/admin"
	"deadalus-orch/server/internal/pkg/config"
	"deadalus-orch/server/internal/pkg/utils"
	"deadalus-orch/server/internal/telemetry"
	"deadalus-orch/shared/constants"
	"fmt"
	"os"
	"sync"
	"time"

	commands "deadalus-orch/server/internal/usecase/command"

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
//     - Parses the current node's address (SelfMemberAddr from global configuration).
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
	MasterNode              *dragonboat.RaftNode // This is a struct from the dragonboat package that was NOT moved
	RestAdminAPI            *rest_api_admin.RestAdminAPI
	NodeReadyWatcherStopper *syncutil.Stopper
	ApiLock                 sync.Mutex
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
	app.MasterNode = masterNode
	if err != nil {
		log.Fatal().
			Err(err).
			Str("package", "app").
			Str("func", "Run").
			Msgf("❌ Failed Init raft Master node")
	}

	log.Info().Interface("", roles).Msg("This node has these roles")

	app.NodeReadyWatcherStopper.RunWorker(func() {
		interval := 3 * time.Second
		readyUpdates := masterNode.StartNodeReadyWatcher(interval)
		defer func() {
			if app.RestAdminAPI != nil {
				log.Info().Msg("🔌 Node readiness watcher stopped, ensuring Admin API is shutdown...")
				app.CloseAdminAPI()
			}
		}()

		for {
			select {
			case isReady, ok := <-readyUpdates:
				if !ok {
					log.Info().Msg("✅ NodeReadyWatcher channel closed.")
					return
				}

				app.MasterNodeIsReady = isReady
				if isReady {
					log.Info().Msg("✅ Node is ready for consensus.")
					if dragonboat.ContainsRole(roles, dragonboat.RoleConsensus) {

						bootstrapRootUserCmd := &commands.BootstrapRootUserCommand{}

						cmd := commands.FSM_Command{
							Now:  utils.GetNowInInt(),
							Type: commands.REPOSITORY_COMMAND,
							CMD:  bootstrapRootUserCmd,
						}

						ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
						defer cancel()
						_, err := masterNode.Write(ctx, cmd)
						log.Fatal().
							Err(err).
							Str("package", "app").
							Str("func", "Run").
							Msgf("❌ Failed to bootstrap root user", err)
					}

					if dragonboat.ContainsRole(roles, dragonboat.RoleAdmin) {
						app.StartAdminAPI(masterNode)
					} else {

						app.CloseAdminAPI()
						app.RestAdminAPI = nil
					}
				} else {
					log.Info().Msg("⚠️ Node is NOT ready for consensus.")
					if app.RestAdminAPI != nil {
						log.Info().Msg("🔌 Node not ready, shutting down Admin API...")
						app.CloseAdminAPI()
					}
				}

			case <-app.NodeReadyWatcherStopper.ShouldStop():

				log.Info().Msg("🛑 NodeReadyWatcher received stop signal.")
				return
			case <-time.After(interval):

			}
		}
	})

}

func (app *Application) Stop() {
	var wg sync.WaitGroup
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if app.MasterNode != nil {
			log.Info().Msg("Stopping Master Node...")
			app.MasterNode.Stop()
			log.Info().Msg("Master Node stopped.")
		} else {
			log.Warn().Msg("No Master Node to stop.")
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if app.RestAdminAPI != nil {
			app.CloseAdminAPI()
		} else {
			log.Warn().Msg("No Admin API to stop.")
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		app.NodeReadyWatcherStopper.Stop()
		log.Info().Msg("NodeReadyWatcher stopped.")
	}()

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
func (app *Application) StartAdminAPI(masterNode *dragonboat.RaftNode) {
	app.ApiLock.Lock()
	defer app.ApiLock.Unlock()
	if app.RestAdminAPI == nil {
		jwtSecret := config.GlobalConfiguration.AdminAPIJWTSecret
		jwtDuration := time.Hour * time.Duration(config.GlobalConfiguration.AdminAPIJWTExpirationHours)

		log.Info().Msg("Admin API JWT Expiration: " + jwtDuration.String())

		// Pass the global log.Logger instance, which is configured in app.Run()
		app.RestAdminAPI = rest_api_admin.NewRestAdminAPI(masterNode, jwtSecret, jwtDuration, log.Logger)

		adminListenAddr := fmt.Sprintf("%s:%d", config.GlobalConfiguration.AdminListenAddrHost, config.GlobalConfiguration.AdminListenAddrPort)
		go func() {
			if err := app.RestAdminAPI.Start(adminListenAddr); err != nil {
				log.Error().Err(err).Msg("❌ Admin API server failed to start or shut down with error")
			}
		}()
		log.Info().Str("address", adminListenAddr).Msg("🚀 Admin API scheduled to start because RoleAdmin is present.")

	} else if app.RestAdminAPI != nil {
		log.Info().Msg("Admin API already running or was previously started.")
	}
}
func (app *Application) CloseAdminAPI() {
	app.ApiLock.Lock()
	defer app.ApiLock.Unlock()
	if app.RestAdminAPI != nil {
		log.Info().Msg("Closing Admin API...")
		shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelShutdown()
		if err := app.RestAdminAPI.Shutdown(shutdownCtx); err != nil {
			log.Error().Err(err).Msg("❌ Error during Admin API shutdown")
		} else {
			log.Info().Msg("✅ Admin API closed successfully.")
		}
		app.RestAdminAPI = nil
	} else {
		log.Warn().Msg("No Admin API to close.")
	}
}
func NewApplication() *Application {
	return &Application{
		MasterNodeIsReady:       false,
		MasterNode:              nil,
		RestAdminAPI:            nil,
		NodeReadyWatcherStopper: syncutil.NewStopper(),
	}
}
