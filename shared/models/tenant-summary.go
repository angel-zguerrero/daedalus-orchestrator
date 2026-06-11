package models

import "time"

type TenantSummary struct {
	ID string `orm:"primary-key"`

	ExchangesCount int
	QueuesCount    int
	BindingsCount  int
	MessagesCount  int
	HasMessages    bool

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (TenantSummary) TableName() string {
	return "tenant-summaries"
}
