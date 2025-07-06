package constants

// MasterTenant is a special tenant identifier used for the master or system-level
// Raft group that manages cluster-wide concerns or metadata, as opposed to
// tenant-specific data.
const MasterTenant = "master-tenant"
const MaxTenantsInProduction = 10000
const MaxTenantsInNonProduction = 100

const MaxReplicationInProduction = 100
const MaxReplicationInNonProduction = 10

const MinSafePort = 5000
const MaxPort = 65535

const AdminPortSafeDistance = 1000
