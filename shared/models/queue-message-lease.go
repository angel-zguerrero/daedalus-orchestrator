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
	LeaseUntil     time.Time

	IsDelivered                       bool // True if the message was successfully delivered to the worker stream.
	JobWorkerCapacityPolicyIndexMatch int  // Index of the capacity policy that matched when the lease was created, used for worker capacity management and scheduling decisions.
	CreatedAt                         time.Time
}

func (QueueMessageLease) TableName() string {
	return "queue_message_leases"
}
