package constants

// Configuration file keys. These constants define the keys expected in a configuration file (e.g., a .conf file).

// ConfigConnectorPortKey is the key for the connector port setting in the configuration file.
const ConfigConnectorPortKey = "connector_port"

// ConfigReplicaIDKey is the key for the replica ID setting in the configuration file.
const ConfigReplicaIDKey = "replica_id"

// ConfigRolesKey is the key for the node roles setting in the configuration file.
const ConfigRolesKey = "roles"

// ConfigSelfMemberAddrKey is the key for the self member address setting in the configuration file.
const ConfigSelfMemberAddrKey = "self_member_addr"

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

// Environment variable keys. These constants define the names of environment variables used for configuration.

// EnvVarEnvKey is the environment variable name for specifying the application environment (e.g., "development", "production").
// Note: The application internally often refers to "DEADALUS_ENV", ensure consistency or map this appropriately.
const EnvVarEnvKey = "ENV" // Consider changing to "DEADALUS_ENV" for clarity if that's the actual external expectation.

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
const EnvVarSelfMemberAddr = "SELF_MEMBER_ADDR"

// EnvVarInitialMembers is the environment variable name for the initial members list (comma-separated).
const EnvVarInitialMembers = "INITIAL_MEMBERS"

// EnvVarJoin is the environment variable name for the join flag ("true" or "false").
const EnvVarJoin = "JOIN"

// EnvVarConnectorPort is the environment variable name for the connector port.
const EnvVarConnectorPort = "CONNECTOR_PORT"

// EnvVarTTLInternalError is the environment variable name for the TTL for internal errors (in seconds).
const EnvVarTTLInternalError = "TTL_INTERNAL_ERROR"

// OpenTelemetry specific environment variables.

// EnvVarOtelActived is the environment variable name to enable or disable OpenTelemetry ("true" or "false").
const EnvVarOtelActived = "OTEL_ACTIVE"

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

// OtelActive defines the activation status for OpenTelemetry, represented as strings "true" or "false".
// Using a specific type alias, even if underlying is string, can improve type safety if used consistently.
// However, it's currently defined as const strings directly.
type OtelActive string // Example of how it could be typed; current usage is string constants.

// Possible string values for OpenTelemetry activation status.
const (
	// OTEL_ACTIVE_TRUE represents the "true" string value for enabling OpenTelemetry.
	OTEL_ACTIVE_TRUE string = "true"
	// OTEL_ACTIVE_FALSE represents the "false" string value for disabling OpenTelemetry.
	OTEL_ACTIVE_FALSE string = "false"
)
