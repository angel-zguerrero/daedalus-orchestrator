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
	ID                   string                 `orm:"primary-key"`
	WorkflowExecutionID string
	WorkflowDefinitionID string
	VNamespace           string
	CurrentNodeID        string
	Status               ExecutionTokenStatus
	ScopeVariables       map[string]interface{}
	ParentTokenID        string
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

func (ExecutionToken) TableName() string {
	return "execution_tokens"
}
