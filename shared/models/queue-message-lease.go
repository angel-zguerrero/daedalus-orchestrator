package models

import "time"

type QueueMessageLeaseStatus string

const (
	QueueMessageLeaseStatusActive   QueueMessageLeaseStatus = "active"
	QueueMessageLeaseStatusReleased QueueMessageLeaseStatus = "released"
	QueueMessageLeaseStatusExpired  QueueMessageLeaseStatus = "expired"
)

type QueueMessageLease struct {
	ID             string `orm:"primary-key"`
	QueueMessageID string
	WorkerID       string
	LeaseStatus    QueueMessageLeaseStatus
	LeaseUntil     time.Time `orm:"data-only"`
}

func (QueueMessageLease) TableName() string {
	return "queue_message_leases"
}
