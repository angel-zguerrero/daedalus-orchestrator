package models

import "time"

type QueueType string

const (
	StandardQueue   QueueType = "standard"
	DelayedQueue    QueueType = "delayed"
	DeadLetterQueue QueueType = "dead-letter"
)

type QueueState string

const (
	QueueActive   QueueState = "active"
	QueuePaused   QueueState = "paused"
	QueueDraining QueueState = "draining"
	QueueStopped  QueueState = "stopped"
)

type Queue struct {
	ID   string `orm:"primary-key"`
	Name string `orm:"unique-compound:0"`
	Code string `orm:"unique"`

	VNamespace string `orm:"unique-compound:0"`

	State QueueState
	Type  QueueType

	PriorityThresholds map[int]int `orm:"data-only"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (Queue) TableName() string {
	return "queues"
}
