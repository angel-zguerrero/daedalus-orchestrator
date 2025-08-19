package models

import "time"

type TenantSummary struct {
	ID       string `orm:"primary-key"`
	TenantId string
	Code     string

	ExchangesCount int
	QueuesCount    int
	MessagesCount  int

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (TenantSummary) TableName() string {
	return "tenant-summaries"
}
