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

func TestLoadOrDefault_ConfigFileAllKeys(t *testing.T) {
	content := `
db_name=my.db
default_root_user=admin
default_root_password=secret
`
	path := writeTempFile(t, content)
	defer os.Remove(path)
	err := config.LoadOrDefault(path)
	if err != nil {
		t.Fatal(err)
	}

	if config.GlobalConfiguration.DefaultRootUser != "admin" {
		t.Errorf("expected default_root_user=admin")
	}
	if config.GlobalConfiguration.DefaultRootPassword != "secret" {
		t.Errorf("expected default_root_password=secret")
	}

}

func TestLoadOrDefault_ConfigFileOverwriteKeysWithEnvVars(t *testing.T) {
	content := `
db_name=my.db
default_root_user=admin
default_root_password=secret
port=50005
`

	setEnv(t, constants.EnvVarDefaultRootPassword, "envPass")
	setEnv(t, constants.EnvVarDefaultRootUser, "envUser")

	path := writeTempFile(t, content)
	defer os.Remove(path)
	err := config.LoadOrDefault(path)
	if err != nil {
		t.Fatal(err)
	}

	if config.GlobalConfiguration.DefaultRootUser != "envUser" {
		t.Errorf("expected DefaultRootUser=envUser, got %s", config.GlobalConfiguration.DefaultRootUser)
	}
	if config.GlobalConfiguration.DefaultRootPassword != "envPass" {
		t.Errorf("expected DefaultRootPassword=envPass, got %s", config.GlobalConfiguration.DefaultRootPassword)
	}

}

func TestLoadOrDefault_ConfigFilePartialKeys_ENVFallback(t *testing.T) {
	content := `
db_name=my.db
`
	path := writeTempFile(t, content)
	defer os.Remove(path)
	setEnv(t, constants.EnvVarDefaultRootUser, "envUser")
	setEnv(t, constants.EnvVarDefaultRootPassword, "envPass")

	err := config.LoadOrDefault(path)
	if err != nil {
		t.Fatal(err)
	}

	if config.GlobalConfiguration.DefaultRootUser != "envUser" {
		t.Errorf("expected default_root_user from env")
	}
	if config.GlobalConfiguration.DefaultRootPassword != "envPass" {
		t.Errorf("expected default_root_password from env")
	}
}

func TestLoadOrDefault_InvalidPath(t *testing.T) {

	err := config.LoadOrDefault("/nonexistent/path.conf")
	if err == nil {
		t.Errorf("expected error but get nil")
	}

}

func TestLoadOrDefault_NoConfigFile_ENVFallback(t *testing.T) {
	setEnv(t, constants.EnvVarDefaultRootUser, "envUser")
	setEnv(t, constants.EnvVarDefaultRootPassword, "envPass")

	err := config.LoadOrDefault("")
	if err != nil {
		t.Fatal(err)
	}

	if config.GlobalConfiguration.DefaultRootUser != "envUser" {
		t.Errorf("expected default_root_user from env")
	}
	if config.GlobalConfiguration.DefaultRootPassword != "envPass" {
		t.Errorf("expected default_root_password from env")
	}
}

func TestLoadOrDefault_NoFile_NoEnv_DefaultFallback(t *testing.T) {
	err := config.LoadOrDefault("")
	if err != nil {
		t.Fatal(err)
	}

	if config.GlobalConfiguration.DefaultRootUser != "admin" {
		t.Errorf("expected fallback default_root_user=admin")
	}

	if config.GlobalConfiguration.DefaultRootPassword != "admin" {
		t.Errorf("expected fallback default_root_password=admin")
	}
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
	defer os.Remove(path)
	_, err := config.LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}

}

func TestLoadOrDefault_ENVSelection(t *testing.T) {
	setEnv(t, constants.EnvVarEnvKey, string(constants.DEVELOPMENT))
	err := config.LoadOrDefault("")
	if err != nil {
		t.Fatal(err)
	}

	setEnv(t, constants.EnvVarEnvKey, string(constants.STAGING))
	err = config.LoadOrDefault("")
	if err != nil {
		t.Fatal(err)
	}

	setEnv(t, constants.EnvVarEnvKey, string(constants.PRODUCTION))
	err = config.LoadOrDefault("")
	if err != nil {
		t.Fatal(err)
	}
}
func TestLoadOrDefault_ENV_DefaultDevelopment(t *testing.T) {
	t.Setenv(constants.EnvVarEnvKey, "")
	err := config.LoadOrDefault("")
	require.NoError(t, err)
}

func TestLoadOrDefault_ENV_Development(t *testing.T) {
	t.Setenv(constants.EnvVarEnvKey, string(constants.DEVELOPMENT))
	err := config.LoadOrDefault("")
	require.NoError(t, err)
}

func TestLoadOrDefault_ENV_Staging_FileMissing(t *testing.T) {
	t.Setenv(constants.EnvVarEnvKey, string(constants.STAGING))
	err := config.LoadOrDefault("")
	require.NoError(t, err)
}

func TestLoadOrDefault_ENV_Production_WithFile(t *testing.T) {
	t.Setenv(constants.EnvVarEnvKey, string(constants.PRODUCTION))
	file := writeTempFile(t, `db_name = my.db`) // simulate file at /etc/daedalus
	defer os.Remove(file)
	err := config.LoadOrDefault(file)
	require.NoError(t, err)
}

func TestLoadOrDefault_CustomPath_FileExists(t *testing.T) {
	file := writeTempFile(t, `db_name = custom.db`)
	defer os.Remove(file)
	err := config.LoadOrDefault(file)
	require.NoError(t, err)

}

func TestLoadOrDefault_CustomPath_FileMissing(t *testing.T) {
	err := config.LoadOrDefault("/tmp/does-not-exist.conf")
	require.Error(t, err)
	assert.True(t, errors.Is(err, os.ErrNotExist))
}

func TestLoadOrDefault_ENVFallbacks(t *testing.T) {
	t.Setenv(constants.EnvVarDefaultRootUser, "root")
	t.Setenv(constants.EnvVarDefaultRootPassword, "rootpass")
	err := config.LoadOrDefault("")
	require.NoError(t, err)
	assert.Equal(t, "root", config.GlobalConfiguration.DefaultRootUser)
	assert.Equal(t, "rootpass", config.GlobalConfiguration.DefaultRootPassword)
}

func TestLoadOrDefault_DefaultRootFallbacks(t *testing.T) {
	err := config.LoadOrDefault("")
	require.NoError(t, err)
	assert.Equal(t, "admin", config.GlobalConfiguration.DefaultRootUser)
	assert.Equal(t, "admin", config.GlobalConfiguration.DefaultRootPassword)
}
