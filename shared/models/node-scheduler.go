package models

import "time"

type ConnectionStatus string

const (
	ConnectionStatusConnected    ConnectionStatus = "connected"
	ConnectionStatusDisconnected ConnectionStatus = "disconnected"
)

type NodeScheduler struct {
	ID   string `orm:"primary-key"`
	Name string `orm:"unique"`

	TTL int64 `orm:"ttl"`

	LastHeartbeat time.Time

	Information map[string]string `orm:"data-only"`

	ConnectionStatus ConnectionStatus
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (NodeScheduler) TableName() string {
	return "node-schedulers"
}
