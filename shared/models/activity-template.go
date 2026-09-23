package models

import "time"

type ActivityTemplateScope string

const (
	ActivityTemplateScopeGlobal ActivityTemplateScope = "global"
	ActivityTemplateScopeTenant ActivityTemplateScope = "tenant"
)

type ActivityTemplate struct {
	ID               string                `orm:"primary-key" json:"id"`
	Code             string                `orm:"unique-compound:0" json:"code"`
	VNamespace       string                `orm:"unique-compound:0" json:"vnamespace"`
	Name             string                `json:"name"`
	Description      string                `orm:"data-only" json:"description"`
	ActivityFamily   string                `json:"activityFamily"`
	ParentTemplateId string                `json:"parentTemplateId"`
	RootActivity     string                `json:"rootActivity"`
	Payload          []byte                `json:"payload"`
	IsActive         bool                  `json:"isActive"`
	Scope            ActivityTemplateScope `json:"scope"`
	TenantID         string                `json:"tenantId"`
	CreatedAt        time.Time             `json:"createdAt"`
	UpdatedAt        time.Time             `json:"updatedAt"`
}

func (ActivityTemplate) TableName() string {
	return "activity_templates"
}
