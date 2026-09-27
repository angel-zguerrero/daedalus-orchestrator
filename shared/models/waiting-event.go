package models

import "time"

type WaitingEventType string

const (
	WaitingEventTypeUserInput     WaitingEventType = "USER_INPUT"
	WaitingEventTypeSystemMessage WaitingEventType = "SYSTEM_MESSAGE"
)

type WaitingEvent struct {
	ID                   string `orm:"primary-key"`
	WorkflowDefinitionID string
	WorkflowExecutionID string
	ExecutionTokenID     string
	EventID              string
	Type                 WaitingEventType
	ExpectedInput        map[string]interface{} `orm:"data-only"`
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

func (WaitingEvent) TableName() string {
	return "waiting_events"
}
