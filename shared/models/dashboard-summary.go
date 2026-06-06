package models

import "time"

// DashboardSummaryID is the fixed primary key used for the single global dashboard record.
const DashboardSummaryID = "global"

// DashboardSummary holds aggregated global counters across all tenants.
// It is stored in the master node and kept up to date by the DashboardSummaryWorker.
type DashboardSummary struct {
	ID string `orm:"primary-key"`

	TenantsCount   int
	ExchangesCount int
	QueuesCount    int
	BindingsCount  int
	MessagesCount  int

	UpdatedAt time.Time
}

func (DashboardSummary) TableName() string {
	return "dashboard-summary"
}
