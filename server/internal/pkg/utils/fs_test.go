package utils_test

import (
	"deadalus-orch/server/internal/pkg/utils"
	"os"
	"path/filepath"
	"testing"
)

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
