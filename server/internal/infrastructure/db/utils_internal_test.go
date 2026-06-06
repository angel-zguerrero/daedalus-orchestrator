package db

import (
	"deadalus-orch/shared/constants"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultPathProvider_OSPaths(t *testing.T) {
	// Backup original functions / variables
	origGetEnv := getEnv
	origGoos := goos
	origUserHomeDir := userHomeDir
	origMkdirAll := mkdirAll

	defer func() {
		getEnv = origGetEnv
		goos = origGoos
		userHomeDir = origUserHomeDir
		mkdirAll = origMkdirAll
	}()

	provider := DefaultPathProvider{}

	t.Run("Linux development", func(t *testing.T) {
		goos = "linux"
		userHomeDir = func() (string, error) { return "/home/testuser", nil }
		getEnv = func(key string) string {
			if key == constants.EnvVarEnvKey {
				return string(constants.DEVELOPMENT)
			}
			return ""
		}
		var createdPath string
		mkdirAll = func(path string, perm os.FileMode) error {
			createdPath = path
			return nil
		}

		path, err := provider.GetDatabasePath()
		assert.NoError(t, err)
		expected := filepath.Join("/home/testuser", ".daedalus", "data")
		assert.Equal(t, expected, path)
		assert.Equal(t, expected, createdPath)
	})

	t.Run("Linux production", func(t *testing.T) {
		goos = "linux"
		userHomeDir = func() (string, error) { return "/home/testuser", nil }
		getEnv = func(key string) string {
			if key == constants.EnvVarEnvKey {
				return string(constants.PRODUCTION)
			}
			return ""
		}
		var createdPath string
		mkdirAll = func(path string, perm os.FileMode) error {
			createdPath = path
			return nil
		}

		path, err := provider.GetDatabasePath()
		assert.NoError(t, err)
		expected := filepath.Join("/", "var", "lib", "daedalus", "data")
		assert.Equal(t, expected, path)
		assert.Equal(t, expected, createdPath)
	})

	t.Run("macOS development", func(t *testing.T) {
		goos = "darwin"
		userHomeDir = func() (string, error) { return "/Users/testuser", nil }
		getEnv = func(key string) string {
			if key == constants.EnvVarEnvKey {
				return string(constants.DEVELOPMENT)
			}
			return ""
		}
		var createdPath string
		mkdirAll = func(path string, perm os.FileMode) error {
			createdPath = path
			return nil
		}

		path, err := provider.GetDatabasePath()
		assert.NoError(t, err)
		expected := filepath.Join("/Users/testuser", ".daedalus", "data")
		assert.Equal(t, expected, path)
		assert.Equal(t, expected, createdPath)
	})

	t.Run("macOS production", func(t *testing.T) {
		goos = "darwin"
		userHomeDir = func() (string, error) { return "/Users/testuser", nil }
		getEnv = func(key string) string {
			if key == constants.EnvVarEnvKey {
				return string(constants.PRODUCTION)
			}
			return ""
		}
		var createdPath string
		mkdirAll = func(path string, perm os.FileMode) error {
			createdPath = path
			return nil
		}

		path, err := provider.GetDatabasePath()
		assert.NoError(t, err)
		expected := filepath.Join("/Users/testuser", "Library", "Application Support", "Daedalus", "data")
		assert.Equal(t, expected, path)
		assert.Equal(t, expected, createdPath)
	})

	t.Run("Windows development", func(t *testing.T) {
		goos = "windows"
		userHomeDir = func() (string, error) { return `C:\Users\testuser`, nil }
		getEnv = func(key string) string {
			if key == constants.EnvVarEnvKey {
				return string(constants.DEVELOPMENT)
			}
			return ""
		}
		var createdPath string
		mkdirAll = func(path string, perm os.FileMode) error {
			createdPath = path
			return nil
		}

		path, err := provider.GetDatabasePath()
		assert.NoError(t, err)
		expected := filepath.Join(`C:\Users\testuser`, ".daedalus", "data")
		assert.Equal(t, expected, path)
		assert.Equal(t, expected, createdPath)
	})

	t.Run("Windows production", func(t *testing.T) {
		goos = "windows"
		userHomeDir = func() (string, error) { return `C:\Users\testuser`, nil }
		getEnv = func(key string) string {
			if key == constants.EnvVarEnvKey {
				return string(constants.PRODUCTION)
			}
			if key == "ProgramData" {
				return `C:\ProgramData`
			}
			return ""
		}
		var createdPath string
		mkdirAll = func(path string, perm os.FileMode) error {
			createdPath = path
			return nil
		}

		path, err := provider.GetDatabasePath()
		assert.NoError(t, err)
		expected := filepath.Join(`C:\ProgramData`, "Daedalus", "data")
		assert.Equal(t, expected, path)
		assert.Equal(t, expected, createdPath)
	})

	t.Run("DAEDALUS_DATA_DIR override", func(t *testing.T) {
		goos = "linux"
		userHomeDir = func() (string, error) { return "/home/testuser", nil }
		getEnv = func(key string) string {
			if key == "DAEDALUS_DATA_DIR" {
				return "/custom/data/dir"
			}
			return ""
		}
		var createdPath string
		mkdirAll = func(path string, perm os.FileMode) error {
			createdPath = path
			return nil
		}

		path, err := provider.GetDatabasePath()
		assert.NoError(t, err)
		assert.Equal(t, "/custom/data/dir", path)
		assert.Equal(t, "/custom/data/dir", createdPath)
	})
}

