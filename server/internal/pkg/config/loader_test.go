package config_test

import (
	"deadalus-orch/server/internal/pkg/config"
	"deadalus-orch/shared/constants"
	"errors"
	"flag" // Added
	"os"
	"path/filepath"
	"strings" // Added
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

// Helper function to manage flag.CommandLine
func withTestFlagSet(t *testing.T, args []string, testFunc func()) {
	originalCommandLine := flag.CommandLine
	defer func() {
		flag.CommandLine = originalCommandLine
		// Reset a few specific flags that might be globally registered by the config package
		// This is a bit of a hack to clean up, ideally flags are not global or are instance members
		config.HelpFlag = flag.Bool("help", false, "Show help message and exit.")
		config.ConfigFilePathFlag = flag.String("config", "", "Path to the application configuration file.")
		config.RoleFlag = flag.String("role", "", "Comma-separated list of roles for this node")
		config.InitialMembersFlag = flag.String("initial-members", "", "Cluster initial members")
		config.SelfMemberAddrFlag = flag.String("self-member-addr", "", "Nodehost address")
		config.JoinFlag = flag.Bool("join", false, "Joining a new node")
		config.ReplicaIDFlag = flag.Uint64("replica", 0, "Nodehost replica ID")
		config.ConnectorPortFlag = flag.Int("connector-port", 0, "Connector port")

	}()

	// Create a new FlagSet
	// The name of the FlagSet can be anything, "test" is common
	// flag.ContinueOnError is used so we can check for errors ourselves
	// instead of the test exiting on flag parse error.
	testFlags := flag.NewFlagSet("test", flag.ContinueOnError)

	// Redefine application flags on this new FlagSet
	// Important: These are local variables in the config package, so we need to assign them
	// to the flag set that config.LoadDefaultConfiguration will use.
	// We also need to use the *original* pointers that the config package uses.
	config.HelpFlag = testFlags.Bool("help", false, "Show help message and exit.")
	config.ConfigFilePathFlag = testFlags.String("config", "", "Path to the application configuration file.")
	config.RoleFlag = testFlags.String("role", "", "Comma-separated list of roles for this node")
	config.InitialMembersFlag = testFlags.String("initial-members", "", "Cluster initial members")
	config.SelfMemberAddrFlag = testFlags.String("self-member-addr", "", "Nodehost address")
	config.JoinFlag = testFlags.Bool("join", false, "Joining a new node")
	config.ReplicaIDFlag = testFlags.Uint64("replica", 0, "Nodehost replica ID")
	config.ConnectorPortFlag = testFlags.Int("connector-port", 0, "Connector port")

	// Assign the new FlagSet to flag.CommandLine
	flag.CommandLine = testFlags

	// Parse the arguments.
	// Note: flag.Parse() is called inside LoadDefaultConfiguration,
	// so we just need to ensure the arguments are set up for it.
	// os.Args will be used by flag.Parse() if flag.CommandLine.Parse() is not called with specific args.
	// We simulate command line args by temporarily changing os.Args
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()
	os.Args = append([]string{"cmd"}, args...) // "cmd" is typically the program name

	testFunc()
}

func TestLoadDefault_FlagOverridesEnvAndFile_ConnectorPort(t *testing.T) {
	// 1. Create a temp config file
	configFileContent := "connector_port=1000"
	configFilePath := writeTempFile(t, configFileContent)

	// 2. Set the CONNECTOR_PORT env var
	setEnv(t, constants.EnvVarConnectorPort, "2000")

	// 3. Set the --connector-port flag
	// This also sets the config file path via flag for this test setup
	withTestFlagSet(t, []string{"--connector-port", "3000", "--config", configFilePath}, func() {
		err := config.LoadDefaultConfiguration()
		require.NoError(t, err)
		assert.Equal(t, 3000, config.GlobalConfiguration.ConnectorPort)
	})
}

func TestLoadDefault_AppliesDefaults_WhenNotJoiningAndFieldsEmpty(t *testing.T) {
	// Ensure flags and ENV vars are unset or default
	// Unset relevant ENV vars
	t.Setenv(constants.EnvVarJoin, "false") // or unset
	t.Setenv(constants.EnvVarSelfMemberAddr, "")
	t.Setenv(constants.EnvVarInitialMembers, "")
	t.Setenv(constants.EnvVarReplicaId, "0") // or unset

	// Use an empty config file or no config file
	emptyConfigPath := writeTempFile(t, "")

	withTestFlagSet(t, []string{"--config", emptyConfigPath, "--join=false", "--replica=0"}, func() {
		err := config.LoadDefaultConfiguration()
		require.NoError(t, err)
		assert.Equal(t, "127.0.0.1:7001", config.GlobalConfiguration.SelfMemberAddr)
		assert.Equal(t, uint64(1), config.GlobalConfiguration.ReplicaID)
		assert.Equal(t, "127.0.0.1:7001", config.GlobalConfiguration.InitialMembers)
		assert.False(t, config.GlobalConfiguration.Join)
	})
}

func TestLoadDefault_EnvVarReplicaID_InvalidFormat(t *testing.T) {
	setEnv(t, constants.EnvVarReplicaId, "abc")
	withTestFlagSet(t, []string{}, func() {
		err := config.LoadDefaultConfiguration()
		assert.Error(t, err)
	})
}

func TestLoadDefault_EnvVarJoin_InvalidFormat(t *testing.T) {
	setEnv(t, constants.EnvVarJoin, "notabool")
	withTestFlagSet(t, []string{}, func() {
		err := config.LoadDefaultConfiguration()
		assert.Error(t, err)
	})
}

func TestLoadDefault_EnvVarConnectorPort_InvalidFormat(t *testing.T) {
	setEnv(t, constants.EnvVarConnectorPort, "abc")
	withTestFlagSet(t, []string{}, func() {
		err := config.LoadDefaultConfiguration()
		assert.Error(t, err)
	})
}

func TestLoadDefault_EnvVarTTLInternalError_InvalidFormat(t *testing.T) {
	setEnv(t, constants.EnvVarTTLInternalError, "abc")
	withTestFlagSet(t, []string{}, func() {
		err := config.LoadDefaultConfiguration()
		assert.Error(t, err)
	})
}

func TestLoadConfigFromPath_InvalidConnectorPortValue(t *testing.T) {
	path := writeTempFile(t, "connector_port=abc")
	_, err := config.LoadConfigFromPath(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error parsing connector_port")
}

func TestLoadConfigFromPath_InvalidReplicaIDValue(t *testing.T) {
	path := writeTempFile(t, "replica_id=abc")
	_, err := config.LoadConfigFromPath(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error parsing replica_id")
}

func TestLoadConfigFromPath_InvalidJoinValue(t *testing.T) {
	path := writeTempFile(t, "join=notabool")
	_, err := config.LoadConfigFromPath(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error parsing join")
}

func TestLoadConfigFromPath_InvalidTTLInternalErrorValue(t *testing.T) {
	path := writeTempFile(t, "ttl_internal_error=abc")
	_, err := config.LoadConfigFromPath(path)
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
	cfg, err := config.LoadConfigFromPath(path)
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
