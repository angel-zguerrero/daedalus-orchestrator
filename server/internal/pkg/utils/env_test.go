package utils_test

import (
	"deadalus-orch/server/internal/pkg/utils"
	"deadalus-orch/shared/constants"
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

func TestENVValidator_ENV_ValidProduction(t *testing.T) {
	t.Setenv(constants.EnvVarEnvKey, string(constants.PRODUCTION))
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
