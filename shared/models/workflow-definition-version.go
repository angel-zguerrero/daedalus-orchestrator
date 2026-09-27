package models

import "time"

type WorkflowDefinitionVersion struct {
	ID                   string                `orm:"primary-key"`
	WorkflowDefinitionID string
	Version              int32
	VNamespace           string
	Payload              []byte                `orm:"data-only"`
	PayloadFormat        WorkflowPayloadFormat `orm:"data-only"`
	StructuralHash       string                `orm:"data-only"`
	CreatedAt            time.Time
}

func (WorkflowDefinitionVersion) TableName() string {
	return "workflow_definition_versions"
}
