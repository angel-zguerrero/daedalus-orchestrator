package db

import (
	"deadalus-orch/shared/constants"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
)

var (
	getEnv         = os.Getenv
	getCurrentUser = user.Current
	mkdirAll       = os.MkdirAll
	goos           = runtime.GOOS
	userHomeDir    = os.UserHomeDir
)

// GetDatabasePath returns the appropriate database storage path based on the environment and operating system.
// It creates the directory if it doesn't exist and returns an error if the directory cannot be created.
func (d DefaultPathProvider) GetDatabasePath() (string, error) {
	path, err := d.getDefaultDataPath()
	if err != nil {
		return "", err
	}

	if err := mkdirAll(path, 0755); err != nil {
		return "", fmt.Errorf("could not create database directory at %q: %w", path, err)
	}

	return path, nil
}

// getDefaultDataPath resolves the default data directory path based on environment and OS.
func (d DefaultPathProvider) getDefaultDataPath() (string, error) {
	if dataDir := getEnv("DAEDALUS_DATA_DIR"); dataDir != "" {
		return dataDir, nil
	}

	env := getEnv(constants.EnvVarEnvKey)
	if env == "" {
		env = string(constants.DEVELOPMENT)
	}

	if env == string(constants.DEVELOPMENT) {
		home, err := userHomeDir()
		if err != nil {
			usr, err2 := getCurrentUser()
			if err2 != nil {
				return "", fmt.Errorf("could not get user home directory: %w", err)
			}
			home = usr.HomeDir
		}
		return filepath.Join(home, ".daedalus", "data"), nil
	}

	// Non-development environments (e.g. production, staging)
	switch goos {
	case "windows":
		programData := getEnv("ProgramData")
		if programData == "" {
			programData = `C:\ProgramData`
		}
		return filepath.Join(programData, "Daedalus", "data"), nil
	case "darwin": // macOS
		home, err := userHomeDir()
		if err != nil {
			usr, err2 := getCurrentUser()
			if err2 != nil {
				return "", fmt.Errorf("could not get user home directory for macOS: %w", err)
			}
			home = usr.HomeDir
		}
		return filepath.Join(home, "Library", "Application Support", "Daedalus", "data"), nil
	default: // Linux and others
		return filepath.Join("/", "var", "lib", "daedalus", "data"), nil
	}
}

