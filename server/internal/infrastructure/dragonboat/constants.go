package dragonboat

const LocalDefaultPort = 7000
const LocalDefaultHost = "127.0.0.1"

const (
	AppliedIndexKey    string = "disk_kv_applied_index"
	CurrentDBFilename  string = "current"
	UpdatingDBFilename string = "current.updating"
)

const (
	prefixData      = "data:"
	prefixTTLIndex  = "ttl-index:"
	prefixTTLExpire = "ttl-expire:"
)

type NodeRole string

const (
	RoleConsensus NodeRole = "consensus"
	RoleScheduler NodeRole = "scheduler"
	RoleConnector NodeRole = "connector"
)

const MasterShardID = 1

type Member struct {
	IP   string
	Port int
}
