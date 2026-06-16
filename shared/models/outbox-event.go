package models

import "time"

const (
	EventTypeTenantActivated = "TenantActivated"
	EventTypeMetricsRelay    = "MetricsRelay"
)

// OutboxEvent tracks edge-triggered events generated inside a Raft State Machine
// that need to be asynchronously relayed to the Master Node.
type OutboxEvent struct {
	ID        string `orm:"primary-key"`
	EventType string
	TenantID  string
	Payload   []byte
	CreatedAt time.Time
}

func (OutboxEvent) TableName() string {
	return "outbox-events"
}
