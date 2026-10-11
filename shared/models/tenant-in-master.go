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
	WorkflowsCount int
	QueuesCount    int
	MessagesCount  int
	HasMessages    bool

	Status TenantInMasterStatus

	UserID         string `json:"userId,omitempty"`
	AccountID      string `json:"accountId,omitempty"`
	AccountName    string `json:"accountName,omitempty"`
	ExternalUserID string `json:"externalUserId,omitempty"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (TenantInMaster) TableName() string {
	return "tenants"
}
