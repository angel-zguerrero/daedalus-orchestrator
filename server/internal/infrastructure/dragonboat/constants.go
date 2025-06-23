package dragonboat

// LocalDefaultPort is the default port used for local Dragonboat communication.
const LocalDefaultPort = 7000

// LocalDefaultHost is the default host address used for local Dragonboat communication.
const LocalDefaultHost = "127.0.0.1"

const (
	// AppliedIndexKey is the key used in the metadata store to save the last applied Raft index for a shard.
	AppliedIndexKey string = "disk_kv_applied_index"
	// CurrentDBFilename is the standard filename for the current RocksDB database instance.
	CurrentDBFilename string = "current"
	// UpdatingDBFilename is the temporary filename used for a RocksDB instance while it's being updated (e.g., during snapshot application).
	UpdatingDBFilename string = "current.updating"
)

const (
	// RoleConsensus indicates that the node participates in Raft consensus for data replication and leadership.
	RoleConsensus NodeRole = "consensus"
	// RoleScheduler indicates that the node is responsible for scheduling tasks or operations.
	RoleScheduler NodeRole = "scheduler"
	// RoleConnector indicates that the node acts as a connector or gateway to external systems or services.
	RoleConnector NodeRole = "connector"
	// RoleAdmin indicates that the node should expose administrator-level APIs.
	RoleAdmin NodeRole = "admin"
)

// MasterShardID is the dedicated Shard ID for the master shard, which handles cluster-wide metadata and coordination.
const MasterShardID = 1
