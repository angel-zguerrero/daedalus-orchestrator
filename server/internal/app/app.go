package app

import (
	"bytes"
	"context"
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	rest_api_admin "deadalus-orch/server/internal/infrastructure/server/rest/admin"
	"deadalus-orch/server/internal/pkg/config"
	"deadalus-orch/server/internal/pkg/utils"
	"deadalus-orch/server/internal/telemetry"
	"deadalus-orch/shared/constants"
	"deadalus-orch/shared/models"
	"encoding/gob"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	commands "deadalus-orch/server/internal/usecase/command"

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
	RestAdminAPI            *rest_api_admin.RestAdminAPI
	NodeReadyWatcherStopper *syncutil.Stopper
	ApiLock                 sync.Mutex
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

	app.NodeReadyWatcherStopper.RunWorker(func() {
		interval := 3 * time.Second
		masterReadyCh := masterNode.StartNodeReadyWatcher(interval)

		tenantReadyChs := make([]<-chan bool, len(app.TenantNodes))
		for i, node := range app.TenantNodes {
			tenantReadyChs[i] = node.StartNodeReadyWatcher(interval)
		}

		readyMap := make(map[int]bool) // key -1 for master, 0..N-1 for tenants
		const masterKey = -1

		defer func() {
			if app.RestAdminAPI != nil {
				log.Info().Msg("🔌 Node readiness watcher stopped, ensuring Admin API is shutdown...")
				app.CloseAdminAPI()
			}
		}()

		for {
			select {
			case isReady, ok := <-masterReadyCh:
				if !ok {
					log.Warn().Msg("🛑 Master node watcher closed.")
					return
				}
				readyMap[masterKey] = isReady

			default:
				for i, ch := range tenantReadyChs {
					select {
					case ready, ok := <-ch:
						if !ok {
							log.Warn().Int("tenant", i).Msg("🛑 Tenant node watcher closed.")
							return
						}
						if !ready && app.MasterNodeIsReady {
							log.Warn().Int("tenant", i).Msg("⚠️ Tenant node does not respond.")
						}
						readyMap[i] = ready
					default:
					}
				}
			}

			allReady := readyMap[masterKey]
			for i := range tenantReadyChs {
				if !readyMap[i] {
					allReady = false
					break
				}
			}

			if allReady && !app.MasterNodeIsReady {
				log.Info().Msg("✅ Master + all tenants ready for consensus.")
				app.MasterNodeIsReady = true

				app.StartAssignTenants()
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
					if err != nil {
						log.Fatal().
							Err(err).
							Str("package", "app").
							Str("func", "Run").
							Msgf("❌ Failed to bootstrap root user")
					}
				}

				if dragonboat.ContainsRole(roles, dragonboat.RoleAdmin) {
					app.StartAdminAPI(masterNode)
				} else {
					app.CloseAdminAPI()
				}
			}

			if !allReady && app.MasterNodeIsReady {
				log.Warn().Msg("⚠️ One or more nodes are not ready. Marking node as not ready.")
				app.MasterNodeIsReady = false
				app.CloseAdminAPI()
			}

			select {
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
		if app.RestAdminAPI != nil {
			app.CloseAdminAPI()
		} else {
			log.Warn().Msg("⚠ No Admin API to stop.")
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

func (app *Application) StartAdminAPI(masterNode *dragonboat.RaftNode) {
	app.ApiLock.Lock()
	defer app.ApiLock.Unlock()
	if app.RestAdminAPI == nil {
		jwtSecret := config.GlobalConfiguration.AdminAPIJWTSecret
		jwtDuration := time.Hour * time.Duration(config.GlobalConfiguration.AdminAPIJWTExpirationHours)

		log.Info().Msg("Admin API JWT Expiration: " + jwtDuration.String())

		// Pass the global log.Logger instance, which is configured in app.Run()
		app.RestAdminAPI = rest_api_admin.NewRestAdminAPI(app.MasterNode, app.TenantNodes, app.TenantNodesDictionary, jwtSecret, jwtDuration, log.Logger)

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
		log.Info().Msg("Closing Admin app...")
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

func (app *Application) StartAssignTenants() {
	cursor := ""
	pageSize := 10

	for {
		paginateTenantsCommand := &commands.PaginateTenantsCommand{
			Cursor:   cursor,
			PageSize: pageSize,
		}

		queryCommand := &commands.Query_Command{
			Command: &commands.Repository_Command{
				CMD: paginateTenantsCommand,
			},
			Now: time.Now().UnixNano(),
		}

		ctx, cancel := context.WithTimeout(context.Background(), config.GlobalConfiguration.ApiRaftTimeout)
		defer cancel()

		result, err := app.MasterNode.Read(ctx, *queryCommand)
		if err != nil {
			log.Fatal().Err(err).Msg("Paginate tenants command failed")

			return
		}

		buf := bytes.NewBuffer(result.([]byte))
		dec := gob.NewDecoder(buf)
		parsedResult := &commands.CommandResult{}
		if err := dec.Decode(parsedResult); err != nil {
			log.Fatal().Err(err).Msg("Paginate tenants command failed (decode)")

			return
		}

		if parsedResult.Error != "" {
			log.Fatal().Str("error", parsedResult.Error).Msg("Paginate tenants command failed (business error)")
			return
		}

		tenantsResult := parsedResult.Result.(db.FindResult[models.TenantInMaster])
		writeCtx, writeCancel := context.WithTimeout(context.Background(), config.GlobalConfiguration.ApiRaftTimeout) // Or a specific timeout for writes
		defer writeCancel()
		for _, tenant := range tenantsResult.Entities {
			var tenantNode *dragonboat.RaftNode
			for i := range app.TenantNodes {
				if app.TenantNodes[i].ShardID == uint64(tenant.ShardId) {
					tenantNode = app.TenantNodes[i]

					if tenant.Status == models.PendingForAssign {
						createColumnFamilyCommand := &commands.CreateColumnFamilyCommand{
							Name: tenant.ID,
						}

						ccfCmd := commands.FSM_Command{
							Now:  utils.GetNowInInt(),
							Type: commands.REPOSITORY_COMMAND,
							CMD:  createColumnFamilyCommand,
						}

						result, err = tenantNode.Write(writeCtx, ccfCmd)
						if err != nil {

							log.Fatal().Err(err).Str("Code", tenant.Code).Msg("Failed to assign tenant")

						}

						assignToShardTenantInMasterCommand := &commands.AssignToShardTenantInMasterCommand{
							TenantCode: tenant.Code,
						}

						atstCmd := commands.FSM_Command{
							Now:  utils.GetNowInInt(),
							Type: commands.REPOSITORY_COMMAND,
							CMD:  assignToShardTenantInMasterCommand,
						}

						result, err = app.MasterNode.Write(writeCtx, atstCmd)
						if err != nil {
							log.Fatal().Err(err).Str("Code", tenant.Code).Msg("Failed to assign tenant")

						}
					}

					if tenant.Status == models.PendingForDeletion {
						deleteColumnFamilyCommand := &commands.DeleteColumnFamilyCommand{
							Name: tenant.ID,
						}

						ccfCmd := commands.FSM_Command{
							Now:  utils.GetNowInInt(),
							Type: commands.REPOSITORY_COMMAND,
							CMD:  deleteColumnFamilyCommand,
						}

						result, err = tenantNode.Write(writeCtx, ccfCmd)
						if err != nil {
							log.Fatal().Err(err).Str("Code", tenant.Code).Msg("Failed to delete column family")
						}

						deleteTenantInMasterCommand := &commands.DeleteTenantInMasterCommand{
							TenantId: tenant.ID,
						}

						atstCmd := commands.FSM_Command{
							Now:  utils.GetNowInInt(),
							Type: commands.REPOSITORY_COMMAND,
							CMD:  deleteTenantInMasterCommand,
						}

						result, err = app.MasterNode.Write(writeCtx, atstCmd)
						if err != nil {
							log.Fatal().Err(err).Str("Code", tenant.Code).Msg("Failed to delete tenant")
						}
					}

					break
				}
			}
			app.TenantNodesDictionary[tenant.ID] = tenantNode
		}

		if tenantsResult.Cursor == "" {
			break
		}

		cursor = tenantsResult.Cursor
	}

}

func RecommendRTTMillisecond() uint64 {
	shardCount := config.GlobalConfiguration.MaxTenants
	switch {
	case shardCount <= 50:
		return 200
	case shardCount <= 100:
		return 250
	case shardCount <= 200:
		return 350
	case shardCount <= 400:
		return 375
	case shardCount <= 800:
		return 450
	case shardCount <= 1600:
		return 500

	default:
		return 300
	}
}
func NewApplication() *Application {
	return &Application{
		MasterNodeIsReady:       false,
		MasterNode:              nil,
		RestAdminAPI:            nil,
		NodeReadyWatcherStopper: syncutil.NewStopper(),
		TenantNodes:             make([]*dragonboat.RaftNode, 0),
		TenantNodesDictionary:   make(map[string]*dragonboat.RaftNode),
	}
}
