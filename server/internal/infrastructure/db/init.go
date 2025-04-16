package db

import (
	"deadalus-orch/server/internal/pkg/utils"
	"deadalus-orch/shared/constants"
	"fmt"
	"os"
	"os/user"
	"path/filepath"

	"github.com/linxGnu/grocksdb"
	"github.com/rs/zerolog/log"
)

var (
	getEnv         = os.Getenv
	getCurrentUser = user.Current
	mkdirAll       = os.MkdirAll
)

type PathProvider interface {
	GetDatabasePath() (string, error)
}

type DefaultPathProvider struct{}

func (d DefaultPathProvider) GetDatabasePath() (string, error) {
	env := getEnv(constants.EnvVarEnvKey)
	if env == "" {
		env = "development"
	}

	if env == "development" {
		usr, err := getCurrentUser()
		if err != nil {
			return "", fmt.Errorf("could not get user: %v", err)
		}
		return filepath.Join(usr.HomeDir, ".daedalus", "data"), nil
	}

	path := "/var/lib/daedalus/data"
	if err := mkdirAll(path, 0755); err != nil {
		return "", fmt.Errorf("could not create path: %v", err)
	}

	return path, nil
}

// InitDB usa un PathProvider para determinar dónde crear la base
func InitDB(dbName string, provider PathProvider) (*grocksdb.DB, error) {
	dbPath, err := provider.GetDatabasePath()
	if err != nil {
		return nil, err
	}

	fullPath := filepath.Join(dbPath, dbName)
	if err := utils.EnsureDirExists(fullPath); err != nil {
		return nil, fmt.Errorf("could not create db dir: %w", err)
	}

	return OpenDB(fullPath)
}

func OpenDB(dbPath string) (*grocksdb.DB, error) {
	log.Info().
		Str("dbPath", dbPath).
		Msg("🗄️  Opening db")
	opts := grocksdb.NewDefaultOptions()
	opts.SetCreateIfMissing(true)
	opts.SetInfoLogLevel(grocksdb.WarnInfoLogLevel)
	db, err := grocksdb.OpenDb(opts, dbPath)
	if err != nil {
		return nil, fmt.Errorf("error opening database: %v", err)
	}

	return db, nil
}
