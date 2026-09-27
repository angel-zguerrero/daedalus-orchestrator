package models

import "time"

type ExecutionTokenStatus string

const (
	ExecutionTokenStatusActive    ExecutionTokenStatus = "active"
	ExecutionTokenStatusWaiting   ExecutionTokenStatus = "waiting"
	ExecutionTokenStatusCompleted ExecutionTokenStatus = "completed"
	ExecutionTokenStatusCancelled ExecutionTokenStatus = "cancelled"
)

type ExecutionToken struct {
	ID                   string                 `orm:"primary-key" json:"id"`
	WorkflowExecutionID string                 `json:"workflowExecutionId"`
	WorkflowDefinitionID string                 `json:"workflowDefinitionId"`
	VNamespace           string                 `json:"vnamespace"`
	CurrentNodeID        string                 `json:"currentNodeId"`
	Status               ExecutionTokenStatus   `json:"status"`
	ScopeVariables       map[string]interface{} `json:"scopeVariables,omitempty"`
	ParentTokenID        string                 `json:"parentTokenId,omitempty"`
	CreatedAt            time.Time              `json:"createdAt"`
	UpdatedAt            time.Time              `json:"updatedAt"`
}

func (ExecutionToken) TableName() string {
	return "execution_tokens"
}
