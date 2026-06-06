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

	assert.Equal(t, "", GlobalConfiguration.DefaultRootUser)
	assert.Equal(t, "", GlobalConfiguration.DefaultRootPassword)
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

	assert.Equal(t, "", GlobalConfiguration.DefaultRootUser)
	assert.Equal(t, "", GlobalConfiguration.DefaultRootPassword)
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
	assert.Equal(t, "", GlobalConfiguration.DefaultRootUser)
	assert.Equal(t, "", GlobalConfiguration.DefaultRootPassword)
}

func TestValidateClusterBasePort_PortTooLow(t *testing.T) {
	cfg := &Config{
		ClusterBasePort: constants.MinSafePort - 1,
		Env:             string(constants.DEVELOPMENT),
		MaxShards:      10,
	}

	expected := fmt.Sprintf("❌ ClusterBasePort (%d) must be between %d and %d",
		cfg.ClusterBasePort, constants.MinSafePort, constants.MaxPort)

	assert.PanicsWithValue(t, expected, func() { validateClusterBasePort(cfg) })
}

// Tests for gRPC Server Configuration
func TestLoadDefault_GrpcServer_Defaults(t *testing.T) {
	os.Clearenv() // Clear environment variables to ensure defaults are tested
	// Reset flags to their default values if they were changed by other tests.
	// This is tricky because flags are global. A better approach might be to run flag-dependent tests in separate processes
	// or use a library that allows temporary flag set/reset. For now, we assume other tests clean up or don't interfere.
	*ConfigFilePathFlag = "" // Ensure no config file is used

	err := LoadDefaultConfiguration()
	require.NoError(t, err)

	assert.Equal(t, "0.0.0.0", GlobalConfiguration.GrpcServerListenAddrHost, "Default gRPC host should be 0.0.0.0")
	assert.Equal(t, 4000, GlobalConfiguration.GrpcServerListenAddrPort, "Default gRPC port should be 4000")
}

func TestLoadDefault_GrpcServer_ConfigFile(t *testing.T) {
	os.Clearenv()
	content := `
grpc_server_listen_addr_host=127.0.0.1
grpc_server_listen_addr_port=50051
`
	path := writeTempFile(t, content)
	*ConfigFilePathFlag = path // Set config file path directly for the test

	err := LoadDefaultConfiguration()
	require.NoError(t, err)

	assert.Equal(t, "127.0.0.1", GlobalConfiguration.GrpcServerListenAddrHost)
	assert.Equal(t, 50051, GlobalConfiguration.GrpcServerListenAddrPort)
	*ConfigFilePathFlag = "" // Reset for other tests
}

func TestLoadDefault_GrpcServer_EnvVars(t *testing.T) {
	os.Clearenv()
	setEnv(t, constants.EnvVarGrpcServerListenAddrHost, "192.168.1.100")
	setEnv(t, constants.EnvVarGrpcServerListenAddrPort, "50052")
	*ConfigFilePathFlag = ""

	err := LoadDefaultConfiguration()
	require.NoError(t, err)

	assert.Equal(t, "192.168.1.100", GlobalConfiguration.GrpcServerListenAddrHost)
	assert.Equal(t, 50052, GlobalConfiguration.GrpcServerListenAddrPort)
}

func TestLoadDefault_GrpcServer_Flags(t *testing.T) {
	os.Clearenv()
	// Simulate setting flags. Note: Direct manipulation of flag variables is the standard way to "set" flags for testing in Go.
	originalHostFlag := *GrpcServerListenAddrHostFlag
	originalPortFlag := *GrpcServerListenAddrPortFlag
	defer func() { // Reset flags after test
		*GrpcServerListenAddrHostFlag = originalHostFlag
		*GrpcServerListenAddrPortFlag = originalPortFlag
	}()

	*GrpcServerListenAddrHostFlag = "my.grpc.server"
	*GrpcServerListenAddrPortFlag = 50053
	*ConfigFilePathFlag = ""

	err := LoadDefaultConfiguration() // This will call flag.Parse() if it hasn't been called
	require.NoError(t, err)

	assert.Equal(t, "my.grpc.server", GlobalConfiguration.GrpcServerListenAddrHost)
	assert.Equal(t, 50053, GlobalConfiguration.GrpcServerListenAddrPort)
}

func TestLoadDefault_GrpcServer_Precedence_FlagOverEnvOverFile(t *testing.T) {
	os.Clearenv()

	// 1. Config File values
	content := `
grpc_server_listen_addr_host=config.host.com
grpc_server_listen_addr_port=11111
`
	path := writeTempFile(t, content)
	*ConfigFilePathFlag = path

	// 2. Environment Variable values (should override config file)
	setEnv(t, constants.EnvVarGrpcServerListenAddrHost, "env.host.com")
	setEnv(t, constants.EnvVarGrpcServerListenAddrPort, "22222")

	// 3. Flag values (should override env vars and config file)
	originalHostFlag := *GrpcServerListenAddrHostFlag
	originalPortFlag := *GrpcServerListenAddrPortFlag
	defer func() {
		*GrpcServerListenAddrHostFlag = originalHostFlag
		*GrpcServerListenAddrPortFlag = originalPortFlag
		*ConfigFilePathFlag = "" // Reset config file path
	}()
	*GrpcServerListenAddrHostFlag = "flag.host.com"
	*GrpcServerListenAddrPortFlag = 33333

	err := LoadDefaultConfiguration()
	require.NoError(t, err)

	assert.Equal(t, "flag.host.com", GlobalConfiguration.GrpcServerListenAddrHost, "Flag host should take precedence")
	assert.Equal(t, 33333, GlobalConfiguration.GrpcServerListenAddrPort, "Flag port should take precedence")
}

