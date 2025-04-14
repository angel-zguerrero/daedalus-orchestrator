package db

import (
	"deadalus-orch/server/internal/pkg/utils"
	"fmt"
	"os"
	"os/user"
	"path/filepath"

	"github.com/linxGnu/grocksdb"
)

func OpenDB(dbPath string) (*grocksdb.DB, error) {
	fmt.Println("🗄️  Opening db:", dbPath)
	opts := grocksdb.NewDefaultOptions()
	opts.SetCreateIfMissing(true)
	opts.SetInfoLogLevel(grocksdb.WarnInfoLogLevel)
	db, err := grocksdb.OpenDb(opts, dbPath)
	if err != nil {
		return nil, fmt.Errorf("error opening database: %v", err)
	}

	return db, nil
}

func InitDB(dbName string) (*grocksdb.DB, error) {

	dbPath, err := getDatabasePath()
	if err != nil {
		return nil, err
	}

	fullPath := filepath.Join(dbPath, dbName)
	if err := utils.EnsureDirExists(fullPath); err != nil {
		return nil, fmt.Errorf("could not create db dir: %w", err)
	}

	return OpenDB(fullPath)
}

func getDatabasePath() (string, error) {
	env := os.Getenv("ENV")
	if env == "" {
		env = "development"
	}

	if env == "development" {
		usr, err := user.Current()
		if err != nil {
			return "", fmt.Errorf("could not get user: %v", err)
		}
		return filepath.Join(usr.HomeDir, ".daedalus", "data"), nil
	}

	path := "/var/lib/daedalus/data"
	if err := os.MkdirAll(path, 0755); err != nil {
		return "", fmt.Errorf("could not create path: %v", err)
	}

	return path, nil
}
