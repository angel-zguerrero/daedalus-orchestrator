package models

import "time"

type JobWorkerConnectionStatus string

const (
	JobWorkerConnectionStatusConnected    JobWorkerConnectionStatus = "connected"
	JobWorkerConnectionStatusDisconnected JobWorkerConnectionStatus = "disconnected"
)

type ClaimWorkFilter struct {
	TenantCodes           []string // empty means all tenants
	ExcludeTenantCodes    []string // empty means no exclusions
	TenantPatterns        []string // empty means no patterns
	ExcludeTenantPatterns []string // empty means no exclusions

	VNamespaces               []string // empty means all vnamespaces
	ExcludeVNamespaces        []string // empty means no exclusions
	VNamespacePatterns        []string // empty means no patterns
	ExcludeVNamespacePatterns []string // empty means no exclusions

	QueueCodes           []string // empty means all queues
	ExcludeQueueCodes    []string // empty means no exclusions
	QueuePatterns        []string // empty means no patterns
	ExcludeQueuePatterns []string // empty means no exclusions
}

type ClaimWorkCapacityPolicy struct {
	MaxQueueMessages     int
	CurrentQueueMessages int
	ClaimWorkFilter      ClaimWorkFilter
}

type JobWorker struct {
	ID   string `orm:"primary-key"`
	Name string `orm:"unique"`

	TTL int64 `orm:"ttl"`

	LastHeartbeat time.Time

	Information map[string]string `orm:"data-only"`

	ConnectionStatus JobWorkerConnectionStatus
	CreatedAt        time.Time
	UpdatedAt        time.Time

	//Claim work information, how to select which queues to claim
	ClaimWorkCapacityPolicies             map[string]ClaimWorkCapacityPolicy `orm:"data-only"`
	TenantCursorByClaimWorkPolicyCode     map[string]string
	VNamespaceCursorByClaimWorkPolicyCode map[string]string
	QueueCursorByClaimWorkPolicyCode      map[string]string
}

func (JobWorker) TableName() string {
	return "job-workers"
}
