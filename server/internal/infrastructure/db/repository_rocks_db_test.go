package db_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/linxGnu/grocksdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"deadalus-orch/server/internal/infrastructure/db"
)

type testEntity struct {
	ID   string `orm:"primaryKey"`
	Name string `orm:"unique"`
}

func (testEntity) TableName() string {
	return "users"
}

func newRocksdbStore(t *testing.T) *db.RocksdbStore {
	tmpDir := t.TempDir()
	opts := grocksdb.NewDefaultOptions()
	opts.SetCreateIfMissing(true)
	opts.SetCreateIfMissingColumnFamilies(true)
	goOp := grocksdb.NewDefaultOptions()

	rocks, cfHs, err := grocksdb.OpenDbColumnFamilies(opts, tmpDir, []string{DefaultFC, TestFC}, []*grocksdb.Options{goOp, goOp})
	require.NoError(t, err)
	t.Cleanup(func() { rocks.Close() })

	columnFamilyNames, err := grocksdb.ListColumnFamilies(opts, tmpDir)
	require.NoError(t, err)

	cfMap := make(map[string]*grocksdb.ColumnFamilyHandle, len(columnFamilyNames))
	for i, name := range columnFamilyNames {
		cfMap[name] = cfHs[i]
	}

	return &db.RocksdbStore{
		DB:                     rocks,
		ColumnFamilyHandles:    cfMap,
		TTLColumnFamilyHandles: make(map[string]*grocksdb.ColumnFamilyHandle),
	}
}

func newTestRepository(t *testing.T) (*db.Repository[testEntity], error) {
	store := newRocksdbStore(t)
	iGF := NewTestIDGeneratorFactory([]string{"123", "456"})
	return db.NewRepository[testEntity](store, TestFC, "test_schema", iGF)
}

func newTestRepositoryDefaultIdGenerator(t *testing.T) (*db.Repository[testEntity], error) {
	store := newRocksdbStore(t)

	return db.NewRepository[testEntity](store, TestFC, "test_schema", &db.DefaultIDGeneratorFactory{})
}

func TestRepository_PutAndGet(t *testing.T) {
	repo, err := newTestRepository(t)
	require.NoError(t, err)
	entity := testEntity{ID: "----", Name: "Alice"}

	id, err := repo.Create(&entity)
	require.NoError(t, err)
	assert.Equal(t, id, "123")

	found, err := repo.FindByField("ID", "123")
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, "123", found.ID)
	assert.Equal(t, entity.Name, found.Name)

	found, err = repo.FindByField("Name", "Alice")
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, "123", found.ID)
	assert.Equal(t, entity.Name, found.Name)
}

func TestRepository_Get_NotFound(t *testing.T) {
	repo, err := newTestRepository(t)
	require.NoError(t, err)

	found, err := repo.FindByField("ID", "5566")
	require.NoError(t, err)
	assert.Nil(t, found)
}

func TestRepository_WriteBatch(t *testing.T) {
	repo, err := newTestRepository(t)
	require.NoError(t, err)

	a := testEntity{ID: "---", Name: "Alpha"}
	b := testEntity{ID: "---", Name: "Beta"}

	id, err := repo.Create(&a)
	assert.Equal(t, id, "123")
	require.NoError(t, err)
	id, err = repo.Create(&b)
	require.NoError(t, err)
	assert.Equal(t, id, "456")

	resA, err := repo.FindByField("ID", "123")
	require.NoError(t, err)
	assert.Equal(t, a.Name, resA.Name)

	resB, err := repo.FindByField("ID", "456")
	require.NoError(t, err)
	assert.Equal(t, b.Name, resB.Name)
}

