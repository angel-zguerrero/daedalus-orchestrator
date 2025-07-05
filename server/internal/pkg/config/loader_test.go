package config

import (
	"deadalus-orch/shared/constants"
	"errors" // Added
	"fmt"
	"os"
	"path/filepath" // Added
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

	err := LoadDefaultConfiguration()
	require.NoError(t, err)

	assert.Equal(t, "admin", GlobalConfiguration.DefaultRootUser)
	assert.Equal(t, "secret", GlobalConfiguration.DefaultRootPassword)
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

	err := LoadDefaultConfiguration()
	require.NoError(t, err)

	assert.Equal(t, "envUser", GlobalConfiguration.DefaultRootUser)
	assert.Equal(t, "envPass", GlobalConfiguration.DefaultRootPassword)
}

func TestLoadDefault_ConfigFilePartialKeys_ENVFallback(t *testing.T) {
	content := `
db_name=my.db
`
	path := writeTempFile(t, content)
	setFlagConfig(t, path)

	setEnv(t, constants.EnvVarDefaultRootUser, "envUser")
	setEnv(t, constants.EnvVarDefaultRootPassword, "envPass")

	err := LoadDefaultConfiguration()
	require.NoError(t, err)

	assert.Equal(t, "envUser", GlobalConfiguration.DefaultRootUser)
	assert.Equal(t, "envPass", GlobalConfiguration.DefaultRootPassword)
}

func TestLoadDefault_InvalidPath(t *testing.T) {
	setFlagConfig(t, "/nonexistent/path.conf")

	err := LoadDefaultConfiguration()
	assert.Error(t, err)
}

func TestLoadDefault_NoConfigFile_ENVFallback(t *testing.T) {
	setFlagConfig(t, "")
	setEnv(t, constants.EnvVarDefaultRootUser, "envUser")
	setEnv(t, constants.EnvVarDefaultRootPassword, "envPass")

	err := LoadDefaultConfiguration()
	require.NoError(t, err)

	assert.Equal(t, "envUser", GlobalConfiguration.DefaultRootUser)
	assert.Equal(t, "envPass", GlobalConfiguration.DefaultRootPassword)
}

func TestLoadDefault_NoFile_NoEnv_DefaultFallback(t *testing.T) {
	setFlagConfig(t, "")

	err := LoadDefaultConfiguration()
	require.NoError(t, err)

	assert.Equal(t, "admin", GlobalConfiguration.DefaultRootUser)
	assert.Equal(t, "admin", GlobalConfiguration.DefaultRootPassword)
}

