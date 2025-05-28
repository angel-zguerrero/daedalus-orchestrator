package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHasDuplicates(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		expected bool
	}{
		{"has duplicates", []string{"a", "b", "a"}, true},
		{"no duplicates", []string{"a", "b", "c"}, false},
		{"empty slice", []string{}, false},
		{"single element", []string{"a"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, hasDuplicates(tt.slice))
		})
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		target   string
		expected bool
	}{
		{"target exists", []string{"a", "b", "c"}, "b", true},
		{"target does not exist", []string{"a", "b", "c"}, "d", false},
		{"empty slice", []string{}, "a", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, contains(tt.slice, tt.target))
		})
	}
}
