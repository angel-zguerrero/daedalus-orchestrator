package constants

// MasterTenant is a special tenant identifier used for the master or system-level
// Raft group that manages cluster-wide concerns or metadata, as opposed to
// tenant-specific data. The typo "mater" should ideally be corrected to "master".
const MasterTenant = "mater-tenant" // TODO: Correct typo to "master-tenant"