func TestLoadConfigFromPath_InvalidConnectorPortValue(t *testing.T) {
	path := writeTempFile(t, "connector_port=abc")
	_, err := LoadConfigFromPath(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error parsing connector_port")
}

func TestLoadConfigFromPath_InvalidReplicaIDValue(t *testing.T) {
	path := writeTempFile(t, "replica_id=abc")
	_, err := LoadConfigFromPath(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error parsing replica_id")
}

func TestLoadConfigFromPath_InvalidJoinValue(t *testing.T) {
	path := writeTempFile(t, "join=notabool")
	_, err := LoadConfigFromPath(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error parsing join")
}

func TestLoadConfigFromPath_InvalidTTLInternalErrorValue(t *testing.T) {
	path := writeTempFile(t, "ttl_internal_error=abc")
	_, err := LoadConfigFromPath(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error parsing ttl_internal_error")
}

func TestLoadConfigFromPath_IgnoresUnknownKeys(t *testing.T) {
	content := `
unknown_key=some_value
# valid key to ensure parsing happens
connector_port=1234
`
	path := writeTempFile(t, content)
	cfg, err := LoadConfigFromPath(path)
	require.NoError(t, err)
	assert.Equal(t, 1234, cfg.ConnectorPort)
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

	_, err := LoadConfigFromPath(path)
	require.NoError(t, err)
}

func TestLoadDefault_ENVSelection(t *testing.T) {
	for _, env := range []string{string(constants.DEVELOPMENT), string(constants.STAGING), string(constants.PRODUCTION)} {
		t.Setenv(constants.EnvVarEnvKey, env)
		setFlagConfig(t, "")
		err := LoadDefaultConfiguration()
		require.NoError(t, err)
	}
}

func TestLoadDefault_ENV_DefaultDevelopment(t *testing.T) {
	t.Setenv(constants.EnvVarEnvKey, "")
	setFlagConfig(t, "")
	err := LoadDefaultConfiguration()
	require.NoError(t, err)
}

func TestLoadDefault_ENV_Staging_FileMissing(t *testing.T) {
	t.Setenv(constants.EnvVarEnvKey, string(constants.STAGING))
	setFlagConfig(t, "")
	err := LoadDefaultConfiguration()
	require.NoError(t, err)
}

func TestLoadDefault_ENV_Production_WithFile(t *testing.T) {
	t.Setenv(constants.EnvVarEnvKey, string(constants.PRODUCTION))
	file := writeTempFile(t, `db_name = my.db`)
	setFlagConfig(t, file)

	err := LoadDefaultConfiguration()
	require.NoError(t, err)
}

func TestLoadDefault_CustomPath_FileExists(t *testing.T) {
	file := writeTempFile(t, `db_name = custom.db`)
	setFlagConfig(t, file)

	err := LoadDefaultConfiguration()
	require.NoError(t, err)
}

func TestLoadDefault_CustomPath_FileMissing(t *testing.T) {
	setFlagConfig(t, "/tmp/does-not-exist.conf")

	err := LoadDefaultConfiguration()
	require.Error(t, err)
	assert.True(t, errors.Is(err, os.ErrNotExist))
}

func TestLoadDefault_ENVFallbacks(t *testing.T) {
	t.Setenv(constants.EnvVarDefaultRootUser, "root")
	t.Setenv(constants.EnvVarDefaultRootPassword, "rootpass")
	setFlagConfig(t, "")

	err := LoadDefaultConfiguration()
	require.NoError(t, err)
	assert.Equal(t, "root", GlobalConfiguration.DefaultRootUser)
	assert.Equal(t, "rootpass", GlobalConfiguration.DefaultRootPassword)
}

func TestLoadDefault_DefaultRootFallbacks(t *testing.T) {
	setFlagConfig(t, "")

	err := LoadDefaultConfiguration()
	require.NoError(t, err)
	assert.Equal(t, "admin", GlobalConfiguration.DefaultRootUser)
	assert.Equal(t, "admin", GlobalConfiguration.DefaultRootPassword)
}

func TestValidateClusterBasePort_PortTooLow(t *testing.T) {
	cfg := &Config{
		ClusterBasePort: constants.MinSafePort - 1,
		Env:             string(constants.DEVELOPMENT),
		MaxTenants:      10,
	}

	expected := fmt.Sprintf("❌ ClusterBasePort (%d) must be between %d and %d",
		cfg.ClusterBasePort, constants.MinSafePort, constants.MaxPort)

	assert.PanicsWithValue(t, expected, func() { validateClusterBasePort(cfg) })
}

func TestValidateClusterBasePort_PortTooHigh(t *testing.T) {
	cfg := &Config{
		ClusterBasePort: constants.MaxPort + 1,
		Env:             string(constants.PRODUCTION),
		MaxTenants:      10,
	}

	expected := fmt.Sprintf("❌ ClusterBasePort (%d) must be between %d and %d",
		cfg.ClusterBasePort, constants.MinSafePort, constants.MaxPort)

	assert.PanicsWithValue(t, expected, func() { validateClusterBasePort(cfg) })
}

func TestValidateClusterBasePort_ValidInProduction(t *testing.T) {
	cfg := &Config{
		ClusterBasePort: 5000,
		Env:             string(constants.PRODUCTION),
		MaxTenants:      100,
	}

	assert.NotPanics(t, func() { validateClusterBasePort(cfg) })
}

func TestValidateClusterBasePort_ExceedsMaxInProduction(t *testing.T) {
	cfg := &Config{
		ClusterBasePort: constants.MaxPort - 10,
		Env:             string(constants.PRODUCTION),
		MaxTenants:      20,
	}

	expected := fmt.Sprintf("❌ ClusterBasePort (%d) with max tenants (%d) exceeds maximum allowed port %d. "+
		"Please adjust the ClusterBasePort or reduce the number of tenants.",
		cfg.ClusterBasePort, cfg.MaxTenants, constants.MaxPort)

	assert.PanicsWithValue(t, expected, func() { validateClusterBasePort(cfg) })
}

func TestValidateClusterBasePort_ValidInNonProduction(t *testing.T) {
	cfg := &Config{
		ClusterBasePort: 5000,
		Env:             string(constants.DEVELOPMENT),
		MaxTenants:      10,
	}

	assert.NotPanics(t, func() { validateClusterBasePort(cfg) })
}

func TestValidateClusterBasePort_ExceedsMaxInNonProduction(t *testing.T) {
	cfg := &Config{
		ClusterBasePort: constants.MaxPort - 10,
		Env:             string(constants.DEVELOPMENT),
		MaxTenants:      50,
	}

	expected := fmt.Sprintf("❌ ClusterBasePort (%d) with max tenants (%d) exceeds maximum allowed port %d. "+
		"Please adjust the ClusterBasePort or reduce the number of tenants.",
		cfg.ClusterBasePort, cfg.MaxTenants, constants.MaxPort)

	assert.PanicsWithValue(t, expected, func() { validateClusterBasePort(cfg) })
}
