package models

import "time"

type EnvVar struct {
	ID          string    `orm:"primary-key"`
	GroupID     string    `orm:"group-index"`
	Key         string    `orm:"unique-compound:0"`
	GroupIDComp string    `orm:"unique-compound:0"`
	Value       string    `orm:"data-only"`
	Description string    `orm:"data-only"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (EnvVar) TableName() string {
	return "env_vars"
}
