package utils_test

import (
	"deadalus-orch/server/internal/pkg/utils"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestENVValidator_ENV_Invalid(t *testing.T) {
	t.Setenv("ENV", "invalid")
	err := utils.ValidateEnvVar()
	require.Error(t, err)
	assert.EqualError(t, err, "invalid ENV value: invalid. Must be one of: development, staging, production")
}

func TestENVValidator_ENV_ValidProduction(t *testing.T) {
	t.Setenv("ENV", "production")
	err := utils.ValidateEnvVar()
	assert.NoError(t, err)
}

func TestENVValidator_ENV_ValidDevelopment(t *testing.T) {
	t.Setenv("ENV", "development")
	err := utils.ValidateEnvVar()
	assert.NoError(t, err)
}

func TestENVValidator_ENV_ValidStagin(t *testing.T) {
	t.Setenv("ENV", "staging")
	err := utils.ValidateEnvVar()
	assert.NoError(t, err)
}
