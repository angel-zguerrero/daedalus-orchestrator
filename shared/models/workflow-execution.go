package models

import "time"

type WorkflowExecutionStatus string

const (
	WorkflowExecutionStatusPending    WorkflowExecutionStatus = "pending"
	WorkflowExecutionStatusRunning    WorkflowExecutionStatus = "running"
	WorkflowExecutionStatusCompleted  WorkflowExecutionStatus = "completed"
	WorkflowExecutionStatusFailed     WorkflowExecutionStatus = "failed"
	WorkflowExecutionStatusTerminated WorkflowExecutionStatus = "terminated"
	WorkflowExecutionStatusCancelled  WorkflowExecutionStatus = "cancelled"
)

type WorkflowExecution struct {
	ID                        string                  `orm:"primary-key" json:"id"`
	WorkflowDefinitionID      string                  `json:"workflowDefinitionId"`
	WorkflowDefinitionVersion int32                   `json:"workflowDefinitionVersion"`
	OnVersionChange           VersionChangePolicy     `orm:"data-only" json:"onVersionChange,omitempty"`
	PayloadSnapshot           []byte                  `orm:"data-only" json:"payloadSnapshot,omitempty"`
	VNamespace                string                  `json:"vnamespace"`
	ExecutionKey              string                  `json:"executionKey"`
	Status                    WorkflowExecutionStatus `json:"status"`
	Input                     map[string]interface{}  `json:"input"`
	Output                    map[string]interface{}  `json:"output"`
	StateData                 map[string]interface{}  `json:"stateData"`
	Error                     string                  `json:"error,omitempty"`
	StartedAt                 *time.Time              `json:"startedAt,omitempty"`
	CompletedAt               *time.Time              `json:"completedAt,omitempty"`
	CreatedAt                 time.Time               `json:"createdAt"`
	UpdatedAt                 time.Time               `json:"updatedAt"`
}

func (WorkflowExecution) TableName() string {
	return "workflow_executions"
}
