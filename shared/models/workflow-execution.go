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
	ID                        string `orm:"primary-key"`
	WorkflowDefinitionID      string
	WorkflowDefinitionVersion int32
	OnVersionChange           VersionChangePolicy `orm:"data-only"`
	PayloadSnapshot           []byte              `orm:"data-only"`
	VNamespace                string
	ExecutionKey              string
	Status                    WorkflowExecutionStatus
	Input                     map[string]interface{}
	Output                    map[string]interface{}
	StateData                 map[string]interface{}
	Error                     string
	StartedAt                 *time.Time
	CompletedAt               *time.Time
	CreatedAt                 time.Time
	UpdatedAt                 time.Time
}

func (WorkflowExecution) TableName() string {
	return "workflow_executions"
}
