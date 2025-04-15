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
	db, err := db.InitDB("mydb", provider)

	assert.NoError(t, err)
	assert.NotNil(t, db)

	db.Close()
	_, err = os.Stat(dbPath)
	assert.NoError(t, err)
}
