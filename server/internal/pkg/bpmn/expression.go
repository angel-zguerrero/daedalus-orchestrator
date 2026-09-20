package bpmn

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var exprRegex = regexp.MustCompile(`\$\{([^}]+)\}`)

// ResolveVariablePath evaluates a dot-notated/indexed path against a state map.
// Supports nested maps (e.g., "user.address.city") and array indexes (e.g., "items[0].name").
func ResolveVariablePath(path string, state map[string]interface{}) interface{} {
	path = strings.TrimSpace(path)
	if path == "" || state == nil {
		return nil
	}

	tokens := parsePathTokens(path)
	var current interface{} = state

	for _, token := range tokens {
		if current == nil {
			return nil
		}

		switch curr := current.(type) {
		case map[string]interface{}:
			current = curr[token.key]
		case map[string]string:
			if val, ok := curr[token.key]; ok {
				current = val
			} else {
				return nil
			}
		default:
			return nil
		}

		if token.hasIndex && current != nil {
			idx := token.index
			switch arr := current.(type) {
			case []interface{}:
				if idx >= 0 && idx < len(arr) {
					current = arr[idx]
				} else {
					return nil
				}
			case []string:
				if idx >= 0 && idx < len(arr) {
					current = arr[idx]
				} else {
					return nil
				}
			case []map[string]interface{}:
				if idx >= 0 && idx < len(arr) {
					current = arr[idx]
				} else {
					return nil
				}
			default:
				return nil
			}
		}
	}

	return current
}

type pathToken struct {
	key      string
	hasIndex bool
	index    int
}

func parsePathTokens(path string) []pathToken {
	parts := strings.Split(path, ".")
	var tokens []pathToken

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		if idxStart := strings.Index(part, "["); idxStart != -1 && strings.HasSuffix(part, "]") {
			key := part[:idxStart]
			idxStr := part[idxStart+1 : len(part)-1]
			idx, err := strconv.Atoi(idxStr)
			if err == nil {
				tokens = append(tokens, pathToken{
					key:      key,
					hasIndex: true,
					index:    idx,
				})
				continue
			}
		}

		tokens = append(tokens, pathToken{
			key:      part,
			hasIndex: false,
		})
	}

	return tokens
}

// EvaluateString resolves ${variableName} expressions in a string.
// If the string is strictly a single expression `${expr}`, it returns the resolved object with its original type.
// Otherwise, it interpolates all `${expr}` occurrences into a formatted string.
func EvaluateString(input string, state map[string]interface{}) interface{} {
	trimmed := strings.TrimSpace(input)

	// Single standalone expression: preserve native object type
	if strings.HasPrefix(trimmed, "${") && strings.HasSuffix(trimmed, "}") && strings.Count(trimmed, "${") == 1 {
		exprInner := trimmed[2 : len(trimmed)-1]
		val := ResolveVariablePath(exprInner, state)
		if val != nil {
			return EvaluateObject(val, state)
		}
		return ""
	}

	// Mixed text or multiple expressions: interpolate to string
	result := exprRegex.ReplaceAllStringFunc(input, func(match string) string {
		exprInner := match[2 : len(match)-1]
		val := ResolveVariablePath(exprInner, state)
		if val == nil {
			return ""
		}
		return fmt.Sprintf("%v", val)
	})

	return result
}

// EvaluateObject recursively traverses data structures (maps, slices, strings) and resolves expressions against state.
func EvaluateObject(val interface{}, state map[string]interface{}) interface{} {
	if val == nil {
		return nil
	}

	switch v := val.(type) {
	case string:
		return EvaluateString(v, state)

	case map[string]interface{}:
		evaluatedMap := make(map[string]interface{}, len(v))
		for k, child := range v {
			evaluatedKey := k
			if strings.Contains(k, "${") {
				if evalKeyStr, ok := EvaluateString(k, state).(string); ok {
					evaluatedKey = evalKeyStr
				}
			}
			evaluatedMap[evaluatedKey] = EvaluateObject(child, state)
		}
		return evaluatedMap

	case map[string]string:
		evaluatedMap := make(map[string]interface{}, len(v))
		for k, childStr := range v {
			evaluatedKey := k
			if strings.Contains(k, "${") {
				if evalKeyStr, ok := EvaluateString(k, state).(string); ok {
					evaluatedKey = evalKeyStr
				}
			}
			evaluatedMap[evaluatedKey] = EvaluateObject(childStr, state)
		}
		return evaluatedMap

	case []interface{}:
		evaluatedSlice := make([]interface{}, len(v))
		for i, elem := range v {
			evaluatedSlice[i] = EvaluateObject(elem, state)
		}
		return evaluatedSlice

	case []string:
		evaluatedSlice := make([]interface{}, len(v))
		for i, elem := range v {
			evaluatedSlice[i] = EvaluateObject(elem, state)
		}
		return evaluatedSlice

	default:
		return val
	}
}
