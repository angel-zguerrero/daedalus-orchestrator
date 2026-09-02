package models

// ActiveQueue represents a queue that has MessagesCount > 0.
// This is used for fast linear scanning by job workers to consume messages without
// having to scan empty queues.
type ActiveQueue struct {
	ID         string `orm:"primary-key"`
	Code       string
	VNamespace string
}

func (ActiveQueue) TableName() string {
	return "active_queues"
}
