package bpmn

import (
	"fmt"
	"strings"

	"github.com/expr-lang/expr"
)

// EvaluateCondition evaluates a BPMN condition expression against state variables using expr library.
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

	if state == nil {
		state = make(map[string]interface{})
	}

	program, err := expr.Compile(condition, expr.Env(state), expr.AsBool())
	if err != nil {
		return false, fmt.Errorf("condition syntax error: %w", err)
	}

	output, err := expr.Run(program, state)
	if err != nil {
		return false, fmt.Errorf("condition execution error: %w", err)
	}

	result, ok := output.(bool)
	if !ok {
		return false, fmt.Errorf("condition output is not boolean")
	}

	return result, nil
}
