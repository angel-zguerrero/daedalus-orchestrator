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
	ID                  string              `orm:"primary-key"`
	Code                string              `orm:"unique-compound:0"`
	VNamespace          string              `orm:"unique-compound:0"`
	Name                string
	Description         string              `orm:"data-only"`
	Version             int32
	OnVersionChange     VersionChangePolicy `orm:"data-only"`
	Payload             []byte
	PayloadFormat       WorkflowPayloadFormat
	MaxDurationSeconds  int32
	IsActive            bool
	Scope               WorkflowScope
	TenantID            string
	HasDesignErrors     bool     `orm:"data-only"`
	DesignErrorMessages []string `orm:"data-only"`
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

func (WorkflowDefinition) TableName() string {
	return "workflow_definitions"
}
