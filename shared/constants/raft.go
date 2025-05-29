package constants

// MasterTenant is a special tenant identifier used for the master or system-level
// Raft group that manages cluster-wide concerns or metadata, as opposed to
// tenant-specific data.
const MasterTenant = "master-tenant"
