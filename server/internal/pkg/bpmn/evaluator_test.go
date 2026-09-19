package bpmn

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEvaluateCondition(t *testing.T) {
	state := map[string]interface{}{
		"monto":  150,
		"name":   "angel",
		"status": "APPROVED",
		"isVIP":  true,
		"age":    25,
	}

	tests := []struct {
		name      string
		condition string
		expected  bool
		hasError  bool
	}{
		{
			name:      "Empty condition",
			condition: "",
			expected:  true,
		},
		{
			name:      "Simple true",
			condition: "true",
			expected:  true,
		},
		{
			name:      "Simple equality with ${}",
			condition: `${status == "APPROVED"}`,
			expected:  true,
		},
		{
			name:      "Simple numeric comparison",
			condition: "monto > 100",
			expected:  true,
		},
		{
			name:      "Nested condition with && and ||",
			condition: `monto > 100 && (name == "angel" || name == "reina")`,
			expected:  true,
		},
		{
			name:      "Nested condition that evaluates to false",
			condition: `monto > 200 && (name == "angel" || name == "reina")`,
			expected:  false,
		},
		{
			name:      "Complex expression with boolean variable and math",
			condition: `isVIP && age >= 18 && (status == "APPROVED" || status == "PENDING")`,
			expected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := EvaluateCondition(tt.condition, state)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
