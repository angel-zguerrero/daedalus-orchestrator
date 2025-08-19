package models

import "time"

type TenantInMasterStatus string

const (
	PendingForAssign   TenantInMasterStatus = "pending-for-assign"
	Assigned           TenantInMasterStatus = "assigned"
	PendingForDeletion TenantInMasterStatus = "pending-for-deletion"
)

type TenantInMaster struct {
	ID   string `orm:"primary-key"`
	Name string
	Code string `orm:"unique"`

	ShardId           int
	ColumnFamilyIndex int

	ExchangesCount int
	QueuesCount    int
	MessagesCount  int

	Status             TenantInMasterStatus
	LastCheckUpdatedAt time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func (TenantInMaster) TableName() string {
	return "tenants"
}
