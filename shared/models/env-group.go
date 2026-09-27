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
	ID          string        `orm:"primary-key" json:"id"`
	Code        string        `orm:"unique-compound:0" json:"code"`
	VNamespace  string        `orm:"unique-compound:0" json:"vnamespace"`
	Name        string        `json:"name"`
	Description string        `orm:"data-only" json:"description"`
	Type        EnvGroupType  `json:"type"` // "config" or "secret"
	Scope       EnvGroupScope `json:"scope"` // "global" or "tenant"
	TenantID    string        `json:"tenantId"`
	CreatedAt   time.Time     `json:"createdAt"`
	UpdatedAt   time.Time     `json:"updatedAt"`
}

func (EnvGroup) TableName() string {
	return "env_groups"
}
