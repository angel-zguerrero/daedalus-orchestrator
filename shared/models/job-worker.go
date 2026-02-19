package models

import "time"

type JobWorkerConnectionStatus string

const (
	JobWorkerConnectionStatusConnected    JobWorkerConnectionStatus = "connected"
	JobWorkerConnectionStatusDisconnected JobWorkerConnectionStatus = "disconnected"
)

type JobWorker struct {
	ID   string `orm:"primary-key"`
	Name string `orm:"unique"`

	TTL int64 `orm:"ttl"`

	LastHeartbeat time.Time

	Information map[string]string `orm:"data-only"`

	ConnectionStatus JobWorkerConnectionStatus
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (JobWorker) TableName() string {
	return "job-workers"
}
