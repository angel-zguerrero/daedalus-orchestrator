package models

import "time"

type QueueType string

const (
	StandardQueue QueueType = "standard"
)

type QueueSupervisionState string

const (
	Supervised   QueueSupervisionState = "supervised"
	Unsupervised QueueSupervisionState = "unsupervised"
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
	Name string
	Code string `orm:"unique-compound:0"`

	VNamespace string `orm:"unique-compound:0"`

	State QueueState
	Type  QueueType

	MessagesCount int

	MaxDeliveringMessages     int // Max number of messages that can be in delivering state at the same time for this queue. 0 means no limit.
	CurrentDeliveringMessages int // Number of messages that are currently in delivering state for this queue.

	DefaultQueueMessageTTL       int
	DefaultQueueMessageDelayTime int
	QueueExpires                 int
	ExpireAt                     *time.Time
	AllowDuplicated              bool
	MaxAttempts                  int
	DesiredPriorityThresholds    map[int]int       `orm:"data-only"`
	PriorityThresholds           map[int]int       `orm:"data-only"`
	Headers                      map[string]string `orm:"virtual"` // Virtual field for queue headers, not stored in DB

	MaxQueueSize int

	DeadLetterExchangeId                  string
	DeadLetterExchangeRoutingKeyOrPattern string

	CreatedAt time.Time
	UpdatedAt time.Time

	NodeSchedulerQueueSupervisionState QueueSupervisionState
	NodeSchedulerSupervisorId          string
	NodeSchedulerSupervisorCode        string `orm:"virtual"`
	NodeSchedulerSupervisorName        string `orm:"virtual"`
}

func (Queue) TableName() string {
	return "queues"
}
