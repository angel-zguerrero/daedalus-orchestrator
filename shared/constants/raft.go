package constants

// MasterTenant is a special tenant identifier used for the master or system-level
// Raft group that manages cluster-wide concerns or metadata, as opposed to
// tenant-specific data.
const MasterTenant = "master-tenant"
const MaxShardsInProduction = 1000
const MaxShardsInNonProduction = 100

const MaxReplicationInProduction = 100
const MaxReplicationInNonProduction = 10

const MinSafePort = 17000
const MaxPort = 65535

const RestPortSafeDistance = 1000
