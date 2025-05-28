package config

import (
	"bufio"
	"deadalus-orch/shared/constants"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
)

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
  --self-member-addr The network address (in ip:port format) that this node will use for communication with other members in the cluster.

Environment Variables:
  CONFIG_PATH                  Path to the configuration file.
  ENV                          Application environment (e.g., development, staging, production).
  DEFAULT_ROOT_USER            Default root username for the application.
  DEFAULT_ROOT_PASSWORD        Default root password for the application.
  REPLICA_ID                   Node's replica ID.
  ROLES                        Comma-separated node roles.
  SELF_MEMBER_ADDR             Node's own member address (ip:port).
  INITIAL_MEMBERS              Initial members of the cluster (ip:port,ip:port,...).
  JOIN                         Set to "true" if the node is joining an existing cluster.
  CONNECTOR_PORT               Port for the connector service.
  TTL_INTERNAL_ERROR           TTL for internal error caching in seconds.
  OTEL_ACTIVE                  Set to "true" or "false" to enable/disable OpenTelemetry.
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
    self_member_addr
    initial_members
    join
    ttl_internal_error

Precedence of Configuration:
  The configuration is loaded in the following order of precedence (highest to lowest):
  1. Command-line flags
  2. Environment variables
  3. Configuration file

  If a setting is specified in multiple places, the value from the source with higher precedence will be used.
`

var ConfigFilePathFlag = flag.String("config", "", "Path to the application configuration file. This flag takes precedence over the CONFIG_PATH environment variable.")
var RoleFlag = flag.String("role", "", "Comma-separated list of roles for this node (e.g., 'consensus,scheduler,connector'). Defines the node's responsibilities within the cluster.")
var InitialMembersFlag = flag.String("initial-members", "", "Comma-separated list of initial member addresses (in ip:port format) for bootstrapping a new cluster. Required when creating a cluster and not using the --join flag.")
var SelfMemberAddrFlag = flag.String("self-member-addr", "", "The network address (in ip:port format) that this node will use for communication with other members in the cluster.")
var JoinFlag = flag.Bool("join", false, "Set this flag to true if this node should attempt to join an existing cluster. When joining, --initial-members should specify addresses of nodes in the existing cluster.")
var ReplicaIDFlag = flag.Uint64("replica", 0, "Unique identifier (positive integer) for this node within the cluster. Required when creating a new cluster or joining an existing one.")
var ConnectorPortFlag = flag.Int("connector-port", 0, "The network port on which the connector service will listen for external client connections. Overrides the 'connector_port' value from the configuration file and the CONNECTOR_PORT environment variable.")
var HelpFlag = flag.Bool("help", false, "Show help message and exit.")

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

	configFilePath := os.Getenv(constants.EnvVarConfigPath)
	if configFilePath == "" {
		configFilePath = *ConfigFilePathFlag
	}

	if configFilePath != "" {
		if _, err := os.Stat(configFilePath); err == nil {
			log.Info().
				Str("path", configFilePath).
				Msg("✅ Using config file")
			cfg, err := LoadConfigFromPath(configFilePath)
			config = cfg
			if err != nil {
				log.Error().
					Str("path", configFilePath).
					Err(err).
					Msg("⚠️ Failed to load config file")
				return err
			}
		} else {
			log.Error().
				Str("path", configFilePath).
				Err(err).
				Msg("⚠️ Failed to load config file")
			return err
		}
	}

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

	if envVal := os.Getenv(constants.EnvVarSelfMemberAddr); envVal != "" {
		config.SelfMemberAddr = envVal
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

	if config.DefaultRootUser == "" {
		config.DefaultRootUser = "admin"
	}

	if config.DefaultRootPassword == "" {
		config.DefaultRootPassword = "admin"
	}
	/* overwrite using flags */
	if *RoleFlag != "" {
		config.Roles = *RoleFlag
	}

	if *JoinFlag {
		config.Join = *JoinFlag
	}

	if *InitialMembersFlag != "" {
		config.InitialMembers = *InitialMembersFlag
	}

	if *SelfMemberAddrFlag != "" {
		config.SelfMemberAddr = *SelfMemberAddrFlag
	}

	if *ReplicaIDFlag != 0 {
		config.ReplicaID = *ReplicaIDFlag
	}
	if *ConnectorPortFlag != 0 {
		config.ConnectorPort = *ConnectorPortFlag
	}

	if !config.Join && config.SelfMemberAddr == "" && config.InitialMembers == "" && config.ReplicaID == 0 {
		if config.SelfMemberAddr == "" {
			config.SelfMemberAddr = "127.0.0.1:7001"
		}
		config.ReplicaID = 1
		config.InitialMembers = "127.0.0.1:7001"
	}

	GlobalConfiguration = config
	return nil
}

func LoadConfigFromPath(path string) (*Config, error) {
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
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if key != "" && value != "" {
			configMap[key] = value
		}
	}
	config, err := mapToConfig(configMap)
	if err != nil {
		return nil, err
	}
	return ConfigFromMapToConfig(*config), scanner.Err()
}

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

		case constants.ConfigSelfMemberAddrKey:
			cfg.self_member_addr = v

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
		}
	}

	return cfg, nil
}
