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
	UserID                    string                  `json:"userId,omitempty"`
	AccountID                 string                  `json:"accountId,omitempty"`
	AccountName               string                  `json:"accountName,omitempty"`
	ExternalUserID            string                  `json:"externalUserId,omitempty"`
	StartedAt                 *time.Time              `json:"startedAt,omitempty"`
	CompletedAt               *time.Time              `json:"completedAt,omitempty"`
	TTL                       int64                   `orm:"ttl" json:"ttl,omitempty"`
	CreatedAt                 time.Time               `json:"createdAt"`
	UpdatedAt                 time.Time               `json:"updatedAt"`

	// Virtual fields returned in command results for metrics tracking
	EnqueuedGauges []QueueGauges `orm:"virtual" json:"enqueuedGauges,omitempty"`
}

func (WorkflowExecution) TableName() string {
	return "workflow_executions"
}
