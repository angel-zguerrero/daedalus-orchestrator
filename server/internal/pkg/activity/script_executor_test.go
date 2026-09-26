package activity

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScriptExecutor_Success_PrimitiveAndObjectReturn(t *testing.T) {
	exec := &ScriptExecutor{}

	// 1. Numeric calculation with workflow input variables
	out, err := exec.Execute(context.Background(), map[string]interface{}{
		"scriptFormat":   "javascript",
		"resultVariable": "totalConIva",
		"precio":         100,
		"cantidad":       3,
		"script": `
			const subtotal = precio * cantidad;
			const iva = subtotal * 0.21;
			return subtotal + iva;
		`,
	})
	require.NoError(t, err)
	assert.Equal(t, "SUCCESS", out["status"])
	assert.InDelta(t, 363.0, out["totalConIva"], 0.001)

	// 2. Object return mapped to resultVariable
	outObj, err := exec.Execute(context.Background(), map[string]interface{}{
		"scriptFormat":   "JavaScript",
		"resultVariable": "evaluacionCredito",
		"nombre":         "Angel",
		"ingresos":       5000,
		"deuda":          800,
		"script": `
			const ratio = deuda / ingresos;
			const aprobado = ratio < 0.35;
			return {
				cliente: nombre.toUpperCase(),
				aprobado: aprobado,
				score: aprobado ? 95 : 40
			};
		`,
	})
	require.NoError(t, err)
	mapped, ok := outObj["evaluacionCredito"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "ANGEL", mapped["cliente"])
	assert.Equal(t, true, mapped["aprobado"])
	assert.Equal(t, int64(95), mapped["score"])
}

func TestScriptExecutor_MandatoryReturnValidation(t *testing.T) {
	exec := &ScriptExecutor{}

	// Missing return statement
	_, err := exec.Execute(context.Background(), map[string]interface{}{
		"scriptFormat":   "javascript",
		"resultVariable": "outVar",
		"script": `
			const x = 10 + 20;
			// return is only in a comment: return x;
		`,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "return")

	// Return statement returning undefined
	_, err = exec.Execute(context.Background(), map[string]interface{}{
		"scriptFormat":   "javascript",
		"resultVariable": "outVar",
		"script": `
			let u = undefined;
			return u;
		`,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "did not return a value")
}

func TestScriptExecutor_UnsupportedFormatAndMissingResultVariable(t *testing.T) {
	exec := &ScriptExecutor{}

	// Unsupported script format (e.g. python)
	_, err := exec.Execute(context.Background(), map[string]interface{}{
		"scriptFormat":   "python",
		"resultVariable": "outVar",
		"script":         "return 42",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported script format")

	// Missing resultVariable
	_, err = exec.Execute(context.Background(), map[string]interface{}{
		"scriptFormat": "javascript",
		"script":       "return 42;",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "resultVariable")
}

func TestScriptExecutor_SandboxIsolationAndTimeout(t *testing.T) {
	exec := &ScriptExecutor{}

	// Verify no network/OS globals exist ("blind, deaf, and mute")
	out, err := exec.Execute(context.Background(), map[string]interface{}{
		"scriptFormat":   "javascript",
		"resultVariable": "isolationCheck",
		"script": `
			return {
				hasFetch: typeof fetch !== 'undefined',
				hasXMLHttpRequest: typeof XMLHttpRequest !== 'undefined',
				hasRequire: typeof require !== 'undefined',
				hasProcess: typeof process !== 'undefined'
			};
		`,
	})
	require.NoError(t, err)
	check := out["isolationCheck"].(map[string]interface{})
	assert.Equal(t, false, check["hasFetch"])
	assert.Equal(t, false, check["hasXMLHttpRequest"])
	assert.Equal(t, false, check["hasRequire"])
	assert.Equal(t, false, check["hasProcess"])

	// Verify infinite loop is interrupted by context timeout
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err = exec.Execute(ctx, map[string]interface{}{
		"scriptFormat":   "javascript",
		"resultVariable": "loopOut",
		"script": `
			while (true) {}
			return 1;
		`,
	})
	require.Error(t, err)
}
