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
  --cluster-base-port The base port number that this node will use for communication with other members in the cluster. (e.g., 5000)
  --master-db-engine         The database engine for the master database (e.g., "pebble", "rocksdb"). Defaults to "pebble".
  --tenant-db-engine         The database engine for tenant databases (e.g., "pebble", "rocksdb"). Defaults to "pebble".
  --admin-api-jwt-expiration-hours JWT expiration time in hours for the Admin API. Default is 3 hours.
  --admin-host                 Host address for the Admin API. Overrides config file and environment variable.
  --admin-port                 Port for the Admin API. Overrides config file and environment variable.
  --api-raft-timeout           Timeout for API to Raft node requests (e.g., 5s, 1m). Default 5s. Overrides config file and environment variable.
  --max-tenants                Maximum number of tenants (default 10, max 10000). Overrides config file and environment variable.

Environment Variables:
  CONFIG_PATH                  Path to the configuration file.
  ENV                          Application environment (e.g., development, staging, production).
  DEFAULT_ROOT_USER            Default root username for the application.
  DEFAULT_ROOT_PASSWORD        Default root password for the application.
  REPLICA_ID                   Node's replica ID.
  ROLES                        Comma-separated node roles.
  SELF_MEMBER_HOST             Node's own member host (e.g., 127.0.0.1). (Corresponds to ` + constants.EnvVarSelfMemberHost + `)
  CLUSTER_BASE_PORT            Node's own cluster base port (e.g., 5000). (Corresponds to ` + constants.EnvVarClusterBasePort + `)
  INITIAL_MEMBERS              Initial members of the cluster (ip:port,ip:port,...).
  JOIN                         Set to "true" if the node is joining an existing cluster.
  CONNECTOR_PORT               Port for the connector service.
  TTL_INTERNAL_ERROR           TTL for internal error caching in seconds.
  MASTER_DB_ENGINE             The database engine for the master database.
  TENANT_DB_ENGINE             The database engine for tenant databases.
  ADMIN_API_JWT_EXPIRATION_HOURS JWT expiration time in hours for the Admin API.
  ADMIN_LISTEN_ADDR_HOST       Host address for the Admin API. (Corresponds to ` + constants.EnvVarAdminListenAddrHost + `)
  ADMIN_LISTEN_ADDR_PORT       Port for the Admin API. (Corresponds to ` + constants.EnvVarAdminListenAddrPort + `)
  ADMIN_API_JWT_SECRET         JWT secret key for the Admin API. (Corresponds to ` + constants.EnvVarAdminAPIJWTSecret + `)
  API_RAFT_TIMEOUT             Timeout for API to Raft node requests (e.g., "5s", "1m"). (Corresponds to ` + constants.EnvVarAPIRaftTimeout + `)
  MAX_TENANTS                  Maximum number of tenants. (Corresponds to ` + constants.EnvVarMaxTenants + `)
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
    admin_api_jwt_expiration_hours
    admin_listen_addr_host
    admin_listen_addr_port
    admin_api_jwt_secret
    api_raft_timeout               Timeout for API to Raft node requests in seconds (e.g., 5 for 5s).
    tenant_port_range              Tenant port range (e.g., "4000-4100").
    max_tenants                    Maximum number of tenants.

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

// AdminAPIJWTExpirationHoursFlag defines the --admin-api-jwt-expiration-hours command-line flag for specifying the Admin API JWT expiration in hours.
var AdminAPIJWTExpirationHoursFlag = flag.Int("admin-api-jwt-expiration-hours", 0, "JWT expiration time in hours for the Admin API. Overrides config file and environment variable.")

// AdminListenAddrHostFlag defines the --admin-host command-line flag for specifying the Admin API listen host.
var AdminListenAddrHostFlag = flag.String("admin-host", "", "Host address for the Admin API. Overrides config file and environment variable.")

// AdminListenAddrPortFlag defines the --admin-port command-line flag for specifying the Admin API listen port.
var AdminListenAddrPortFlag = flag.Int("admin-port", 0, "Port for the Admin API. Overrides config file and environment variable.")

// ApiRaftTimeoutFlag defines the --api-raft-timeout command-line flag for specifying the API to Raft node request timeout.
var ApiRaftTimeoutFlag = flag.Duration("api-raft-timeout", 5*time.Second, "Timeout for API to Raft node requests (e.g., 5s, 1m). Overrides config file and environment variable.")

// MaxTenantsFlag defines the --max-tenants command-line flag.
var MaxTenantsFlag = flag.Int(
	constants.MaxTenantsFlagName,
	0,
	fmt.Sprintf(
		"Maximum number of tenants (default: 10, max: %d in production, %d in non-production environments).",
		constants.MaxTenantsInProduction,
		constants.MaxTenantsInNonProduction,
	),
)

