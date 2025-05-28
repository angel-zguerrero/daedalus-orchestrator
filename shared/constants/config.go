package constants

const ConfigConnectorPortKey = "connector_port"
const ConfigReplicaIDKey = "replica_id"
const ConfigRolesKey = "roles"
const ConfigSelfMemberAddrKey = "self_member_addr"
const ConfigInitialMembersKey = "initial_members"
const ConfigJoinKey = "join"
const ConfigTTLInternalErrorKey = "ttl_internal_error"
const ConfigDefaultRootUser = "default_root_user"
const ConfigDefaultRootPassword = "default_root_password"

const EnvVarEnvKey = "ENV"
const EnvVarConfigPath = "CONFIG_PATH"
const EnvVarDefaultRootUser = "DEFAULT_ROOT_USER"
const EnvVarDefaultRootPassword = "DEFAULT_ROOT_PASSWORD"
const EnvVarReplicaId = "REPLICA_ID"
const EnvVarRoles = "ROLES"
const EnvVarSelfMemberAddr = "SELF_MEMBER_ADDR"

const EnvVarInitialMembers = "INITIAL_MEMBERS"

const EnvVarJoin = "JOIN"
const EnvVarConnectorPort = "CONNECTOR_PORT"
const EnvVarTTLInternalError = "TTL_INTERNAL_ERROR"

const EnvVarOtelActived = "OTEL_ACTIVE"
const EnvVarOtelEndpoint = "OTEL_ENDPOINT"
const EnvVarOtelTracerServiceName = "OTEL_TRACER_SERVICE_NAME"

type Env string

const (
	PRODUCTION  Env = "production"
	DEVELOPMENT Env = "development"
	STAGING     Env = "staging"
)

const (
	OTEL_ACTIVE_TRUE  string = "true"
	OTEL_ACTIVE_FALSE string = "false"
)
