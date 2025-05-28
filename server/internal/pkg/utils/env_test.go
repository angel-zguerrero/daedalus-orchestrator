package utils_test

import (
	"deadalus-orch/server/internal/pkg/utils"
	"deadalus-orch/shared/constants"
	"os" // Added for os.Getenv
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestENVValidator_ENV_Invalid(t *testing.T) {
	t.Setenv(constants.EnvVarEnvKey, "invalid")
	err := utils.ValidateEnvVar()
	require.Error(t, err)
	assert.EqualError(t, err, "invalid ENV value: invalid. Must be one of: development, staging, production")
}

func TestENVValidator_OtelActived_Invalid(t *testing.T) {
	t.Setenv(constants.EnvVarOtelActived, "invalid")
	err := utils.ValidateEnvVar()
	require.Error(t, err)
	assert.EqualError(t, err, "invalid OtelActived value: invalid. Must be one of: true, false")
}

func TestENVValidator_ENV_ValidProduction(t *testing.T) {
	t.Setenv(constants.EnvVarEnvKey, string(constants.PRODUCTION))
	err := utils.ValidateEnvVar()
	assert.NoError(t, err)
}

func TestENVValidator_ENV_DefaultsToDevelopmentWhenEmpty(t *testing.T) {
	t.Setenv(constants.EnvVarEnvKey, "")
	// Also ensure OTEL_ACTIVE is valid or empty to not interfere
	t.Setenv(constants.EnvVarOtelActived, constants.OTEL_ACTIVE_TRUE)
	err := utils.ValidateEnvVar()
	assert.NoError(t, err)
	assert.Equal(t, string(constants.DEVELOPMENT), os.Getenv(constants.EnvVarEnvKey))
}

func TestENVValidator_OTEL_ACTIVE_DefaultsToTrueWhenEmpty(t *testing.T) {
	// Set a valid ENV var to isolate the test to OTEL_ACTIVE
	t.Setenv(constants.EnvVarEnvKey, string(constants.DEVELOPMENT))
	t.Setenv(constants.EnvVarOtelActived, "")
	err := utils.ValidateEnvVar()
	assert.NoError(t, err)
	assert.Equal(t, constants.OTEL_ACTIVE_TRUE, os.Getenv(constants.EnvVarOtelActived))
}

func TestENVValidator_OTEL_ACTIVE_ValidTrue(t *testing.T) {
	t.Setenv(constants.EnvVarEnvKey, string(constants.DEVELOPMENT))
	t.Setenv(constants.EnvVarOtelActived, constants.OTEL_ACTIVE_TRUE)
	err := utils.ValidateEnvVar()
	assert.NoError(t, err)
}

func TestENVValidator_OTEL_ACTIVE_ValidFalse(t *testing.T) {
	t.Setenv(constants.EnvVarEnvKey, string(constants.DEVELOPMENT))
	t.Setenv(constants.EnvVarOtelActived, constants.OTEL_ACTIVE_FALSE)
	err := utils.ValidateEnvVar()
	assert.NoError(t, err)
}

func TestENVValidator_ValidProductionAndOtelFalse(t *testing.T) {
	t.Setenv(constants.EnvVarEnvKey, string(constants.PRODUCTION))
	t.Setenv(constants.EnvVarOtelActived, constants.OTEL_ACTIVE_FALSE)
	err := utils.ValidateEnvVar()
	assert.NoError(t, err)
}

func TestENVValidator_ENV_ValidDevelopment(t *testing.T) {
	t.Setenv(constants.EnvVarEnvKey, string(constants.DEVELOPMENT))
	err := utils.ValidateEnvVar()
	assert.NoError(t, err)
}

func TestENVValidator_ENV_ValidStagin(t *testing.T) {
	t.Setenv(constants.EnvVarEnvKey, string(constants.STAGING))
	err := utils.ValidateEnvVar()
	assert.NoError(t, err)
}