// HelpFlag defines the --help command-line flag to display the help message.
var HelpFlag = flag.Bool("help", false, "Show help message and exit.")

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
					Msg("⚠️ Failed to load map from config file")
				return err
			}

			cfgFromMapIntermediate, err := mapToConfig(configMap)
			if err != nil {
				log.Error().
					Str("path", configFilePath).
					Err(err).
					Msg("⚠️ Failed to parse config file data into struct")
				return err
			}
			// Apply all other values from config file to the main config object
			config = ConfigFromMapToConfig(*cfgFromMapIntermediate)

		} else {
			log.Error().
				Str("path", configFilePath).
				Err(err).
				Msg("⚠️ Failed to load config file")
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

	if envVal := os.Getenv(constants.EnvVarAdminAPIJWTExpirationHours); envVal != "" {
		jwtExpirationHours, err := strconv.Atoi(envVal)
		if err != nil {
			return err
		}
		config.AdminAPIJWTExpirationHours = jwtExpirationHours
	}

	if envVal := os.Getenv(constants.EnvVarAdminListenAddrHost); envVal != "" {
		config.AdminListenAddrHost = envVal
	}

	if envVal := os.Getenv(constants.EnvVarAdminListenAddrPort); envVal != "" {
		adminPort, err := strconv.Atoi(envVal)
		if err != nil {
			return err
		}
		config.AdminListenAddrPort = adminPort
	}

	if envVal := os.Getenv(constants.EnvVarAdminAPIJWTSecret); envVal != "" {
		config.AdminAPIJWTSecret = envVal
	}

	if envVal := os.Getenv(constants.EnvVarAPIRaftTimeout); envVal != "" {
		apiRaftTimeout, err := time.ParseDuration(envVal)
		if err != nil {
			return fmt.Errorf("error parsing %s environment variable: %w", constants.EnvVarAPIRaftTimeout, err)
		}
		config.ApiRaftTimeout = apiRaftTimeout
	}

	// MaxTenants from environment variable
	if envVal := os.Getenv(constants.EnvVarMaxTenants); envVal != "" {
		maxTenants, err := strconv.Atoi(envVal)
		if err != nil {
			return fmt.Errorf("error parsing %s environment variable: %w", constants.EnvVarMaxTenants, err)
		}
		config.MaxTenants = maxTenants
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

	if *AdminAPIJWTExpirationHoursFlag != 0 {
		config.AdminAPIJWTExpirationHours = *AdminAPIJWTExpirationHoursFlag
	}

	if *AdminListenAddrHostFlag != "" {
		config.AdminListenAddrHost = *AdminListenAddrHostFlag
	}

	if *AdminListenAddrPortFlag != 0 {
		config.AdminListenAddrPort = *AdminListenAddrPortFlag
	}

	// Note: ApiRaftTimeoutFlag is a time.Duration. If it's different from its default, it means it was set.
	// The default value for the flag (5s) is applied if not overridden by env or file,
	// or if the flag itself is not used.
	// If ApiRaftTimeout is still zero duration here, it means it wasn't set by file or env.
	// The flag's value (*ApiRaftTimeoutFlag) will then be used, which includes its own default.
	// Assign flag value (user-set, or flag's own default e.g. 5s). This overrides env/file.
	config.ApiRaftTimeout = *ApiRaftTimeoutFlag

	if *MaxTenantsFlag != 0 {
		config.MaxTenants = *MaxTenantsFlag
	}

	// Apply defaults if values are not set by any source
	if config.DefaultRootUser == "" {
		config.DefaultRootUser = "admin"
	}
	if config.DefaultRootPassword == "" {
		config.DefaultRootPassword = "admin"
	}
	if config.MasterDBEngine == "" {
		config.MasterDBEngine = "pebble"
	}
	if config.TenantDBEngine == "" {
		config.TenantDBEngine = "pebble"
	}
	if config.AdminAPIJWTExpirationHours == 0 { // Note: 0 is the default for int if not set by flag/env/file
		config.AdminAPIJWTExpirationHours = 3 // Default to 3 hours
	}
	if config.AdminListenAddrHost == "" {
		config.AdminListenAddrHost = "0.0.0.0"
	}
	if config.AdminListenAddrPort == 0 { // Note: 0 is the default for int if not set by flag/env/file
		config.AdminListenAddrPort = 3000
	}
	if config.AdminAPIJWTSecret == "" {
		config.AdminAPIJWTSecret = "super-secret-default-jwt-key-please-change"
		log.Warn().Msgf("⚠️ WARNING: Admin API JWT Secret is not set, using default insecure key. Please set the %s environment variable or the admin_api_jwt_secret key in your configuration file.", constants.EnvVarAdminAPIJWTSecret)
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
		log.Warn().Msgf("API Raft Timeout was configured to %v, which is invalid or zero. Resetting to default 5s.", config.ApiRaftTimeout)
		config.ApiRaftTimeout = 5 * time.Second
	}

	// Specific default logic for cluster setup
	if !config.Join && config.SelfMemberHost == "" && config.ClusterBasePort == 0 && config.InitialMembers == "" && config.ReplicaID == 0 {
		if config.SelfMemberHost == "" {
			config.SelfMemberHost = "127.0.0.1"
		}
		if config.ClusterBasePort == 0 {
			config.ClusterBasePort = 5000
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

	// Apply default for MaxTenants if not set by any source
	if config.MaxTenants == 0 {
		config.MaxTenants = 10 // Default to 10 tenants
		log.Info().Msgf("Max tenants not specified, defaulting to %d", config.MaxTenants)
	}

	var MaxTenants int
	if env == string(constants.PRODUCTION) {
		MaxTenants = constants.MaxTenantsInProduction
	} else {
		MaxTenants = constants.MaxTenantsInNonProduction
	}

	if config.MaxTenants > MaxTenants {
		log.Error().Msgf("❌ Max tenants (%d) exceeds the maximum allowed (%d). Capping at %d.", config.MaxTenants, MaxTenants, MaxTenants)
		config.MaxTenants = MaxTenants
	}
	if config.MaxTenants <= 0 {
		config.MaxTenants = 10
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
	validateAdminPortAgainstClusterBasePort(config.AdminListenAddrPort, config.ClusterBasePort)

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

		case constants.ConfigDefaultRootUser:
			cfg.default_root_user = v

		case constants.ConfigDefaultRootPassword:
			cfg.default_root_password = v

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
		case constants.ConfigAdminAPIJWTExpirationHoursKey:
			p, err := strconv.Atoi(v)
			if err != nil {
				return nil, fmt.Errorf("error parsing %s: %w", k, err)
			}
			cfg.admin_api_jwt_expiration_hours = p
		case "admin_listen_addr_host":
			cfg.admin_listen_addr_host = v
		case "admin_listen_addr_port":
			p, err := strconv.Atoi(v)
			if err != nil {
				return nil, fmt.Errorf("error parsing %s: %w", k, err)
			}
			cfg.admin_listen_addr_port = p
		case constants.ConfigAdminAPIJWTSecretKey:
			cfg.admin_api_jwt_secret = v
		case constants.ConfigAPIRaftTimeoutKey:
			p, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("error parsing %s: %w", k, err)
			}
			cfg.api_raft_timeout = p
		case constants.ConfigMaxTenantsKey:
			mt, err := strconv.Atoi(v)
			if err != nil {
				return nil, fmt.Errorf("error parsing %s: %w", k, err)
			}
			cfg.max_tenants = mt
		}
	}

	return cfg, nil
}

func validateClusterBasePort(config *Config) {
	clusterBasePort := config.ClusterBasePort
	env := config.Env
	maxTenants := config.MaxTenants

	if clusterBasePort < constants.MinSafePort || clusterBasePort > constants.MaxPort {
		log.Panic().Msgf("❌ ClusterBasePort (%d) must be between %d and %d",
			clusterBasePort, constants.MinSafePort, constants.MaxPort)
	}

	var maxUsedPort int

	if env != string(constants.PRODUCTION) {
		maxReplicaID := constants.MaxReplicationInNonProduction
		portSpan := maxReplicaID*maxReplicaID*constants.MaxReplicationInNonProduction + maxTenants - 1
		maxUsedPort = clusterBasePort + portSpan
	} else {
		portSpan := maxTenants - 1
		maxUsedPort = clusterBasePort + portSpan
	}

	if maxUsedPort > constants.MaxPort {
		log.Panic().Msgf("❌ ClusterBasePort (%d) with max tenants (%d) exceeds maximum allowed port %d. "+
			"Please adjust the ClusterBasePort or reduce the number of tenants.",
			clusterBasePort, maxTenants, constants.MaxPort)
	}
}

func validateAdminPortAgainstClusterBasePort(adminPort int, clusterBasePort int) {
	if adminPort >= clusterBasePort-constants.AdminPortSafeDistance {
		log.Panic().Msgf("❌ Admin API port (%d) must be at least %d ports below ClusterBasePort (%d) to avoid conflicts. "+
			"Please choose a lower admin port.", adminPort, constants.AdminPortSafeDistance, clusterBasePort)
	}
}
