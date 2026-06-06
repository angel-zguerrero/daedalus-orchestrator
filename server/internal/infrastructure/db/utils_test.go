package db_test

import (
	"deadalus-orch/shared/constants" // Added
	"os"
	"os/user" // Added
	"path/filepath"
	"runtime"
	"strings" // Added
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require" // Added

	"deadalus-orch/server/internal/infrastructure/db"
)

// Fake provider para pruebas
type FakePathProvider struct {
	Path string
	Err  error
}

func (f FakePathProvider) GetDatabasePath() (string, error) {
	return f.Path, f.Err
}

func TestDefaultPathProvider_DevelopmentEnv(t *testing.T) {
	t.Setenv(constants.EnvVarEnvKey, string(constants.DEVELOPMENT))

	currentUser, err := user.Current()
	require.NoError(t, err, "Failed to get current user")

	provider := db.DefaultPathProvider{}
	path, err := provider.GetDatabasePath()

	assert.NoError(t, err)
	expectedPathSuffix := filepath.Join(".daedalus", "data")
	assert.True(t, strings.HasSuffix(path, expectedPathSuffix), "Path should end with "+expectedPathSuffix)
	assert.True(t, strings.HasPrefix(path, currentUser.HomeDir), "Path should start with user's home directory")

	// Clean up created directory if it exists, be careful with this in real tests
	// For this test, we are primarily checking the path string, but if it creates, we can clean.
	// The function GetDatabasePath itself creates the dir.
	if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
		os.RemoveAll(filepath.Join(currentUser.HomeDir, ".daedalus")) // remove parent .daedalus
	}
}

func TestDefaultPathProvider_ProductionEnv(t *testing.T) {
	t.Setenv(constants.EnvVarEnvKey, string(constants.PRODUCTION))

	currentUser, err := user.Current()
	require.NoError(t, err, "Failed to get current user")

	provider := db.DefaultPathProvider{}
	path, err := provider.GetDatabasePath()

	var expectedPath string
	switch osName := runtime.GOOS; osName {
	case "darwin":
		expectedPath = filepath.Join(currentUser.HomeDir, "Library", "Application Support", "Daedalus", "data")
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
		if os.IsPermission(err) {
			assert.Equal(t, expectedPath, path)
		} else {
			assert.Fail(t, "unexpected error: %v", err)
		}
	} else {
		assert.Equal(t, expectedPath, path)
		// Clean up created path
		if runtime.GOOS == "darwin" {
			os.RemoveAll(filepath.Join(currentUser.HomeDir, "Library", "Application Support", "Daedalus"))
		} else if runtime.GOOS == "windows" {
			// standard program data path clean
			programData := os.Getenv("ProgramData")
			if programData == "" {
				programData = `C:\ProgramData`
			}
			os.RemoveAll(filepath.Join(programData, "Daedalus"))
		} else {
			os.RemoveAll("/var/lib/daedalus")
		}
	}
}

