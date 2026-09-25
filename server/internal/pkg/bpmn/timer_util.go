package bpmn

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var durationRegex = regexp.MustCompile(`(?i)^\s*(\d+(?:\.\d+)?)\s*([a-z]+)?\s*$`)

// ParseHumanDuration parses human readable duration strings like "5m", "5 minutes", "2h", "1 day", "30s", etc.
func ParseHumanDuration(input string) (time.Duration, error) {
	s := strings.TrimSpace(input)
	if s == "" {
		return 0, fmt.Errorf("empty duration string")
	}

	// First try standard Go duration (e.g., "5m", "2h30m", "100s")
	if d, err := time.ParseDuration(s); err == nil {
		return d, nil
	}

	// Try ISO 8601 duration format (e.g., "PT5M", "PT1H30M", "P1D")
	if strings.HasPrefix(strings.ToUpper(s), "P") {
		if d, err := parseISO8601Duration(s); err == nil {
			return d, nil
		}
	}

	// Match number + unit (e.g., "5 minutes", "10 min", "2 hours", "3 days", "1 week")
	matches := durationRegex.FindStringSubmatch(s)
	if len(matches) < 2 {
		return 0, fmt.Errorf("invalid duration format: %s", s)
	}

	val, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return 0, fmt.Errorf("invalid duration number: %s", matches[1])
	}

	unit := ""
	if len(matches) >= 3 {
		unit = strings.ToLower(strings.TrimSpace(matches[2]))
	}

	switch unit {
	case "", "s", "sec", "secs", "second", "seconds":
		return time.Duration(val * float64(time.Second)), nil
	case "m", "min", "mins", "minute", "minutes":
		return time.Duration(val * float64(time.Minute)), nil
	case "h", "hr", "hrs", "hour", "hours":
		return time.Duration(val * float64(time.Hour)), nil
	case "d", "day", "days":
		return time.Duration(val * 24 * float64(time.Hour)), nil
	case "w", "wk", "wks", "week", "weeks":
		return time.Duration(val * 7 * 24 * float64(time.Hour)), nil
	case "ms", "millisecond", "milliseconds":
		return time.Duration(val * float64(time.Millisecond)), nil
	default:
		return 0, fmt.Errorf("unknown duration unit: %s", unit)
	}
}

// parseISO8601Duration handles basic ISO 8601 strings like PT5M, PT1H, P1D
func parseISO8601Duration(s string) (time.Duration, error) {
	s = strings.ToUpper(strings.TrimSpace(s))
	if !strings.HasPrefix(s, "P") {
		return 0, fmt.Errorf("invalid ISO 8601 duration")
	}

	var total time.Duration
	s = s[1:] // strip 'P'

	isTime := false
	numBuf := ""

	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == 'T' {
			isTime = true
			continue
		}
		if c >= '0' && c <= '9' || c == '.' {
			numBuf += string(c)
			continue
		}

		if numBuf == "" {
			continue
		}
		val, err := strconv.ParseFloat(numBuf, 64)
		if err != nil {
			return 0, err
		}
		numBuf = ""

		switch c {
		case 'Y':
			total += time.Duration(val * 365 * 24 * float64(time.Hour))
		case 'M':
			if isTime {
				total += time.Duration(val * float64(time.Minute))
			} else {
				total += time.Duration(val * 30 * 24 * float64(time.Hour))
			}
		case 'W':
			total += time.Duration(val * 7 * 24 * float64(time.Hour))
		case 'D':
			total += time.Duration(val * 24 * float64(time.Hour))
		case 'H':
			total += time.Duration(val * float64(time.Hour))
		case 'S':
			total += time.Duration(val * float64(time.Second))
		}
	}
	return total, nil
}

// ParseDateTime parses timestamp strings formatted as ISO 8601 / RFC3339 or datetime-local formats.
func ParseDateTime(input string) (time.Time, error) {
	s := strings.TrimSpace(input)
	if s == "" {
		return time.Time{}, fmt.Errorf("empty date string")
	}

	layouts := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05-07:00",
		"2006-01-02T15:04:05.000Z07:00",
		"2006-01-02T15:04:05",
		"2006-01-02T15:04",
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
	}

	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("failed to parse date string: %s", s)
}
