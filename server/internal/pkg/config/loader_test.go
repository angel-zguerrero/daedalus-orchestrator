package config_test

import (
	"deadalus-orch/server/internal/pkg/config"
	"deadalus-orch/shared/constants"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func TestMain(m *testing.M) {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	code := m.Run()
	os.Exit(code)
}

func writeTempFile(t *testing.T, content string) string {
	t.Helper()
	tmp := t.TempDir()
	path := filepath.Join(tmp, "test.conf")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	return path
}

func setEnv(t *testing.T, key, value string) {
	t.Helper()
	t.Setenv(key, value)
}

func setFlagConfig(t *testing.T, path string) {
	t.Helper()
	setEnv(t, constants.EnvVarConfigPath, path)
}

func TestLoadDefault_ConfigFileAllKeys(t *testing.T) {
	content := `
db_name=my.db
default_root_user=admin
default_root_password=secret
`
	path := writeTempFile(t, content)
	setFlagConfig(t, path)

	err := config.LoadDefaultConfiguration()
	require.NoError(t, err)

	assert.Equal(t, "admin", config.GlobalConfiguration.DefaultRootUser)
	assert.Equal(t, "secret", config.GlobalConfiguration.DefaultRootPassword)
}

func TestLoadDefault_ConfigFileOverwriteWithEnv(t *testing.T) {
	content := `
db_name=my.db
default_root_user=admin
default_root_password=secret
`
	path := writeTempFile(t, content)
	setFlagConfig(t, path)

	setEnv(t, constants.EnvVarDefaultRootUser, "envUser")
	setEnv(t, constants.EnvVarDefaultRootPassword, "envPass")

	err := config.LoadDefaultConfiguration()
	require.NoError(t, err)

	assert.Equal(t, "envUser", config.GlobalConfiguration.DefaultRootUser)
	assert.Equal(t, "envPass", config.GlobalConfiguration.DefaultRootPassword)
}

func TestLoadDefault_ConfigFilePartialKeys_ENVFallback(t *testing.T) {
	content := `
db_name=my.db
`
	path := writeTempFile(t, content)
	setFlagConfig(t, path)

	setEnv(t, constants.EnvVarDefaultRootUser, "envUser")
	setEnv(t, constants.EnvVarDefaultRootPassword, "envPass")

	err := config.LoadDefaultConfiguration()
	require.NoError(t, err)

	assert.Equal(t, "envUser", config.GlobalConfiguration.DefaultRootUser)
	assert.Equal(t, "envPass", config.GlobalConfiguration.DefaultRootPassword)
}

func TestLoadDefault_InvalidPath(t *testing.T) {
	setFlagConfig(t, "/nonexistent/path.conf")

	err := config.LoadDefaultConfiguration()
	assert.Error(t, err)
}

func TestLoadDefault_NoConfigFile_ENVFallback(t *testing.T) {
	setFlagConfig(t, "")
	setEnv(t, constants.EnvVarDefaultRootUser, "envUser")
	setEnv(t, constants.EnvVarDefaultRootPassword, "envPass")

	err := config.LoadDefaultConfiguration()
	require.NoError(t, err)

	assert.Equal(t, "envUser", config.GlobalConfiguration.DefaultRootUser)
	assert.Equal(t, "envPass", config.GlobalConfiguration.DefaultRootPassword)
}

func TestLoadDefault_NoFile_NoEnv_DefaultFallback(t *testing.T) {
	setFlagConfig(t, "")

	err := config.LoadDefaultConfiguration()
	require.NoError(t, err)

	assert.Equal(t, "admin", config.GlobalConfiguration.DefaultRootUser)
	assert.Equal(t, "admin", config.GlobalConfiguration.DefaultRootPassword)
}

func TestLoadConfig_ValidAndInvalidLines(t *testing.T) {
	content := `
# Comment
db_name=my.db
invalidline
key_without_value=
=onlyvalue
valid_key = value
  spaced_key= spaced_value
`
	path := writeTempFile(t, content)

	_, err := config.LoadConfigFromPath(path)
	require.NoError(t, err)
}

func TestLoadDefault_ENVSelection(t *testing.T) {
	for _, env := range []string{string(constants.DEVELOPMENT), string(constants.STAGING), string(constants.PRODUCTION)} {
		t.Setenv(constants.EnvVarEnvKey, env)
		setFlagConfig(t, "")
		err := config.LoadDefaultConfiguration()
		require.NoError(t, err)
	}
}

func TestLoadDefault_ENV_DefaultDevelopment(t *testing.T) {
	t.Setenv(constants.EnvVarEnvKey, "")
	setFlagConfig(t, "")
	err := config.LoadDefaultConfiguration()
	require.NoError(t, err)
}

func TestLoadDefault_ENV_Staging_FileMissing(t *testing.T) {
	t.Setenv(constants.EnvVarEnvKey, string(constants.STAGING))
	setFlagConfig(t, "")
	err := config.LoadDefaultConfiguration()
	require.NoError(t, err)
}

func TestLoadDefault_ENV_Production_WithFile(t *testing.T) {
	t.Setenv(constants.EnvVarEnvKey, string(constants.PRODUCTION))
	file := writeTempFile(t, `db_name = my.db`)
	setFlagConfig(t, file)

	err := config.LoadDefaultConfiguration()
	require.NoError(t, err)
}

func TestLoadDefault_CustomPath_FileExists(t *testing.T) {
	file := writeTempFile(t, `db_name = custom.db`)
	setFlagConfig(t, file)

	err := config.LoadDefaultConfiguration()
	require.NoError(t, err)
}

func TestLoadDefault_CustomPath_FileMissing(t *testing.T) {
	setFlagConfig(t, "/tmp/does-not-exist.conf")

	err := config.LoadDefaultConfiguration()
	require.Error(t, err)
	assert.True(t, errors.Is(err, os.ErrNotExist))
}

func TestLoadDefault_ENVFallbacks(t *testing.T) {
	t.Setenv(constants.EnvVarDefaultRootUser, "root")
	t.Setenv(constants.EnvVarDefaultRootPassword, "rootpass")
	setFlagConfig(t, "")

	err := config.LoadDefaultConfiguration()
	require.NoError(t, err)
	assert.Equal(t, "root", config.GlobalConfiguration.DefaultRootUser)
	assert.Equal(t, "rootpass", config.GlobalConfiguration.DefaultRootPassword)
}

func TestLoadDefault_DefaultRootFallbacks(t *testing.T) {
	setFlagConfig(t, "")

	err := config.LoadDefaultConfiguration()
	require.NoError(t, err)
	assert.Equal(t, "admin", config.GlobalConfiguration.DefaultRootUser)
	assert.Equal(t, "admin", config.GlobalConfiguration.DefaultRootPassword)
}
