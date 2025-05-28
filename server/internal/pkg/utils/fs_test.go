package utils_test

import (
	"deadalus-orch/server/internal/pkg/utils"
	"os"
	"path/filepath"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"  // Added
	"github.com/stretchr/testify/require" // Added
)

func TestMain(m *testing.M) {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	code := m.Run()
	os.Exit(code)
}

func TestEnsureDirExists(t *testing.T) {
	// Base para los directorios temporales
	baseDir := os.TempDir()

	// Casos de prueba
	tests := []struct {
		name        string
		path        string
		expectedErr bool
	}{
		{
			name:        "Valid Path With File",
			path:        filepath.Join(baseDir, "test_daedalus", "valid_dir", "file.txt"),
			expectedErr: false,
		},
		{
			name:        "Valid Path  Without File",
			path:        filepath.Join(baseDir, "test_daedalus", "valid_dir"),
			expectedErr: false,
		},
		{
			name:        "Empty Path",
			path:        "",
			expectedErr: true,
		},
		{
			name:        "Existing Path",
			path:        filepath.Join(baseDir, "test_daedalus", "existing_dir", "file.txt"),
			expectedErr: false,
		},
		{
			name:        "Invalid Permissions",
			path:        "/root/test_invalid_perm/file.txt", // Supone que no tenemos permisos para escribir en /root
			expectedErr: true,
		},
		{
			name:        "Complex Path",
			path:        filepath.Join(baseDir, "test_daedalus", "subdir", "nested", "file.txt"),
			expectedErr: false,
		},
	}

	t.Cleanup(func() {
		_ = os.RemoveAll(filepath.Join(baseDir, "test_daedalus"))
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := utils.EnsureDirExists(tt.path)

			if (err != nil) != tt.expectedErr {
				t.Errorf("Expected error: %v, but got: %v", tt.expectedErr, err)
			}
			if err == nil {
				dir := filepath.Dir(tt.path)
				_, statErr := os.Stat(dir)
				if os.IsNotExist(statErr) {
					t.Errorf("Expected directory %s to be created, but it does not exist", dir)
				}
			}
		})
	}
}

func TestDirExists_PathIsDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	exists, err := utils.DirExists(tmpDir)
	assert.True(t, exists)
	assert.NoError(t, err)
}

func TestDirExists_PathIsFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "testfile.txt")
	f, err := os.Create(filePath)
	require.NoError(t, err, "Failed to create test file")
	f.Close()

	exists, err := utils.DirExists(filePath)
	assert.False(t, exists)
	assert.NoError(t, err)
}

func TestDirExists_PathDoesNotExist(t *testing.T) {
	tmpDir := t.TempDir()
	nonExistentPath := filepath.Join(tmpDir, "nonexistent")
	// Ensure we remove it if it was accidentally created by a previous test or setup
	_ = os.RemoveAll(nonExistentPath)

	exists, err := utils.DirExists(nonExistentPath)
	assert.False(t, exists)
	assert.NoError(t, err) // DirExists returns false, nil for non-existent paths
}

func TestDirExists_EmptyPath(t *testing.T) {
	exists, err := utils.DirExists("")
	assert.False(t, exists)
	assert.NoError(t, err)
}

func TestCreateFileWithDirs_CreatesFileAndNewDirs(t *testing.T) {
	tmpDir := t.TempDir()
	newFilePath := filepath.Join(tmpDir, "newsubdir1", "newsubdir2", "testfile.txt")

	file, err := utils.CreateFileWithDirs(newFilePath)
	require.NoError(t, err, "CreateFileWithDirs should not return an error")
	require.NotNil(t, file, "Returned file should not be nil")
	defer file.Close()

	_, statErr := os.Stat(newFilePath)
	assert.NoError(t, statErr, "File should exist at the new path")

	// Verify directories were created
	dirPath1 := filepath.Join(tmpDir, "newsubdir1")
	dirInfo1, statErr1 := os.Stat(dirPath1)
	assert.NoError(t, statErr1, "newsubdir1 should exist")
	assert.True(t, dirInfo1.IsDir(), "newsubdir1 should be a directory")

	dirPath2 := filepath.Join(tmpDir, "newsubdir1", "newsubdir2")
	dirInfo2, statErr2 := os.Stat(dirPath2)
	assert.NoError(t, statErr2, "newsubdir2 should exist")
	assert.True(t, dirInfo2.IsDir(), "newsubdir2 should be a directory")
}

func TestCreateFileWithDirs_CreatesFileInExistingDir(t *testing.T) {
	existingDir := t.TempDir() // t.TempDir() already creates the directory
	filePath := filepath.Join(existingDir, "testfile.txt")

	file, err := utils.CreateFileWithDirs(filePath)
	require.NoError(t, err, "CreateFileWithDirs should not return an error for existing dir")
	require.NotNil(t, file, "Returned file should not be nil")
	defer file.Close()

	_, statErr := os.Stat(filePath)
	assert.NoError(t, statErr, "File should exist in the existing directory")
}

func TestCreateFileWithDirs_EmptyPath(t *testing.T) {
	file, err := utils.CreateFileWithDirs("")
	assert.Error(t, err, "CreateFileWithDirs should return an error for an empty path")
	assert.Nil(t, file, "Returned file should be nil on error")
}
