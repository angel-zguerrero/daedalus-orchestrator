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
			return false, nil
		}
		return false, err
	}
	return info.IsDir(), nil
}

func CreateFileWithDirs(path string) (*os.File, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	file, err := os.Create(path)
	if err != nil {
		return nil, err
	}

	return file, nil
}
