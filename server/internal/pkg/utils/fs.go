package utils

import (
	"errors"
	"os"
	"path/filepath"
)

// EnsureDirExists checks if the directory for the given path exists,
// and creates it if it does not. It uses os.MkdirAll, so it will create
// parent directories as needed. The permissions for created directories are 0755.
//
// Parameters:
//   - path: The file path for which the directory structure should be ensured.
//     If `path` itself is a directory, its parent directory will be used as the base for MkdirAll.
//     If `path` is a file, its containing directory will be created.
//
// Returns:
//   - An error if the path is empty or if os.MkdirAll fails. Returns nil on success.
func EnsureDirExists(path string) error {
	if path == "" {
		return errors.New("path cannot be empty")
	}
	// filepath.Dir returns the directory containing path.
	// If path is already a directory, Dir returns the parent directory.
	// If path is a file, Dir returns its containing directory.
	// This is suitable for os.MkdirAll which creates all necessary parent directories.
	dir := filepath.Dir(path)
	// os.MkdirAll creates a directory named path, along with any necessary parents,
	// and returns nil, or else returns an error. The permission bits perm (before umask)
	// are used for all directories that MkdirAll creates.
	return os.MkdirAll(dir, 0755) // 0755 gives rwx for owner, rx for group and others.
}

// DirExists checks if a given path exists and is a directory.
//
// Parameters:
//   - path: The path to check.
//
// Returns:
//   - A boolean indicating whether the path exists and is a directory (true) or not (false).
//   - An error if os.Stat fails for any reason other than the path not existing.
//     If the path does not exist, it returns (false, nil).
func DirExists(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil // Path does not exist, so it's not a directory.
		}
		return false, err // Another error occurred (e.g., permission denied).
	}
	return info.IsDir(), nil // Path exists, return whether it's a directory.
}

// FileExists checks if a given path exists and is a regular file.
//
// Parameters:
//   - path: The path to check.
//
// Returns:
//   - A boolean indicating whether the path exists and is a regular file (true) or not (false).
//   - An error if os.Stat fails for any reason other than the path not existing.
//     If the path does not exist, it returns (false, nil).
func FileExists(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil // Path does not exist, so it's not a file.
		}
		return false, err // Another error occurred (e.g., permission denied).
	}
	return info.Mode().IsRegular(), nil // Path exists, return whether it's a regular file.
}

// CreateFileWithDirs creates a new file at the specified path.
// If the directory structure leading to the file does not exist, it creates all necessary directories
// with 0755 permissions.
//
// Parameters:
//   - path: The full path where the file should be created.
//
// Returns:
//   - A pointer to the created os.File, opened for reading and writing.
//   - An error if creating the directories or the file fails.
func CreateFileWithDirs(path string) (*os.File, error) {
	dir := filepath.Dir(path)
	// Ensure all parent directories exist, creating them if necessary.
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	// Create the file. If it already exists, it will be truncated.
	file, err := os.Create(path)
	if err != nil {
		return nil, err
	}

	return file, nil
}
