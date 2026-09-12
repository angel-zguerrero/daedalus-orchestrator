package models

import "time"

type ScheduledJobTrackerStatus string

const (
	ScheduledJobTrackerPending   ScheduledJobTrackerStatus = "pending"
	ScheduledJobTrackerCompleted ScheduledJobTrackerStatus = "completed"
	ScheduledJobTrackerExhausted ScheduledJobTrackerStatus = "exhausted"
)

type ScheduledJobTracker struct {
	ID             string `orm:"primary-key"`
	ScheduledJobID string `orm:"unique-compound:0"`
	ExecutionID    string `orm:"unique-compound:0"`
	QueueID        string `orm:"unique-compound:0"`
	Status         ScheduledJobTrackerStatus

	CreatedAt time.Time `orm:"data-only"`
	UpdatedAt time.Time `orm:"data-only"`
}

func (ScheduledJobTracker) TableName() string {
	return "scheduled_job_trackers"
}
