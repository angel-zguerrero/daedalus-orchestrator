package models

import "time"

type EnvVar struct {
	ID          string    `orm:"primary-key" json:"id"`
	GroupID     string    `orm:"group-index" json:"groupId"`
	Key         string    `orm:"unique-compound:0" json:"key"`
	GroupIDComp string    `orm:"unique-compound:0" json:"groupIdComp"`
	Value       string    `orm:"data-only" json:"value"`
	Description string    `orm:"data-only" json:"description"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (EnvVar) TableName() string {
	return "env_vars"
}
