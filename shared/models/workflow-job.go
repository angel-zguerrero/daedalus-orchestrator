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
	ID                   string `orm:"primary-key"`
	WorkflowExecutionID string
	ExecutionTokenID     string
	WorkflowDefinitionID string
	VNamespace           string
	ActivityID           string
	ActivityName         string
	ActivityType         string
	Status               WorkflowJobStatus
	Input                map[string]interface{}
	Output               map[string]interface{}
	Error                string
	Retries              int32
	MaxRetries           int32
	TimeoutSeconds       int32
	AssignedWorkerID     string
	LockExpiresAt        *time.Time
	CompletedAt          *time.Time
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

func (WorkflowJob) TableName() string {
	return "workflow_jobs"
}
