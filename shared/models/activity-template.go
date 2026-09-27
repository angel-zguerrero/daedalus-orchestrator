package models

import "time"

type ActivityTemplateScope string

const (
	ActivityTemplateScopeGlobal ActivityTemplateScope = "global"
	ActivityTemplateScopeTenant ActivityTemplateScope = "tenant"
)

type ActivityTemplate struct {
	ID               string                `orm:"primary-key"`
	Code             string                `orm:"unique-compound:0"`
	VNamespace       string                `orm:"unique-compound:0"`
	Name             string
	Description      string                `orm:"data-only"`
	ActivityFamily   string
	ParentTemplateId string
	RootActivity     string
	Payload          []byte
	IsActive         bool
	Scope            ActivityTemplateScope
	TenantID         string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (ActivityTemplate) TableName() string {
	return "activity_templates"
}
