package models

// QueueGauges represents the current size counters for a queue
type QueueGauges struct {
	QueueCode  string
	VNamespace string
	Pending    uint64
	InProcess  uint64
}
