package models

import "time"

type ScheduledJobState string

const (
	ScheduledJobIdle      ScheduledJobState = "idle"
	ScheduledJobDelivered ScheduledJobState = "delivered"
)

type ScheduledJobType string

const (
	ScheduledJobOneOff    ScheduledJobType = "OneOff"
	ScheduledJobRecurring ScheduledJobType = "Recurring"
)

type ScheduledJobTargetType string

const (
	ScheduledJobTargetQueue    ScheduledJobTargetType = "queue"
	ScheduledJobTargetExchange ScheduledJobTargetType = "exchange"
)

type ScheduledJob struct {
	ID string `orm:"primary-key"`

	TenantID                       string
	TargetType                     string // "queue" or "exchange"
	TargetID                       string // resolved queue ID or exchange ID
	TargetCode                     string
	RoutingKeyOrPatternOrQueueCode string

	VNamespace  string
	Content     string            `orm:"data-only"`
	ContentType string            `orm:"data-only"`
	Headers     map[string]string `orm:"data-only"`
	Handler     string            `orm:"data-only"`
	Parameters  map[string]string `orm:"data-only"`
	Priority    int               `orm:"data-only"`

	State ScheduledJobState
	Type  ScheduledJobType

	Every          string
	CronExpression string
	RunAt          *time.Time
	RunAfter       string

	NextRunAt time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (ScheduledJob) TableName() string {
	return "scheduled_jobs"
}
