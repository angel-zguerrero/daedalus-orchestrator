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

func newRocksdbStore(t *testing.T) *db.RocksdbStore {
	tmpDir := t.TempDir()
	opts := grocksdb.NewDefaultOptions()
	opts.SetCreateIfMissing(true)
	opts.SetCreateIfMissingColumnFamilies(true)
	goOp := grocksdb.NewDefaultOptions()

	rocks, cfHs, err := grocksdb.OpenDbColumnFamilies(opts, tmpDir, []string{DefaultFC, TestFC, TemporalFC}, []*grocksdb.Options{goOp, goOp, goOp})
	require.NoError(t, err)
	t.Cleanup(func() { rocks.Close() })

	columnFamilyNames, err := grocksdb.ListColumnFamilies(opts, tmpDir)
	require.NoError(t, err)

	cfMap := make(map[string]*grocksdb.ColumnFamilyHandle, len(columnFamilyNames)-1)
	for i, name := range columnFamilyNames {
		if name != TemporalFC {
			cfMap[name] = cfHs[i]
		}
	}

	ttlCFMap := make(map[string]*grocksdb.ColumnFamilyHandle, len(columnFamilyNames)-2)
	for i, name := range columnFamilyNames {
		if name == TemporalFC {
			ttlCFMap[name] = cfHs[i]
		}
	}

	return &db.RocksdbStore{
		DB:                     rocks,
		ColumnFamilyHandles:    cfMap,
		TTLColumnFamilyHandles: ttlCFMap,
	}
}

func newTestRepository(t *testing.T) (*db.Repository[testEntity], error) {
	store := newRocksdbStore(t)
	iGF := NewTestIDGeneratorFactory([]string{"123", "456"})
	return db.NewRepository[testEntity](store, TestFC, "test_schema", iGF)
}

func newTestTTLRepository(t *testing.T) (*db.Repository[testEntity], *db.RocksdbStore, error) {
	store := newRocksdbStore(t)
	iGF := NewTestIDGeneratorFactory([]string{"123", "456"})
	repository, err := db.NewRepository[testEntity](store, TemporalFC, "test_schema", iGF)
	return repository, store, err
}

func newTestRepositorySpesificIds(t *testing.T, ids []string) (*db.Repository[testEntity], error) {
	store := newRocksdbStore(t)
	iGF := NewTestIDGeneratorFactory(ids)
	return db.NewRepository[testEntity](store, TestFC, "test_schema", iGF)
}

func newTestTTLRepositorySpesificIds(t *testing.T, ids []string) (*db.Repository[testEntity], error) {
	store := newRocksdbStore(t)
	iGF := NewTestIDGeneratorFactory(ids)
	return db.NewRepository[testEntity](store, TemporalFC, "test_schema", iGF)
}

func newTestRepositoryDefaultIdGenerator(t *testing.T) (*db.Repository[testEntity], error) {
	store := newRocksdbStore(t)

	return db.NewRepository[testEntity](store, TestFC, "test_schema", &db.DefaultIDGeneratorFactory{})
}

