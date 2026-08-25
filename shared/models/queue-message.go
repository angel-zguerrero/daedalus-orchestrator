package models

import "time"

type QueueMessage struct {
	ID string `orm:"primary-key"`

	MessageID string `orm:"unique-compound:0"`
	QueueID   string `orm:"unique-compound:0"`

	QueuePartitionID string `orm:"data-only"`
	Priority         int    `orm:"data-only"`

	Attempts int // Kept indexed: mutated by DequeueCommand

	NextQueueMessageID string // Kept indexed: mutated by EnqueueCommand (chaining)

	Parameters map[string]string `orm:"data-only"`

	ContentType   string `orm:"data-only"`
	ContentLength int64  `orm:"data-only"`
	Content       []byte `orm:"data-only"`

	Headers map[string]string `orm:"virtual"` // Virtual field for queue headers, not stored in DB

	Handler string `orm:"data-only"`

	VNamespace string `orm:"data-only"`

	CreatedAt time.Time `orm:"data-only"`
	UpdatedAt time.Time `orm:"data-only"`
}

func (QueueMessage) TableName() string {
	return "queue_messages"
}
