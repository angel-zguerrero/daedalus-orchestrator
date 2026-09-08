package utils

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	models "deadalus-orch/shared/models"

	"github.com/robfig/cron/v3"
)

// ParseExtendedDuration parses durations like "5m", "1h", "10s", "2d", "500ms".
func ParseExtendedDuration(dStr string) (time.Duration, error) {
	dStr = strings.TrimSpace(dStr)
	if dStr == "" {
		return 0, fmt.Errorf("duration string is empty")
	}

	if strings.HasSuffix(dStr, "d") {
		daysStr := strings.TrimSuffix(dStr, "d")
		days, err := strconv.ParseFloat(daysStr, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid days duration: %s", dStr)
		}
		return time.Duration(days * 24 * float64(time.Hour)), nil
	}

	dur, err := time.ParseDuration(dStr)
	if err != nil {
		return 0, fmt.Errorf("invalid duration format %s: %w", dStr, err)
	}
	return dur, nil
}

// CalculateNextRunAt calculates the next execution time based on job type and time parameters.
func CalculateNextRunAt(
	jobType models.ScheduledJobType,
	runAt *time.Time,
	runAfter string,
	every string,
	cronExpr string,
	baseTime time.Time,
) (time.Time, error) {
	baseTime = baseTime.UTC()

	switch jobType {
	case models.ScheduledJobOneOff:
		if runAt != nil && !runAt.IsZero() && runAfter != "" {
			return time.Time{}, fmt.Errorf("runAt and runAfter are mutually exclusive")
		}
		if runAt != nil && !runAt.IsZero() {
			return runAt.UTC(), nil
		}
		if runAfter != "" {
			dur, err := ParseExtendedDuration(runAfter)
			if err != nil {
				return time.Time{}, fmt.Errorf("failed to parse runAfter: %w", err)
			}
			return baseTime.Add(dur), nil
		}
		return baseTime, nil

	case models.ScheduledJobRecurring:
		if every != "" && cronExpr != "" {
			return time.Time{}, fmt.Errorf("every and cronExpression are mutually exclusive")
		}
		if every == "" && cronExpr == "" {
			return time.Time{}, fmt.Errorf("either every or cronExpression must be provided for recurring job")
		}

		if every != "" {
			dur, err := ParseExtendedDuration(every)
			if err != nil {
				return time.Time{}, fmt.Errorf("failed to parse every: %w", err)
			}
			if dur <= 0 {
				return time.Time{}, fmt.Errorf("every duration must be positive")
			}
			return baseTime.Add(dur), nil
		}

		if cronExpr != "" {
			parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
			sched, err := parser.Parse(cronExpr)
			if err != nil {
				return time.Time{}, fmt.Errorf("invalid cron expression %s: %w", cronExpr, err)
			}
			next := sched.Next(baseTime)
			if next.IsZero() {
				return time.Time{}, fmt.Errorf("could not calculate next run time for cron expression %s", cronExpr)
			}
			return next.UTC(), nil
		}

	default:
		return time.Time{}, fmt.Errorf("unsupported scheduled job type: %s", jobType)
	}

	return baseTime, nil
}
