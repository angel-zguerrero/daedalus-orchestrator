package models

import "time"

const (
	EventTypeTenantActivated = "TenantActivated"
)

// OutboxEvent tracks edge-triggered events generated inside a Raft State Machine
// that need to be asynchronously relayed to the Master Node.
type OutboxEvent struct {
	ID        string `orm:"primary-key"`
	EventType string
	TenantID  string
	CreatedAt time.Time
}

func (OutboxEvent) TableName() string {
	return "outbox-events"
}
