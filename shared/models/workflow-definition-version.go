package models

import "time"

type WorkflowDefinitionVersion struct {
	ID                   string                `orm:"primary-key" json:"id"`
	WorkflowDefinitionID string                `json:"workflowDefinitionId"`
	Version              int32                 `json:"version"`
	VNamespace           string                `json:"vnamespace"`
	Payload              []byte                `orm:"data-only" json:"payload"`
	PayloadFormat        WorkflowPayloadFormat `orm:"data-only" json:"payloadFormat"`
	StructuralHash       string                `orm:"data-only" json:"structuralHash"`
	CreatedAt            time.Time             `json:"createdAt"`
}

func (WorkflowDefinitionVersion) TableName() string {
	return "workflow_definition_versions"
}
