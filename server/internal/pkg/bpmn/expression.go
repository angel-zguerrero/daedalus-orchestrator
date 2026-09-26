package bpmn

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type EnvResolver interface {
	ResolveEnvVar(groupType string, scope string, groupRef string, varKey string) (string, error)
}

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

var (
	exprRegex         = regexp.MustCompile(`\$\{([^}]+)\}`)
	configSecretRegex = regexp.MustCompile(`^(config|secret)\.(global|tenant)\s*\[\s*["']?([^"'\]]+?)["']?\s*\]\s*\[\s*["']?([^"'\]]+?)["']?\s*\]$`)
)


func resolveSingleExpr(exprInner string, state map[string]interface{}, resolver EnvResolver) (interface{}, error) {
	exprInner = strings.TrimSpace(exprInner)
	if matches := configSecretRegex.FindStringSubmatch(exprInner); len(matches) == 5 {
		if resolver == nil {
			return nil, fmt.Errorf("environment resolver is required to resolve %q", exprInner)
		}
		val, err := resolver.ResolveEnvVar(matches[1], matches[2], matches[3], matches[4])
		if err != nil {
			return nil, err
		}
		return val, nil
	}

	val := ResolveVariablePath(exprInner, state)
	if val != nil {
		return EvaluateObjectResolvable(val, state, resolver)
	}
	return "", nil
}

// EvaluateStringResolvable resolves ${variableName} or ${config...}/${secret...} expressions in a string.
func EvaluateStringResolvable(input string, state map[string]interface{}, resolver EnvResolver) (interface{}, error) {
	trimmed := strings.TrimSpace(input)

	// Single standalone expression: preserve native object type
	if strings.HasPrefix(trimmed, "${") && strings.HasSuffix(trimmed, "}") && strings.Count(trimmed, "${") == 1 {
		exprInner := trimmed[2 : len(trimmed)-1]
		return resolveSingleExpr(exprInner, state, resolver)
	}

	// Mixed text or multiple expressions: interpolate to string
	var firstErr error
	var hasError bool

	result := exprRegex.ReplaceAllStringFunc(input, func(match string) string {
		if hasError {
			return match
		}
		exprInner := match[2 : len(match)-1]
		resolved, err := resolveSingleExpr(exprInner, state, resolver)
		if err != nil {
			hasError = true
			firstErr = err
			return match
		}
		if resolved == nil {
			return ""
		}
		return fmt.Sprintf("%v", resolved)
	})

	if hasError {
		return nil, firstErr
	}

	return result, nil
}

// EvaluateScriptStringResolvable resolves ${config...}/${secret...} and known state variables while preserving
// unknown ${localJsVar} expressions so JavaScript template literals work natively at runtime.
func EvaluateScriptStringResolvable(input string, state map[string]interface{}, resolver EnvResolver) (interface{}, error) {
	var firstErr error
	var hasError bool

	result := exprRegex.ReplaceAllStringFunc(input, func(match string) string {
		if hasError {
			return match
		}
		exprInner := strings.TrimSpace(match[2 : len(match)-1])
		if matches := configSecretRegex.FindStringSubmatch(exprInner); len(matches) == 5 {
			if resolver == nil {
				hasError = true
				firstErr = fmt.Errorf("environment resolver is required to resolve %q", exprInner)
				return match
			}
			val, err := resolver.ResolveEnvVar(matches[1], matches[2], matches[3], matches[4])
			if err != nil {
				hasError = true
				firstErr = err
				return match
			}
			return val
		}
		if state != nil {
			if val := ResolveVariablePath(exprInner, state); val != nil {
				return fmt.Sprintf("%v", val)
			}
		}
		// Preserve ${...} for local JS template literals
		return match
	})

	if hasError {
		return nil, firstErr
	}
	return result, nil
}

// EvaluateObjectResolvable recursively traverses data structures and resolves expressions using state & optional resolver.
func EvaluateObjectResolvable(val interface{}, state map[string]interface{}, resolver EnvResolver) (interface{}, error) {
	if val == nil {
		return nil, nil
	}

	switch v := val.(type) {
	case string:
		return EvaluateStringResolvable(v, state, resolver)

	case map[string]interface{}:
		evaluatedMap := make(map[string]interface{}, len(v))
		for k, child := range v {
			evaluatedKey := k
			if strings.Contains(k, "${") {
				evalKeyStr, err := EvaluateStringResolvable(k, state, resolver)
				if err != nil {
					return nil, err
				}
				if str, ok := evalKeyStr.(string); ok {
					evaluatedKey = str
				}
			}
			if (evaluatedKey == "script" || evaluatedKey == "scriptBody") && child != nil {
				if childStr, ok := child.(string); ok {
					childRes, err := EvaluateScriptStringResolvable(childStr, state, resolver)
					if err != nil {
						return nil, err
					}
					evaluatedMap[evaluatedKey] = childRes
					continue
				}
			}
			childRes, err := EvaluateObjectResolvable(child, state, resolver)
			if err != nil {
				return nil, err
			}
			evaluatedMap[evaluatedKey] = childRes
		}
		return evaluatedMap, nil

	case map[string]string:
		evaluatedMap := make(map[string]interface{}, len(v))
		for k, childStr := range v {
			evaluatedKey := k
			if strings.Contains(k, "${") {
				evalKeyStr, err := EvaluateStringResolvable(k, state, resolver)
				if err != nil {
					return nil, err
				}
				if str, ok := evalKeyStr.(string); ok {
					evaluatedKey = str
				}
			}
			if evaluatedKey == "script" || evaluatedKey == "scriptBody" {
				childRes, err := EvaluateScriptStringResolvable(childStr, state, resolver)
				if err != nil {
					return nil, err
				}
				evaluatedMap[evaluatedKey] = childRes
				continue
			}
			childRes, err := EvaluateObjectResolvable(childStr, state, resolver)
			if err != nil {
				return nil, err
			}
			evaluatedMap[evaluatedKey] = childRes
		}
		return evaluatedMap, nil

	case []interface{}:
		evaluatedSlice := make([]interface{}, len(v))
		for i, elem := range v {
			childRes, err := EvaluateObjectResolvable(elem, state, resolver)
			if err != nil {
				return nil, err
			}
			evaluatedSlice[i] = childRes
		}
		return evaluatedSlice, nil

	case []string:
		evaluatedSlice := make([]interface{}, len(v))
		for i, elem := range v {
			childRes, err := EvaluateObjectResolvable(elem, state, resolver)
			if err != nil {
				return nil, err
			}
			evaluatedSlice[i] = childRes
		}
		return evaluatedSlice, nil

	default:
		return val, nil
	}
}

// EvaluateString resolves ${variableName} expressions in a string.
func EvaluateString(input string, state map[string]interface{}) interface{} {
	res, _ := EvaluateStringResolvable(input, state, nil)
	return res
}

// EvaluateObject recursively traverses data structures (maps, slices, strings) and resolves expressions against state.
func EvaluateObject(val interface{}, state map[string]interface{}) interface{} {
	res, _ := EvaluateObjectResolvable(val, state, nil)
	return res
}

