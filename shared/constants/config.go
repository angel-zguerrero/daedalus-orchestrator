package constants

const ConfigEnvKey = "env"
const ConfigPortKey = "port"
const ConfigDBName = "db_name"
const ConfigDefaultRootUser = "default_root_user"
const ConfigDefaultRootPassword = "default_root_password"

const EnvVarEnvKey = "ENV"
const EnvVarPortKey = "PORT"
const EnvVarDBName = "DB_NAME"
const EnvVarDefaultRootUser = "DEFAULT_ROOT_USER"
const EnvVarDefaultRootPassword = "DEFAULT_ROOT_PASSWORD"

type Env string

const (
	PRODUCTION  Env = "production"
	DEVELOPMENT Env = "development"
	STAGING     Env = "staging"
)
