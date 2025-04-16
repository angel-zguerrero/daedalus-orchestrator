package config_test

import (
	"deadalus-orch/server/internal/pkg/config"
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
	cfg, err := config.LoadOrDefault(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DBname != "my.db" {
		t.Errorf("expected db_name=my.db, got %s", cfg.DBname)
	}
	if cfg.DefaultRootUser != "admin" {
		t.Errorf("expected default_root_user=admin")
	}
	if cfg.DefaultRootPassword != "secret" {
		t.Errorf("expected default_root_password=secret")
	}

	if cfg.Port != 50052 {
		t.Errorf("expected port=50052, got %d", cfg.Port)
	}
}

func TestLoadOrDefault_ConfigFileOverwriteKeysWithEnvVars(t *testing.T) {
	content := `
db_name=my.db
default_root_user=admin
default_root_password=secret
port=50005
`

	setEnv(t, "DB_NAME", "other-name-db")
	setEnv(t, "DEFAULT_ROOT_PASSWORD", "envPass")
	setEnv(t, "DEFAULT_ROOT_USER", "envUser")
	setEnv(t, "PORT", "50006")

	path := writeTempFile(t, content)
	defer os.Remove(path)
	cfg, err := config.LoadOrDefault(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DBname != "other-name-db" {
		t.Errorf("expected DBname=other-name-db, got %s", cfg.DBname)
	}
	if cfg.DefaultRootUser != "envUser" {
		t.Errorf("expected DefaultRootUser=envUser, got %s", cfg.DefaultRootUser)
	}
	if cfg.DefaultRootPassword != "envPass" {
		t.Errorf("expected DefaultRootPassword=envPass, got %s", cfg.DefaultRootPassword)
	}

	if cfg.Port != 50006 {
		t.Errorf("expected port=50006, got %d", cfg.Port)
	}
}

func TestLoadOrDefault_InvalidPort(t *testing.T) {
	content := `
port=9
`
	path := writeTempFile(t, content)
	defer os.Remove(path)
	_, err := config.LoadOrDefault(path)
	require.Error(t, err)
	assert.EqualError(t, err, "invalid 'port' in config: '9'")
}

func TestLoadOrDefault_ConfigFilePartialKeys_ENVFallback(t *testing.T) {
	content := `
db_name=my.db
`
	path := writeTempFile(t, content)
	defer os.Remove(path)
	setEnv(t, "DEFAULT_ROOT_USER", "envUser")
	setEnv(t, "DEFAULT_ROOT_PASSWORD", "envPass")

	cfg, err := config.LoadOrDefault(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DBname != "my.db" {
		t.Errorf("expected db_name=my.db, got %s", cfg.DBname)
	}
	if cfg.DefaultRootUser != "envUser" {
		t.Errorf("expected default_root_user from env")
	}
	if cfg.DefaultRootPassword != "envPass" {
		t.Errorf("expected default_root_password from env")
	}
}

func TestLoadOrDefault_InvalidPath(t *testing.T) {

	_, err := config.LoadOrDefault("/nonexistent/path.conf")
	if err == nil {
		t.Errorf("expected error but get nil")
	}

}

func TestLoadOrDefault_NoConfigFile_ENVFallback(t *testing.T) {
	setEnv(t, "DB_NAME", "env.db")
	setEnv(t, "DEFAULT_ROOT_USER", "envUser")
	setEnv(t, "DEFAULT_ROOT_PASSWORD", "envPass")
	setEnv(t, "PORT", "5050")

	cfg, err := config.LoadOrDefault("")
	if err != nil {
		t.Fatal(err)
	}

	if cfg.DBname != "env.db" {
		t.Errorf("expected db_name=env.db, got %s", cfg.DBname)
	}
	if cfg.Port != 5050 {
		t.Errorf("expected port=5050, got %d", cfg.Port)
	}
	if cfg.DefaultRootUser != "envUser" {
		t.Errorf("expected default_root_user from env")
	}
	if cfg.DefaultRootPassword != "envPass" {
		t.Errorf("expected default_root_password from env")
	}
}

func TestLoadOrDefault_NoFile_NoEnv_DefaultFallback(t *testing.T) {
	cfg, err := config.LoadOrDefault("")
	if err != nil {
		t.Fatal(err)
	}

	if cfg.DBname != "daedalus.db" {
		t.Errorf("expected fallback db_name=daedalus.db")
	}

	if cfg.DefaultRootUser != "admin" {
		t.Errorf("expected fallback default_root_user=admin")
	}

	if cfg.DefaultRootPassword != "admin" {
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
	cfg, err := config.LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.DBname != "my.db" {
		t.Errorf("expected db_name=my.db, got %s", cfg.DBname)
	}

	if cfg.Port != 0 {
		t.Errorf("expected Port 0, got %d", cfg.Port)
	}

}

func TestLoadOrDefault_ENVSelection(t *testing.T) {
	setEnv(t, "ENV", "development")
	setEnv(t, "DB_NAME", "dev.db")
	cfg, err := config.LoadOrDefault("")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DBname != "dev.db" {
		t.Errorf("expected db_name from ENV in development")
	}

	setEnv(t, "ENV", "staging")
	setEnv(t, "DB_NAME", "stage.db")
	cfg, err = config.LoadOrDefault("")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DBname != "stage.db" {
		t.Errorf("expected db_name from ENV in staging")
	}

	setEnv(t, "ENV", "production")
	setEnv(t, "DB_NAME", "prod.db")
	cfg, err = config.LoadOrDefault("")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DBname != "prod.db" {
		t.Errorf("expected db_name from ENV in production")
	}
}
func TestLoadOrDefault_ENV_DefaultDevelopment(t *testing.T) {
	t.Setenv("ENV", "")
	cfg, err := config.LoadOrDefault("")
	require.NoError(t, err)
	assert.Equal(t, "daedalus.db", cfg.DBname)
}

func TestLoadOrDefault_ENV_Development(t *testing.T) {
	t.Setenv("ENV", "development")
	cfg, err := config.LoadOrDefault("")
	require.NoError(t, err)
	assert.Equal(t, "daedalus.db", cfg.DBname)
}

func TestLoadOrDefault_ENV_Staging_FileMissing(t *testing.T) {
	t.Setenv("ENV", "staging")
	cfg, err := config.LoadOrDefault("")
	require.NoError(t, err)
	assert.Equal(t, "daedalus.db", cfg.DBname)
}

func TestLoadOrDefault_ENV_Production_WithFile(t *testing.T) {
	t.Setenv("ENV", "production")
	file := writeTempFile(t, `db_name = my.db`) // simulate file at /etc/daedalus
	defer os.Remove(file)
	cfg, err := config.LoadOrDefault(file)
	require.NoError(t, err)
	assert.Equal(t, "my.db", cfg.DBname)
}

func TestLoadOrDefault_CustomPath_FileExists(t *testing.T) {
	file := writeTempFile(t, `db_name = custom.db`)
	defer os.Remove(file)
	cfg, err := config.LoadOrDefault(file)
	require.NoError(t, err)
	assert.Equal(t, "custom.db", cfg.DBname)
}

func TestLoadOrDefault_CustomPath_FileMissing(t *testing.T) {
	_, err := config.LoadOrDefault("/tmp/does-not-exist.conf")
	require.Error(t, err)
	assert.True(t, errors.Is(err, os.ErrNotExist))
}

func TestLoadOrDefault_ENVFallbacks(t *testing.T) {
	t.Setenv("DB_NAME", "fromenv.db")
	t.Setenv("DEFAULT_ROOT_USER", "root")
	t.Setenv("DEFAULT_ROOT_PASSWORD", "rootpass")
	cfg, err := config.LoadOrDefault("")
	require.NoError(t, err)
	assert.Equal(t, "fromenv.db", cfg.DBname)
	assert.Equal(t, "root", cfg.DefaultRootUser)
	assert.Equal(t, "rootpass", cfg.DefaultRootPassword)
}

func TestLoadOrDefault_DefaultRootFallbacks(t *testing.T) {
	cfg, err := config.LoadOrDefault("")
	require.NoError(t, err)
	assert.Equal(t, "admin", cfg.DefaultRootUser)
	assert.Equal(t, "admin", cfg.DefaultRootPassword)
}
