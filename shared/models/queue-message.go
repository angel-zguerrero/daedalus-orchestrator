package models

import "time"

type QueueMessage struct {
	ID string `orm:"primary-key"`

	MessageID string `orm:"unique-compound:0"`
	QueueID   string `orm:"unique-compound:0"`

	QueuePartitionID string
	Priority         int

	NextQueueMessageID string

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (QueueMessage) TableName() string {
	return "queue_messages"
}
