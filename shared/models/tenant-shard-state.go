package models

import "time"

// TenantShardState tracks the internal state of a Tenant within its assigned Shard Node.
// This allows the Shard to definitively know if a tenant currently has messages without
// querying all queues, enabling edge-triggered outbox events.
type TenantShardState struct {
	ID          string `orm:"primary-key"` // Corresponds to TenantID
	HasMessages bool
	UpdatedAt   time.Time
}

func (TenantShardState) TableName() string {
	return "tenant-shard-states"
}
