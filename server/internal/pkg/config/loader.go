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

var configFilePathFlag = flag.String("config", "", "Configuration file path")
var roleFlag = flag.String("role", "", "Comma-separated node roles: consensus, scheduler, connector")
var initialMembersFlag = flag.String("initial-members", "", "Cluster initial members as ip:port,ip:port,...")
var selfMemberAddrFlag = flag.String("self-member-addr", "", "Nodehost address (ip:port)")
var joinFlag = flag.Bool("join", false, "Joining a new node")
var replicaIDFlag = flag.Uint64("replica", 0, "Nodehost replica")
var connectorPortFlag = flag.Int("connector-port", 0, "Connector port")

func LoadDefaultConfiguration() error {
	config := &Config{}
	env := os.Getenv(constants.EnvVarEnvKey)
	if env == "" {
		env = string(constants.DEVELOPMENT)
	}

	flag.Parse()
	configFilePath := os.Getenv(constants.EnvVarConfigPath)
	if configFilePath == "" {
		configFilePath = *configFilePathFlag
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
	if *roleFlag != "" {
		config.Roles = *roleFlag
	}

	if *joinFlag {
		config.Join = *joinFlag
	}

	if *initialMembersFlag != "" {
		config.InitialMembers = *initialMembersFlag
	}

	if *selfMemberAddrFlag != "" {
		config.SelfMemberAddr = *selfMemberAddrFlag
	}

	if *replicaIDFlag != 0 {
		config.ReplicaID = *replicaIDFlag
	}
	if *connectorPortFlag != 0 {
		config.ConnectorPort = *connectorPortFlag
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
