package config

type Config struct {
	ReplicaID           uint64
	Roles               string
	SelfMemberAddr      string
	InitialMembers      string
	Join                bool
	ConnectorPort       int
	TTLInternalError    uint64
	DefaultRootUser     string
	DefaultRootPassword string
}

type ConfigFromMap struct {
	replica_id            uint64
	roles                 string
	self_member_addr      string
	initial_members       string
	join                  bool
	connector_port        int
	ttl_internal_error    uint64
	default_root_user     string
	default_root_password string
}

func ConfigFromMapToConfig(configFromMap ConfigFromMap) *Config {
	c := &Config{
		ReplicaID:           configFromMap.replica_id,
		Roles:               configFromMap.roles,
		SelfMemberAddr:      configFromMap.self_member_addr,
		InitialMembers:      configFromMap.initial_members,
		Join:                configFromMap.join,
		ConnectorPort:       configFromMap.connector_port,
		TTLInternalError:    configFromMap.ttl_internal_error,
		DefaultRootUser:     configFromMap.default_root_user,
		DefaultRootPassword: configFromMap.default_root_password,
	}
	return c
}

var GlobalConfiguration *Config