func TestLoadDefault_GrpcServer_Precedence_EnvOverFile(t *testing.T) {
	os.Clearenv()
	// 1. Config File values
	content := `
grpc_server_listen_addr_host=config.host.com
grpc_server_listen_addr_port=11111
`
	path := writeTempFile(t, content)
	*ConfigFilePathFlag = path

	// 2. Environment Variable values (should override config file)
	setEnv(t, constants.EnvVarGrpcServerListenAddrHost, "env.host.com")
	setEnv(t, constants.EnvVarGrpcServerListenAddrPort, "22222")

	// Ensure flags are not set or are default (empty for string, 0 for int for these specific flags)
	originalHostFlag := *GrpcServerListenAddrHostFlag
	originalPortFlag := *GrpcServerListenAddrPortFlag
	defer func() {
		*GrpcServerListenAddrHostFlag = originalHostFlag
		*GrpcServerListenAddrPortFlag = originalPortFlag
		*ConfigFilePathFlag = ""
	}()
	*GrpcServerListenAddrHostFlag = "" // Default/unset state for string flag
	*GrpcServerListenAddrPortFlag = 0  // Default/unset state for int flag

	err := LoadDefaultConfiguration()
	require.NoError(t, err)

	assert.Equal(t, "env.host.com", GlobalConfiguration.GrpcServerListenAddrHost, "Env host should take precedence over file")
	assert.Equal(t, 22222, GlobalConfiguration.GrpcServerListenAddrPort, "Env port should take precedence over file")
}

func TestLoadDefault_GrpcServer_InvalidPort_Env(t *testing.T) {
	os.Clearenv()
	setEnv(t, constants.EnvVarGrpcServerListenAddrPort, "not-a-port")
	*ConfigFilePathFlag = ""

	err := LoadDefaultConfiguration()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "error parsing GRPC_SERVER_LISTEN_ADDR_PORT environment variable")
}

func TestLoadConfigFromPath_GrpcServerConfig(t *testing.T) {
	os.Clearenv()
	content := `
grpc_server_listen_addr_host=file.grpc.local
grpc_server_listen_addr_port=54321
unknown_grpc_key=some_value
`
	path := writeTempFile(t, content)
	cfg, err := LoadConfigFromPath(path)
	require.NoError(t, err)
	assert.Equal(t, "file.grpc.local", cfg.GrpcServerListenAddrHost)
	assert.Equal(t, 54321, cfg.GrpcServerListenAddrPort)
}

func TestLoadConfigFromPath_InvalidGrpcPortValue(t *testing.T) {
	os.Clearenv()
	path := writeTempFile(t, "grpc_server_listen_addr_port=abc")
	_, err := LoadConfigFromPath(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error parsing grpc_server_listen_addr_port")
}

func TestValidateClusterBasePort_PortTooHigh(t *testing.T) {
	cfg := &Config{
		ClusterBasePort: constants.MaxPort + 1,
		Env:             string(constants.PRODUCTION),
		MaxShards:      10,
	}

	expected := fmt.Sprintf("❌ ClusterBasePort (%d) must be between %d and %d",
		cfg.ClusterBasePort, constants.MinSafePort, constants.MaxPort)

	assert.PanicsWithValue(t, expected, func() { validateClusterBasePort(cfg) })
}

func TestValidateClusterBasePort_ValidInProduction(t *testing.T) {
	cfg := &Config{
		ClusterBasePort: 17000,
		Env:             string(constants.PRODUCTION),
		MaxShards:      100,
	}

	assert.NotPanics(t, func() { validateClusterBasePort(cfg) })
}

func TestValidateClusterBasePort_ExceedsMaxInProduction(t *testing.T) {
	cfg := &Config{
		ClusterBasePort: constants.MaxPort - 10,
		Env:             string(constants.PRODUCTION),
		MaxShards:      20,
	}

	expected := fmt.Sprintf("❌ ClusterBasePort (%d) with max shards (%d) exceeds maximum allowed port %d. "+
		"Please adjust the ClusterBasePort or reduce the number of shards.",
		cfg.ClusterBasePort, cfg.MaxShards, constants.MaxPort)

	assert.PanicsWithValue(t, expected, func() { validateClusterBasePort(cfg) })
}

func TestValidateClusterBasePort_ValidInNonProduction(t *testing.T) {
	cfg := &Config{
		ClusterBasePort: 17000,
		Env:             string(constants.DEVELOPMENT),
		MaxShards:      10,
	}

	assert.NotPanics(t, func() { validateClusterBasePort(cfg) })
}

func TestValidateClusterBasePort_ExceedsMaxInNonProduction(t *testing.T) {
	cfg := &Config{
		ClusterBasePort: constants.MaxPort - 10,
		Env:             string(constants.DEVELOPMENT),
		MaxShards:      50,
	}

	expected := fmt.Sprintf("❌ ClusterBasePort (%d) with max shards (%d) exceeds maximum allowed port %d. "+
		"Please adjust the ClusterBasePort or reduce the number of shards.",
		cfg.ClusterBasePort, cfg.MaxShards, constants.MaxPort)

	assert.PanicsWithValue(t, expected, func() { validateClusterBasePort(cfg) })
}
