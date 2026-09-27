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
	ID                        string                  `json:"id" orm:"primary-key"`
	WorkflowDefinitionID      string                  `json:"workflowDefinitionId"`
	WorkflowDefinitionVersion int32                   `json:"workflowDefinitionVersion"`
	OnVersionChange           VersionChangePolicy     `json:"onVersionChange" orm:"data-only"`
	PayloadSnapshot           []byte                  `json:"payloadSnapshot,omitempty" orm:"data-only"`
	VNamespace                string                  `json:"vnamespace"`
	ExecutionKey              string                  `json:"executionKey"`
	Status                    WorkflowExecutionStatus `json:"status"`
	Input                     map[string]interface{}  `json:"input,omitempty"`
	Output                    map[string]interface{} `json:"output,omitempty"`
	StateData                 map[string]interface{} `json:"stateData,omitempty"`
	Error                     string                  `json:"error,omitempty"`
	StartedAt                 *time.Time              `json:"startedAt,omitempty"`
	CompletedAt               *time.Time              `json:"completedAt,omitempty"`
	CreatedAt                 time.Time               `json:"createdAt"`
	UpdatedAt                 time.Time               `json:"updatedAt"`
}

func (WorkflowExecution) TableName() string {
	return "workflow_executions"
}
