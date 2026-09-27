package models

import "time"

type TenantSummary struct {
	ID string `orm:"primary-key"`

	QueuesCount   int
	MessagesCount int
	HasMessages   bool

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (TenantSummary) TableName() string {
	return "tenant-summaries"
}
