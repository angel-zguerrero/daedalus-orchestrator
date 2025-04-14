package utils

import (
	"errors"
	"os"
	"path/filepath"
)

func EnsureDirExists(path string) error {
	if path == "" {
		return errors.New("path cannot be empty")
	}
	dir := filepath.Dir(path)
	return os.MkdirAll(dir, 0755)
}
