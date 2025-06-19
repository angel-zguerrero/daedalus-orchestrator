package db_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"deadalus-orch/server/internal/infrastructure/db"
)

func newTestRepositoryPebble(t *testing.T) (*db.Repository[testEntity], error) {
	store := newPebbleStore(t)
	iGF := NewTestIDGeneratorFactory([]string{"123", "456"})
	return db.NewRepository[testEntity](store, TestFC, "test_schema", iGF)
}

func newTestRepositorySpesificIdsPebble(t *testing.T, ids []string) (*db.Repository[testEntity], error) {
	store := newPebbleStore(t)
	iGF := NewTestIDGeneratorFactory(ids)
	return db.NewRepository[testEntity](store, TestFC, "test_schema", iGF)
}

func newTestRepositoryDefaultIdGeneratorPebble(t *testing.T) (*db.Repository[testEntity], error) {
	store := newPebbleStore(t)

	return db.NewRepository[testEntity](store, TestFC, "test_schema", &db.DefaultIDGeneratorFactory{})
}

func TestRepository_PutAndGet_Pebble(t *testing.T) {
	repo, err := newTestRepositoryPebble(t)
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

func TestRepository_Get_NotFound_Pebble(t *testing.T) {
	repo, err := newTestRepositoryPebble(t)
	require.NoError(t, err)

	found, err := repo.FindByField("ID", "5566")
	require.NoError(t, err)
	assert.Nil(t, found)
}

// --- Conditional Uniqueness Tests ---

type ConditionalUniqueEntityPebble struct {
	ID                     string `orm:"primary-key"`
	Name                   string
	UniqueValue            string `orm:"unique,ignore-is-true:ShouldIgnoreUniqueness"`
	ShouldIgnoreUniqueness bool
}

func (e ConditionalUniqueEntityPebble) TableName() string {
	return "cond_unique_pebble"
}

func newTestConditionalUniqueRepoPebble(t *testing.T, initialIDs []string) (*db.Repository[ConditionalUniqueEntityPebble], db.KVStore) {
	store := newPebbleStore(t) // Uses a new temp dir for each call
	idGenerator := NewTestIDGeneratorFactory(initialIDs)
	repo, err := db.NewRepository[ConditionalUniqueEntityPebble](store, TestFC, "test_schema_cond", idGenerator)
	require.NoError(t, err, "Failed to create repository for ConditionalUniqueEntityPebble")
	return repo, store
}

func TestPebbleConditionalUniquenessCreate(t *testing.T) {
	t.Run("IgnoreUniqueness", func(t *testing.T) {
		ids := []string{"id1", "id2", "id3"}
		repo, _ := newTestConditionalUniqueRepoPebble(t, ids)

		entity1 := &ConditionalUniqueEntityPebble{Name: "E1", UniqueValue: "uv1", ShouldIgnoreUniqueness: true}
		_, err := repo.Create(entity1)
		require.NoError(t, err, "Create entity1 should succeed")

		entity2 := &ConditionalUniqueEntityPebble{Name: "E2", UniqueValue: "uv1", ShouldIgnoreUniqueness: true}
		_, err = repo.Create(entity2)
		require.NoError(t, err, "Create entity2 with same UniqueValue (ignored) should succeed")

		entity3 := &ConditionalUniqueEntityPebble{Name: "E3", UniqueValue: "uv1", ShouldIgnoreUniqueness: false}
		_, err = repo.Create(entity3)
		require.NoError(t, err, "Create entity3 with same UniqueValue (enforced, but not previously by E1/E2) should succeed")

		entity4 := &ConditionalUniqueEntityPebble{Name: "E3", UniqueValue: "uv1", ShouldIgnoreUniqueness: false}
		_, err = repo.Create(entity4)
		require.Error(t, err)
		require.Contains(t, err.Error(), "duplicate unique field: UniqueValue = uv1")

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
		repo, _ := newTestConditionalUniqueRepoPebble(t, ids)

		entity4 := &ConditionalUniqueEntityPebble{Name: "E4", UniqueValue: "uv2", ShouldIgnoreUniqueness: false}
		_, err := repo.Create(entity4)
		require.NoError(t, err, "Create entity4 should succeed")

		entity5 := &ConditionalUniqueEntityPebble{Name: "E5", UniqueValue: "uv2", ShouldIgnoreUniqueness: false}
		_, err = repo.Create(entity5)
		require.Error(t, err, "Create entity5 with same UniqueValue (enforced) should fail")
		assert.Contains(t, err.Error(), "duplicate unique field")

		entity6 := &ConditionalUniqueEntityPebble{Name: "E6", UniqueValue: "uv2", ShouldIgnoreUniqueness: true}
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

func TestPebbleConditionalUniquenessUpdate(t *testing.T) {
	t.Run("UpdateWithFlagTrueBypassesConflict", func(t *testing.T) {
		ids := []string{"idA", "idB"}
		repo, _ := newTestConditionalUniqueRepoPebble(t, ids)

		entityA := &ConditionalUniqueEntityPebble{ID: ids[0], Name: "EntityA", UniqueValue: "uva", ShouldIgnoreUniqueness: false}
		_, err := repo.Create(entityA)
		require.NoError(t, err)

		entityB := &ConditionalUniqueEntityPebble{ID: ids[1], Name: "EntityB", UniqueValue: "uvb", ShouldIgnoreUniqueness: false}
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
		repo, _ := newTestConditionalUniqueRepoPebble(t, ids)

		entityC := &ConditionalUniqueEntityPebble{ID: ids[0], Name: "EntityC", UniqueValue: "uvc", ShouldIgnoreUniqueness: false}
		_, err := repo.Create(entityC)
		require.NoError(t, err)

		entityD := &ConditionalUniqueEntityPebble{ID: ids[1], Name: "EntityD", UniqueValue: "uvd", ShouldIgnoreUniqueness: false}
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
		repo, _ := newTestConditionalUniqueRepoPebble(t, ids)

		entityE := &ConditionalUniqueEntityPebble{ID: ids[0], Name: "EntityE", UniqueValue: "uve", ShouldIgnoreUniqueness: false}
		_, err := repo.Create(entityE)
		require.NoError(t, err)

		// Update entityE to set ShouldIgnoreUniqueness = true
		entityE.ShouldIgnoreUniqueness = true
		updated, err := repo.Update(entityE)
		require.NoError(t, err, "Update entityE should succeed")
		assert.True(t, updated)

		entityF := &ConditionalUniqueEntityPebble{ID: ids[1], Name: "EntityF", UniqueValue: "uve", ShouldIgnoreUniqueness: true}
		_, err = repo.Create(entityF)
		require.NoError(t, err, "Create entityF should succeed as E is ignoring uniqueness")

		// Verify F
		fCreated, _ := repo.FindByField("ID", ids[1])
		require.NotNil(t, fCreated)
		assert.Equal(t, "uve", fCreated.UniqueValue)
		assert.True(t, fCreated.ShouldIgnoreUniqueness)
	})
}

func TestRepository_WriteBatch_Pebble(t *testing.T) {
	repo, err := newTestRepositoryPebble(t)
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

func TestRepository_SearchByPatternPaginatedKV_MatchSingle_Pebble(t *testing.T) {
	repo, err := newTestRepositoryPebble(t)
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

func TestRepository_Update_SimpleFieldChange_Pebble(t *testing.T) {
	repo, err := newTestRepositoryPebble(t)
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

func TestRepository_Update_UniqueIndexChange_Pebble(t *testing.T) {
	repo, err := newTestRepositoryPebble(t)
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

func TestRepository_Update_IndexCollisionShouldFail_Pebble(t *testing.T) {
	repo, err := newTestRepositoryPebble(t)
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

func TestRepository_Update_NotFound_Pebble(t *testing.T) {
	repo, err := newTestRepositoryPebble(t)
	require.NoError(t, err)

	entity := testEntity{ID: "999", Name: "Zoe"}
	ok, err := repo.Update(&entity)
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestRepository_Delete_Success_Pebble(t *testing.T) {
	repo, err := newTestRepositoryPebble(t)
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

func TestRepository_Delete_NotFound_Pebble(t *testing.T) {
	repo, err := newTestRepositoryPebble(t)
	require.NoError(t, err)

	deleted, err := repo.Delete("nonexistent-id")
	require.NoError(t, err)
	assert.False(t, deleted)
}
func TestRepository_Find_FilteringAndPagination_Pebble(t *testing.T) {
	repo, err := newTestRepositoryDefaultIdGeneratorPebble(t)
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

func TestRepository_Find_PaginationLoop_Pebble(t *testing.T) {
	repo, err := newTestRepositoryDefaultIdGeneratorPebble(t)
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
func TestRepository_Find_Operators_Pebble(t *testing.T) {
	repo, err := newTestRepositoryDefaultIdGeneratorPebble(t)
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
func TestRepository_Find_LargeDatasetWithPaginationAndComplexFilters_Pebble(t *testing.T) {
	repo, err := newTestRepositoryDefaultIdGeneratorPebble(t)
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

func TestRepository_Find_ComplexNestedFilters_Pebble(t *testing.T) {
	repo, err := newTestRepositoryDefaultIdGeneratorPebble(t)
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

func TestRepository_BulkCreate_Pebble(t *testing.T) {
	t.Run("Basic bulk insert", func(t *testing.T) {
		repo, err := newTestRepositoryDefaultIdGeneratorPebble(t)
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
		repo, err := newTestRepositoryDefaultIdGeneratorPebble(t)
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
		repo, err := newTestRepositoryDefaultIdGeneratorPebble(t)
		require.NoError(t, err)

		ids, err := repo.BulkCreate([]*testEntity{})
		require.NoError(t, err)
		assert.Empty(t, ids)
	})

	t.Run("Can bulk insert large batch", func(t *testing.T) {
		repo, err := newTestRepositoryDefaultIdGeneratorPebble(t)
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
		repo, err := newTestRepositoryDefaultIdGeneratorPebble(t)
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
		repo, err := newTestRepositoryDefaultIdGeneratorPebble(t)

		batch := []*testEntity{
			{Name: "duplicate-name", Age: 100},
			{Name: "duplicate-name", Age: 200}, // mismo Name
		}

		_, err = repo.BulkCreate(batch)
		require.Error(t, err)
		require.Contains(t, err.Error(), "duplicate") // adapta esto según el mensaje real
	})
}

func TestRepository_BulkDelete_Pebble(t *testing.T) {
	t.Run("Delete multiple existing entities", func(t *testing.T) {
		ids := []string{"123", "456", "789"}
		repo, err := newTestRepositorySpesificIdsPebble(t, ids)
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
		repo, err := newTestRepositoryPebble(t)
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
		repo, err := newTestRepositoryPebble(t)
		require.NoError(t, err)
		_, err = repo.BulkDelete([]string{})
		require.NoError(t, err)
	})

	t.Run("Bulk delete should also remove unique indices", func(t *testing.T) {
		ids := []string{"111", "222"}
		repo, err := newTestRepositorySpesificIdsPebble(t, ids)
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
func TestRepository_BulkUpdate_Pebble(t *testing.T) {
	t.Run("Basic bulk update", func(t *testing.T) {
		repo, err := newTestRepositoryDefaultIdGeneratorPebble(t)
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
		repo, err := newTestRepositoryDefaultIdGeneratorPebble(t)
		require.NoError(t, err)

		result, err := repo.BulkUpdate([]*testEntity{})
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("Should return false for missing records", func(t *testing.T) {
		repo, err := newTestRepositoryDefaultIdGeneratorPebble(t)
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
		repo, err := newTestRepositoryDefaultIdGeneratorPebble(t)
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
		repo, err := newTestRepositoryDefaultIdGeneratorPebble(t)
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
		repo, err := newTestRepositoryDefaultIdGeneratorPebble(t)
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
		repo, err := newTestRepositoryDefaultIdGeneratorPebble(t)
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

func newTestNRepositoryPebble(t *testing.T) (*db.Repository[UserComplexN], error) {
	store := newPebbleStore(t)
	iGF := NewTestIDGeneratorFactory([]string{"123", "456"})
	return db.NewRepository[UserComplexN](store, TestFC, "test_schema", iGF)
}

func newNestedEntityTestPebbleRepositoryPebble(t *testing.T) (*db.Repository[NestedEntityTestPebble], error) {
	store := newPebbleStore(t)                                                              // Assumes newPebbleStore is defined in this file
	iGF := NewTestIDGeneratorFactory([]string{"pnid1", "pnid2", "pnid3", "pnid4", "pnid5"}) // Example IDs for Pebble
	return db.NewRepository[NestedEntityTestPebble](store, TestFC, "nested_entity_schema_pebble", iGF)
}

type NestedMetaTestPebble struct {
	UniqueID    string `orm:"unique"`
	OTValue     string
	Description string
}

type NestedEntityTestPebble struct {
	ID   string `orm:"primary-key"`
	Data string
	Meta NestedMetaTestPebble
}

func (NestedEntityTestPebble) TableName() string {
	return "nested_entities_test"
}

func TestRepository_PutAndGet_Nested_Pebble(t *testing.T) {
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

func TestRepository_BulkCreate_Nested_Pebble(t *testing.T) {
	repo, err := newNestedEntityTestPebbleRepositoryPebble(t)
	require.NoError(t, err, "Failed to create repository for NestedEntityTestPebble (Pebble)")

	t.Run("Successful bulk creation with nested structs pebble", func(t *testing.T) {
		entities := []*NestedEntityTestPebble{
			{ID: "---", Data: "EntityP1", Meta: NestedMetaTestPebble{UniqueID: "uniqueP1", OTValue: "ttlP1", Description: "DescP1"}},
			{ID: "---", Data: "EntityP2", Meta: NestedMetaTestPebble{UniqueID: "uniqueP2", OTValue: "ttlP2", Description: "DescP2"}},
			{ID: "---", Data: "EntityP3", Meta: NestedMetaTestPebble{UniqueID: "uniqueP3", OTValue: "ttlP3", Description: "DescP3"}},
		}

		ids, err := repo.BulkCreate(entities)
		require.NoError(t, err, "BulkCreate failed for valid nested entities (Pebble)")
		require.Len(t, ids, len(entities), "BulkCreate should return an ID for each entity (Pebble)")

		for i, id := range ids {
			require.NotEmpty(t, id, "Expected a non-empty ID for entity %d (Pebble)", i)
			entities[i].ID = id // Assign returned ID for later checks

			found, err := repo.FindByField("ID", id)
			require.NoError(t, err, "FindByField by ID failed for created entity %s (Pebble)", id)
			require.NotNil(t, found, "Should find entity by ID %s (Pebble)", id)
			assert.Equal(t, entities[i].Data, found.Data, "Data field mismatch for entity %s (Pebble)", id)
			assert.Equal(t, entities[i].Meta.UniqueID, found.Meta.UniqueID, "Meta.UniqueID field mismatch for entity %s (Pebble)", id)
			assert.Equal(t, entities[i].Meta.OTValue, found.Meta.OTValue, "Meta.TTLValue field mismatch for entity %s (Pebble)", id)
			assert.Equal(t, entities[i].Meta.Description, found.Meta.Description, "Meta.Description field mismatch for entity %s (Pebble)", id)
		}

		// Verify finding by nested unique index
		foundByNestedUnique, err := repo.FindByField("Meta.UniqueID", "uniqueP2")
		require.NoError(t, err, "FindByField by Meta.UniqueID failed (Pebble)")
		require.NotNil(t, foundByNestedUnique, "Should find entity by Meta.UniqueID 'uniqueP2' (Pebble)")
		assert.Equal(t, entities[1].ID, foundByNestedUnique.ID, "ID mismatch when finding by Meta.UniqueID (Pebble)")
		assert.Equal(t, "EntityP2", foundByNestedUnique.Data)
		assert.Equal(t, "uniqueP2", foundByNestedUnique.Meta.UniqueID)
	})

	t.Run("Bulk creation with duplicate UniqueID within the batch pebble", func(t *testing.T) {
		repoFresh, err := newNestedEntityTestPebbleRepositoryPebble(t)
		require.NoError(t, err)

		entities := []*NestedEntityTestPebble{
			{ID: "---", Data: "EntityPX", Meta: NestedMetaTestPebble{UniqueID: "duplicateKeyInBatchP", OTValue: "ttlPX", Description: "DescPX"}},
			{ID: "---", Data: "EntityPY", Meta: NestedMetaTestPebble{UniqueID: "anotherUniqueInBatchP", OTValue: "ttlPY", Description: "DescPY"}},
			{ID: "---", Data: "EntityPZ", Meta: NestedMetaTestPebble{UniqueID: "duplicateKeyInBatchP", OTValue: "ttlPZ", Description: "DescPZ"}}, // Duplicate UniqueID
		}

		ids, err := repoFresh.BulkCreate(entities)
		require.Error(t, err, "BulkCreate should fail due to duplicate UniqueID within the batch (Pebble)")
		assert.Nil(t, ids, "IDs should be nil on batch creation failure (Pebble)")
		require.Contains(t, err.Error(), "duplicate", "Error message should indicate a duplicate key problem (Pebble)")

		// Verify no entities were partially inserted
		found, err := repoFresh.FindByField("Meta.UniqueID", "duplicateKeyInBatchP")
		require.NoError(t, err)
		assert.Nil(t, found, "No entity should be found with the conflicting UniqueID if batch failed (Pebble)")

		found, err = repoFresh.FindByField("Meta.UniqueID", "anotherUniqueInBatchP")
		require.NoError(t, err)
		assert.Nil(t, found, "No entity should be found with a non-conflicting UniqueID if batch failed (Pebble)")
	})

	t.Run("Bulk creation with UniqueID conflicting with existing data pebble", func(t *testing.T) {
		repoClean, err := newNestedEntityTestPebbleRepositoryPebble(t) // Fresh repo
		require.NoError(t, err)

		// Pre-existing entity
		existingEntity := NestedEntityTestPebble{ID: "---", Data: "ExistingDataP", Meta: NestedMetaTestPebble{UniqueID: "conflictWithExistingP", OTValue: "ttlPE", Description: "DescPE"}}
		_, err = repoClean.Create(&existingEntity)
		require.NoError(t, err, "Setup: Failed to create initial entity (Pebble)")

		entitiesToBulkCreate := []*NestedEntityTestPebble{
			{ID: "---", Data: "NewEntityP1", Meta: NestedMetaTestPebble{UniqueID: "newUniqueP1", OTValue: "ttlPN1", Description: "DescPN1"}},
			{ID: "---", Data: "NewEntityP2Conflicting", Meta: NestedMetaTestPebble{UniqueID: "conflictWithExistingP", OTValue: "ttlPN2", Description: "DescPN2"}}, // Conflicts
			{ID: "---", Data: "NewEntityP3", Meta: NestedMetaTestPebble{UniqueID: "newUniqueP3", OTValue: "ttlPN3", Description: "DescPN3"}},
		}

		ids, err := repoClean.BulkCreate(entitiesToBulkCreate)
		require.Error(t, err, "BulkCreate should fail due to conflict with existing UniqueID (Pebble)")
		assert.Nil(t, ids, "IDs should be nil on batch creation failure due to existing conflict (Pebble)")
		require.Contains(t, err.Error(), "duplicate", "Error message should indicate a duplicate key problem (Pebble)")

		foundNew, err := repoClean.FindByField("Meta.UniqueID", "newUniqueP1")
		require.NoError(t, err)
		assert.Nil(t, foundNew, "Non-conflicting entity from failed batch should not be inserted (Pebble)")

		foundExisting, err := repoClean.FindByField("Meta.UniqueID", "conflictWithExistingP")
		require.NoError(t, err)
		require.NotNil(t, foundExisting, "Original entity with the conflicting key should still exist (Pebble)")
		assert.Equal(t, existingEntity.Data, foundExisting.Data)
	})
}

func TestRepository_BulkUpdate_Nested_Pebble(t *testing.T) {
	t.Run("Successful bulk update of nested structs pebble", func(t *testing.T) {
		repo, err := newNestedEntityTestPebbleRepositoryPebble(t) // Uses specific IDs: pnid1, pnid2, ...
		require.NoError(t, err)

		initialEntities := []*NestedEntityTestPebble{
			{ID: "---", Data: "DataOneP", Meta: NestedMetaTestPebble{UniqueID: "uniquePU1", OTValue: "ttlPU1", Description: "DescPU1"}},
			{ID: "---", Data: "DataTwoP", Meta: NestedMetaTestPebble{UniqueID: "uniquePU2", OTValue: "ttlPU2", Description: "DescPU2"}},
		}
		createdIds, err := repo.BulkCreate(initialEntities)
		require.NoError(t, err)
		require.Len(t, createdIds, 2)

		// Prepare updates
		updatedEntities := []*NestedEntityTestPebble{
			{ID: createdIds[0], Data: "DataOneUpdatedP", Meta: NestedMetaTestPebble{UniqueID: "uniquePU1_new", OTValue: "ttlPU1_new", Description: "DescPU1_new"}},
			{ID: createdIds[1], Data: "DataTwoUpdatedP", Meta: NestedMetaTestPebble{UniqueID: "uniquePU2_new", OTValue: "ttlPU2_new", Description: "DescPU2_new"}},
		}

		results, err := repo.BulkUpdate(updatedEntities)
		require.NoError(t, err, "BulkUpdate failed for valid nested entity updates (Pebble)")
		require.Len(t, results, len(updatedEntities), "BulkUpdate should return a result for each entity (Pebble)")
		for i, success := range results {
			assert.True(t, success, "Expected update for entity ID %s to succeed (Pebble)", updatedEntities[i].ID)
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

			oldUniqueValue := initialEntities[i].Meta.UniqueID
			foundByOldUnique, err := repo.FindByField("Meta.UniqueID", oldUniqueValue)
			require.NoError(t, err)
			assert.Nil(t, foundByOldUnique, "Entity should not be found by old Meta.UniqueID %s (Pebble)", oldUniqueValue)

			foundByNewUnique, err := repo.FindByField("Meta.UniqueID", updatedEntity.Meta.UniqueID)
			require.NoError(t, err)
			require.NotNil(t, foundByNewUnique, "Entity should be found by new Meta.UniqueID %s (Pebble)", updatedEntity.Meta.UniqueID)
			assert.Equal(t, updatedEntity.ID, foundByNewUnique.ID)
		}
	})

	t.Run("Bulk update with UniqueID conflict within the batch pebble", func(t *testing.T) {
		repo, err := newNestedEntityTestPebbleRepositoryPebble(t)
		require.NoError(t, err)

		initialEntities := []*NestedEntityTestPebble{
			{ID: "---", Data: "AlphaP", Meta: NestedMetaTestPebble{UniqueID: "alphaUniqueP", OTValue: "ttlAP", Description: "DescAP"}},
			{ID: "---", Data: "BetaP", Meta: NestedMetaTestPebble{UniqueID: "betaUniqueP", OTValue: "ttlBP", Description: "DescBP"}},
		}
		createdIds, err := repo.BulkCreate(initialEntities)
		require.NoError(t, err)
		require.Len(t, createdIds, 2)
		initialEntities[0].ID = createdIds[0]
		initialEntities[1].ID = createdIds[1]

		conflictingUpdates := []*NestedEntityTestPebble{
			{ID: createdIds[0], Data: "AlphaUpdatedP", Meta: NestedMetaTestPebble{UniqueID: "conflictKeyP", OTValue: "ttlAP_new", Description: "DescAP_new"}},
			{ID: createdIds[1], Data: "BetaUpdatedP", Meta: NestedMetaTestPebble{UniqueID: "conflictKeyP", OTValue: "ttlBP_new", Description: "DescBP_new"}},
		}

		_, err = repo.BulkUpdate(conflictingUpdates)
		require.Error(t, err, "BulkUpdate should fail due to UniqueID conflict within the batch (Pebble)")
		require.Contains(t, err.Error(), "duplicate", "Error message should indicate a duplicate key problem (Pebble)")

		for _, originalEntity := range initialEntities {
			found, err := repo.FindByField("ID", originalEntity.ID)
			require.NoError(t, err)
			require.NotNil(t, found)
			assert.Equal(t, originalEntity.Data, found.Data)
			assert.Equal(t, originalEntity.Meta.UniqueID, found.Meta.UniqueID)
		}
	})

	t.Run("Bulk update with UniqueID conflicting with another existing (untouched) entity pebble", func(t *testing.T) {
		repo, err := newNestedEntityTestPebbleRepositoryPebble(t)
		require.NoError(t, err)

		entities := []*NestedEntityTestPebble{
			{ID: "---", Data: "EntityToUpdateP", Meta: NestedMetaTestPebble{UniqueID: "originalUniqueP1", OTValue: "ttlP1", Description: "DescP1"}},
			{ID: "---", Data: "EntityToConflictWithP", Meta: NestedMetaTestPebble{UniqueID: "existingUniqueP2", OTValue: "ttlP2", Description: "DescP2"}},
		}
		createdIds, err := repo.BulkCreate(entities)
		require.NoError(t, err)
		require.Len(t, createdIds, 2)
		entities[0].ID = createdIds[0]
		entities[1].ID = createdIds[1]

		updateAttempt := []*NestedEntityTestPebble{
			{ID: createdIds[0], Data: "EntityToUpdateModifiedP", Meta: NestedMetaTestPebble{UniqueID: "existingUniqueP2", OTValue: "ttlP1_mod", Description: "DescP1_mod"}},
		}

		_, err = repo.BulkUpdate(updateAttempt)
		require.Error(t, err, "BulkUpdate should fail due to conflict with another existing entity's UniqueID (Pebble)")
		require.Contains(t, err.Error(), "duplicate", "Error message should indicate a duplicate key problem (Pebble)")

		foundOriginal, err := repo.FindByField("ID", createdIds[0])
		require.NoError(t, err)
		require.NotNil(t, foundOriginal)
		assert.Equal(t, "EntityToUpdateP", foundOriginal.Data)
		assert.Equal(t, "originalUniqueP1", foundOriginal.Meta.UniqueID)

		foundUntouched, err := repo.FindByField("ID", createdIds[1])
		require.NoError(t, err)
		require.NotNil(t, foundUntouched)
		assert.Equal(t, "EntityToConflictWithP", foundUntouched.Data)
		assert.Equal(t, "existingUniqueP2", foundUntouched.Meta.UniqueID)
	})

	t.Run("Bulk update including non-existent entities pebble", func(t *testing.T) {
		repo, err := newNestedEntityTestPebbleRepositoryPebble(t)
		require.NoError(t, err)

		existingEntity := NestedEntityTestPebble{ID: "---", Data: "RealDataP", Meta: NestedMetaTestPebble{UniqueID: "realUniqueP", OTValue: "ttlRealP", Description: "DescRealP"}}
		createdIds, err := repo.BulkCreate([]*NestedEntityTestPebble{&existingEntity})
		require.NoError(t, err)
		require.Len(t, createdIds, 1)
		existingEntity.ID = createdIds[0]

		updates := []*NestedEntityTestPebble{
			{ID: existingEntity.ID, Data: "RealDataUpdatedP", Meta: NestedMetaTestPebble{UniqueID: "realUniqueUpdatedP", OTValue: "ttlRealUpdatedP", Description: "DescRealUpdatedP"}},
			{ID: "nonExistentIDP1", Data: "PhantomDataP1", Meta: NestedMetaTestPebble{UniqueID: "phantomUniqueP1", OTValue: "ttlPhantomP1", Description: "DescPhantomP1"}},
			{ID: "nonExistentIDP2", Data: "PhantomDataP2", Meta: NestedMetaTestPebble{UniqueID: "phantomUniqueP2", OTValue: "ttlPhantomP2", Description: "DescPhantomP2"}},
		}

		results, err := repo.BulkUpdate(updates)
		require.NoError(t, err, "BulkUpdate with non-existent IDs should not error out (Pebble)")
		require.Len(t, results, len(updates))
		assert.True(t, results[0], "Update for existing entity should succeed (Pebble)")
		assert.False(t, results[1], "Update for non-existent entity nonExistentIDP1 should be marked as false (Pebble)")
		assert.False(t, results[2], "Update for non-existent entity nonExistentIDP2 should be marked as false (Pebble)")

		found, err := repo.FindByField("ID", existingEntity.ID)
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, "RealDataUpdatedP", found.Data)
		assert.Equal(t, "realUniqueUpdatedP", found.Meta.UniqueID)

		foundPhantom1, err := repo.FindByField("ID", "nonExistentIDP1")
		require.NoError(t, err)
		assert.Nil(t, foundPhantom1)
	})
}
