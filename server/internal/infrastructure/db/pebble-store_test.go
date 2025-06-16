package db_test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"deadalus-orch/server/internal/infrastructure/db"
)

const TestFC = "test_fc"
const DefaultFC = "default"
const TemporalFC = "temporal_fc"

type testEntity struct {
	ID       string `orm:"primary-key"`
	Name     string `orm:"unique"`
	LastName string
	Age      int
	TTL      int `orm:"ttl"`
}

func (testEntity) TableName() string {
	return "users"
}

type TestIDGeneratorFactory struct {
	ids   []string
	index int
	mu    sync.Mutex
}

func (g *TestIDGeneratorFactory) GenerateID() string {
	g.mu.Lock()
	defer g.mu.Unlock()

	if len(g.ids) == 0 {
		return ""
	}

	id := g.ids[g.index]
	g.index = (g.index + 1) % len(g.ids) // avance circular
	return id
}

func NewTestIDGeneratorFactory(ids []string) *TestIDGeneratorFactory {
	return &TestIDGeneratorFactory{
		ids: ids,
	}
}

func newPebbleStore(t *testing.T) db.KVStore {
	tmpDir := t.TempDir()
	store, err := db.CreatePebbleStore(tmpDir, []string{DefaultFC, TestFC}, []string{TemporalFC})
	require.NoError(t, err)
	return store
}

func newTestTTLRepositoryDefaultIdGeneratorPebble(t *testing.T) (*db.Repository[testEntity], db.KVStore, error) {
	store := newPebbleStore(t)
	repository, err := db.NewRepository[testEntity](store, TemporalFC, "test_schema", &db.DefaultIDGeneratorFactory{})
	return repository, store, err
}

func newTestTTLRepositoryPebble(t *testing.T) (*db.Repository[testEntity], db.KVStore, error) {
	store := newPebbleStore(t)
	iGF := NewTestIDGeneratorFactory([]string{"123", "456"})
	repository, err := db.NewRepository[testEntity](store, TemporalFC, "test_schema", iGF)
	return repository, store, err
}

func TestPebbleStore_PutAndGet(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := db.CreatePebbleStore(tmpDir, []string{DefaultFC, TestFC}, nil)
	require.NoError(t, err)
	defer store.Close()

	key := "key"
	value := []byte("value")

	err = store.Put(TestFC, key, value, 0)
	require.NoError(t, err)

	result, err := store.Get(TestFC, key)
	require.NoError(t, err)
	assert.Equal(t, value, result)
}

func TestPebbleStore_Get_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := db.CreatePebbleStore(tmpDir, []string{DefaultFC, TestFC}, nil)
	require.NoError(t, err)
	defer store.Close()

	result, err := store.Get(TestFC, "nonexistent")
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestPebbleStore_WriteBatch(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := db.CreatePebbleStore(tmpDir, []string{DefaultFC, TestFC}, nil)
	require.NoError(t, err)
	defer store.Close()

	batch := db.NewWriteBatch()
	batch.Put(TestFC, "a", []byte("valueA"))
	batch.Put(TestFC, "b", []byte("valueB"))

	err = store.Write(batch)
	require.NoError(t, err)

	resultA, err := store.Get(TestFC, "a")
	require.NoError(t, err)
	assert.Equal(t, []byte("valueA"), resultA)

	resultB, err := store.Get(TestFC, "b")
	require.NoError(t, err)
	assert.Equal(t, []byte("valueB"), resultB)
}

func TestPebbleStore_SearchByPatternPaginatedKV_MatchSingle(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := db.CreatePebbleStore(tmpDir, []string{DefaultFC, TestFC}, nil)
	require.NoError(t, err)
	defer store.Close()

	require.NoError(t, store.Put(TestFC, "user:123:name", []byte("Alice"), 0))

	results, next, err := store.SearchByPatternPaginatedKV(TestFC, "user:123:*", "", 10)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "user:123:name", results[0].Key)
	assert.Equal(t, []byte("Alice"), results[0].Value)
	assert.Equal(t, "", next)
}

func TestPebbleStore_SearchByPatternPaginatedKV_MatchMultiplePages(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := db.CreatePebbleStore(tmpDir, []string{DefaultFC, TestFC}, nil)
	require.NoError(t, err)
	defer store.Close()

	require.NoError(t, store.Put(TestFC, "user:1", []byte("a"), 0))
	require.NoError(t, store.Put(TestFC, "user:2", []byte("b"), 0))
	require.NoError(t, store.Put(TestFC, "user:3", []byte("c"), 0))

	var all []db.KeyValuePair
	cursor := ""
	for {
		page, next, err := store.SearchByPatternPaginatedKV(TestFC, "user:*", cursor, 2)
		require.NoError(t, err)
		all = append(all, page...)
		if next == "" {
			break
		}
		cursor = next
	}
	require.Len(t, all, 3)
}

