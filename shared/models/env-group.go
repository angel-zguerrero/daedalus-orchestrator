package models

import "time"

type EnvGroupType string

const (
	EnvGroupTypeConfig EnvGroupType = "config"
	EnvGroupTypeSecret EnvGroupType = "secret"
)

type EnvGroupScope string

const (
	EnvGroupScopeGlobal EnvGroupScope = "global"
	EnvGroupScopeTenant EnvGroupScope = "tenant"
)

type EnvGroup struct {
	ID          string        `orm:"primary-key"`
	Code        string        `orm:"unique-compound:0"`
	VNamespace  string        `orm:"unique-compound:0"`
	Name        string
	Description string        `orm:"data-only"`
	Type        EnvGroupType  // "config" or "secret"
	Scope       EnvGroupScope // "global" or "tenant"
	TenantID    string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (EnvGroup) TableName() string {
	return "env_groups"
}
