package config

import (
	"bufio" // Added for IsValidPort
	"deadalus-orch/shared/constants"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// comprehensiveHelpMessage is a detailed help message that describes the application's
// usage, available command-line flags, environment variables, configuration file format,
// and the precedence of configuration sources. It is displayed when the --help flag is used.
const comprehensiveHelpMessage = `
Usage: ./server [flags]

Available Flags:
  --config			 Path to the application configuration file. This flag takes precedence over the CONFIG_PATH environment variable.
  --connector-port	 The network port on which the connector service will listen for external client connections. Overrides the 'connector_port' value from the configuration file and the CONNECTOR_PORT environment variable.
  --help			 Show help message and exit.
  --initial-members	 Comma-separated list of initial member addresses (in ip:port format) for bootstrapping a new cluster. Required when creating a cluster and not using the --join flag.
  --join			 Set this flag to true if this node should attempt to join an existing cluster. When joining, --initial-members should specify addresses of nodes in the existing cluster.
  --replica			 Unique identifier (positive integer) for this node within the cluster. Required when creating a new cluster or joining an existing one.
  --role			 Comma-separated list of roles for this node (e.g., 'consensus,scheduler,connector'). Defines the node's responsibilities within the cluster.
  --self-member-host The IP address or hostname that this node will use for communication with other members in the cluster. (e.g., 127.0.0.1)
  --cluster-base-port The base port number that this node will use for communication with other members in the cluster. (e.g., 17000)
  --master-db-engine         The database engine for the master database (e.g., "pebble", "rocksdb"). Defaults to "pebble".
  --tenant-db-engine         The database engine for tenant databases (e.g., "pebble", "rocksdb"). Defaults to "pebble".
  --rest-api-jwt-expiration-hours JWT expiration time in hours for the Rest API. Default is 3 hours.
  --rest-host                 Host address for the Rest API. Overrides config file and environment variable.
  --rest-port                 Port for the Rest API. Overrides config file and environment variable.
  --api-raft-timeout           Timeout for API to Raft node requests (e.g., 5s, 1m). Default 5s. Overrides config file and environment variable.
  --max-shards                Maximum number of shards (default 10, max 1000). Overrides config file and environment variable.
  --max-column-families       Maximum number of column families (default 10, max 100 in production, 10 in non-production). Overrides config file and environment variable.
  --grpc-host                  Host address for the gRPC server. Overrides config file and environment variable.
  --grpc-port                  Port for the gRPC server. Default 4000. Overrides config file and environment variable.
  --tenant-summary-worker-interval  Interval for tenant summary worker in seconds. Minimum 10. Default 30. Overrides config file and environment variable.
  --max-headers                Maximum number of headers. Default 100, minimum 5, maximum 1000. Overrides config file and environment variable.
  --deployment-id              Unique identifier for cluster isolation (default 1). Overrides config file and environment variable.


Environment Variables:
  CONFIG_PATH                  Path to the configuration file.
  ENV                          Application environment (e.g., development, staging, production).
  DEFAULT_ROOT_USER            Default root username for the application.
  DEFAULT_ROOT_PASSWORD        Default root password for the application.
  REPLICA_ID                   Node's replica ID.
  ROLES                        Comma-separated node roles.
  SELF_MEMBER_HOST             Node's own member host (e.g., 127.0.0.1). (Corresponds to ` + constants.EnvVarSelfMemberHost + `)
  CLUSTER_BASE_PORT            Node's own cluster base port (e.g., 17000). (Corresponds to ` + constants.EnvVarClusterBasePort + `)
  INITIAL_MEMBERS              Initial members of the cluster (ip:port,ip:port,...).
  JOIN                         Set to "true" if the node is joining an existing cluster.
  CONNECTOR_PORT               Port for the connector service.
  TTL_INTERNAL_ERROR           TTL for internal error caching in seconds.
  MASTER_DB_ENGINE             The database engine for the master database.
  TENANT_DB_ENGINE             The database engine for tenant databases.
  REST_API_JWT_EXPIRATION_HOURS JWT expiration time in hours for the Rest API.
  REST_LISTEN_ADDR_HOST       Host address for the Rest API. (Corresponds to ` + constants.EnvVarRestListenAddrHost + `)
  REST_LISTEN_ADDR_PORT       Port for the Rest API. (Corresponds to ` + constants.EnvVarRestListenAddrPort + `)
  REST_API_JWT_SECRET         JWT secret key for the Rest API. (Corresponds to ` + constants.EnvVarRestAPIJWTSecret + `)
  API_RAFT_TIMEOUT             Timeout for API to Raft node requests (e.g., "5s", "1m"). (Corresponds to ` + constants.EnvVarAPIRaftTimeout + `)
  MAX_SHARDS                  Maximum number of shards. (Corresponds to ` + constants.EnvVarMaxShards + `)
  MAX_COLUMN_FAMILIES         Maximum number of column families. (Corresponds to ` + constants.EnvVarMaxColumnFamilies + `)
  GRPC_SERVER_LISTEN_ADDR_HOST Host address for the gRPC server. (Corresponds to ` + constants.EnvVarGrpcServerListenAddrHost + `)
  GRPC_SERVER_LISTEN_ADDR_PORT Port for the gRPC server. (Corresponds to ` + constants.EnvVarGrpcServerListenAddrPort + `)
  TENANT_SUMMARY_WORKER_INTERVAL  Interval for tenant summary worker in seconds. (Corresponds to ` + constants.EnvVarTenantSummaryWorkerInterval + `)
  MAX_HEADERS                 Maximum number of headers. (Corresponds to ` + constants.EnvVarMaxHeaders + `)
  DEPLOYMENT_ID                Unique identifier for cluster isolation. (Corresponds to ` + constants.EnvVarDeploymentID + `)

  OTEL_ACTIVED                  Set to "true" or "false" to enable/disable OpenTelemetry.
  OTEL_ENDPOINT                OpenTelemetry collector endpoint.
  OTEL_TRACER_SERVICE_NAME     OpenTelemetry service name.

Configuration File:
  The application can be configured using a file specified by the --config flag or the CONFIG_PATH environment variable.
  The file should use key=value pairs, with one pair per line. Lines starting with # are treated as comments and are ignored.

  Available keys:
    connector_port
    default_root_user
    default_root_password
    replica_id
    roles
    self_member_host
    cluster_base_port
    initial_members
    join
    ttl_internal_error
    master_db_engine
    tenant_db_engine
    rest_api_jwt_expiration_hours
    rest_listen_addr_host
    rest_listen_addr_port
    rest_api_jwt_secret
    api_raft_timeout               Timeout for API to Raft node requests in seconds (e.g., 5 for 5s).
    tenant_port_range              Tenant port range (e.g., "4000-4100").
    max_shards                    Maximum number of shards.
    max_column_families           Maximum number of column families.
    grpc_server_listen_addr_host   Host address for the gRPC server.
    grpc_server_listen_addr_port   Port for the gRPC server.
    tenant_summary_worker_interval  Interval for tenant summary worker in seconds.
    max_headers                   Maximum number of headers.
    deployment_id                 Unique identifier for cluster isolation.


Precedence of Configuration:
  The configuration is loaded in the following order of precedence (highest to lowest):
  1. Command-line flags
  2. Environment variables
  3. Configuration file

  If a setting is specified in multiple places, the value from the source with higher precedence will be used.
`

// ConfigFilePathFlag defines the --config command-line flag for specifying the configuration file path.
var ConfigFilePathFlag = flag.String("config", "", "Path to the application configuration file. This flag takes precedence over the CONFIG_PATH environment variable.")

// RoleFlag defines the --role command-line flag for specifying node roles.
var RoleFlag = flag.String("role", "", "Comma-separated list of roles for this node (e.g., 'consensus,scheduler,connector'). Defines the node's responsibilities within the cluster.")

// InitialMembersFlag defines the --initial-members command-line flag for specifying initial cluster members.
var InitialMembersFlag = flag.String("initial-members", "", "Comma-separated list of initial member addresses (in ip:port format) for bootstrapping a new cluster. Required when creating a cluster and not using the --join flag.")

// SelfMemberAddrFlag defines the --self-member-addr command-line flag for specifying the node's own Raft address.
// var SelfMemberAddrFlag = flag.String("self-member-addr", "", "The network address (in ip:port format) that this node will use for communication with other members in the cluster.") // Deprecated

// SelfMemberHostFlag defines the --self-member-host command-line flag.
var SelfMemberHostFlag = flag.String(constants.SelfMemberHostFlagName, "", "The IP address or hostname that this node will use for communication with other members in the cluster.")

// ClusterBasePortFlag defines the --cluster-base-port command-line flag.
var ClusterBasePortFlag = flag.Int(constants.ClusterBasePortFlagName, 0, "The base port number that this node will use for communication with other members in the cluster.")

// JoinFlag defines the --join command-line flag to indicate if the node should join an existing cluster.
var JoinFlag = flag.Bool("join", false, "Set this flag to true if this node should attempt to join an existing cluster. When joining, --initial-members should specify addresses of nodes in the existing cluster.")

// ReplicaIDFlag defines the --replica command-line flag for specifying the node's replica ID.
var ReplicaIDFlag = flag.Uint64("replica", 0, "Unique identifier (positive integer) for this node within the cluster. Required when creating a new cluster or joining an existing one.")

// ConnectorPortFlag defines the --connector-port command-line flag for specifying the connector service port.
var ConnectorPortFlag = flag.Int("connector-port", 0, "The network port on which the connector service will listen for external client connections. Overrides the 'connector_port' value from the configuration file and the CONNECTOR_PORT environment variable.")

// MasterDBEngineFlag defines the --master-db-engine command-line flag for specifying the master database engine.
var MasterDBEngineFlag = flag.String(constants.MasterDBEngineFlagName, "", "The database engine for the master database (e.g., \"pebble\", \"rocksdb\").")

// TenantDBEngineFlag defines the --tenant-db-engine command-line flag for specifying the tenant database engine.
var TenantDBEngineFlag = flag.String(constants.TenantDBEngineFlagName, "", "The database engine for tenant databases (e.g., \"pebble\", \"rocksdb\").")

// RestAPIJWTExpirationHoursFlag defines the --rest-api-jwt-expiration-hours command-line flag for specifying the Rest API JWT expiration in hours.
var RestAPIJWTExpirationHoursFlag = flag.Int("rest-api-jwt-expiration-hours", 0, "JWT expiration time in hours for the Rest API. Overrides config file and environment variable.")

// RestListenAddrHostFlag defines the --rest-host command-line flag for specifying the Rest API listen host.
var RestListenAddrHostFlag = flag.String("rest-host", "", "Host address for the Rest API. Overrides config file and environment variable.")

// RestListenAddrPortFlag defines the --rest-port command-line flag for specifying the Rest API listen port.
var RestListenAddrPortFlag = flag.Int("rest-port", 0, "Port for the Rest API. Overrides config file and environment variable.")

// ApiRaftTimeoutFlag defines the --api-raft-timeout command-line flag for specifying the API to Raft node request timeout.
var ApiRaftTimeoutFlag = flag.Duration("api-raft-timeout", 30*time.Second, "Timeout for API to Raft node requests (e.g., 5s, 1m). Overrides config file and environment variable.")

// MaxShardsFlag defines the --max-shards command-line flag.
var MaxShardsFlag = flag.Int(
	constants.MaxShardsFlagName,
	0,
	fmt.Sprintf(
		"Maximum number of shards (default: 10, max: %d in production, %d in non-production environments).",
		constants.MaxShardsInProduction,
		constants.MaxShardsInNonProduction,
	),
)

// HelpFlag defines the --help command-line flag to display the help message.
var HelpFlag = flag.Bool("help", false, "Show help message and exit.")

// MaxColumnFamiliesFlag defines the --max-column-families command-line flag.
var MaxColumnFamiliesFlag = flag.Int(
	constants.MaxColumnFamiliesFlagName,
	0,
	fmt.Sprintf(
		"Maximum number of column families (default: 10, max: %d in production, %d in non-production environments).",
		constants.MaxColumnFamiliesInProduction,
		constants.MaxColumnFamiliesInNonProduction,
	),
)

// GrpcServerListenAddrHostFlag defines the --grpc-host command-line flag for specifying the gRPC server listen host.
var GrpcServerListenAddrHostFlag = flag.String(constants.GrpcServerListenAddrHostFlagName, "", "Host address for the gRPC server. Overrides config file and environment variable.")

// GrpcServerListenAddrPortFlag defines the --grpc-port command-line flag for specifying the gRPC server listen port.
var GrpcServerListenAddrPortFlag = flag.Int(constants.GrpcServerListenAddrPortFlagName, 0, "Port for the gRPC server. Default 4000. Overrides config file and environment variable.")

// TenantSummaryWorkerIntervalFlag defines the --tenant-summary-worker-interval command-line flag for specifying the tenant summary worker interval.
var TenantSummaryWorkerIntervalFlag = flag.Int64(constants.TenantSummaryWorkerIntervalFlagName, 30, "Interval for tenant summary worker in seconds. Minimum 10. Overrides config file and environment variable.")

// MaxHeadersFlag defines the --max-headers command-line flag for specifying the maximum number of headers.
var MaxHeadersFlag = flag.Int(constants.MaxHeadersFlagName, 0, "Maximum number of headers (default 100, minimum 5, maximum 1000). Overrides config file and environment variable.")

// DeploymentIDFlag defines the --deployment-id command-line flag.
var DeploymentIDFlag = flag.Uint64(constants.DeploymentIDFlagName, 0, "Unique identifier for cluster isolation. Overrides config file and environment variable.")

// MessageLeaseDurationFlag defines the --message-lease-duration command-line flag.
// It specifies, in seconds, how long a dequeued message is locked to a JobWorker
// before the lease expires automatically.
var MessageLeaseDurationFlag = flag.Int64(constants.MessageLeaseDurationFlagName, 0, fmt.Sprintf("Duration in seconds a dequeued message is leased to a JobWorker (default: %d, minimum: 5). Overrides config file and environment variable.", constants.DefaultMessageLeaseDurationSeconds))

// LoadDefaultConfiguration loads the application configuration from various sources
// and populates the GlobalConfiguration variable.
// The loading order of precedence is:
// 1. Command-line flags (highest precedence)
// 2. Environment variables
// 3. Configuration file (lowest precedence)
// If the --help flag is set, it prints a comprehensive help message and exits.
// Default values are applied for some settings if they are not provided by any source.
//
// Returns:
//   - An error if loading the configuration from a file fails, or if parsing
//     values from environment variables fails. Returns nil on successful configuration loading.
func LoadDefaultConfiguration() error {
	flag.Parse() // Parse command-line flags first

	// Check if the help flag is set
	if HelpFlag != nil && *HelpFlag {
		fmt.Print(comprehensiveHelpMessage)
		os.Exit(0)
	}

	config := &Config{}

	env := os.Getenv(constants.EnvVarEnvKey)
	if env == "" {
		env = string(constants.DEVELOPMENT)
	}

	config.Env = env

	configFilePath := os.Getenv(constants.EnvVarConfigPath)
	if configFilePath == "" {
		configFilePath = *ConfigFilePathFlag
	}

	if configFilePath != "" {
		if _, err := os.Stat(configFilePath); err == nil {
			log.Info().
				Str("path", configFilePath).
				Msg("✅ Using config file")

			configMap, err := loadMapFromPath(configFilePath)
			if err != nil {
				log.Error().
					Str("path", configFilePath).
					Err(err).
					Msg("⚠️️ Failed to load map from config file")
				return err
			}

			cfgFromMapIntermediate, err := mapToConfig(configMap)
			if err != nil {
				log.Error().
					Str("path", configFilePath).
					Err(err).
					Msg("⚠️️ Failed to parse config file data into struct")
				return err
			}
			// Apply all other values from config file to the main config object
			config = ConfigFromMapToConfig(*cfgFromMapIntermediate)

		} else {
			log.Error().
				Str("path", configFilePath).
				Err(err).
				Msg("⚠️️ Failed to load config file")
			return err
		}
	}

	// Environment variables override config file values (except for rawTenantPortRange which is handled specially)
	if envVal := os.Getenv(constants.EnvVarReplicaId); envVal != "" {
		replicaID, err := strconv.ParseUint(envVal, 10, 64)
		if err != nil {
			return err
		}
		config.ReplicaID = replicaID
	}

	if envVal := os.Getenv(constants.EnvVarRoles); envVal != "" {
		config.Roles = envVal
	}

	if envVal := os.Getenv(constants.EnvVarSelfMemberHost); envVal != "" {
		config.SelfMemberHost = envVal
	}
	if envVal := os.Getenv(constants.EnvVarClusterBasePort); envVal != "" {
		port, err := strconv.Atoi(envVal)
		if err != nil {
			return fmt.Errorf("error parsing %s environment variable: %w", constants.EnvVarClusterBasePort, err)
		}
		config.ClusterBasePort = port
	}

	if envVal := os.Getenv(constants.EnvVarInitialMembers); envVal != "" {
		config.InitialMembers = envVal
	}

	if envVal := os.Getenv(constants.EnvVarJoin); envVal != "" {
		join, err := strconv.ParseBool(envVal)
		if err != nil {
			return err
		}
		config.Join = join
	}

	if envVal := os.Getenv(constants.EnvVarConnectorPort); envVal != "" {
		connectorPort, err := strconv.Atoi(envVal)
		if err != nil {
			return err
		}
		config.ConnectorPort = connectorPort
	}

	if envVal := os.Getenv(constants.EnvVarTTLInternalError); envVal != "" {
		ttlInternalError, err := strconv.ParseUint(envVal, 10, 64)
		if err != nil {
			return err
		}
		config.TTLInternalError = ttlInternalError
	}

	if os.Getenv(constants.EnvVarDefaultRootUser) != "" {
		config.DefaultRootUser = os.Getenv(constants.EnvVarDefaultRootUser)
	}

	if os.Getenv(constants.EnvVarDefaultRootPassword) != "" {
		config.DefaultRootPassword = os.Getenv(constants.EnvVarDefaultRootPassword)
	}

	if envVal := os.Getenv(constants.EnvVarMasterDBEngine); envVal != "" {
		config.MasterDBEngine = envVal
	}

	if envVal := os.Getenv(constants.EnvVarTenantDBEngine); envVal != "" {
		config.TenantDBEngine = envVal
	}

	if envVal := os.Getenv(constants.EnvVarRestAPIJWTExpirationHours); envVal != "" {
		jwtExpirationHours, err := strconv.Atoi(envVal)
		if err != nil {
			return err
		}
		config.RestAPIJWTExpirationHours = jwtExpirationHours
	}

	if envVal := os.Getenv(constants.EnvVarRestListenAddrHost); envVal != "" {
		config.RestListenAddrHost = envVal
	}

	if envVal := os.Getenv(constants.EnvVarRestListenAddrPort); envVal != "" {
		restPort, err := strconv.Atoi(envVal)
		if err != nil {
			return err
		}
		config.RestListenAddrPort = restPort
	}

	if envVal := os.Getenv(constants.EnvVarRestAPIJWTSecret); envVal != "" {
		config.RestAPIJWTSecret = envVal
	}

	if envVal := os.Getenv(constants.EnvVarAPIRaftTimeout); envVal != "" {
		apiRaftTimeout, err := time.ParseDuration(envVal)
		if err != nil {
			return fmt.Errorf("error parsing %s environment variable: %w", constants.EnvVarAPIRaftTimeout, err)
		}
		config.ApiRaftTimeout = apiRaftTimeout
	}

	// MaxShards from environment variable
	if envVal := os.Getenv(constants.EnvVarMaxShards); envVal != "" {
		maxShards, err := strconv.Atoi(envVal)
		if err != nil {
			return fmt.Errorf("error parsing %s environment variable: %w", constants.EnvVarMaxShards, err)
		}
		config.MaxShards = maxShards
	}

	if envVal := os.Getenv(constants.EnvVarMaxColumnFamilies); envVal != "" {
		maxColumnFamilies, err := strconv.Atoi(envVal)
		if err != nil {
			return fmt.Errorf("error parsing %s environment variable: %w", constants.EnvVarMaxColumnFamilies, err)
		}
		config.MaxColumnFamilies = maxColumnFamilies
	}

	if envVal := os.Getenv(constants.EnvVarGrpcServerListenAddrHost); envVal != "" {
		config.GrpcServerListenAddrHost = envVal
	}

	if envVal := os.Getenv(constants.EnvVarGrpcServerListenAddrPort); envVal != "" {
		grpcPort, err := strconv.Atoi(envVal)
		if err != nil {
			return fmt.Errorf("error parsing %s environment variable: %w", constants.EnvVarGrpcServerListenAddrPort, err)
		}
		config.GrpcServerListenAddrPort = grpcPort
	}

	if envVal := os.Getenv(constants.EnvVarTenantSummaryWorkerInterval); envVal != "" {
		tenantSummaryWorkerInterval, err := strconv.ParseInt(envVal, 10, 64)
		if err != nil {
			return fmt.Errorf("error parsing %s environment variable: %w", constants.EnvVarTenantSummaryWorkerInterval, err)
		}
		config.TenantSummaryWorkerInterval = tenantSummaryWorkerInterval
	}

	if envVal := os.Getenv(constants.EnvVarMaxHeaders); envVal != "" {
		maxHeaders, err := strconv.Atoi(envVal)
		if err != nil {
			return fmt.Errorf("error parsing %s environment variable: %w", constants.EnvVarMaxHeaders, err)
		}
		config.MaxHeaders = maxHeaders
	}

	if envVal := os.Getenv(constants.EnvVarDeploymentID); envVal != "" {
		deploymentID, err := strconv.ParseUint(envVal, 10, 64)
		if err != nil {
			return fmt.Errorf("error parsing %s environment variable: %w", constants.EnvVarDeploymentID, err)
		}
		config.DeploymentID = deploymentID
	}

	if envVal := os.Getenv(constants.EnvVarMessageLeaseDuration); envVal != "" {
		messageLeaseDuration, err := strconv.ParseInt(envVal, 10, 64)
		if err != nil {
			return fmt.Errorf("error parsing %s environment variable: %w", constants.EnvVarMessageLeaseDuration, err)
		}
		config.MessageLeaseDuration = time.Duration(messageLeaseDuration) * time.Second
	}

	// Flags override environment variables and config file
	if *RoleFlag != "" {
		config.Roles = *RoleFlag
	}

	if *JoinFlag {
		config.Join = *JoinFlag
	}

	if *InitialMembersFlag != "" {
		config.InitialMembers = *InitialMembersFlag
	}

	if *SelfMemberHostFlag != "" {
		config.SelfMemberHost = *SelfMemberHostFlag
	}
	if *ClusterBasePortFlag != 0 {
		config.ClusterBasePort = *ClusterBasePortFlag
	}

	if *ReplicaIDFlag != 0 {
		config.ReplicaID = *ReplicaIDFlag
	}
	if *ConnectorPortFlag != 0 {
		config.ConnectorPort = *ConnectorPortFlag
	}

	if *MasterDBEngineFlag != "" {
		config.MasterDBEngine = *MasterDBEngineFlag
	}

	if *TenantDBEngineFlag != "" {
		config.TenantDBEngine = *TenantDBEngineFlag
	}

	if *RestAPIJWTExpirationHoursFlag != 0 {
		config.RestAPIJWTExpirationHours = *RestAPIJWTExpirationHoursFlag
	}

	if *RestListenAddrHostFlag != "" {
		config.RestListenAddrHost = *RestListenAddrHostFlag
	}

	if *RestListenAddrPortFlag != 0 {
		config.RestListenAddrPort = *RestListenAddrPortFlag
	}

	// Note: ApiRaftTimeoutFlag is a time.Duration. If it's different from its default, it means it was set.
	// The default value for the flag (5s) is applied if not overridden by env or file,
	// or if the flag itself is not used.
	// If ApiRaftTimeout is still zero duration here, it means it wasn't set by file or env.
	// The flag's value (*ApiRaftTimeoutFlag) will then be used, which includes its own default.
	// Assign flag value (user-set, or flag's own default e.g. 5s). This overrides env/file.
	config.ApiRaftTimeout = *ApiRaftTimeoutFlag

	if *MaxShardsFlag != 0 {
		config.MaxShards = *MaxShardsFlag
	}

	if *MaxColumnFamiliesFlag != 0 {
		config.MaxColumnFamilies = *MaxColumnFamiliesFlag
	}

	if *GrpcServerListenAddrHostFlag != "" {
		config.GrpcServerListenAddrHost = *GrpcServerListenAddrHostFlag
	}
	if *GrpcServerListenAddrPortFlag != 0 {
		config.GrpcServerListenAddrPort = *GrpcServerListenAddrPortFlag
	}

	// TenantSummaryWorker flag
	if *TenantSummaryWorkerIntervalFlag != 30 { // Only override if different from default
		config.TenantSummaryWorkerInterval = *TenantSummaryWorkerIntervalFlag
	}

	// MaxHeaders flag
	if *MaxHeadersFlag != 0 {
		config.MaxHeaders = *MaxHeadersFlag
	}

	if *DeploymentIDFlag != 0 { // Only override if different from default
		config.DeploymentID = *DeploymentIDFlag
	}

	if *MessageLeaseDurationFlag != 0 {
		config.MessageLeaseDuration = time.Duration(*MessageLeaseDurationFlag) * time.Second
	}

	// Apply defaults if values are not set by any source
	if config.MasterDBEngine == "" {
		config.MasterDBEngine = "pebble"
	}
	if config.TenantDBEngine == "" {
		config.TenantDBEngine = "pebble"
	}
	if config.RestAPIJWTExpirationHours == 0 { // Note: 0 is the default for int if not set by flag/env/file
		config.RestAPIJWTExpirationHours = 3 // Default to 3 hours
	}
	if config.RestListenAddrHost == "" {
		config.RestListenAddrHost = "0.0.0.0"
	}
	if config.RestListenAddrPort == 0 { // Note: 0 is the default for int if not set by flag/env/file
		config.RestListenAddrPort = 3000
	}
	if config.RestAPIJWTSecret == "" {
		config.RestAPIJWTSecret = "super-secret-default-jwt-key-please-change"
		log.Warn().Msgf("⚠️️ WARNING: Rest API JWT Secret is not set, using default insecure key. Please set the %s environment variable or the rest_api_jwt_secret key in your configuration file.", constants.EnvVarRestAPIJWTSecret)
	}

	if config.GrpcServerListenAddrHost == "" {
		config.GrpcServerListenAddrHost = "0.0.0.0"
	}
	if config.GrpcServerListenAddrPort == 0 { // Note: 0 is the default for int if not set by flag/env/file
		config.GrpcServerListenAddrPort = 4000 // Default gRPC port
	}

	if config.TenantSummaryWorkerInterval == 0 {
		config.TenantSummaryWorkerInterval = 30 // Default to 30 seconds
	}
	if config.TenantSummaryWorkerInterval < 10 {
		log.Warn().Msgf("TenantSummaryWorkerInterval (%d seconds) is less than minimum 10 seconds. Setting to 10 seconds.", config.TenantSummaryWorkerInterval)
		config.TenantSummaryWorkerInterval = 10
	}

	if config.MaxHeaders == 0 {
		config.MaxHeaders = 100 // Default to 100 headers
	}
	if config.MaxHeaders < 5 {
		log.Warn().Msgf("MaxHeaders (%d) is less than minimum 5. Setting to 5.", config.MaxHeaders)
		config.MaxHeaders = 5
	}
	if config.MaxHeaders > 1000 {
		log.Warn().Msgf("MaxHeaders (%d) exceeds maximum 1000. Setting to 1000.", config.MaxHeaders)
		config.MaxHeaders = 1000
	}

	if config.DeploymentID == 0 {
		// Default DeploymentID is 0 to maintain compatibility with existing Dragonboat clusters
		config.DeploymentID = 0
	}

	// Default for ApiRaftTimeout if not set by file, env, or flag (flag itself has a default of 5s)
	// This ensures that if config file parsing of api_raft_timeout results in 0 (e.g. key not present or invalid),
	// and env var is not set, and flag is not explicitly used, it still gets the flag's default value.
	// The previous block for flags `if *ApiRaftTimeoutFlag != 0` correctly assigns the flag's value (which could be its default).
	// So, if config.ApiRaftTimeout is *still* 0 here, it implies it wasn't set by config file (parsed to 0),
	// not by env var (parsed to 0), and the flag assignment logic also resulted in 0 (which shouldn't happen for time.Duration unless explicitly set to 0s).
	// A time.Duration field defaults to 0. The flag has a default of 5s.
	// If ApiRaftTimeout is 0 after all loading stages, it means it was explicitly set to "0s" or 0 in the config.
	// We'll rely on the flag's default being propagated correctly.
	// If `config.ApiRaftTimeout` is zero (e.g. after `ConfigFromMapToConfig` if `api_raft_timeout` was 0 and not overridden by ENV or Flag)
	// we should ensure it gets a sensible default. The flag `*ApiRaftTimeoutFlag` will carry its default of 5s if not changed.
	// The assignment `config.ApiRaftTimeout = *ApiRaftTimeoutFlag` handles this.
	// So, an explicit check here for `config.ApiRaftTimeout == 0` and setting it to a default is redundant if the flag logic is correct.
	// Let's verify the order: Config file (parsed into int64 seconds, then to duration), then ENV (parsed as duration), then Flag (is a duration).
	// If config file has api_raft_timeout = 0, then config.ApiRaftTimeout becomes 0.
	// Then if ENV is not set, it remains 0.
	// Then `config.ApiRaftTimeout = *ApiRaftTimeoutFlag` will set it to the flag's value (e.g. 5s default, or user-set value).
	// This logic seems fine.

	if config.ApiRaftTimeout <= 0 {
		log.Warn().Msgf("API Raft Timeout was configured to %v, which is invalid or zero. Resetting to default 30s.", config.ApiRaftTimeout)
		config.ApiRaftTimeout = 30 * time.Second
	}

	// Apply default and validation for MessageLeaseDuration.
	// Priority: args > env var > config file, handled above; this block covers the
	// "not set by any source" case.
	if config.MessageLeaseDuration == 0 {
		config.MessageLeaseDuration = time.Duration(constants.DefaultMessageLeaseDurationSeconds) * time.Second
		log.Info().Msgf("MessageLeaseDuration not specified, defaulting to %v", config.MessageLeaseDuration)
	}
	if config.MessageLeaseDuration < 5*time.Second {
		log.Warn().Msgf("MessageLeaseDuration (%v) is less than minimum 5 seconds. Setting to 5 seconds.", config.MessageLeaseDuration)
		config.MessageLeaseDuration = 5 * time.Second
	}

	// Specific default logic for cluster setup
	if !config.Join && config.SelfMemberHost == "" && config.ClusterBasePort == 0 && config.InitialMembers == "" && config.ReplicaID == 0 {
		if config.SelfMemberHost == "" {
			config.SelfMemberHost = "127.0.0.1"
		}
		if config.ClusterBasePort == 0 {
			config.ClusterBasePort = 17000
		}
		config.ReplicaID = 1
		// Construct InitialMembers from SelfMemberHost and ClusterBasePort if not specified
		if config.InitialMembers == "" {
			config.InitialMembers = fmt.Sprintf("%s:r%d", config.SelfMemberHost, config.ReplicaID)
		}
	} else if !config.Join && config.SelfMemberHost != "" && config.ClusterBasePort != 0 && config.InitialMembers == "" && config.ReplicaID != 0 {
		// If host and port are set, and replica ID is set, but initial members is not, default initial members to self.
		config.InitialMembers = fmt.Sprintf("%s:r%d", config.SelfMemberHost, config.ReplicaID)
	}

	// Apply default for MaxShards if not set by any source
	if config.MaxShards == 0 {
		config.MaxShards = constants.MaxShardsInNonProduction
		log.Info().Msgf("Max shards not specified, defaulting to %d", config.MaxShards)
	}

	var MaxShards int
	if env == string(constants.PRODUCTION) {
		MaxShards = constants.MaxShardsInProduction
	} else {
		MaxShards = constants.MaxShardsInNonProduction
	}

	if config.MaxShards > MaxShards {
		log.Error().Msgf("❌ Max shards (%d) exceeds the maximum allowed (%d). Capping at %d.", config.MaxShards, MaxShards, MaxShards)
		config.MaxShards = MaxShards
	}
	if config.MaxShards <= 0 {
		config.MaxShards = 100
	}

	// Apply default for MaxColumnFamilies if not set by any source
	if config.MaxColumnFamilies == 0 {
		config.MaxColumnFamilies = constants.MaxColumnFamiliesInNonProduction
		log.Info().Msgf("Max column families not specified, defaulting to %d", config.MaxColumnFamilies)
	}

	var maxColumnFamilies int
	if env == string(constants.PRODUCTION) {
		maxColumnFamilies = constants.MaxColumnFamiliesInProduction
	} else {
		maxColumnFamilies = constants.MaxColumnFamiliesInNonProduction
	}

	if config.MaxColumnFamilies > maxColumnFamilies {
		log.Error().Msgf("❌ Max column families (%d) exceeds the maximum allowed (%d). Capping at %d.", config.MaxColumnFamilies, maxColumnFamilies, maxColumnFamilies)
		config.MaxColumnFamilies = maxColumnFamilies
	}
	if config.MaxColumnFamilies <= 0 {
		config.MaxColumnFamilies = 10
	}

	var MaxReplicaId int
	if env == string(constants.PRODUCTION) {
		MaxReplicaId = constants.MaxReplicationInProduction
	} else {
		MaxReplicaId = constants.MaxReplicationInNonProduction
	}

	if int(config.ReplicaID) > MaxReplicaId {
		log.Fatal().Msgf("❌ Replica ID (%d) exceeds the maximum allowed (%d)", config.ReplicaID, MaxReplicaId)
	}

	validateClusterBasePort(config)
	validateRestPortAgainstClusterBasePort(config.RestListenAddrPort, config.ClusterBasePort)

	GlobalConfiguration = config
	return nil
}

// loadMapFromPath reads a configuration file from the given path and parses its key-value pairs
// into a map[string]string.
// Lines starting with '#' are treated as comments and ignored.
// Empty lines are also ignored. Each line should be in 'key=value' format.
//
// Parameters:
//   - path: The file system path to the configuration file.
//
// Returns:
//   - A map[string]string containing configuration key-value pairs from the file.
//   - An error if opening the file or reading it fails.
func loadMapFromPath(path string) (map[string]string, error) {
	configMap := make(map[string]string)

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			// Optionally log a warning for malformed lines
			log.Warn().Msgf("Skipping malformed line in config file %s: %s", path, line)
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if key != "" { // Allow empty values
			configMap[key] = value
		}
	}
	return configMap, scanner.Err()
}

