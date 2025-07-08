package models

type TenantInMaster struct {
	ID string `orm:"primary-key"`

	Code string `orm:"unique"`

	ShardId int

	IsAssignedToShard bool
}

func (TenantInMaster) TableName() string {
	return "users"
}
