package models

import "time"

type QueuePartition struct {
	ID string `orm:"primary-key"`

	QueueID  string `orm:"unique-compound:0"`
	Priority int    `orm:"unique-compound:0"`

	MessagesCount int

	FirstQueueMessageID string
	LastQueueMessageID  string

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (QueuePartition) TableName() string {
	return "queue_partitions"
}
