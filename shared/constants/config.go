package constants

// Configuration file keys. These constants define the keys expected in a configuration file (e.g., a .conf file).

// ConfigConnectorPortKey is the key for the connector port setting in the configuration file.
const ConfigConnectorPortKey = "connector_port"

// ConfigReplicaIDKey is the key for the replica ID setting in the configuration file.
const ConfigReplicaIDKey = "replica_id"

// ConfigRolesKey is the key for the node roles setting in the configuration file.
const ConfigRolesKey = "roles"

// ConfigSelfMemberAddrKey is the key for the self member address setting in the configuration file.
// const ConfigSelfMemberAddrKey = "self_member_addr" // Deprecated

// ConfigSelfMemberHostKey is the key for the self member host setting in the configuration file.
const ConfigSelfMemberHostKey = "self_member_host"

// ConfigClusterBasePortKey is the key for the cluster base port setting in the configuration file.
const ConfigClusterBasePortKey = "cluster_base_port"

// ConfigInitialMembersKey is the key for the initial members list setting in the configuration file.
const ConfigInitialMembersKey = "initial_members"

// ConfigJoinKey is the key for the join flag setting in the configuration file.
const ConfigJoinKey = "join"

// ConfigTTLInternalErrorKey is the key for the TTL for internal errors setting in the configuration file.
const ConfigTTLInternalErrorKey = "ttl_internal_error"

// ConfigDefaultRootUser is the key for the default root username setting in the configuration file.
const ConfigDefaultRootUser = "default_root_user"

// ConfigDefaultRootPassword is the key for the default root password setting in the configuration file.
const ConfigDefaultRootPassword = "default_root_password"

// ConfigMasterDBEngineKey is the key for the master database engine setting in the configuration file.
const ConfigMasterDBEngineKey = "master_db_engine"

// ConfigTenantDBEngineKey is the key for the tenant database engine setting in the configuration file.
const ConfigTenantDBEngineKey = "tenant_db_engine"

// ConfigRestAPIJWTExpirationHoursKey is the key for the Rest API JWT expiration hours setting in the configuration file.
const ConfigRestAPIJWTExpirationHoursKey = "rest_api_jwt_expiration_hours"

// ConfigRestListenAddrHostKey is the key for the Rest API listen host setting in the configuration file.
const ConfigRestListenAddrHostKey = "rest_listen_addr_host"

// ConfigRestListenAddrPortKey is the key for the Rest API listen port setting in the configuration file.
const ConfigRestListenAddrPortKey = "rest_listen_addr_port"

// ConfigRestAPIJWTSecretKey is the key for the Rest API JWT secret setting in the configuration file.
const ConfigRestAPIJWTSecretKey = "rest_api_jwt_secret"

// ConfigAPIRaftTimeoutKey is the key for the API Raft timeout setting in the configuration file (in seconds).
const ConfigAPIRaftTimeoutKey = "api_raft_timeout"

// ConfigMaxShardsKey is the key for the maximum number of shards setting in the configuration file.
const ConfigMaxShardsKey = "max_shards"

// ConfigMaxColumnFamiliesKey is the key for the maximum number of column families setting in the configuration file.
const ConfigMaxColumnFamiliesKey = "max_column_families"

// ConfigGrpcServerListenAddrHostKey is the key for the gRPC server listen host setting in the configuration file.
const ConfigGrpcServerListenAddrHostKey = "grpc_server_listen_addr_host"

// ConfigGrpcServerListenAddrPortKey is the key for the gRPC server listen port setting in the configuration file.
const ConfigGrpcServerListenAddrPortKey = "grpc_server_listen_addr_port"

// ConfigNodeSchedulerHeartbeatTimeoutKey is the key for the node scheduler heartbeat timeout setting in the configuration file (in seconds).
const ConfigNodeSchedulerHeartbeatTimeoutKey = "node_scheduler_heartbeat_timeout"

// ConfigNodeSchedulerTTLKey is the key for the node scheduler TTL setting in the configuration file (in minutes).
const ConfigNodeSchedulerTTLKey = "node_scheduler_ttl"

