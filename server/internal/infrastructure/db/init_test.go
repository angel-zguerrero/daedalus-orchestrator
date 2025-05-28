package db_test

import (
	"deadalus-orch/shared/constants" // Added
	"os"
	"os/user" // Added
	"path/filepath"
	"strings" // Added
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require" // Added

	"deadalus-orch/server/internal/infrastructure/db"
)

// Fake provider para pruebas
type FakePathProvider struct {
	Path string
	Err  error
}

func (f FakePathProvider) GetDatabasePath() (string, error) {
	return f.Path, f.Err
}

func TestInitDB_CreatesDBAtCorrectPath(t *testing.T) {
	tmp := t.TempDir()

	provider := FakePathProvider{Path: tmp}

	dbPath := filepath.Join(tmp, "mydb")
	db, _, _, err := db.InitDB("mydb", provider, []string{db.AdminFC, db.MetaFC}, []string{})

	assert.NoError(t, err)
	assert.NotNil(t, db)

	db.Close()
	_, err = os.Stat(dbPath)
	assert.NoError(t, err)
}

func TestInitDB_ErrorIfDuplicateInSameCFList(t *testing.T) {
	tmp := t.TempDir()
	provider := FakePathProvider{Path: tmp}

	_, _, _, err := db.InitDB("testdb", provider, []string{"cf1", "cf1"}, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "duplicated names in columnFamilyNames")
}

func TestInitDB_ErrorIfDuplicateBetweenNormalAndTTL(t *testing.T) {
	tmp := t.TempDir()
	provider := FakePathProvider{Path: tmp}

	_, _, _, err := db.InitDB("testdb", provider, []string{"sharedcf"}, []string{"sharedcf"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exists in both normal and TTL sets")
}

func TestInitDB_OpensDBWithNormalAndTTLCFs(t *testing.T) {
	tmp := t.TempDir()
	provider := FakePathProvider{Path: tmp}

	dbInstance, normalCFs, ttlCFs, err := db.InitDB("testdb", provider, []string{"normal1"}, []string{"ttl1"})
	assert.NoError(t, err)
	assert.NotNil(t, dbInstance)
	assert.Contains(t, normalCFs, "normal1")
	assert.Contains(t, ttlCFs, "ttl1")
	dbInstance.Close()
}

func TestInitDB_ErrorOnInvalidPath(t *testing.T) {
	provider := FakePathProvider{
		Path: "",
		Err:  os.ErrInvalid,
	}

	_, _, _, err := db.InitDB("testdb", provider, []string{"cf1"}, []string{})
	assert.Error(t, err)
}

func TestDefaultPathProvider_DevelopmentEnv(t *testing.T) {
	t.Setenv(constants.EnvVarEnvKey, string(constants.DEVELOPMENT))

	currentUser, err := user.Current()
	require.NoError(t, err, "Failed to get current user")

	provider := db.DefaultPathProvider{}
	path, err := provider.GetDatabasePath()

	assert.NoError(t, err)
	expectedPathSuffix := filepath.Join(".daedalus", "data")
	assert.True(t, strings.HasSuffix(path, expectedPathSuffix), "Path should end with "+expectedPathSuffix)
	assert.True(t, strings.HasPrefix(path, currentUser.HomeDir), "Path should start with user's home directory")

	// Clean up created directory if it exists, be careful with this in real tests
	// For this test, we are primarily checking the path string, but if it creates, we can clean.
	// The function GetDatabasePath itself creates the dir.
	if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
		os.RemoveAll(filepath.Join(currentUser.HomeDir, ".daedalus")) // remove parent .daedalus
	}
}

func TestDefaultPathProvider_ProductionEnv(t *testing.T) {
	t.Setenv(constants.EnvVarEnvKey, string(constants.PRODUCTION))

	provider := db.DefaultPathProvider{}
	path, err := provider.GetDatabasePath()

	if err != nil {
		if os.IsPermission(err) {
			assert.Equal(t, "/var/lib/daedalus/data", path, "Path should be /var/lib/daedalus/data")
		} else {
			assert.Error(t, err, "mkdir /var/lib/daedalus: permission denied")
		}
	} else {
		assert.Equal(t, "/var/lib/daedalus/data", path, "Path should be /var/lib/daedalus/data")
		os.RemoveAll("/var/lib/daedalus")
	}
}
