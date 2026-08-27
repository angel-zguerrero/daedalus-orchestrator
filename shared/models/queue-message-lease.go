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
	QueueMessageID string `orm:"data-only"`
	WorkerID       string `orm:"data-only"`
	LeaseStatus    QueueMessageLeaseStatus
	LeaseUntil     time.Time

	IsDelivered                       bool `orm:"data-only"`
	JobWorkerCapacityPolicyIndexMatch int  `orm:"data-only"`
	CreatedAt                         time.Time `orm:"data-only"`
}

func (QueueMessageLease) TableName() string {
	return "queue_message_leases"
}