// Environment variable keys. These constants define the names of environment variables used for configuration.

// EnvVarEnvKey is the environment variable name for specifying the application environment (e.g., "development", "production").
// Note: The application internally often refers to "ENV", ensure consistency or map this appropriately.
const EnvVarEnvKey = "ENV" // Consider changing to "ENV" for clarity if that's the actual external expectation.

// EnvVarConfigPath is the environment variable name for the configuration file path.
const EnvVarConfigPath = "CONFIG_PATH"

// EnvVarDefaultRootUser is the environment variable name for the default root username.
const EnvVarDefaultRootUser = "DEFAULT_ROOT_USER"

// EnvVarDefaultRootPassword is the environment variable name for the default root password.
const EnvVarDefaultRootPassword = "DEFAULT_ROOT_PASSWORD"

// EnvVarReplicaId is the environment variable name for the replica ID.
const EnvVarReplicaId = "REPLICA_ID"

// EnvVarRoles is the environment variable name for the node roles (comma-separated).
const EnvVarRoles = "ROLES"

// EnvVarSelfMemberAddr is the environment variable name for the self member address.
// const EnvVarSelfMemberAddr = "SELF_MEMBER_ADDR" // Deprecated

// EnvVarSelfMemberHost is the environment variable name for the self member host.
const EnvVarSelfMemberHost = "SELF_MEMBER_HOST"

// EnvVarClusterBasePort is the environment variable name for the cluster base port.
const EnvVarClusterBasePort = "CLUSTER_BASE_PORT"

// EnvVarInitialMembers is the environment variable name for the initial members list (comma-separated).
const EnvVarInitialMembers = "INITIAL_MEMBERS"

// EnvVarJoin is the environment variable name for the join flag ("true" or "false").
const EnvVarJoin = "JOIN"

// EnvVarConnectorPort is the environment variable name for the connector port.
const EnvVarConnectorPort = "CONNECTOR_PORT"

// EnvVarTTLInternalError is the environment variable name for the TTL for internal errors (in seconds).
const EnvVarTTLInternalError = "TTL_INTERNAL_ERROR"

// EnvVarMasterDBEngine is the environment variable name for the master database engine.
const EnvVarMasterDBEngine = "MASTER_DB_ENGINE"

// EnvVarTenantDBEngine is the environment variable name for the tenant database engine.
const EnvVarTenantDBEngine = "TENANT_DB_ENGINE"

// EnvVarRestAPIJWTExpirationHours is the environment variable name for the Rest API JWT expiration hours.
const EnvVarRestAPIJWTExpirationHours = "REST_API_JWT_EXPIRATION_HOURS"

// EnvVarRestListenAddrHost is the environment variable name for the Rest API listen host.
const EnvVarRestListenAddrHost = "REST_LISTEN_ADDR_HOST"

// EnvVarRestListenAddrPort is the environment variable name for the Rest API listen port.
const EnvVarRestListenAddrPort = "REST_LISTEN_ADDR_PORT"

// EnvVarRestAPIJWTSecret is the environment variable name for the Rest API JWT secret.
const EnvVarRestAPIJWTSecret = "REST_API_JWT_SECRET"

// EnvVarAPIRaftTimeout is the environment variable name for the API Raft timeout (e.g., "5s", "1m").
const EnvVarAPIRaftTimeout = "API_RAFT_TIMEOUT"

// EnvVarMaxShards is the environment variable name for the maximum number of shards.
const EnvVarMaxShards = "MAX_SHARDS"

// EnvVarMaxColumnFamilies is the environment variable name for the maximum number of column families.
const EnvVarMaxColumnFamilies = "MAX_COLUMN_FAMILIES"

// EnvVarGrpcServerListenAddrHost is the environment variable name for the gRPC server listen host.
const EnvVarGrpcServerListenAddrHost = "GRPC_SERVER_LISTEN_ADDR_HOST"

