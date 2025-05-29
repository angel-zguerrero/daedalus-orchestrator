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

// PathProvider is an interface for determining the database storage path.
type PathProvider interface {
	// GetDatabasePath returns the path where the database should be stored.
	// It returns an error if the path cannot be determined or accessed.
	GetDatabasePath() (string, error)
}

// DefaultPathProvider is the default implementation of PathProvider.
// It determines the database path based on the environment:
// - In development (DEADALUS_ENV is "development" or not set), it uses a subdirectory in the user's home directory (`~/.daedalus/data`).
// - In other environments (e.g., production), it uses `/var/lib/daedalus/data` and attempts to create it if it doesn't exist.
type DefaultPathProvider struct{}

// GetDatabasePath returns the appropriate database storage path based on the environment.
// For development, it's typically `~/.daedalus/data`.
// For production or other environments, it's `/var/lib/daedalus/data`.
// It creates the directory if it doesn't exist (for non-development environments).
// Returns an error if the user's home directory cannot be found (in development)
// or if the directory cannot be created.
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

// InitDB initializes and opens a RocksDB database using a PathProvider to determine its location.
// It ensures the directory for the database exists and then calls OpenDB to handle the actual opening.
//
// Parameters:
//   - dbName: The name of the database (will be a subdirectory under the path provided by PathProvider).
//   - provider: The PathProvider implementation to get the base database path.
//   - columnFamilyNames: A list of names for regular column families to be opened.
//   - ttlColumnFamilyNames: A list of names for column families that should be opened with TTL support.
//
// Returns:
//   - A pointer to the opened grocksdb.DB instance.
//   - A map of normal column family names to their grocksdb.ColumnFamilyHandle.
//   - A map of TTL column family names to their grocksdb.ColumnFamilyHandle.
//   - An error if any step of the initialization fails (e.g., path resolution, directory creation, opening the DB).
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

// OpenDB opens a RocksDB database at the specified dbPath, creating and/or opening the specified column families.
// It handles the logic for listing existing column families, merging them with the requested ones,
// and ensuring default ("default", "meta") column families exist.
// It also separates handles for normal and TTL column families.
//
// Parameters:
//   - dbPath: The full file system path where the RocksDB database is located or will be created.
//   - columnFamilyNames: A list of names for regular column families to be opened/created.
//   - ttlColumnFamilyNames: A list of names for column families that should be treated as TTL column families.
//     These names must not overlap with columnFamilyNames.
//
// Returns:
//   - A pointer to the opened grocksdb.DB instance.
//   - A map of normal column family names to their grocksdb.ColumnFamilyHandle.
//   - A map of TTL column family names to their grocksdb.ColumnFamilyHandle.
//   - An error if the database cannot be opened, if column family names are duplicated,
//     or if there's an issue listing or creating column families.
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

// hasDuplicates checks if a slice of strings contains any duplicate items.
// It uses a map to keep track of items seen so far.
//
// Parameters:
//   - items: A slice of strings to check for duplicates.
//
// Returns:
//   - true if duplicates are found, false otherwise.
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

// contains checks if a slice of strings contains a specific target string.
//
// Parameters:
//   - list: The slice of strings to search within.
//   - target: The string to search for.
//
// Returns:
//   - true if the target string is found in the list, false otherwise.
func contains(list []string, target string) bool {
	for _, item := range list {
		if item == target {
			return true
		}
	}
	return false
}
