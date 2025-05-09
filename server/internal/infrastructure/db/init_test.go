package db_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

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