// EnvVarGrpcServerListenAddrPort is the environment variable name for the gRPC server listen port.
const EnvVarGrpcServerListenAddrPort = "GRPC_SERVER_LISTEN_ADDR_PORT"

// EnvVarNodeSchedulerHeartbeatTimeout is the environment variable name for the node scheduler heartbeat timeout (e.g., "3m", "5m").
const EnvVarNodeSchedulerHeartbeatTimeout = "NODE_SCHEDULER_HEARTBEAT_TIMEOUT"

// EnvVarNodeSchedulerTTL is the environment variable name for the node scheduler TTL (in minutes).
const EnvVarNodeSchedulerTTL = "NODE_SCHEDULER_TTL"

// OpenTelemetry specific environment variables.

// EnvVarOtelActived is the environment variable name to enable or disable OpenTelemetry ("true" or "false").
const EnvVarOtelActived = "OTEL_ACTIVED"

// EnvVarOtelEndpoint is the environment variable name for the OpenTelemetry collector endpoint (e.g., "localhost:4317").
const EnvVarOtelEndpoint = "OTEL_ENDPOINT"

// EnvVarOtelTracerServiceName is the environment variable name for the OpenTelemetry tracer service name.
const EnvVarOtelTracerServiceName = "OTEL_TRACER_SERVICE_NAME"

// Env defines the application's operating environment (e.g., production, development, staging).
// It is a string type to allow for easy comparison and extensibility.
type Env string

// Possible values for the Env type.
const (
	// PRODUCTION environment, typically for live, user-facing deployments.
	PRODUCTION Env = "production"
	// DEVELOPMENT environment, typically for local development and testing.
	DEVELOPMENT Env = "development"
	// STAGING environment, typically for pre-production testing and QA.
	STAGING Env = "staging"
)

const (
	// MaxColumnFamiliesInProduction is the maximum number of column families allowed in a production environment.
	MaxColumnFamiliesInProduction = 100
	// MaxColumnFamiliesInNonProduction is the maximum number of column families allowed in a non-production environment.
	MaxColumnFamiliesInNonProduction = 10
)

// Possible string values for OpenTelemetry activation status.
const (
	// OTEL_ACTIVE_TRUE represents the "true" string value for enabling OpenTelemetry.
	OTEL_ACTIVE_TRUE string = "true"
	// OTEL_ACTIVE_FALSE represents the "false" string value for disabling OpenTelemetry.
	OTEL_ACTIVE_FALSE string = "false"
)

// Command-line flag names. These constants define the names of command-line flags used for configuration.

// MasterDBEngineFlagName is the command-line flag name for the master database engine.
const MasterDBEngineFlagName = "master-db-engine"

// TenantDBEngineFlagName is the command-line flag name for the tenant database engine.
const TenantDBEngineFlagName = "tenant-db-engine"

// MaxShardsFlagName is the command-line flag name for the maximum number of shards.
const MaxShardsFlagName = "max-shards"

// MaxColumnFamiliesFlagName is the command-line flag name for the maximum number of column families.
const MaxColumnFamiliesFlagName = "max-column-families"

// SelfMemberHostFlagName is the command-line flag name for the self member host.
const SelfMemberHostFlagName = "self-member-host"

// ClusterBasePortFlagName is the command-line flag name for the cluster base port.
const ClusterBasePortFlagName = "cluster-base-port"

// GrpcServerListenAddrHostFlagName is the command-line flag name for the gRPC server listen host.
const GrpcServerListenAddrHostFlagName = "grpc-host"

// GrpcServerListenAddrPortFlagName is the command-line flag name for the gRPC server listen port.
const GrpcServerListenAddrPortFlagName = "grpc-port"

// NodeSchedulerHeartbeatTimeoutFlagName is the command-line flag name for the node scheduler heartbeat timeout.
const NodeSchedulerHeartbeatTimeoutFlagName = "node-scheduler-heartbeat-timeout"

// NodeSchedulerTTLFlagName is the command-line flag name for the node scheduler TTL (in minutes).
const NodeSchedulerTTLFlagName = "node-scheduler-ttl"
