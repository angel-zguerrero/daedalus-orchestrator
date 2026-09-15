package bpmn

import (
	"fmt"
	"strconv"
	"strings"
)

func EvaluateCondition(condition string, state map[string]interface{}) (bool, error) {
	condition = strings.TrimSpace(condition)
	if condition == "" {
		return true, nil
	}

	// Clean wrapper like ${...}
	if strings.HasPrefix(condition, "${") && strings.HasSuffix(condition, "}") {
		condition = strings.TrimSpace(condition[2 : len(condition)-1])
	}

	if condition == "" || condition == "true" {
		return true, nil
	}
	if condition == "false" {
		return false, nil
	}

	// Handle equality "==" or "!="
	if strings.Contains(condition, "==") {
		parts := strings.SplitN(condition, "==", 2)
		leftKey := strings.TrimSpace(parts[0])
		rightVal := strings.TrimSpace(parts[1])

		val, ok := resolveStateVar(leftKey, state)
		if !ok {
			return false, nil
		}
		return formatVal(val) == cleanQuotes(rightVal), nil
	}

	if strings.Contains(condition, "!=") {
		parts := strings.SplitN(condition, "!=", 2)
		leftKey := strings.TrimSpace(parts[0])
		rightVal := strings.TrimSpace(parts[1])

		val, ok := resolveStateVar(leftKey, state)
		if !ok {
			return true, nil
		}
		return formatVal(val) != cleanQuotes(rightVal), nil
	}

	// Handle > or <
	if strings.Contains(condition, ">") {
		parts := strings.SplitN(condition, ">", 2)
		leftKey := strings.TrimSpace(parts[0])
		rightValStr := strings.TrimSpace(parts[1])

		val, ok := resolveStateVar(leftKey, state)
		if !ok {
			return false, nil
		}
		numVal, err1 := toFloat(val)
		targetNum, err2 := strconv.ParseFloat(cleanQuotes(rightValStr), 64)
		if err1 == nil && err2 == nil {
			return numVal > targetNum, nil
		}
	}

	if strings.Contains(condition, "<") {
		parts := strings.SplitN(condition, "<", 2)
		leftKey := strings.TrimSpace(parts[0])
		rightValStr := strings.TrimSpace(parts[1])

		val, ok := resolveStateVar(leftKey, state)
		if !ok {
			return false, nil
		}
		numVal, err1 := toFloat(val)
		targetNum, err2 := strconv.ParseFloat(cleanQuotes(rightValStr), 64)
		if err1 == nil && err2 == nil {
			return numVal < targetNum, nil
		}
	}

	// Fallback to boolean lookup in state map
	if val, ok := resolveStateVar(condition, state); ok {
		if boolVal, isBool := val.(bool); isBool {
			return boolVal, nil
		}
		if strVal, isStr := val.(string); isStr {
			return strVal == "true", nil
		}
	}

	return false, nil
}

func resolveStateVar(key string, state map[string]interface{}) (interface{}, bool) {
	if state == nil {
		return nil, false
	}
	val, ok := state[key]
	return val, ok
}

func cleanQuotes(s string) string {
	s = strings.TrimSpace(s)
	if (strings.HasPrefix(s, "\"") && strings.HasSuffix(s, "\"")) ||
		(strings.HasPrefix(s, "'") && strings.HasSuffix(s, "'")) {
		return s[1 : len(s)-1]
	}
	return s
}

func formatVal(v interface{}) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}

func toFloat(v interface{}) (float64, error) {
	switch num := v.(type) {
	case float64:
		return num, nil
	case float32:
		return float64(num), nil
	case int:
		return float64(num), nil
	case int64:
		return float64(num), nil
	case int32:
		return float64(num), nil
	case string:
		return strconv.ParseFloat(num, 64)
	default:
		return 0, fmt.Errorf("not a number")
	}
}
