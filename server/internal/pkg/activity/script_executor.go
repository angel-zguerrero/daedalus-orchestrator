package activity

import (
	"context"
	"fmt"
	"strings"

	"deadalus-orch/server/internal/pkg/bpmn"

	"github.com/dop251/goja"
	"github.com/rs/zerolog/log"
)

// ScriptExecutor executes isolated JavaScript tasks using the pure-Go goja ECMAScript engine.
// The runtime has no access to network, filesystem, or OS environment ("blind, deaf, and mute").
type ScriptExecutor struct{}

func (e *ScriptExecutor) Execute(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	if input == nil {
		input = make(map[string]interface{})
	}

	scriptFormat := "javascript"
	if sf, ok := input["scriptFormat"].(string); ok && strings.TrimSpace(sf) != "" {
		scriptFormat = strings.TrimSpace(sf)
	}
	if !bpmn.IsSupportedScriptFormat(scriptFormat) {
		return nil, fmt.Errorf("unsupported script format %q: only 'javascript' is supported", scriptFormat)
	}

	scriptBody := ""
	if s, ok := input["script"].(string); ok && strings.TrimSpace(s) != "" {
		scriptBody = strings.TrimSpace(s)
	} else if sb, ok := input["scriptBody"].(string); ok && strings.TrimSpace(sb) != "" {
		scriptBody = strings.TrimSpace(sb)
	}
	if scriptBody == "" {
		return nil, fmt.Errorf("script body is required for JavaScript task execution")
	}

	if !bpmn.HasMandatoryReturnStatement(scriptBody) {
		return nil, fmt.Errorf("a 'return <value>;' statement is mandatory in the JavaScript script")
	}

	resultVariable := ""
	for _, key := range []string{"resultVariable", "camunda:resultVariable", "outputVariable"} {
		if rv, ok := input[key].(string); ok && strings.TrimSpace(rv) != "" {
			resultVariable = strings.TrimSpace(rv)
			break
		}
	}
	if resultVariable == "" {
		return nil, fmt.Errorf("resultVariable (output variable name) is required to map the script return value")
	}

	vm := goja.New()

	// Enforce context cancellation / timeout to prevent infinite loops
	done := make(chan struct{})
	defer close(done)
	if ctx != nil {
		go func() {
			select {
			case <-ctx.Done():
				vm.Interrupt("javascript execution cancelled or timed out")
			case <-done:
			}
		}()
	}

	// Inject workflow variables into isolated JS global scope
	reservedProps := map[string]bool{
		"script":                 true,
		"scriptBody":             true,
		"scriptFormat":           true,
		"resultVariable":         true,
		"camunda:resultVariable": true,
		"outputVariable":         true,
		"_templateCode":          true,
	}

	workflowVars := make(map[string]interface{})
	for k, v := range input {
		if reservedProps[k] {
			continue
		}
		workflowVars[k] = v
		_ = vm.Set(k, v)
	}
	_ = vm.Set("input", workflowVars)
	_ = vm.Set("variables", workflowVars)

	wrappedScript := fmt.Sprintf("(function() {\n%s\n})()", scriptBody)
	val, err := vm.RunString(wrappedScript)
	if err != nil {
		return nil, fmt.Errorf("javascript execution error: %w", err)
	}

	if val == nil || goja.IsUndefined(val) {
		return nil, fmt.Errorf("javascript execution did not return a value: a 'return <value>;' statement is mandatory")
	}

	exported := val.Export()

	log.Info().
		Str("resultVariable", resultVariable).
		Interface("returnValue", exported).
		Msg("📜 [SCRIPT TASK]: Successfully executed JavaScript task")

	return map[string]interface{}{
		"status":       "SUCCESS",
		"result":       exported,
		resultVariable: exported,
	}, nil
}
