package bpmn

import (
	"testing"
	"time"
)

func TestParseHumanDuration(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
		wantErr  bool
	}{
		{"5m", 5 * time.Minute, false},
		{"5 minutes", 5 * time.Minute, false},
		{"10 min", 10 * time.Minute, false},
		{"2h", 2 * time.Hour, false},
		{"2 hours", 2 * time.Hour, false},
		{"30s", 30 * time.Second, false},
		{"30 seconds", 30 * time.Second, false},
		{"1d", 24 * time.Hour, false},
		{"1 day", 24 * time.Hour, false},
		{"2 days", 48 * time.Hour, false},
		{"1w", 7 * 24 * time.Hour, false},
		{"1 week", 7 * 24 * time.Hour, false},
		{"500ms", 500 * time.Millisecond, false},
		{"PT5M", 5 * time.Minute, false},
		{"PT1H", 1 * time.Hour, false},
		{"invalid_str", 0, true},
	}

	for _, tt := range tests {
		got, err := ParseHumanDuration(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseHumanDuration(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && got != tt.expected {
			t.Errorf("ParseHumanDuration(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestParseDateTime(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
	}{
		{"2026-09-25T18:30:00-03:00", false},
		{"2026-09-25T21:30:00Z", false},
		{"2026-09-25T18:30:00", false},
		{"2026-09-25T18:30", false},
		{"invalid-date", true},
	}

	for _, tt := range tests {
		_, err := ParseDateTime(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseDateTime(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
		}
	}
}
