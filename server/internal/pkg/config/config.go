package config

// Config holds the application's configuration parameters.
// These parameters are typically loaded from command-line flags, environment variables,
// or a configuration file.
type Config struct {
	// ReplicaID is the unique identifier for this node in a Raft cluster.
	// It's essential for distinguishing replicas within a shard.
	ReplicaID uint64
	// Roles specifies the roles this node will perform in the cluster (e.g., "consensus", "scheduler", "connector").
	// It's typically a comma-separated string.
	Roles string
	// SelfMemberAddr is the network address (IP:port) that this node uses for Raft communication.
	SelfMemberAddr string
	// InitialMembers is a comma-separated list of Raft addresses for all members in a new cluster.
	// This is used when bootstrapping the cluster.
	InitialMembers string
	// Join indicates whether this node should attempt to join an existing cluster.
	// If true, InitialMembers might be used to discover the cluster.
	Join bool
	// ConnectorPort is the port on which the connector role (if enabled) will listen for external connections.
	ConnectorPort int
	// TTLInternalError is the Time-To-Live (in seconds) for internal error messages stored in the database.
	TTLInternalError uint64
	// DefaultRootUser is the username for the default administrative user, created during initial setup.
	DefaultRootUser string
	// DefaultRootPassword is the password for the default administrative user.
	DefaultRootPassword string
}

// ConfigFromMap is an unexported struct used as an intermediary when loading
// configuration data from a map (e.g., from a JSON or YAML file parsed into a map).
// Its fields are lowercase and use underscores, matching common conventions for map keys from config files.
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

// ConfigFromMapToConfig converts a configFromMap struct (typically derived from a config file)
// into the primary Config struct used by the application.
//
// Parameters:
//   - configFromMapInstance: An instance of the unexported configFromMap struct.
//
// Returns:
//   - A pointer to a Config struct populated with values from configFromMapInstance.
func ConfigFromMapToConfig(configFromMapInstance ConfigFromMap) *Config {
	c := &Config{
		ReplicaID:           configFromMapInstance.replica_id,
		Roles:               configFromMapInstance.roles,
		SelfMemberAddr:      configFromMapInstance.self_member_addr,
		InitialMembers:      configFromMapInstance.initial_members,
		Join:                configFromMapInstance.join,
		ConnectorPort:       configFromMapInstance.connector_port,
		TTLInternalError:    configFromMapInstance.ttl_internal_error,
		DefaultRootUser:     configFromMapInstance.default_root_user,
		DefaultRootPassword: configFromMapInstance.default_root_password,
	}
	return c
}

// GlobalConfiguration holds the globally accessible configuration for the application.
// It is populated by LoadDefaultConfiguration (or other loading mechanisms) at startup.
// Access to this variable should be read-only after initialization to prevent race conditions.
var GlobalConfiguration *Config
