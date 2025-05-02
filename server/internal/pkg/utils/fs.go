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

func DirExists(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil // No existe
		}
		return false, err // Otro error (permisos, etc)
	}
	return info.IsDir(), nil
}
