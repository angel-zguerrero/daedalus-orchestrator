package db_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/shared/constants"

	"github.com/stretchr/testify/assert"
)

// Fake provider for tests
type FakePathProvider struct {
	Path string
	Err  error
}

func (f FakePathProvider) GetDatabasePath() (string, error) {
	return f.Path, f.Err
}

func TestDefaultPathProvider_DevelopmentEnv(t *testing.T) {
	t.Setenv(constants.EnvVarEnvKey, string(constants.DEVELOPMENT))

	// Isolate home directory using t.TempDir() so tests NEVER delete ~/.daedalus
	fakeHome := t.TempDir()
	t.Setenv("HOME", fakeHome)
	t.Setenv("USERPROFILE", fakeHome)

	provider := db.DefaultPathProvider{}
	path, err := provider.GetDatabasePath()

	assert.NoError(t, err)
	expectedPathSuffix := filepath.Join(".daedalus", "data")
	assert.True(t, strings.HasSuffix(path, expectedPathSuffix), "Path should end with "+expectedPathSuffix)
	assert.True(t, strings.HasPrefix(path, fakeHome), "Path should start with fake home directory")
}

func TestDefaultPathProvider_ProductionEnv(t *testing.T) {
	t.Setenv(constants.EnvVarEnvKey, string(constants.PRODUCTION))

	// Isolate home directory using t.TempDir() so tests NEVER touch user home
	fakeHome := t.TempDir()
	t.Setenv("HOME", fakeHome)
	t.Setenv("USERPROFILE", fakeHome)

	provider := db.DefaultPathProvider{}
	path, err := provider.GetDatabasePath()

	var expectedPath string
	switch osName := runtime.GOOS; osName {
	case "darwin":
		expectedPath = filepath.Join(fakeHome, "Library", "Application Support", "Daedalus", "data")
	case "windows":
		programData := os.Getenv("ProgramData")
		if programData == "" {
			programData = `C:\ProgramData`
		}
		expectedPath = filepath.Join(programData, "Daedalus", "data")
	default:
		expectedPath = "/var/lib/daedalus/data"
	}

	if err != nil {
		if os.IsPermission(err) || strings.Contains(err.Error(), "permission denied") {
			assert.Equal(t, expectedPath, path)
		} else {
			assert.Fail(t, "unexpected error: %v", err)
		}
	} else {
		assert.Equal(t, expectedPath, path)
	}
}

func TestDefaultPathProvider_CustomDataDir(t *testing.T) {
	customDir := t.TempDir()
	t.Setenv("DAEDALUS_DATA_DIR", customDir)

	provider := db.DefaultPathProvider{}
	path, err := provider.GetDatabasePath()

	assert.NoError(t, err)
	assert.Equal(t, customDir, path)
}
