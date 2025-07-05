package constants

// MasterTenant is a special tenant identifier used for the master or system-level
// Raft group that manages cluster-wide concerns or metadata, as opposed to
// tenant-specific data.
const MasterTenant = "master-tenant"
const MaxTenantsInProduction = 1000
const MaxTenantsInNonProduction = 50

const MaxReplicationInProduction = 100
const MaxReplicationInNonProduction = 10
