package db_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"deadalus-orch/server/internal/infrastructure/db"
)

func newPebbleStore(t *testing.T) db.KVStore {
	tmpDir := t.TempDir()
	store, err := db.CreatePebbleStore(tmpDir, []string{DefaultFC, TestFC}, nil)
	require.NoError(t, err)
	return store
}

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
