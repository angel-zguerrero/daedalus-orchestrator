package models

import "time"

type WorkflowJobStatus string

const (
	WorkflowJobStatusPending   WorkflowJobStatus = "pending"
	WorkflowJobStatusAssigned  WorkflowJobStatus = "assigned"
	WorkflowJobStatusCompleted WorkflowJobStatus = "completed"
	WorkflowJobStatusFailed    WorkflowJobStatus = "failed"
	WorkflowJobStatusTimeout   WorkflowJobStatus = "timeout"
)

type WorkflowJob struct {
	ID                   string                 `orm:"primary-key" json:"id"`
	WorkflowExecutionID string                 `json:"workflowExecutionId"`
	ExecutionTokenID     string                 `json:"executionTokenId"`
	WorkflowDefinitionID string                 `json:"workflowDefinitionId"`
	VNamespace           string                 `json:"vnamespace"`
	ActivityID           string            `json:"activityId"`
	ActivityName         string            `json:"activityName"`
	ActivityType         string            `json:"activityType"`
	Status               WorkflowJobStatus `json:"status"`
	Input                map[string]interface{} `json:"input"`
	Output               map[string]interface{} `json:"output"`
	Error                string            `json:"error,omitempty"`
	Retries              int32             `json:"retries"`
	MaxRetries           int32             `json:"maxRetries"`
	TimeoutSeconds       int32             `json:"timeoutSeconds"`
	AssignedWorkerID     string            `json:"assignedWorkerId,omitempty"`
	LockExpiresAt        *time.Time        `json:"lockExpiresAt,omitempty"`
	CompletedAt          *time.Time        `json:"completedAt,omitempty"`
	CreatedAt            time.Time         `json:"createdAt"`
	UpdatedAt            time.Time         `json:"updatedAt"`
}

func (WorkflowJob) TableName() string {
	return "workflow_jobs"
}
