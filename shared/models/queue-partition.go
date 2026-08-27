package models

import "time"

type QueuePartition struct {
	ID string `orm:"primary-key"`

	QueueID  string `orm:"unique-compound:0"`
	Priority int    `orm:"unique-compound:0"`

	MessagesCount int

	FirstQueueMessageID string `orm:"data-only"`
	LastQueueMessageID  string `orm:"data-only"`

	CreatedAt time.Time `orm:"data-only"`
	UpdatedAt time.Time `orm:"data-only"`
}

func (QueuePartition) TableName() string {
	return "queue_partitions"
}