func newTestTTLRepositoryDefaultIdGenerator(t *testing.T) (*db.Repository[testEntity], db.KVStore, error) {
	store := newRocksdbStore(t)
	repository, err := db.NewRepository[testEntity](store, TemporalFC, "test_schema", &db.DefaultIDGeneratorFactory{})
	return repository, store, err
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

// --- Conditional Uniqueness Tests ---

type ConditionalUniqueEntityRocksDB struct {
	ID                     string `orm:"primary-key"`
	Name                   string
	UniqueValue            string `orm:"unique,ignore-is-true:ShouldIgnoreUniqueness"`
	ShouldIgnoreUniqueness bool
}

func (e ConditionalUniqueEntityRocksDB) TableName() string {
	return "cond_unique_rocks_db"
}

func newTestConditionalUniqueRepoRocksDB(t *testing.T, initialIDs []string) (*db.Repository[ConditionalUniqueEntityRocksDB], db.KVStore) {
	store := newRocksdbStore(t) // Uses a new temp dir for each call
	idGenerator := NewTestIDGeneratorFactory(initialIDs)
	repo, err := db.NewRepository[ConditionalUniqueEntityRocksDB](store, TestFC, "test_schema_cond", idGenerator)
	require.NoError(t, err, "Failed to create repository for ConditionalUniqueEntityRocksDB")
	return repo, store
}

func TestRocksDBConditionalUniquenessCreate(t *testing.T) {
	t.Run("IgnoreUniqueness", func(t *testing.T) {
		ids := []string{"id1", "id2", "id3"}
		repo, _ := newTestConditionalUniqueRepoRocksDB(t, ids)

		entity1 := &ConditionalUniqueEntityRocksDB{Name: "E1", UniqueValue: "uv1", ShouldIgnoreUniqueness: true}
		_, err := repo.Create(entity1)
		require.NoError(t, err, "Create entity1 should succeed")

		entity2 := &ConditionalUniqueEntityRocksDB{Name: "E2", UniqueValue: "uv1", ShouldIgnoreUniqueness: true}
		_, err = repo.Create(entity2)
		require.NoError(t, err, "Create entity2 with same UniqueValue (ignored) should succeed")

		entity3 := &ConditionalUniqueEntityRocksDB{Name: "E3", UniqueValue: "uv1", ShouldIgnoreUniqueness: false}
		_, err = repo.Create(entity3)
		require.NoError(t, err, "Create entity3 with same UniqueValue (enforced, but not previously by E1/E2) should succeed")

		// Verify all created
		e1, _ := repo.FindByField("ID", ids[0])
		require.NotNil(t, e1)
		assert.Equal(t, "uv1", e1.UniqueValue)

		e2, _ := repo.FindByField("ID", ids[1])
		require.NotNil(t, e2)
		assert.Equal(t, "uv1", e2.UniqueValue)

		e3, _ := repo.FindByField("ID", ids[2])
		require.NotNil(t, e3)
		assert.Equal(t, "uv1", e3.UniqueValue)
		assert.False(t, e3.ShouldIgnoreUniqueness)
	})

	t.Run("EnforceUniqueness", func(t *testing.T) {
		ids := []string{"id4", "id5", "id6"} // Fresh IDs for this subtest
		repo, _ := newTestConditionalUniqueRepoRocksDB(t, ids)

		entity4 := &ConditionalUniqueEntityRocksDB{Name: "E4", UniqueValue: "uv2", ShouldIgnoreUniqueness: false}
		_, err := repo.Create(entity4)
		require.NoError(t, err, "Create entity4 should succeed")

		entity5 := &ConditionalUniqueEntityRocksDB{Name: "E5", UniqueValue: "uv2", ShouldIgnoreUniqueness: false}
		_, err = repo.Create(entity5)
		require.Error(t, err, "Create entity5 with same UniqueValue (enforced) should fail")
		assert.Contains(t, err.Error(), "duplicate unique field")

		entity6 := &ConditionalUniqueEntityRocksDB{Name: "E6", UniqueValue: "uv2", ShouldIgnoreUniqueness: true}
		_, err = repo.Create(entity6)
		require.NoError(t, err, "Create entity6 with same UniqueValue (ignored) should succeed")

		e4, _ := repo.FindByField("ID", ids[0])
		require.NotNil(t, e4)
		assert.Equal(t, "uv2", e4.UniqueValue)

		e6, _ := repo.FindByField("ID", ids[2]) // id5 failed, so entity6 gets ids[2] if factory generates sequentially
		require.NotNil(t, e6)
		assert.Equal(t, "uv2", e6.UniqueValue)
		assert.True(t, e6.ShouldIgnoreUniqueness)
	})
}

func TestRocksDBConditionalUniquenessUpdate(t *testing.T) {
	t.Run("UpdateWithFlagTrueBypassesConflict", func(t *testing.T) {
		ids := []string{"idA", "idB"}
		repo, _ := newTestConditionalUniqueRepoRocksDB(t, ids)

		entityA := &ConditionalUniqueEntityRocksDB{ID: ids[0], Name: "EntityA", UniqueValue: "uva", ShouldIgnoreUniqueness: false}
		_, err := repo.Create(entityA)
		require.NoError(t, err)

		entityB := &ConditionalUniqueEntityRocksDB{ID: ids[1], Name: "EntityB", UniqueValue: "uvb", ShouldIgnoreUniqueness: false}
		_, err = repo.Create(entityB)
		require.NoError(t, err)

		// Update entityB to have UniqueValue "uva" (conflicts with A) but with ShouldIgnoreUniqueness = true
		entityB.UniqueValue = "uva"
		entityB.ShouldIgnoreUniqueness = true
		updated, err := repo.Update(entityB)
		require.NoError(t, err, "Update entityB should succeed")
		assert.True(t, updated)

		// Verify B is updated
		bUpdated, _ := repo.FindByField("ID", ids[1])
		require.NotNil(t, bUpdated)
		assert.Equal(t, "uva", bUpdated.UniqueValue)
		assert.True(t, bUpdated.ShouldIgnoreUniqueness)
	})

	t.Run("UpdateWithFlagFalseHitsConflict", func(t *testing.T) {
		ids := []string{"idC", "idD"}
		repo, _ := newTestConditionalUniqueRepoRocksDB(t, ids)

		entityC := &ConditionalUniqueEntityRocksDB{ID: ids[0], Name: "EntityC", UniqueValue: "uvc", ShouldIgnoreUniqueness: false}
		_, err := repo.Create(entityC)
		require.NoError(t, err)

		entityD := &ConditionalUniqueEntityRocksDB{ID: ids[1], Name: "EntityD", UniqueValue: "uvd", ShouldIgnoreUniqueness: false}
		_, err = repo.Create(entityD)
		require.NoError(t, err)

		// Attempt to update entityD to have UniqueValue "uvc" (conflicts with C) with ShouldIgnoreUniqueness = false
		entityD.UniqueValue = "uvc"
		entityD.ShouldIgnoreUniqueness = false
		updated, err := repo.Update(entityD)
		require.Error(t, err, "Update entityD should fail due to unique constraint")
		assert.False(t, updated)
		assert.Contains(t, err.Error(), "duplicate unique field")
	})

	t.Run("UpdateFlagFromFalseToTrueThenCreateNew", func(t *testing.T) {
		ids := []string{"idE", "idF"}
		repo, _ := newTestConditionalUniqueRepoRocksDB(t, ids)

		entityE := &ConditionalUniqueEntityRocksDB{ID: ids[0], Name: "EntityE", UniqueValue: "uve", ShouldIgnoreUniqueness: false}
		_, err := repo.Create(entityE)
		require.NoError(t, err)

		// Update entityE to set ShouldIgnoreUniqueness = true
		entityE.ShouldIgnoreUniqueness = true
		updated, err := repo.Update(entityE)
		require.NoError(t, err, "Update entityE should succeed")
		assert.True(t, updated)

		// Now, try to create entityF with UniqueValue "uve" and ShouldIgnoreUniqueness = false
		// This should succeed because entityE is no longer enforcing uniqueness on "uve"
		entityF := &ConditionalUniqueEntityRocksDB{ID: ids[1], Name: "EntityF", UniqueValue: "uve", ShouldIgnoreUniqueness: true}
		_, err = repo.Create(entityF)
		require.NoError(t, err, "Create entityF should succeed as E is ignoring uniqueness")

		// Verify F
		fCreated, _ := repo.FindByField("ID", ids[1])
		require.NotNil(t, fCreated)
		assert.Equal(t, "uve", fCreated.UniqueValue)
		assert.True(t, fCreated.ShouldIgnoreUniqueness)
	})
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
	entity := testEntity{ID: "123", Name: "Alice"}
	id, err := repo.Create(&entity)
	require.NoError(t, err)
	assert.Equal(t, "123", id)

	// Change Name
	entity.Name = "Alice Smith"
	updated, err := repo.Update(&entity)
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

	entity := testEntity{ID: "123", Name: "Alice"}
	_, err = repo.Create(&entity)
	require.NoError(t, err)

	entity.Name = "Bob"
	updated, err := repo.Update(&entity)
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
	a := testEntity{ID: "123", Name: "Alice"}
	b := testEntity{ID: "---", Name: "Bob"}

	_, err = repo.Create(&a)
	require.NoError(t, err)
	_, err = repo.Create(&b)
	require.NoError(t, err)

	// Try to rename Alice to "Bob" → should fail (unique collision)
	a.Name = "Bob"
	updated, err := repo.Update(&a)
	assert.Error(t, err)
	assert.False(t, updated)
}

func TestRepository_Update_NotFound(t *testing.T) {
	repo, err := newTestRepository(t)
	require.NoError(t, err)

	entity := testEntity{ID: "999", Name: "Zoe"}
	ok, err := repo.Update(&entity)
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
func TestRepository_Find_Operators(t *testing.T) {
	repo, err := newTestRepositoryDefaultIdGenerator(t)
	require.NoError(t, err)

	seed := []testEntity{
		{ID: "---", Name: "Ana", LastName: "Zuluaga", Age: 20},
		{ID: "---", Name: "Bea", LastName: "Yanez", Age: 30},
		{ID: "---", Name: "Cleo", LastName: "Ximenez", Age: 40},
		{ID: "---", Name: "Dana", LastName: "White", Age: 25},
		{ID: "---", Name: "Eva", LastName: "Velasco", Age: 35},
	}
	for _, e := range seed {
		_, err := repo.Create(&e)
		require.NoError(t, err)
	}

	t.Run("Equal operator", func(t *testing.T) {
		res, err := repo.Find("Age=30", 10, "")
		require.NoError(t, err)
		require.Len(t, res.Entities, 1)
		assert.Equal(t, "Bea", res.Entities[0].Name)
	})

	t.Run("Not equal operator", func(t *testing.T) {
		res, err := repo.Find("Age!=30", 10, "")
		require.NoError(t, err)
		assert.Len(t, res.Entities, 4)
		for _, e := range res.Entities {
			assert.NotEqual(t, 30, e.Age)
		}
	})

	t.Run("Greater than operator", func(t *testing.T) {
		res, err := repo.Find("Age>30", 10, "")
		require.NoError(t, err)
		assert.Len(t, res.Entities, 2)
		assert.ElementsMatch(t, []string{"Cleo", "Eva"}, []string{res.Entities[0].Name, res.Entities[1].Name})
	})

	t.Run("Greater than or equal", func(t *testing.T) {
		res, err := repo.Find("Age>=30", 10, "")
		require.NoError(t, err)
		assert.Len(t, res.Entities, 3)
	})

	t.Run("Less than operator", func(t *testing.T) {
		res, err := repo.Find("Age<30", 10, "")
		require.NoError(t, err)
		assert.Len(t, res.Entities, 2)
	})

	t.Run("Less than or equal", func(t *testing.T) {
		res, err := repo.Find("Age<=25", 10, "")
		require.NoError(t, err)
		assert.Len(t, res.Entities, 2)
	})

	t.Run("LIKE operator - prefix", func(t *testing.T) {
		res, err := repo.Find("LastName LIKE Z*", 10, "")
		require.NoError(t, err)
		require.Len(t, res.Entities, 1)
		assert.Equal(t, "Ana", res.Entities[0].Name)
	})

	t.Run("LIKE operator - suffix", func(t *testing.T) {
		res, err := repo.Find("LastName LIKE *nez", 10, "")
		require.NoError(t, err)
		require.Len(t, res.Entities, 2)
		assert.ElementsMatch(t, []string{"Cleo", "Bea"}, []string{res.Entities[0].Name, res.Entities[1].Name})
	})

	t.Run("LIKE operator - contains", func(t *testing.T) {
		res, err := repo.Find("LastName LIKE *ela*", 10, "")
		require.NoError(t, err)
		assert.Len(t, res.Entities, 1)
		assert.Equal(t, "Eva", res.Entities[0].Name)
	})

	t.Run("BETWEEN operator", func(t *testing.T) {
		res, err := repo.Find("Age BETWEEN 25 AND 35", 10, "")
		require.NoError(t, err)
		assert.Len(t, res.Entities, 3)
		names := []string{res.Entities[0].Name, res.Entities[1].Name, res.Entities[2].Name}
		assert.ElementsMatch(t, []string{"Bea", "Dana", "Eva"}, names)
	})

	t.Run("Combined operators", func(t *testing.T) {
		res, err := repo.Find("Age>=25 & Age<=35", 10, "")
		require.NoError(t, err)
		assert.Len(t, res.Entities, 3)
	})
}
func TestRepository_Find_LargeDatasetWithPaginationAndComplexFilters(t *testing.T) {
	repo, err := newTestRepositoryDefaultIdGenerator(t)
	require.NoError(t, err)

	total := 2500
	names := make(map[string]bool)
	ages := []int{18, 25, 30, 35, 40, 45, 50}
	for i := 0; i < total; i++ {
		name := fmt.Sprintf("User_%04d", i)
		age := ages[i%len(ages)]
		entity := testEntity{
			ID:       "---",
			Name:     name,
			LastName: fmt.Sprintf("Last_%04d", i),
			Age:      age,
		}
		_, err := repo.Create(&entity)
		require.NoError(t, err)
		names[name] = true
	}

	t.Run("Simple pagination loop with 100 per page", func(t *testing.T) {
		cursor := ""
		found := make(map[string]bool)
		limit := 100

		for {
			res, err := repo.Find("Age>=25", limit, cursor)
			require.NoError(t, err)

			for _, e := range res.Entities {
				assert.GreaterOrEqual(t, e.Age, 25)
				assert.False(t, found[e.Name], "Duplicated entity: %s", e.Name)
				found[e.Name] = true
			}
			if res.Cursor == "" {
				break
			}
			cursor = res.Cursor
		}
		totalExpected := 2500 * 6 / 7 // 6 edades >= 25 de 7 totales
		assert.Len(t, found, totalExpected)
	})

	t.Run("Complex OR filter with pagination", func(t *testing.T) {
		filter := []string{}
		for i := 0; i < 50; i++ {
			filter = append(filter, fmt.Sprintf("Name=User_%04d", i))
		}
		filterStr := strings.Join(filter, " | ")

		cursor := ""
		found := map[string]bool{}
		limit := 10

		for {
			res, err := repo.Find(filterStr, limit, cursor)
			require.NoError(t, err)
			for _, e := range res.Entities {
				found[e.Name] = true
			}
			if res.Cursor == "" {
				break
			}
			cursor = res.Cursor
		}
		assert.Len(t, found, 50)
	})

	t.Run("AND filter with age and name", func(t *testing.T) {
		// Name should exist and age should match
		res, err := repo.Find("Name=User_0001 & Age=25", 10, "")
		require.NoError(t, err)
		if len(res.Entities) == 1 {
			assert.Equal(t, "User_0001", res.Entities[0].Name)
			assert.Equal(t, 25, res.Entities[0].Age)
		} else {
			assert.Empty(t, res.Entities)
		}
	})

	t.Run("LIKE operator with many matches", func(t *testing.T) {
		res, err := repo.Find("LastName LIKE Last_1*", 100, "")
		require.NoError(t, err)
		assert.Greater(t, len(res.Entities), 10)
		for _, e := range res.Entities {
			assert.True(t, strings.HasPrefix(e.LastName, "Last_1"))
		}
	})
}

func TestRepository_Find_ComplexNestedFilters(t *testing.T) {
	repo, err := newTestRepositoryDefaultIdGenerator(t)
	require.NoError(t, err)

	// Semilla para pruebas
	seed := []testEntity{
		{ID: "---", Name: "Ana", LastName: "Zuluaga", Age: 20},
		{ID: "---", Name: "Bea", LastName: "Yanez", Age: 30},
		{ID: "---", Name: "Cleo", LastName: "Ximenez", Age: 40},
		{ID: "---", Name: "Dana", LastName: "White", Age: 25},
		{ID: "---", Name: "Eva", LastName: "Velasco", Age: 35},
		{ID: "---", Name: "Fina", LastName: "White", Age: 50},
	}
	for _, e := range seed {
		_, err := repo.Create(&e)
		require.NoError(t, err)
	}

	tests := []struct {
		name     string
		filter   string
		expected []string // Nombres esperados en el resultado
	}{
		{
			name:     "Simple nested AND",
			filter:   "(Age>20 & Age<40)",
			expected: []string{"Bea", "Dana", "Eva"},
		},
		{
			name:     "Nested AND OR mix",
			filter:   "(Age>20 & (LastName LIKE W* | LastName LIKE V*))",
			expected: []string{"Dana", "Eva", "Fina"},
		},
		{
			name:     "Nested OR with NOT EQUAL",
			filter:   "(Age!=30 | Name=Ana) & LastName LIKE *ez*",
			expected: []string{"Cleo"},
		},
		{
			name:     "Complex nested with BETWEEN and OR",
			filter:   "((Age BETWEEN 25 AND 40) & (LastName LIKE Y* | LastName LIKE X*)) | Name=Ana",
			expected: []string{"Ana", "Bea", "Cleo"},
		},
		{
			name:     "Deep nesting with multiple operators",
			filter:   "((Age>=25 & Age<=35) & ((Name=Bea | Name=Dana) | LastName=White))",
			expected: []string{"Bea", "Dana"},
		},
		{
			name:     "Negative nested case - no match",
			filter:   "(Age<20 & (Name=Bea | Name=Cleo))",
			expected: []string{},
		},
		{
			name:     "Multiple nested OR groups",
			filter:   "(Name=Ana | Name=Eva) & (LastName LIKE Z* | LastName LIKE V*)",
			expected: []string{"Ana", "Eva"},
		},
		{
			name:     "Nested with NOT equal and LIKE",
			filter:   "Age!=20 & LastName LIKE *ite",
			expected: []string{"Dana", "Fina"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := repo.Find(tt.filter, 100, "")
			require.NoError(t, err)

			var gotNames []string
			for _, e := range res.Entities {
				gotNames = append(gotNames, e.Name)
			}
			assert.ElementsMatch(t, tt.expected, gotNames)
		})
	}
}

func TestRepository_BulkCreate(t *testing.T) {
	t.Run("Basic bulk insert", func(t *testing.T) {
		repo, err := newTestRepositoryDefaultIdGenerator(t)
		require.NoError(t, err)

		entities := []*testEntity{
			{ID: "---", Name: "UserA"},
			{ID: "---", Name: "UserB"},
			{ID: "---", Name: "UserC"},
		}
		ids, err := repo.BulkCreate(entities)
		require.NoError(t, err)
		assert.Len(t, ids, 3)

		for _, id := range ids {
			found, err := repo.FindByField("ID", id)
			require.NoError(t, err)
			assert.NotNil(t, found)
		}
	})

	t.Run("Duplicate unique index should fail", func(t *testing.T) {
		repo, err := newTestRepositoryDefaultIdGenerator(t)
		require.NoError(t, err)

		first := []*testEntity{
			{ID: "---", Name: "UniqueName1"},
			{ID: "---", Name: "UniqueName2"},
		}
		_, err = repo.BulkCreate(first)
		require.NoError(t, err)

		second := []*testEntity{
			{ID: "---", Name: "UniqueName2"}, // duplicate
			{ID: "---", Name: "UniqueName3"},
		}
		_, err = repo.BulkCreate(second)
		assert.Error(t, err)
		require.Contains(t, err.Error(), "duplicate")
	})

	t.Run("Empty list should return empty result", func(t *testing.T) {
		repo, err := newTestRepositoryDefaultIdGenerator(t)
		require.NoError(t, err)

		ids, err := repo.BulkCreate([]*testEntity{})
		require.NoError(t, err)
		assert.Empty(t, ids)
	})

	t.Run("Can bulk insert large batch", func(t *testing.T) {
		repo, err := newTestRepositoryDefaultIdGenerator(t)
		require.NoError(t, err)

		var bulk []*testEntity
		for i := 0; i < 500; i++ {
			bulk = append(bulk, &testEntity{ID: "---", Name: fmt.Sprintf("Bulk_%d", i)})
		}
		ids, err := repo.BulkCreate(bulk)
		require.NoError(t, err)
		assert.Len(t, ids, 500)

		// Spot check
		found, err := repo.FindByField("Name", "Bulk_42")
		require.NoError(t, err)
		require.NotNil(t, found)
	})

	t.Run("Partially duplicate should fail all", func(t *testing.T) {
		repo, err := newTestRepositoryDefaultIdGenerator(t)
		require.NoError(t, err)

		_, err = repo.Create(&testEntity{ID: "---", Name: "ConflictName"})

		conflicting := []*testEntity{
			{ID: "---", Name: "NewName1"},
			{ID: "---", Name: "ConflictName"},
			{ID: "---", Name: "NewName2"},
		}
		_, err = repo.BulkCreate(conflicting)
		assert.Error(t, err)
		require.Contains(t, err.Error(), "duplicate")
	})
	t.Run("should fail when input slice contains duplicate unique index values", func(t *testing.T) {
		repo, err := newTestRepositoryDefaultIdGenerator(t)

		batch := []*testEntity{
			{Name: "duplicate-name", Age: 100},
			{Name: "duplicate-name", Age: 200}, // mismo Name
		}

		_, err = repo.BulkCreate(batch)
		require.Error(t, err)
		require.Contains(t, err.Error(), "duplicate") // adapta esto según el mensaje real
	})
}

func TestRepository_BulkDelete(t *testing.T) {
	t.Run("Delete multiple existing entities", func(t *testing.T) {
		ids := []string{"123", "456", "789"}
		repo, err := newTestRepositorySpesificIds(t, ids)
		require.NoError(t, err)

		for _, id := range ids {
			entity := testEntity{ID: id, Name: "Name_" + id}
			_, err := repo.Create(&entity)
			require.NoError(t, err)
		}

		for _, id := range ids {
			found, err := repo.FindByField("ID", id)
			require.NoError(t, err)
			require.NotNil(t, found)
		}

		_, err = repo.BulkDelete(ids)
		require.NoError(t, err)

		for _, id := range ids {
			found, err := repo.FindByField("ID", id)
			require.NoError(t, err)
			assert.Nil(t, found)
		}
	})

	t.Run("Bulk delete with some non-existing IDs", func(t *testing.T) {
		repo, err := newTestRepository(t)
		require.NoError(t, err)

		entity := testEntity{ID: "999", Name: "Alive"}
		_, err = repo.Create(&entity)
		require.NoError(t, err)

		// Mixed list: existing and non-existing
		ids := []string{"999", "does_not_exist", "also_missing"}
		_, err = repo.BulkDelete(ids)
		require.NoError(t, err)

		// Ensure the one that existed is deleted
		found, err := repo.FindByField("ID", "999")
		require.NoError(t, err)
		assert.Nil(t, found)
	})

	t.Run("Empty list should not fail", func(t *testing.T) {
		repo, err := newTestRepository(t)
		require.NoError(t, err)
		_, err = repo.BulkDelete([]string{})
		require.NoError(t, err)
	})

	t.Run("Bulk delete should also remove unique indices", func(t *testing.T) {
		ids := []string{"111", "222"}
		repo, err := newTestRepositorySpesificIds(t, ids)
		require.NoError(t, err)

		a := testEntity{ID: "111", Name: "Alpha"}
		b := testEntity{ID: "222", Name: "Beta"}
		_, err = repo.Create(&a)
		require.NoError(t, err)
		_, err = repo.Create(&b)
		require.NoError(t, err)

		_, err = repo.BulkDelete([]string{"111", "222"})
		require.NoError(t, err)

		foundA, err := repo.FindByField("Name", "Alpha")
		require.NoError(t, err)
		assert.Nil(t, foundA)

		foundB, err := repo.FindByField("Name", "Beta")
		require.NoError(t, err)
		assert.Nil(t, foundB)
	})
}
func TestRepository_BulkUpdate(t *testing.T) {
	t.Run("Basic bulk update", func(t *testing.T) {
		repo, err := newTestRepositoryDefaultIdGenerator(t)
		require.NoError(t, err)

		// Creamos los datos base
		original := []*testEntity{
			{ID: "---", Name: "UserA"},
			{ID: "---", Name: "UserB"},
			{ID: "---", Name: "UserC"},
		}
		ids, err := repo.BulkCreate(original)
		require.NoError(t, err)
		require.Len(t, ids, 3)

		// Preparamos la actualización
		updated := []*testEntity{
			{ID: ids[0], Name: "UpdatedA"},
			{ID: ids[1], Name: "UpdatedB"},
			{ID: ids[2], Name: "UpdatedC"},
		}
		result, err := repo.BulkUpdate(updated)
		require.NoError(t, err)
		assert.Equal(t, []bool{true, true, true}, result)

		for i, id := range ids {
			entity, err := repo.FindByField("ID", id)
			require.NoError(t, err)
			assert.Equal(t, updated[i].Name, entity.Name)
		}
	})

	t.Run("Empty input should return empty result", func(t *testing.T) {
		repo, err := newTestRepositoryDefaultIdGenerator(t)
		require.NoError(t, err)

		result, err := repo.BulkUpdate([]*testEntity{})
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("Should return false for missing records", func(t *testing.T) {
		repo, err := newTestRepositoryDefaultIdGenerator(t)
		require.NoError(t, err)

		// Insert uno solo
		entity := &testEntity{ID: "---", Name: "UserX"}
		createdID, err := repo.Create(entity)
		require.NoError(t, err)

		// Uno existe, otro no
		toUpdate := []*testEntity{
			{ID: createdID, Name: "UpdatedX"},
			{ID: "non-existent-id", Name: "ShouldFail"},
		}
		result, err := repo.BulkUpdate(toUpdate)
		require.NoError(t, err)
		assert.Equal(t, []bool{true, false}, result)
	})

	t.Run("Update should not insert new if ID not exists", func(t *testing.T) {
		repo, err := newTestRepositoryDefaultIdGenerator(t)
		require.NoError(t, err)

		toUpdate := []*testEntity{
			{ID: "ghost-id", Name: "Ghost"},
		}
		result, err := repo.BulkUpdate(toUpdate)
		require.NoError(t, err)
		assert.Equal(t, []bool{false}, result)

		found, err := repo.FindByField("ID", "ghost-id")
		require.NoError(t, err)
		assert.Nil(t, found)
	})

	t.Run("Can bulk update large batch", func(t *testing.T) {
		repo, err := newTestRepositoryDefaultIdGenerator(t)
		require.NoError(t, err)

		var original []*testEntity
		for i := 0; i < 500; i++ {
			original = append(original, &testEntity{ID: "---", Name: fmt.Sprintf("User_%d", i)})
		}
		ids, err := repo.BulkCreate(original)
		require.NoError(t, err)
		require.Len(t, ids, 500)

		var updated []*testEntity
		for i, id := range ids {
			updated = append(updated, &testEntity{ID: id, Name: fmt.Sprintf("Updated_%d", i)})
		}
		result, err := repo.BulkUpdate(updated)
		require.NoError(t, err)
		assert.Len(t, result, 500)
		for i := range result {
			assert.True(t, result[i], "index %d should be true", i)
		}

		// Spot check
		check, err := repo.FindByField("Name", "Updated_42")
		require.NoError(t, err)
		assert.NotNil(t, check)
	})

	t.Run("Duplicate IDs in input should update once", func(t *testing.T) {
		repo, err := newTestRepositoryDefaultIdGenerator(t)
		require.NoError(t, err)

		entity := &testEntity{ID: "---", Name: "Original"}
		id, err := repo.Create(entity)
		require.NoError(t, err)

		// Duplicado el mismo ID dos veces, con valores diferentes
		toUpdate := []*testEntity{
			{ID: id, Name: "FirstUpdate"},
			{ID: id, Name: "SecondUpdate"},
		}
		result, err := repo.BulkUpdate(toUpdate)
		require.NoError(t, err)
		assert.Equal(t, []bool{true, true}, result)

		final, err := repo.FindByField("ID", id)
		require.NoError(t, err)
		assert.Equal(t, "SecondUpdate", final.Name) // el último debe prevalecer
	})

	t.Run("Nil entries should be skipped or fail gracefully", func(t *testing.T) {
		repo, err := newTestRepositoryDefaultIdGenerator(t)
		require.NoError(t, err)

		entity := &testEntity{ID: "---", Name: "Valid"}
		id, err := repo.Create(entity)
		require.NoError(t, err)

		updates := []*testEntity{
			nil,
			{ID: id, Name: "ValidUpdate"},
		}
		result, err := repo.BulkUpdate(updates)
		require.NoError(t, err)
		assert.Equal(t, []bool{false, true}, result) // nil => false, válido => true

		final, err := repo.FindByField("ID", id)
		require.NoError(t, err)
		assert.Equal(t, "ValidUpdate", final.Name)
	})
}

type MetaN struct {
	Tag         string `orm:"unique"`
	ConfigCode  int
	Description string
}

type UserComplexN struct {
	ID     string `orm:"primary-key"`
	Email  string `orm:"unique"`
	Meta   MetaN  // Named field
	Status string
}

func (UserComplexN) TableName() string {
	return "users_complex_n"
}
func newTestNRepository(t *testing.T) (*db.Repository[UserComplexN], error) {
	store := newRocksdbStore(t)
	iGF := NewTestIDGeneratorFactory([]string{"123", "456"})
	return db.NewRepository[UserComplexN](store, TestFC, "test_schema", iGF)
}

func newNestedEntityTestRepositoryRocksDB(t *testing.T) (*db.Repository[NestedEntityTest], error) {
	store := newRocksdbStore(t)                                                        // Assumes newRocksdbStore is defined in this file
	iGF := NewTestIDGeneratorFactory([]string{"nid1", "nid2", "nid3", "nid4", "nid5"}) // Example IDs
	return db.NewRepository[NestedEntityTest](store, TestFC, "nested_entity_schema", iGF)
}

type NestedMetaTest struct {
	UniqueID    string `orm:"unique"`
	OTValue     string
	Description string
}

type NestedEntityTest struct {
	ID   string `orm:"primary-key"`
	Data string
	Meta NestedMetaTest
}

func (NestedEntityTest) TableName() string {
	return "nested_entities_test"
}

func TestRepository_PutAndGet_Nested(t *testing.T) {
	repo, err := newTestNRepository(t)
	require.NoError(t, err)
	entity := UserComplexN{ID: "----", Email: "Alice@x.com", Status: "active", Meta: MetaN{
		Tag:         "t1",
		ConfigCode:  55,
		Description: "None",
	}}

	id, err := repo.Create(&entity)
	require.NoError(t, err)
	assert.Equal(t, id, "123")

	found, err := repo.FindByField("ID", "123")
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, "123", found.ID)
	assert.Equal(t, entity.Email, found.Email)

	found, err = repo.FindByField("Email", "Alice@x.com")
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, "123", found.ID)
	assert.Equal(t, entity.Email, found.Email)

	found, err = repo.FindByField("Meta.Tag", "t1")
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, "123", found.ID)
	assert.Equal(t, entity.Email, found.Email)

	found, err = repo.FindByField("Meta.Tag1", "t1")
	require.Error(t, err)
	require.Contains(t, err.Error(), "Unknown field Meta.Tag1")
}

func TestRepository_BulkCreate_Nested_RocksDB(t *testing.T) {
	repo, err := newNestedEntityTestRepositoryRocksDB(t)
	require.NoError(t, err, "Failed to create repository for NestedEntityTest")

	t.Run("Successful bulk creation with nested structs", func(t *testing.T) {
		entities := []*NestedEntityTest{
			{ID: "---", Data: "Entity1", Meta: NestedMetaTest{UniqueID: "uniqueA1", OTValue: "ttl1", Description: "Desc1"}},
			{ID: "---", Data: "Entity2", Meta: NestedMetaTest{UniqueID: "uniqueA2", OTValue: "ttl2", Description: "Desc2"}},
			{ID: "---", Data: "Entity3", Meta: NestedMetaTest{UniqueID: "uniqueA3", OTValue: "ttl3", Description: "Desc3"}},
		}

		ids, err := repo.BulkCreate(entities)
		require.NoError(t, err, "BulkCreate failed for valid nested entities")
		require.Len(t, ids, len(entities), "BulkCreate should return an ID for each entity")

		for i, id := range ids {
			require.NotEmpty(t, id, "Expected a non-empty ID for entity %d", i)
			entities[i].ID = id // Assign returned ID for later checks

			found, err := repo.FindByField("ID", id)
			require.NoError(t, err, "FindByField by ID failed for created entity %s", id)
			require.NotNil(t, found, "Should find entity by ID %s", id)
			assert.Equal(t, entities[i].Data, found.Data, "Data field mismatch for entity %s", id)
			assert.Equal(t, entities[i].Meta.UniqueID, found.Meta.UniqueID, "Meta.UniqueID field mismatch for entity %s", id)
			assert.Equal(t, entities[i].Meta.OTValue, found.Meta.OTValue, "Meta.TTLValue field mismatch for entity %s", id)
			assert.Equal(t, entities[i].Meta.Description, found.Meta.Description, "Meta.Description field mismatch for entity %s", id)
		}

		// Verify finding by nested unique index
		foundByNestedUnique, err := repo.FindByField("Meta.UniqueID", "uniqueA2")
		require.NoError(t, err, "FindByField by Meta.UniqueID failed")
		require.NotNil(t, foundByNestedUnique, "Should find entity by Meta.UniqueID 'uniqueA2'")
		assert.Equal(t, entities[1].ID, foundByNestedUnique.ID, "ID mismatch when finding by Meta.UniqueID")
		assert.Equal(t, "Entity2", foundByNestedUnique.Data)
		assert.Equal(t, "uniqueA2", foundByNestedUnique.Meta.UniqueID)
	})

	t.Run("Bulk creation with duplicate UniqueID within the batch", func(t *testing.T) {
		// Need a fresh repo instance or ensure DB is clean if tests share state,
		// but newNestedEntityTestRepositoryRocksDB creates a new store each time.
		repoFresh, err := newNestedEntityTestRepositoryRocksDB(t)
		require.NoError(t, err)

		entities := []*NestedEntityTest{
			{ID: "---", Data: "EntityX", Meta: NestedMetaTest{UniqueID: "duplicateKeyInBatch", OTValue: "ttlX", Description: "DescX"}},
			{ID: "---", Data: "EntityY", Meta: NestedMetaTest{UniqueID: "anotherUniqueInBatch", OTValue: "ttlY", Description: "DescY"}},
			{ID: "---", Data: "EntityZ", Meta: NestedMetaTest{UniqueID: "duplicateKeyInBatch", OTValue: "ttlZ", Description: "DescZ"}}, // Duplicate UniqueID
		}

		ids, err := repoFresh.BulkCreate(entities)
		require.Error(t, err, "BulkCreate should fail due to duplicate UniqueID within the batch")
		assert.Nil(t, ids, "IDs should be nil on batch creation failure")
		// The exact error message depends on the implementation, but it should indicate a unique constraint violation.
		// Example: assert.Contains(t, err.Error(), "duplicate key", "Error message should indicate a duplicate key problem")
		// Or, more generically for unique constraints:
		require.Contains(t, err.Error(), "duplicate", "Error message should indicate a unique constraint violation")

		// Verify no entities were partially inserted (transactional behavior)
		found, err := repoFresh.FindByField("Meta.UniqueID", "duplicateKeyInBatch")
		require.NoError(t, err)
		assert.Nil(t, found, "No entity should be found with the conflicting UniqueID if batch failed")

		found, err = repoFresh.FindByField("Meta.UniqueID", "anotherUniqueInBatch")
		require.NoError(t, err)
		assert.Nil(t, found, "No entity should be found with a non-conflicting UniqueID if batch failed")
	})

	t.Run("Bulk creation with UniqueID conflicting with existing data", func(t *testing.T) {
		repoClean, err := newNestedEntityTestRepositoryRocksDB(t) // Fresh repo
		require.NoError(t, err)

		// Pre-existing entity
		existingEntity := NestedEntityTest{ID: "---", Data: "ExistingData", Meta: NestedMetaTest{UniqueID: "conflictWithExisting", OTValue: "ttlE", Description: "DescE"}}
		_, err = repoClean.Create(&existingEntity) // Use Create for single setup
		require.NoError(t, err, "Setup: Failed to create initial entity")

		entitiesToBulkCreate := []*NestedEntityTest{
			{ID: "---", Data: "NewEntity1", Meta: NestedMetaTest{UniqueID: "newUnique1", OTValue: "ttlN1", Description: "DescN1"}},
			{ID: "---", Data: "NewEntity2Conflicting", Meta: NestedMetaTest{UniqueID: "conflictWithExisting", OTValue: "ttlN2", Description: "DescN2"}}, // Conflicts with existingEntity.Meta.UniqueID
			{ID: "---", Data: "NewEntity3", Meta: NestedMetaTest{UniqueID: "newUnique3", OTValue: "ttlN3", Description: "DescN3"}},
		}

		ids, err := repoClean.BulkCreate(entitiesToBulkCreate)
		require.Error(t, err, "BulkCreate should fail due to conflict with existing UniqueID")
		assert.Nil(t, ids, "IDs should be nil on batch creation failure due to existing conflict")
		require.Contains(t, err.Error(), "duplicate", "Error message should indicate a unique constraint violation")

		// Verify the conflicting entity was not inserted and non-conflicting ones from the batch also weren't (if transactional)
		foundNew, err := repoClean.FindByField("Meta.UniqueID", "newUnique1")
		require.NoError(t, err)
		assert.Nil(t, foundNew, "Non-conflicting entity from failed batch should not be inserted")

		// Verify existing entity is still there
		foundExisting, err := repoClean.FindByField("Meta.UniqueID", "conflictWithExisting")
		require.NoError(t, err)
		require.NotNil(t, foundExisting, "Original entity with the conflicting key should still exist")
		assert.Equal(t, existingEntity.Data, foundExisting.Data)
	})
}

func TestRepository_BulkUpdate_Nested_RocksDB(t *testing.T) {
	t.Run("Successful bulk update of nested structs", func(t *testing.T) {
		repo, err := newNestedEntityTestRepositoryRocksDB(t) // Uses specific IDs: nid1, nid2, nid3, ...
		require.NoError(t, err)

		initialEntities := []*NestedEntityTest{
			{ID: "---", Data: "DataOne", Meta: NestedMetaTest{UniqueID: "uniqueU1", OTValue: "ttlU1", Description: "DescU1"}},
			{ID: "---", Data: "DataTwo", Meta: NestedMetaTest{UniqueID: "uniqueU2", OTValue: "ttlU2", Description: "DescU2"}},
		}
		createdIds, err := repo.BulkCreate(initialEntities)
		require.NoError(t, err)
		require.Len(t, createdIds, 2)

		// Prepare updates
		updatedEntities := []*NestedEntityTest{
			{ID: createdIds[0], Data: "DataOneUpdated", Meta: NestedMetaTest{UniqueID: "uniqueU1_new", OTValue: "ttlU1_new", Description: "DescU1_new"}},
			{ID: createdIds[1], Data: "DataTwoUpdated", Meta: NestedMetaTest{UniqueID: "uniqueU2_new", OTValue: "ttlU2_new", Description: "DescU2_new"}},
		}

		results, err := repo.BulkUpdate(updatedEntities)
		require.NoError(t, err, "BulkUpdate failed for valid nested entity updates")
		require.Len(t, results, len(updatedEntities), "BulkUpdate should return a result for each entity")
		for i, success := range results {
			assert.True(t, success, "Expected update for entity ID %s to succeed", updatedEntities[i].ID)
		}

		// Verify updates
		for i, updatedEntity := range updatedEntities {
			found, err := repo.FindByField("ID", updatedEntity.ID)
			require.NoError(t, err)
			require.NotNil(t, found)
			assert.Equal(t, updatedEntity.Data, found.Data)
			assert.Equal(t, updatedEntity.Meta.UniqueID, found.Meta.UniqueID)
			assert.Equal(t, updatedEntity.Meta.OTValue, found.Meta.OTValue)
			assert.Equal(t, updatedEntity.Meta.Description, found.Meta.Description)

			// Verify old unique index is gone
			oldUniqueValue := initialEntities[i].Meta.UniqueID
			foundByOldUnique, err := repo.FindByField("Meta.UniqueID", oldUniqueValue)
			require.NoError(t, err)
			assert.Nil(t, foundByOldUnique, "Entity should not be found by old Meta.UniqueID %s", oldUniqueValue)

			// Verify new unique index is present
			foundByNewUnique, err := repo.FindByField("Meta.UniqueID", updatedEntity.Meta.UniqueID)
			require.NoError(t, err)
			require.NotNil(t, foundByNewUnique, "Entity should be found by new Meta.UniqueID %s", updatedEntity.Meta.UniqueID)
			assert.Equal(t, updatedEntity.ID, foundByNewUnique.ID)
		}
	})

	t.Run("Bulk update with UniqueID conflict within the batch", func(t *testing.T) {
		repo, err := newNestedEntityTestRepositoryRocksDB(t)
		require.NoError(t, err)

		initialEntities := []*NestedEntityTest{
			{ID: "---", Data: "Alpha", Meta: NestedMetaTest{UniqueID: "alphaUnique", OTValue: "ttlA", Description: "DescA"}},
			{ID: "---", Data: "Beta", Meta: NestedMetaTest{UniqueID: "betaUnique", OTValue: "ttlB", Description: "DescB"}},
		}
		createdIds, err := repo.BulkCreate(initialEntities)
		require.NoError(t, err)
		require.Len(t, createdIds, 2)
		initialEntities[0].ID = createdIds[0]
		initialEntities[1].ID = createdIds[1]

		// Try to update both to have the same UniqueID
		conflictingUpdates := []*NestedEntityTest{
			{ID: createdIds[0], Data: "AlphaUpdated", Meta: NestedMetaTest{UniqueID: "conflictKey", OTValue: "ttlA_new", Description: "DescA_new"}},
			{ID: createdIds[1], Data: "BetaUpdated", Meta: NestedMetaTest{UniqueID: "conflictKey", OTValue: "ttlB_new", Description: "DescB_new"}},
		}

		_, err = repo.BulkUpdate(conflictingUpdates)
		require.Error(t, err, "BulkUpdate should fail due to UniqueID conflict within the batch")
		// The exact behavior of `results` on error might vary. Some implementations might return nil, others might return []bool{false, false}
		// Based on existing BulkUpdate tests, it seems an error is returned and results might be nil or reflect failure.
		// Let's assume the primary check is the error.
		require.Contains(t, err.Error(), "duplicate", "Error message should indicate a duplicate key problem")

		// Verify original entities are unchanged
		for _, originalEntity := range initialEntities {
			found, err := repo.FindByField("ID", originalEntity.ID)
			require.NoError(t, err)
			require.NotNil(t, found)
			assert.Equal(t, originalEntity.Data, found.Data, "Data should not have changed for ID %s", originalEntity.ID)
			assert.Equal(t, originalEntity.Meta.UniqueID, found.Meta.UniqueID, "Meta.UniqueID should not have changed for ID %s", originalEntity.ID)
		}
	})

	t.Run("Bulk update with UniqueID conflicting with another existing (untouched) entity", func(t *testing.T) {
		repo, err := newNestedEntityTestRepositoryRocksDB(t)
		require.NoError(t, err)

		entities := []*NestedEntityTest{
			{ID: "---", Data: "EntityToUpdate", Meta: NestedMetaTest{UniqueID: "originalUnique1", OTValue: "ttl1", Description: "Desc1"}},       // Will be nid1
			{ID: "---", Data: "EntityToConflictWith", Meta: NestedMetaTest{UniqueID: "existingUnique2", OTValue: "ttl2", Description: "Desc2"}}, // Will be nid2
		}
		createdIds, err := repo.BulkCreate(entities)
		require.NoError(t, err)
		require.Len(t, createdIds, 2)
		entities[0].ID = createdIds[0]
		entities[1].ID = createdIds[1]

		// Attempt to update EntityToUpdate's UniqueID to match EntityToConflictWith's UniqueID
		updateAttempt := []*NestedEntityTest{
			{ID: createdIds[0], Data: "EntityToUpdateModified", Meta: NestedMetaTest{UniqueID: "existingUnique2", OTValue: "ttl1_mod", Description: "Desc1_mod"}},
		}

		_, err = repo.BulkUpdate(updateAttempt)
		require.Error(t, err, "BulkUpdate should fail due to conflict with another existing entity's UniqueID")
		require.Contains(t, err.Error(), "duplicate", "Error message should indicate a duplicate key problem")
		// We expect results to be nil or indicate failure if the batch fails entirely.
		// If the API returns per-entity status even on overarching error, it might be []bool{false}

		// Verify EntityToUpdate was not actually updated
		foundOriginal, err := repo.FindByField("ID", createdIds[0])
		require.NoError(t, err)
		require.NotNil(t, foundOriginal)
		assert.Equal(t, "EntityToUpdate", foundOriginal.Data, "EntityToUpdate's data should not have changed")
		assert.Equal(t, "originalUnique1", foundOriginal.Meta.UniqueID, "EntityToUpdate's UniqueID should not have changed")

		// Verify EntityToConflictWith is also unchanged
		foundUntouched, err := repo.FindByField("ID", createdIds[1])
		require.NoError(t, err)
		require.NotNil(t, foundUntouched)
		assert.Equal(t, "EntityToConflictWith", foundUntouched.Data)
		assert.Equal(t, "existingUnique2", foundUntouched.Meta.UniqueID)
	})

	t.Run("Bulk update including non-existent entities", func(t *testing.T) {
		repo, err := newNestedEntityTestRepositoryRocksDB(t)
		require.NoError(t, err)

		existingEntity := NestedEntityTest{ID: "---", Data: "RealData", Meta: NestedMetaTest{UniqueID: "realUnique", OTValue: "ttlReal", Description: "DescReal"}}
		createdIds, err := repo.BulkCreate([]*NestedEntityTest{&existingEntity})
		require.NoError(t, err)
		require.Len(t, createdIds, 1)
		existingEntity.ID = createdIds[0]

		updates := []*NestedEntityTest{
			{ID: existingEntity.ID, Data: "RealDataUpdated", Meta: NestedMetaTest{UniqueID: "realUniqueUpdated", OTValue: "ttlRealUpdated", Description: "DescRealUpdated"}},
			{ID: "nonExistentID1", Data: "PhantomData1", Meta: NestedMetaTest{UniqueID: "phantomUnique1", OTValue: "ttlPhantom1", Description: "DescPhantom1"}},
			{ID: "nonExistentID2", Data: "PhantomData2", Meta: NestedMetaTest{UniqueID: "phantomUnique2", OTValue: "ttlPhantom2", Description: "DescPhantom2"}},
		}

		results, err := repo.BulkUpdate(updates)
		require.NoError(t, err, "BulkUpdate with non-existent IDs should not error out if some are valid (depends on exact error handling, but usually it's per-item status)")
		// This behavior (no error, but false for non-existent) is common for bulk ops.
		// If the design is to error out if *any* ID is not found, this test needs adjustment.
		// The existing `TestRepository_BulkUpdate` -> "Should return false for missing records" subtest suggests `NoError` is correct.
		require.Len(t, results, len(updates))
		assert.True(t, results[0], "Update for existing entity should succeed")
		assert.False(t, results[1], "Update for non-existent entity nonExistentID1 should be marked as false")
		assert.False(t, results[2], "Update for non-existent entity nonExistentID2 should be marked as false")

		// Verify the existing entity was updated
		found, err := repo.FindByField("ID", existingEntity.ID)
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, "RealDataUpdated", found.Data)
		assert.Equal(t, "realUniqueUpdated", found.Meta.UniqueID)

		// Verify non-existent entities were not created
		foundPhantom1, err := repo.FindByField("ID", "nonExistentID1")
		require.NoError(t, err)
		assert.Nil(t, foundPhantom1)
	})
}
