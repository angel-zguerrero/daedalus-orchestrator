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

const (
	DefaultFC = "default"
	MetaFC    = "meta"
)

type PathProvider interface {
	GetDatabasePath() (string, error)
}

type DefaultPathProvider struct{}

func (d DefaultPathProvider) GetDatabasePath() (string, error) {
	env := getEnv(constants.EnvVarEnvKey)
	if env == "" {
		env = string(constants.DEVELOPMENT)
	}

	if env == string(constants.DEVELOPMENT) {
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
func InitDB(dbName string, provider PathProvider, columnFamilyNames []string) (*grocksdb.DB, map[string]*grocksdb.ColumnFamilyHandle, error) {
	dbPath, err := provider.GetDatabasePath()
	if err != nil {
		return nil, nil, err
	}

	fullPath := filepath.Join(dbPath, dbName)
	if err := utils.EnsureDirExists(fullPath); err != nil {
		return nil, nil, fmt.Errorf("could not create db dir: %w", err)
	}

	return OpenDB(fullPath, columnFamilyNames)
}

func OpenDB(dbPath string, columnFamilyNames []string) (*grocksdb.DB, map[string]*grocksdb.ColumnFamilyHandle, error) {
	log.Info().
		Str("dbPath", dbPath).
		Msg("🗄️  Opening index db")
	opts := grocksdb.NewDefaultOptions()
	opts.SetCreateIfMissing(true)
	opts.SetInfoLogLevel(grocksdb.WarnInfoLogLevel)

	opts.SetCreateIfMissingColumnFamilies(true)

	var err error
	var currentColumnFamilies []string
	uniqueCF := make(map[string]struct{})
	if exists, _ := utils.DirExists(dbPath); exists {
		currentColumnFamilies, err = grocksdb.ListColumnFamilies(opts, dbPath)
		if err != nil {
			return nil, nil, err
		}
		for _, cf := range currentColumnFamilies {
			uniqueCF[cf] = struct{}{}
		}
	}

	for _, cf := range columnFamilyNames {
		uniqueCF[cf] = struct{}{}
	}

	var allCFs []string
	for cf := range uniqueCF {
		allCFs = append(allCFs, cf)
	}

	cfSet := make(map[string]struct{}, len(allCFs))
	for _, name := range allCFs {
		cfSet[name] = struct{}{}
	}

	if _, ok := cfSet[DefaultFC]; !ok {
		allCFs = append(allCFs, DefaultFC)
	}

	if _, ok := cfSet[MetaFC]; !ok {
		allCFs = append(allCFs, MetaFC)
	}

	cfOpts := make([]*grocksdb.Options, len(allCFs))

	for index, _ := range allCFs {
		cfOpts[index] = grocksdb.NewDefaultOptions()
		defer cfOpts[index].Destroy()
	}

	db, cfHs, err := grocksdb.OpenDbColumnFamilies(opts, dbPath, allCFs, cfOpts)
	if err != nil {
		return nil, nil, fmt.Errorf("error opening database: %v", err)
	}
	ColumnFamilyHandles := make(map[string]*grocksdb.ColumnFamilyHandle, len(allCFs))
	for index, name := range allCFs {
		ColumnFamilyHandles[name] = cfHs[index]
	}

	return db, ColumnFamilyHandles, nil
}
