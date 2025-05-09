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
func InitDB(dbName string, provider PathProvider, columnFamilyNames []string, ttlColumnFamilyNames []string) (*grocksdb.DB, map[string]*grocksdb.ColumnFamilyHandle, map[string]*grocksdb.ColumnFamilyHandle, error) {
	dbPath, err := provider.GetDatabasePath()
	if err != nil {
		return nil, nil, nil, err
	}

	fullPath := filepath.Join(dbPath, dbName)
	if err := utils.EnsureDirExists(fullPath); err != nil {
		return nil, nil, nil, fmt.Errorf("could not create db dir: %w", err)
	}

	return OpenDB(fullPath, columnFamilyNames, ttlColumnFamilyNames)
}

func OpenDB(
	dbPath string,
	columnFamilyNames []string,
	ttlColumnFamilyNames []string,
) (*grocksdb.DB, map[string]*grocksdb.ColumnFamilyHandle, map[string]*grocksdb.ColumnFamilyHandle, error) {

	log.Info().
		Str("dbPath", dbPath).
		Msg("🗄️  Opening index db")

	// Validar duplicados dentro de cada lista
	if hasDuplicates(columnFamilyNames) {
		return nil, nil, nil, fmt.Errorf("duplicated names in columnFamilyNames")
	}
	if hasDuplicates(ttlColumnFamilyNames) {
		return nil, nil, nil, fmt.Errorf("duplicated names in ttlColumnFamilyNames")
	}

	nameSet := make(map[string]struct{})
	for _, name := range columnFamilyNames {
		nameSet[name] = struct{}{}
	}
	for _, name := range ttlColumnFamilyNames {
		if _, exists := nameSet[name]; exists {
			return nil, nil, nil, fmt.Errorf("column family name '%s' exists in both normal and TTL sets", name)
		}
	}

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
			return nil, nil, nil, err
		}
		for _, cf := range currentColumnFamilies {
			uniqueCF[cf] = struct{}{}
		}
	}

	for _, cf := range columnFamilyNames {
		uniqueCF[cf] = struct{}{}
	}
	for _, cf := range ttlColumnFamilyNames {
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
	for i := range allCFs {
		cfOpts[i] = grocksdb.NewDefaultOptions()
		defer cfOpts[i].Destroy()
	}

	db, cfHs, err := grocksdb.OpenDbColumnFamilies(opts, dbPath, allCFs, cfOpts)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error opening database: %v", err)
	}

	normalCFHandles := make(map[string]*grocksdb.ColumnFamilyHandle)
	ttlCFHandles := make(map[string]*grocksdb.ColumnFamilyHandle)

	for i, name := range allCFs {
		handle := cfHs[i]
		if contains(ttlColumnFamilyNames, name) {
			ttlCFHandles[name] = handle
		} else {
			normalCFHandles[name] = handle
		}
	}

	return db, normalCFHandles, ttlCFHandles, nil
}

func hasDuplicates(items []string) bool {
	seen := make(map[string]struct{})
	for _, item := range items {
		if _, ok := seen[item]; ok {
			return true
		}
		seen[item] = struct{}{}
	}
	return false
}

func contains(list []string, target string) bool {
	for _, item := range list {
		if item == target {
			return true
		}
	}
	return false
}