func TestPebbleStore_SearchByPatternPaginatedKV_NoMatch(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := db.CreatePebbleStore(tmpDir, []string{DefaultFC, TestFC}, nil)
	require.NoError(t, err)
	defer store.Close()

	require.NoError(t, store.Put(TestFC, "product:1", []byte("item"), 0))

	results, next, err := store.SearchByPatternPaginatedKV(TestFC, "user:*", "", 10)
	require.NoError(t, err)
	require.Empty(t, results)
	require.Equal(t, "", next)
}

func TestPebbleStore_SearchByPatternPaginatedKV_InvalidColumnFamily(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := db.CreatePebbleStore(tmpDir, []string{DefaultFC}, nil)
	require.NoError(t, err)
	defer store.Close()

	_, _, err = store.SearchByPatternPaginatedKV("nonexistent", "pattern:*", "", 10)
	require.Error(t, err)
	require.Contains(t, err.Error(), "column family")
}

func TestPebbleStore_Delete_ExistingKey(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := db.CreatePebbleStore(tmpDir, []string{DefaultFC, TestFC}, nil)
	require.NoError(t, err)
	defer store.Close()

	key := "delete-key"
	value := []byte("to-delete")

	require.NoError(t, store.Put(TestFC, key, value, 0))
	require.NoError(t, store.Delete(TestFC, key))

	result, err := store.Get(TestFC, key)
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestPebbleStore_Delete_NonExistentKey(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := db.CreatePebbleStore(tmpDir, []string{DefaultFC, TestFC}, nil)
	require.NoError(t, err)
	defer store.Close()

	err = store.Delete(TestFC, "nonexistent")
	assert.NoError(t, err)
}

func TestPebbleStore_Delete_InvalidColumnFamily(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := db.CreatePebbleStore(tmpDir, []string{DefaultFC}, nil)
	require.NoError(t, err)
	defer store.Close()

	err = store.Delete("nonexistent_cf", "key")
	require.Error(t, err)
	require.Contains(t, err.Error(), "column family")
}

func TestPebbleStore_Delete_TTLColumnFamily(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := db.CreatePebbleStore(tmpDir, []string{}, []string{TestFC})
	require.NoError(t, err)
	defer store.Close()

	key := "ttl-key"
	value := []byte("ttl-value")

	require.NoError(t, store.Put(TestFC, key, value, 0))
	require.NoError(t, store.Delete(TestFC, key))

	result, err := store.Get(TestFC, key)
	require.NoError(t, err)
	assert.Nil(t, result)
}
func TestPebbleRepository_TTL_BasicExpiration(t *testing.T) {
	repo, store, err := newTestTTLRepositoryPebble(t)
	require.NoError(t, err)

	entity := testEntity{
		Name:     "ttlTestEntity",
		Age:      20,
		LastName: "Gomez",
		TTL:      2, // 2 seconds
	}

	createdID, err := repo.Create(&entity)
	require.NoError(t, err)
	require.NotEmpty(t, createdID)

	// Verify immediately after creation
	found, err := repo.FindByField("ID", createdID)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, entity.Name, found.Name)
	// TTL field is not usually part of the retrieved data unless explicitly handled by the ORM layer
	// So we don't assert found.TTL == entity.TTL

	schema := "test_schema"
	table := entity.TableName()

	// Verify directly from kvStore (TemporalFC) before ttl

	mainDataKey := fmt.Sprintf("%s:%s:data:%s", schema, table, createdID)
	dataBytes, err := store.Get(TemporalFC, mainDataKey)
	require.NoError(t, err)
	assert.NotNil(t, dataBytes)

	uniqueIndexKey := fmt.Sprintf("%s:%s:idx-u:%s:%s", schema, table, "Name", entity.Name)
	idxBytes, err := store.Get(TemporalFC, uniqueIndexKey)
	require.NoError(t, err)
	assert.NotNil(t, idxBytes)

	generalIndexKeyName := fmt.Sprintf("%s:%s:idx:%s:%s:%s", schema, table, "Name", entity.Name, createdID)
	idxBytes, err = store.Get(TemporalFC, generalIndexKeyName)
	require.NoError(t, err)
	assert.NotNil(t, idxBytes)

	generalIndexKeyLastName := fmt.Sprintf("%s:%s:idx:%s:%s:%s", schema, table, "LastName", entity.LastName, createdID)
	idxBytes, err = store.Get(TemporalFC, generalIndexKeyLastName)
	require.NoError(t, err)
	assert.NotNil(t, idxBytes)

	generalIndexKeyAge := fmt.Sprintf("%s:%s:idx:%s:%d:%s", schema, table, "Age", entity.Age, createdID)
	idxBytes, err = store.Get(TemporalFC, generalIndexKeyAge)
	require.NoError(t, err)
	assert.NotNil(t, idxBytes)

	generalIndexKeyID := fmt.Sprintf("%s:%s:idx:%s:%s:%s", schema, table, "ID", createdID, createdID)
	idxBytes, err = store.Get(TemporalFC, generalIndexKeyID)
	require.NoError(t, err)
	assert.NotNil(t, idxBytes)

	// Wait for TTL to expire
	time.Sleep(3 * time.Second)

	// Verify entity is gone from repository
	notFound, err := repo.FindByField("ID", createdID)
	require.NoError(t, err)
	assert.Nil(t, notFound)

	// Verify directly from kvStore (TemporalFC) after ttl

	mainDataKey = fmt.Sprintf("%s:%s:data:%s", schema, table, createdID)
	dataBytes, err = store.Get(TemporalFC, mainDataKey)
	require.NoError(t, err)
	assert.Nil(t, dataBytes)

	uniqueIndexKey = fmt.Sprintf("%s:%s:idx-u:%s:%s", schema, table, "Name", entity.Name)
	idxBytes, err = store.Get(TemporalFC, uniqueIndexKey)
	require.NoError(t, err)
	assert.Nil(t, idxBytes)

	generalIndexKeyName = fmt.Sprintf("%s:%s:idx:%s:%s:%s", schema, table, "Name", entity.Name, createdID)
	idxBytes, err = store.Get(TemporalFC, generalIndexKeyName)
	require.NoError(t, err)
	assert.Nil(t, idxBytes)

	generalIndexKeyLastName = fmt.Sprintf("%s:%s:idx:%s:%s:%s", schema, table, "LastName", entity.LastName, createdID)
	idxBytes, err = store.Get(TemporalFC, generalIndexKeyLastName)
	require.NoError(t, err)
	assert.Nil(t, idxBytes)

	generalIndexKeyAge = fmt.Sprintf("%s:%s:idx:%s:%d:%s", schema, table, "Age", entity.Age, createdID)
	idxBytes, err = store.Get(TemporalFC, generalIndexKeyAge)
	require.NoError(t, err)
	assert.Nil(t, idxBytes)

	generalIndexKeyID = fmt.Sprintf("%s:%s:idx:%s:%s:%s", schema, table, "ID", createdID, createdID)
	idxBytes, err = store.Get(TemporalFC, generalIndexKeyID)
	require.NoError(t, err)
	assert.Nil(t, idxBytes)
}
func TestPebbleRepository_TTL_BulkUpdateExpiration(t *testing.T) {
	repo, _, err := newTestTTLRepositoryDefaultIdGeneratorPebble(t)
	require.NoError(t, err)

	// 1. & 2. Initialize and Create initial entities
	entityAInitialName := "BulkUpdateA_Initial_"
	entityBInitialName := "BulkUpdateB_Initial"
	entityCInitialName := "BulkUpdateC_Initial"

	entitiesToCreate := []*testEntity{
		{Name: entityAInitialName, TTL: 0}, // Will gain TTL
		{Name: entityBInitialName, TTL: 2}, // Will extend TTL
		{Name: entityCInitialName, TTL: 8}, // Name change, TTL constant
	}
	createdIds, err := repo.BulkCreate(entitiesToCreate)
	require.NoError(t, err)
	require.Len(t, createdIds, 3)
	_, idB, idC := createdIds[0], createdIds[1], createdIds[2]

	// 3. Verify initial retrieval
	for i, currentID := range createdIds {
		found, err := repo.FindByField("ID", currentID)
		require.NoError(t, err)
		if i != 0 {
			require.NotNil(t, found)
			assert.Equal(t, entitiesToCreate[i].Name, found.Name)
		}

		entitiesToCreate[i].ID = currentID // Assign ID for later reference
	}
	entityA := entitiesToCreate[0]
	entityB := entitiesToCreate[1]
	entityC := entitiesToCreate[2]

	// 4. Prepare entities for BulkUpdate
	entityAUpdateName := "BulkUpdateA_NewTTL"
	entityBUpdateName := "BulkUpdateB_ExtendedTTL"
	entityCUpdateName := "BulkUpdateC_NameChange"

	entityA.Name = entityAUpdateName
	entityA.TTL = 3 // New TTL: 3s

	entityB.Name = entityBUpdateName
	entityB.TTL = 6 // Extended TTL: 6s

	entityC.Name = entityCUpdateName
	// entityC.TTL remains 8s

	// 5. Perform BulkUpdate
	updateResults, err := repo.BulkUpdate([]*testEntity{entityB, entityC})
	require.NoError(t, err)
	for _, res := range updateResults {
		assert.True(t, res)
	}

	// 6. Verify entities immediately after update

	foundB, _ := repo.FindByField("ID", idB)
	require.NotNil(t, foundB)
	assert.Equal(t, entityBUpdateName, foundB.Name)
	foundC, _ := repo.FindByField("ID", idC)
	require.NotNil(t, foundC)
	assert.Equal(t, entityCUpdateName, foundC.Name)

	// 7. Timing and Expiration Checks
	// Wait 4 seconds (entityA TTL 3s; entityB initial TTL 2s, now 6s; entityC TTL 8s)
	time.Sleep(4 * time.Second)

	// idA (3s TTL) should be GONE

	// idB (orig 2s, now 6s TTL) should EXIST
	foundB, err = repo.FindByField("ID", idB)
	require.NoError(t, err)
	require.NotNil(t, foundB)
	assert.Equal(t, entityBUpdateName, foundB.Name)

	// idC (8s TTL) should EXIST
	foundC, err = repo.FindByField("ID", idC)
	require.NoError(t, err)
	require.NotNil(t, foundC)
	assert.Equal(t, entityCUpdateName, foundC.Name)

	// Wait another 3 seconds (total 7 seconds)
	// (entityB 6s TTL; entityC 8s TTL)
	time.Sleep(3 * time.Second)

	// idB (6s TTL) should be GONE
	foundB, err = repo.FindByField("ID", idB)
	require.NoError(t, err)
	assert.Nil(t, foundB)

	// idC (8s TTL) should EXIST
	foundC, err = repo.FindByField("ID", idC)
	require.NoError(t, err)
	require.NotNil(t, foundC)
	assert.Equal(t, entityCUpdateName, foundC.Name)

	// Wait another 2 seconds (total 9 seconds)
	// (entityC 8s TTL)
	time.Sleep(2 * time.Second)

	// idC (8s TTL) should be GONE
	foundC, err = repo.FindByField("ID", idC)
	require.NoError(t, err)
	assert.Nil(t, foundC)

}
func TestPebbleRepository_TTL_BulkCreateExpiration(t *testing.T) {
	repo, store, err := newTestTTLRepositoryDefaultIdGeneratorPebble(t)
	require.NoError(t, err)

	entitiesToCreate := []*testEntity{
		{Name: "ttlBulkEntity1", TTL: 2}, // 2 seconds
		{Name: "ttlBulkEntity2", TTL: 3}, // 3 seconds
		{Name: "ttlBulkEntity3", TTL: 2}, // 2 seconds
	}

	createdIds, err := repo.BulkCreate(entitiesToCreate)
	require.NoError(t, err)
	require.Len(t, createdIds, len(entitiesToCreate))

	maxTTL := 0
	for i, createdID := range createdIds {
		originalEntity := entitiesToCreate[i]
		if originalEntity.TTL > maxTTL {
			maxTTL = originalEntity.TTL
		}

		// Assign created ID for later direct KV store verification by name
		originalEntity.ID = createdID

		found, err := repo.FindByField("ID", createdID)
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, originalEntity.Name, found.Name)
	}

	// Wait for the max TTL to expire + a little buffer
	time.Sleep(time.Duration(maxTTL+1) * time.Second)

	schema := "test_schema"
	table := entitiesToCreate[0].TableName() // All entities are of the same type

	for i, createdID := range createdIds {
		originalEntity := entitiesToCreate[i] // Now has ID assigned

		// Verify entity is gone from repository
		notFoundInRepo, err := repo.FindByField("ID", createdID)
		require.NoError(t, err)
		assert.Nil(t, notFoundInRepo)

		// Verify directly from kvStore (TemporalFC)
		mainDataKey := fmt.Sprintf("%s:%s:data:%s", schema, table, createdID)
		dataBytes, err := store.Get(TemporalFC, mainDataKey)
		require.NoError(t, err)
		assert.Nil(t, dataBytes, "Main data key should be nil for ID %s", createdID)

		uniqueIndexKey := fmt.Sprintf("%s:%s:idx-u:%s:%s", schema, table, "Name", originalEntity.Name)
		idxBytes, err := store.Get(TemporalFC, uniqueIndexKey)
		require.NoError(t, err)
		assert.Nil(t, idxBytes, "Unique index key should be nil for Name %s", originalEntity.Name)

		generalIndexKeyName := fmt.Sprintf("%s:%s:idx:%s:%s:%s", schema, table, "Name", originalEntity.Name, createdID)
		idxBytes, err = store.Get(TemporalFC, generalIndexKeyName)
		require.NoError(t, err)
		assert.Nil(t, idxBytes, "General name index key should be nil for Name %s, ID %s", originalEntity.Name, createdID)

		generalIndexKeyID := fmt.Sprintf("%s:%s:idx:%s:%s:%s", schema, table, "ID", createdID, createdID)
		idxBytes, err = store.Get(TemporalFC, generalIndexKeyID)
		require.NoError(t, err)
		assert.Nil(t, idxBytes, "General ID index key should be nil for ID %s", createdID)
	}
}
