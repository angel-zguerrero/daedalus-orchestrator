package utils_test

import (
	"deadalus-orch/server/internal/pkg/utils"
	"testing"
)

func TestIsValidPort(t *testing.T) {
	tests := []struct {
		name     string
		port     int
		expected bool
	}{
		{
			name:     "Port below lower bound",
			port:     1023,
			expected: false,
		},
		{
			name:     "Port at lower bound",
			port:     1024,
			expected: true,
		},
		{
			name:     "Port in valid range",
			port:     17000,
			expected: true,
		},
		{
			name:     "Port at upper bound",
			port:     65535,
			expected: true,
		},
		{
			name:     "Port above upper bound",
			port:     65536,
			expected: false,
		},
		{
			name:     "Zero port",
			port:     0,
			expected: false,
		},
		{
			name:     "Negative port",
			port:     -1,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.IsValidPort(tt.port)
			if result != tt.expected {
				t.Errorf("IsValidPort(%d) = %v; want %v", tt.port, result, tt.expected)
			}
		})
	}
}
