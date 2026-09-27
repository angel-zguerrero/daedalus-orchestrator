package business_logic

import (
	"deadalus-orch/server/internal/pkg/bpmn"
)

// ValidateEventInput strictly validates an incoming payload against the ExpectedInput schema.
// Returns an error if any required field is missing, data types mismatch, or constraints are violated.
func ValidateEventInput(expectedInput map[string]interface{}, payload map[string]interface{}) error {
	return bpmn.ValidateEventInput(expectedInput, payload)
}