// LoadConfigFromPath reads a configuration file from the given path, parses its key-value pairs,
// and converts them into a Config struct.
// Lines starting with '#' are treated as comments and ignored.
// Empty lines are also ignored. Each line should be in 'key=value' format.
//
// Parameters:
//   - path: The file system path to the configuration file.
//
// Returns:
//   - A pointer to a Config struct populated from the file.
//   - An error if opening the file, reading it, or parsing its content fails.
func LoadConfigFromPath(path string) (*Config, error) {
	configMap, err := loadMapFromPath(path)
	if err != nil {
		return nil, fmt.Errorf("error loading map from path %s: %w", path, err)
	}

	configFromMap, err := mapToConfig(configMap)
	if err != nil {
		return nil, fmt.Errorf("error converting map to configFromMap for path %s: %w", path, err)
	}

	return ConfigFromMapToConfig(*configFromMap), nil
}

// mapToConfig converts a map of string key-value pairs (typically parsed from a config file)
// into an unexported configFromMap struct. It handles type conversions for specific keys.
//
// Parameters:
//   - data: A map[string]string containing configuration key-value pairs.
//
// Returns:
//   - A pointer to a populated configFromMap struct.
//   - An error if parsing any known key into its target type fails (e.g., string to int or bool).
func mapToConfig(data map[string]string) (*ConfigFromMap, error) {
	cfg := &ConfigFromMap{}

	for k, v := range data {
		switch k {
		case constants.ConfigConnectorPortKey:
			p, err := strconv.Atoi(v)
			if err != nil {
				return nil, fmt.Errorf("error parsing %s: %w", k, err)
			}
			cfg.connector_port = p
		case constants.ConfigReplicaIDKey:
			id, err := strconv.ParseUint(v, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("error parsing %s: %w", k, err)
			}
			cfg.replica_id = id

		case constants.ConfigRolesKey:
			cfg.roles = v

		// case constants.ConfigSelfMemberAddrKey: // Deprecated
		//	cfg.self_member_addr = v
		case constants.ConfigSelfMemberHostKey:
			cfg.self_member_host = v
		case constants.ConfigClusterBasePortKey:
			p, err := strconv.Atoi(v)
			if err != nil {
				return nil, fmt.Errorf("error parsing %s: %w", k, err)
			}
			cfg.cluster_base_port = p

		case constants.ConfigInitialMembersKey:
			cfg.initial_members = v

		case constants.ConfigJoinKey:
			join, err := strconv.ParseBool(v)
			if err != nil {
				return nil, fmt.Errorf("error parsing %s: %w", k, err)
			}
			cfg.join = join

		case constants.ConfigTTLInternalErrorKey:
			ttl, err := strconv.ParseUint(v, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("error parsing %s: %w", k, err)
			}
			cfg.ttl_internal_error = ttl

		case constants.ConfigMasterDBEngineKey:
			cfg.master_db_engine = v

		case constants.ConfigTenantDBEngineKey:
			cfg.tenant_db_engine = v
		case constants.ConfigRestAPIJWTExpirationHoursKey:
			p, err := strconv.Atoi(v)
			if err != nil {
				return nil, fmt.Errorf("error parsing %s: %w", k, err)
			}
			cfg.rest_api_jwt_expiration_hours = p
		case "rest_listen_addr_host":
			cfg.rest_listen_addr_host = v
		case "rest_listen_addr_port":
			p, err := strconv.Atoi(v)
			if err != nil {
				return nil, fmt.Errorf("error parsing %s: %w", k, err)
			}
			cfg.rest_listen_addr_port = p
		case constants.ConfigRestAPIJWTSecretKey:
			cfg.rest_api_jwt_secret = v
		case constants.ConfigAPIRaftTimeoutKey:
			p, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("error parsing %s: %w", k, err)
			}
			cfg.api_raft_timeout = p
		case constants.ConfigMaxShardsKey:
			mt, err := strconv.Atoi(v)
			if err != nil {
				return nil, fmt.Errorf("error parsing %s: %w", k, err)
			}
			cfg.max_shards = mt
		case constants.ConfigMaxColumnFamiliesKey:
			mt, err := strconv.Atoi(v)
			if err != nil {
				return nil, fmt.Errorf("error parsing %s: %w", k, err)
			}
			cfg.max_column_families = mt
		case constants.ConfigGrpcServerListenAddrHostKey:
			cfg.grpc_server_listen_addr_host = v
		case constants.ConfigGrpcServerListenAddrPortKey:
			p, err := strconv.Atoi(v)
			if err != nil {
				return nil, fmt.Errorf("error parsing %s: %w", k, err)
			}
			cfg.grpc_server_listen_addr_port = p

		case constants.ConfigTenantSummaryWorkerIntervalKey:
			p, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("error parsing %s: %w", k, err)
			}
			cfg.tenant_summary_worker_interval = p
		case constants.ConfigMaxHeadersKey:
			p, err := strconv.Atoi(v)
			if err != nil {
				return nil, fmt.Errorf("error parsing %s: %w", k, err)
			}
			cfg.max_headers = p

		case constants.ConfigDeploymentIDKey:
			id, err := strconv.ParseUint(v, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("error parsing %s: %w", k, err)
			}
			cfg.deployment_id = id
		case constants.ConfigMessageLeaseDurationKey:
			d, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("error parsing %s: %w", k, err)
			}
			cfg.message_lease_duration = d

		}
	}

	return cfg, nil
}

