package models

import "time"

type WorkflowScope string

const (
	WorkflowScopeGlobal WorkflowScope = "global"
	WorkflowScopeTenant WorkflowScope = "tenant"
)

type WorkflowPayloadFormat string

const (
	WorkflowPayloadFormatJSON WorkflowPayloadFormat = "json"
	WorkflowPayloadFormatYAML WorkflowPayloadFormat = "yaml"
)

type VersionChangePolicy string

const (
	VersionChangePolicyDefinedInExecution VersionChangePolicy = "defined_in_execution"
	VersionChangePolicyContinue           VersionChangePolicy = "continue"
	VersionChangePolicyRestart            VersionChangePolicy = "restart"
)

type WorkflowDefinition struct {
	ID                  string                `orm:"primary-key" json:"id"`
	Code                string                `orm:"unique-compound:0" json:"code"`
	VNamespace          string                `orm:"unique-compound:0" json:"vnamespace"`
	Name                string                `json:"name"`
	Description         string                `orm:"data-only" json:"description"`
	Version             int32                 `json:"version"`
	OnVersionChange     VersionChangePolicy   `orm:"data-only" json:"onVersionChange"`
	Payload             []byte                `json:"payload"`
	PayloadFormat       WorkflowPayloadFormat `json:"payloadFormat"`
	MaxDurationSeconds  int32                 `json:"maxDurationSeconds"`
	IsActive            bool                  `json:"isActive"`
	Scope               WorkflowScope         `json:"scope"`
	TenantID            string                `json:"tenantId"`
	HasDesignErrors     bool                  `orm:"data-only" json:"hasDesignErrors"`
	DesignErrorMessages []string              `orm:"data-only" json:"designErrorMessages,omitempty"`
	CreatedAt           time.Time             `json:"createdAt"`
	UpdatedAt           time.Time             `json:"updatedAt"`
}

func (WorkflowDefinition) TableName() string {
	return "workflow_definitions"
}