func TestRepository_SearchByPatternPaginatedKV_MatchSingle(t *testing.T) {
	repo, err := newTestRepository(t)
	require.NoError(t, err)

	entity := testEntity{ID: "user:123:name", Name: "Alice"}
	id, err := repo.Create(&entity)
	assert.Equal(t, id, "123")
	require.NoError(t, err)

	results, err := repo.Find("Name=Alice", 1000, "")
	require.NoError(t, err)
	require.Len(t, results.Entities, 1)
	assert.Equal(t, entity.ID, results.Entities[0].ID)
	assert.Equal(t, entity.Name, results.Entities[0].Name)
}

func TestRepository_Update_SimpleFieldChange(t *testing.T) {
	repo, err := newTestRepository(t)
	require.NoError(t, err)

	// Create original entity
	entity := testEntity{ID: "---", Name: "Alice"}
	id, err := repo.Create(&entity)
	require.NoError(t, err)
	assert.Equal(t, "123", id)

	// Change Name
	entity.Name = "Alice Smith"
	updated, err := repo.Update("123", &entity)
	require.NoError(t, err)
	assert.True(t, updated)

	// Verify updated value
	res, err := repo.FindByField("ID", "123")
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Equal(t, "Alice Smith", res.Name)
}

func TestRepository_Update_UniqueIndexChange(t *testing.T) {
	repo, err := newTestRepository(t)
	require.NoError(t, err)

	entity := testEntity{ID: "---", Name: "Alice"}
	_, err = repo.Create(&entity)
	require.NoError(t, err)

	entity.Name = "Bob"
	updated, err := repo.Update("123", &entity)
	require.NoError(t, err)
	assert.True(t, updated)

	// Should no longer be found by old index
	old, err := repo.FindByField("Name", "Alice")
	require.NoError(t, err)
	assert.Nil(t, old)

	// Should now be found by new index
	newFound, err := repo.FindByField("Name", "Bob")
	require.NoError(t, err)
	require.NotNil(t, newFound)
	assert.Equal(t, "123", newFound.ID)
}

func TestRepository_Update_IndexCollisionShouldFail(t *testing.T) {
	repo, err := newTestRepository(t)
	require.NoError(t, err)

	// Create two users with different names
	a := testEntity{ID: "---", Name: "Alice"}
	b := testEntity{ID: "---", Name: "Bob"}

	_, err = repo.Create(&a)
	require.NoError(t, err)
	_, err = repo.Create(&b)
	require.NoError(t, err)

	// Try to rename Alice to "Bob" → should fail (unique collision)
	a.Name = "Bob"
	updated, err := repo.Update("123", &a)
	assert.Error(t, err)
	assert.False(t, updated)
}