func validateClusterBasePort(config *Config) {
	clusterBasePort := config.ClusterBasePort
	env := config.Env
	maxShards := config.MaxShards

	if clusterBasePort < constants.MinSafePort || clusterBasePort > constants.MaxPort {
		log.Panic().Msgf("❌ ClusterBasePort (%d) must be between %d and %d",
			clusterBasePort, constants.MinSafePort, constants.MaxPort)
	}

	var maxUsedPort int

	if env != string(constants.PRODUCTION) {
		maxReplicaID := constants.MaxReplicationInNonProduction
		portSpan := maxReplicaID*maxReplicaID*constants.MaxReplicationInNonProduction + maxShards - 1
		maxUsedPort = clusterBasePort + portSpan
	} else {
		portSpan := maxShards - 1
		maxUsedPort = clusterBasePort + portSpan
	}

	if maxUsedPort > constants.MaxPort {
		log.Panic().Msgf("❌ ClusterBasePort (%d) with max shards (%d) exceeds maximum allowed port %d. "+
			"Please adjust the ClusterBasePort or reduce the number of shards.",
			clusterBasePort, maxShards, constants.MaxPort)
	}
}

func validateRestPortAgainstClusterBasePort(restPort int, clusterBasePort int) {
	if restPort >= clusterBasePort-constants.RestPortSafeDistance {
		log.Panic().Msgf("❌ Rest API port (%d) must be at least %d ports below ClusterBasePort (%d) to avoid conflicts. "+
			"Please choose a lower restport.", restPort, constants.RestPortSafeDistance, clusterBasePort)
	}
}
