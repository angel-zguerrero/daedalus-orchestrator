package models

import "time"

type QueueMessage struct {
	ID string `orm:"primary-key"`

	MessageID string `orm:"unique-compound:0"`
	QueueID   string `orm:"unique-compound:0"`

	QueuePartitionID string
	Priority         int

	NextQueueMessageID string

	Parameters map[string]string `orm:"data-only"`

	ContentType string
	Content     []byte `orm:"data-only"`

	Headers map[string]string `orm:"virtual"` // Virtual field for queue headers, not stored in DB

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (QueueMessage) TableName() string {
	return "queue_messages"
}
