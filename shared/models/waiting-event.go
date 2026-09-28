package models

import "time"

type WaitingEventType string

const (
	WaitingEventTypeUserInput     WaitingEventType = "USER_INPUT"
	WaitingEventTypeSystemMessage WaitingEventType = "SYSTEM_MESSAGE"
)

type WaitingEvent struct {
	ID                   string                 `orm:"primary-key" json:"id"`
	WorkflowDefinitionID string                 `json:"workflowDefinitionId"`
	WorkflowExecutionID  string                 `json:"workflowExecutionId"`
	ExecutionTokenID     string                 `json:"executionTokenId"`
	EventID              string                 `json:"eventId"`
	Type                 WaitingEventType       `json:"type"`
	ExpectedInput        map[string]interface{} `orm:"data-only" json:"expectedInput,omitempty"`
	TTL                  int64                  `orm:"ttl" json:"ttl,omitempty"`
	CreatedAt            time.Time              `json:"createdAt"`
	UpdatedAt            time.Time              `json:"updatedAt"`
}

func (WaitingEvent) TableName() string {
	return "waiting_events"
}