func TestRepository_Update_NotFound(t *testing.T) {
	repo, err := newTestRepository(t)
	require.NoError(t, err)

	entity := testEntity{ID: "999", Name: "Zoe"}
	ok, err := repo.Update("999", &entity)
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestRepository_Delete_Success_RocksDB(t *testing.T) {
	repo, err := newTestRepository(t)
	require.NoError(t, err)

	// Crear entidad
	entity := testEntity{ID: "---", Name: "Charlie"}
	id, err := repo.Create(&entity)
	require.NoError(t, err)
	assert.Equal(t, "123", id)

	// Asegurar que se puede encontrar
	found, err := repo.FindByField("ID", "123")
	require.NoError(t, err)
	require.NotNil(t, found)

	// Eliminar entidad
	deleted, err := repo.Delete("123")
	require.NoError(t, err)
	assert.True(t, deleted)

	// Verificar que ya no se encuentra
	found, err = repo.FindByField("ID", "123")
	require.NoError(t, err)
	assert.Nil(t, found)

	// Verificar que el índice único también se eliminó
	found, err = repo.FindByField("Name", "Charlie")
	require.NoError(t, err)
	assert.Nil(t, found)
}

func TestRepository_Delete_NotFound_RocksDB(t *testing.T) {
	repo, err := newTestRepository(t)
	require.NoError(t, err)

	deleted, err := repo.Delete("nonexistent-id")
	require.NoError(t, err)
	assert.False(t, deleted)
}
func TestRepository_Find_FilteringAndPagination(t *testing.T) {
	repo, err := newTestRepositoryDefaultIdGenerator(t)
	require.NoError(t, err)

	// Crear 1200 entidades con Name único
	total := 1200
	for i := 0; i < total; i++ {
		name := fmt.Sprintf("Name_%04d", i)
		entity := testEntity{ID: "---", Name: name}
		_, err := repo.Create(&entity)
		require.NoError(t, err)
	}

	// === Filtro simple ===
	targetName := "Name_0003"
	results, err := repo.Find(fmt.Sprintf("Name=%s", targetName), 10, "")
	require.NoError(t, err)
	require.Len(t, results.Entities, 1)
	assert.Equal(t, targetName, results.Entities[0].Name)

	firstPage, err := repo.Find("Name=Name_0003 | Name=Name_0004 | Name=Name_0005", 2, "")
	require.NoError(t, err)
	assert.Len(t, firstPage.Entities, 2)
	assert.NotEmpty(t, firstPage.Cursor)

	secondPage, err := repo.Find("Name=Name_0003 | Name=Name_0004 | Name=Name_0005", 2, firstPage.Cursor)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(secondPage.Entities), 1) // solo hay 3 en total

	orFilter := "Name=Name_0007 | Name=Name_0010"
	orResults, err := repo.Find(orFilter, 10, "")
	require.NoError(t, err)
	assert.Len(t, orResults.Entities, 2)
	names := map[string]bool{
		"Name_0007": true,
		"Name_0010": true,
	}
	for _, e := range orResults.Entities {
		assert.True(t, names[e.Name])
	}

	andResults, err := repo.Find("Name=Name_0003 & Name=Name_0004", 10, "")
	require.NoError(t, err)
	assert.Empty(t, andResults.Entities)

	quoted, err := repo.Find("Name='Name_0003'", 10, "")
	require.NoError(t, err)
	require.Len(t, quoted.Entities, 1)
	assert.Equal(t, "Name_0003", quoted.Entities[0].Name)

	_, err = repo.Find("NameName_0003", 10, "")
	assert.Error(t, err)

	entity := testEntity{ID: "abc123", Name: "Zeta_Unique"}
	_, err = repo.Create(&entity)
	require.NoError(t, err)

	multiField := fmt.Sprintf("ID=%s & Name=%s", entity.ID, entity.Name)
	multiResults, err := repo.Find(multiField, 10, "")
	require.NoError(t, err)
	require.Len(t, multiResults.Entities, 1)
	assert.Equal(t, entity.ID, multiResults.Entities[0].ID)
	assert.Equal(t, entity.Name, multiResults.Entities[0].Name)
}

func TestRepository_Find_PaginationLoop(t *testing.T) {
	repo, err := newTestRepositoryDefaultIdGenerator(t)
	require.NoError(t, err)

	total := 100
	names := make(map[string]bool)
	for i := 0; i < total; i++ {
		name := fmt.Sprintf("Name_%04d", i)
		entity := testEntity{ID: "---", Name: name}
		_, err := repo.Create(&entity)
		require.NoError(t, err)
		names[name] = true
	}

	var filterParts []string
	for i := 10; i < 60; i++ {
		filterParts = append(filterParts, fmt.Sprintf("Name=Name_%04d", i))
	}
	filter := strings.Join(filterParts, " | ")

	limit := 7
	cursor := ""
	found := make(map[string]bool)

	for {
		page, err := repo.Find(filter, limit, cursor)
		require.NoError(t, err)

		for _, e := range page.Entities {
			assert.False(t, found[e.Name], "Entidad repetida en paginación: %s", e.Name)
			found[e.Name] = true
			assert.True(t, names[e.Name], "Entidad fuera del conjunto esperado: %s", e.Name)
		}

		if page.Cursor == "" {
			break
		}
		cursor = page.Cursor
	}
	assert.Len(t, found, 50)
}
