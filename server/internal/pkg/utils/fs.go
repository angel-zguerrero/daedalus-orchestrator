package utils

import (
	"os"
	"path/filepath"
)

func EnsureDirExists(path string) error {
	dir := filepath.Dir(path)
	return os.MkdirAll(dir, 0755)
}
