package db_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"deadalus-orch/server/internal/infrastructure/db"
)

func newTestTTLRepositoryDefaultIdGeneratorPebble(t *testing.T) (*db.Repository[testEntity], db.KVStore, error) {
	store := newTestPebbleStore(t, []string{DefaultFC, TestFC}, []string{TemporalFC})
	repository, err := db.NewRepository[testEntity](store, TemporalFC, testColumnFamilySector, "test_schema", &db.DefaultIDGeneratorFactory{})
	return repository, store, err
}

func newTestTTLRepositoryPebble(t *testing.T) (*db.Repository[testEntity], db.KVStore, error) {
	store := newTestPebbleStore(t, []string{DefaultFC, TestFC}, []string{TemporalFC})
	iGF := NewTestIDGeneratorFactory([]string{"123", "456"})
	repository, err := db.NewRepository[testEntity](store, TemporalFC, testColumnFamilySector, "test_schema", iGF)
	return repository, store, err
}

func newTestRepositoryPebble(t *testing.T) (*db.Repository[testEntity], error) {
	store := newTestPebbleStore(t, []string{DefaultFC, TestFC}, []string{TemporalFC})
	iGF := NewTestIDGeneratorFactory([]string{"123", "456"})
	return db.NewRepository[testEntity](store, TestFC, testColumnFamilySector, "test_schema", iGF)
}

func newTestDeterministicRepositoryPebble(t *testing.T) (*db.Repository[testEntity], error) {
	store := newTestPebbleStore(t, []string{DefaultFC, TestFC}, []string{TemporalFC})
	iGF := &db.DeterministicIDGeneratorFactory{}
	return db.NewRepository[testEntity](store, TestFC, testColumnFamilySector, "test_schema", iGF)
}

func newTestRepositorySpesificIdsPebble(t *testing.T, ids []string) (*db.Repository[testEntity], error) {
	store := newTestPebbleStore(t, []string{DefaultFC, TestFC}, []string{TemporalFC})
	iGF := NewTestIDGeneratorFactory(ids)
	return db.NewRepository[testEntity](store, TestFC, testColumnFamilySector, "test_schema", iGF)
}

func newTestRepositoryDefaultIdGeneratorPebble(t *testing.T) (*db.Repository[testEntity], error) {
	store := newTestPebbleStore(t, []string{DefaultFC, TestFC}, []string{TemporalFC})

	return db.NewRepository[testEntity](store, TestFC, testColumnFamilySector, "test_schema", &db.DefaultIDGeneratorFactory{})
}

func newTestNRepositoryPebble(t *testing.T) (*db.Repository[UserComplex], error) {
	store := newTestPebbleStore(t, []string{DefaultFC, TestFC}, []string{TemporalFC})
	iGF := NewTestIDGeneratorFactory([]string{"123", "456"})
	return db.NewRepository[UserComplex](store, TestFC, testColumnFamilySector, "test_schema", iGF)
}

func newNestedEntityTestPebbleRepositoryPebble(t *testing.T) (*db.Repository[NestedEntityTest], error) {
	store := newTestPebbleStore(t, []string{DefaultFC, TestFC}, []string{TemporalFC})       // Assumes newPebbleStore is defined in this file
	iGF := NewTestIDGeneratorFactory([]string{"pnid1", "pnid2", "pnid3", "pnid4", "pnid5"}) // Example IDs for Pebble
	return db.NewRepository[NestedEntityTest](store, TestFC, testColumnFamilySector, "nested_entity_schema_pebble", iGF)
}

// --- Conditional Uniqueness Tests ---

func newTestConditionalUniqueRepoPebble(t *testing.T, initialIDs []string) (*db.Repository[ConditionalUniqueEntity], db.KVStore) {
	store := newTestPebbleStore(t, []string{DefaultFC, TestFC}, []string{TemporalFC})
	idGenerator := NewTestIDGeneratorFactory(initialIDs)
	repo, err := db.NewRepository[ConditionalUniqueEntity](store, TestFC, testColumnFamilySector, "test_schema_cond", idGenerator)
	require.NoError(t, err, "Failed to create repository for ConditionalUniqueEntity")
	return repo, store
}

func TestRepository_PutAndGet_Pebble(t *testing.T) {
	repo, err := newTestRepositoryPebble(t)
	require.NoError(t, err)
	entity := testEntity{ID: "----", Name: "Alice"}
	now := time.Now()

	id, err := repo.Create(&entity, now)
	require.NoError(t, err)
	assert.Equal(t, id, "123")

	found, err := repo.FindByField("ID", "123", now)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, "123", found.ID)
	assert.Equal(t, entity.Name, found.Name)

	found, err = repo.FindByField("Name", "Alice", now)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, "123", found.ID)
	assert.Equal(t, entity.Name, found.Name)
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
	creationTime := time.Now()

	createdID, err := repo.Create(&entity, creationTime)
	require.NoError(t, err)
	require.NotEmpty(t, createdID)

	// Verify immediately after creation
	found, err := repo.FindByField("ID", createdID, time.Now())
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, entity.Name, found.Name)
	// TTL field is not usually part of the retrieved data unless explicitly handled by the ORM layer
	// So we don't assert found.TTL == entity.TTL

	schema := "test_schema"
	table := entity.TableName()
	now := time.Now()

	// Verify directly from kvStore (TemporalFC) before ttl

	mainDataKey := fmt.Sprintf("%s:%s:data:%s", schema, table, createdID)
	dataBytes, err := store.Get(TemporalFC, testColumnFamilySector, mainDataKey, now)
	require.NoError(t, err)
	assert.NotNil(t, dataBytes)

	uniqueIndexKey := fmt.Sprintf("%s:%s:idx-u:%s:%s", schema, table, "Name", entity.Name)
	idxBytes, err := store.Get(TemporalFC, testColumnFamilySector, uniqueIndexKey, now)
	require.NoError(t, err)
	assert.NotNil(t, idxBytes)

	generalIndexKeyName := fmt.Sprintf("%s:%s:idx:%s:%s:%s", schema, table, "Name", entity.Name, createdID)
	idxBytes, err = store.Get(TemporalFC, testColumnFamilySector, generalIndexKeyName, now)
	require.NoError(t, err)
	assert.NotNil(t, idxBytes)

	generalIndexKeyLastName := fmt.Sprintf("%s:%s:idx:%s:%s:%s", schema, table, "LastName", entity.LastName, createdID)
	idxBytes, err = store.Get(TemporalFC, testColumnFamilySector, generalIndexKeyLastName, now)
	require.NoError(t, err)
	assert.NotNil(t, idxBytes)

	generalIndexKeyAge := fmt.Sprintf("%s:%s:idx:%s:%d:%s", schema, table, "Age", entity.Age, createdID)
	idxBytes, err = store.Get(TemporalFC, testColumnFamilySector, generalIndexKeyAge, now)
	require.NoError(t, err)
	assert.NotNil(t, idxBytes)

	generalIndexKeyID := fmt.Sprintf("%s:%s:idx:%s:%s:%s", schema, table, "ID", createdID, createdID)
	idxBytes, err = store.Get(TemporalFC, testColumnFamilySector, generalIndexKeyID, now)
	require.NoError(t, err)
	assert.NotNil(t, idxBytes)

	// Wait for TTL to expire
	time.Sleep(3 * time.Second)
	afterSleepNow := time.Now()

	// Verify entity is gone from repository
	notFound, err := repo.FindByField("ID", createdID, afterSleepNow)
	require.NoError(t, err)
	assert.Nil(t, notFound)

	// Verify directly from kvStore (TemporalFC) after ttl
	// We use a 'now' that is definitely after expiry for these checks
	expiredTimeCheck := creationTime.Add(time.Duration(entity.TTL+1) * time.Second)

	mainDataKey = fmt.Sprintf("%s:%s:data:%s", schema, table, createdID)
	dataBytes, err = store.Get(TemporalFC, testColumnFamilySector, mainDataKey, expiredTimeCheck)
	require.NoError(t, err)
	assert.Nil(t, dataBytes)

	uniqueIndexKey = fmt.Sprintf("%s:%s:idx-u:%s:%s", schema, table, "Name", entity.Name)
	idxBytes, err = store.Get(TemporalFC, testColumnFamilySector, uniqueIndexKey, expiredTimeCheck)
	require.NoError(t, err)
	assert.Nil(t, idxBytes)

	generalIndexKeyName = fmt.Sprintf("%s:%s:idx:%s:%s:%s", schema, table, "Name", entity.Name, createdID)
	idxBytes, err = store.Get(TemporalFC, testColumnFamilySector, generalIndexKeyName, expiredTimeCheck)
	require.NoError(t, err)
	assert.Nil(t, idxBytes)

	generalIndexKeyLastName = fmt.Sprintf("%s:%s:idx:%s:%s:%s", schema, table, "LastName", entity.LastName, createdID)
	idxBytes, err = store.Get(TemporalFC, testColumnFamilySector, generalIndexKeyLastName, expiredTimeCheck)
	require.NoError(t, err)
	assert.Nil(t, idxBytes)

	generalIndexKeyAge = fmt.Sprintf("%s:%s:idx:%s:%d:%s", schema, table, "Age", entity.Age, createdID)
	idxBytes, err = store.Get(TemporalFC, testColumnFamilySector, generalIndexKeyAge, expiredTimeCheck)
	require.NoError(t, err)
	assert.Nil(t, idxBytes)

	generalIndexKeyID = fmt.Sprintf("%s:%s:idx:%s:%s:%s", schema, table, "ID", createdID, createdID)
	idxBytes, err = store.Get(TemporalFC, testColumnFamilySector, generalIndexKeyID, expiredTimeCheck)
	require.NoError(t, err)
	assert.Nil(t, idxBytes)

	err = store.CleanExpiredKeys(time.Now())
	require.NoError(t, err)

	// Dump all data and verify that the keys are gone
	dumpX, err := store.DumpAll()
	require.NoError(t, err)

	// The dump should be empty or not contain the test column family
	assert.Empty(t, dumpX)
}

func TestPebbleRepository_TTL_BasicExpiration_UpdateTTL(t *testing.T) {
	repo, store, err := newTestTTLRepositoryPebble(t)
	require.NoError(t, err)

	entity := testEntity{
		Name:     "ttlTestEntity",
		Age:      20,
		LastName: "Gomez",
		TTL:      2, // 2 seconds
	}
	creationTime := time.Now()

	createdID, err := repo.Create(&entity, creationTime)
	require.NoError(t, err)
	require.NotEmpty(t, createdID)

	// Verify immediately after creation
	found, err := repo.FindByField("ID", createdID, time.Now())
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, entity.Name, found.Name)
	// TTL field is not usually part of the retrieved data unless explicitly handled by the ORM layer
	// So we don't assert found.TTL == entity.TTL

	schema := "test_schema"
	table := entity.TableName()
	now := time.Now()

	// Verify directly from kvStore (TemporalFC) before ttl

	mainDataKey := fmt.Sprintf("%s:%s:data:%s", schema, table, createdID)
	dataBytes, err := store.Get(TemporalFC, testColumnFamilySector, mainDataKey, now)
	require.NoError(t, err)
	assert.NotNil(t, dataBytes)

	uniqueIndexKey := fmt.Sprintf("%s:%s:idx-u:%s:%s", schema, table, "Name", entity.Name)
	idxBytes, err := store.Get(TemporalFC, testColumnFamilySector, uniqueIndexKey, now)
	require.NoError(t, err)
	assert.NotNil(t, idxBytes)

	generalIndexKeyName := fmt.Sprintf("%s:%s:idx:%s:%s:%s", schema, table, "Name", entity.Name, createdID)
	idxBytes, err = store.Get(TemporalFC, testColumnFamilySector, generalIndexKeyName, now)
	require.NoError(t, err)
	assert.NotNil(t, idxBytes)

	generalIndexKeyLastName := fmt.Sprintf("%s:%s:idx:%s:%s:%s", schema, table, "LastName", entity.LastName, createdID)
	idxBytes, err = store.Get(TemporalFC, testColumnFamilySector, generalIndexKeyLastName, now)
	require.NoError(t, err)
	assert.NotNil(t, idxBytes)

	generalIndexKeyAge := fmt.Sprintf("%s:%s:idx:%s:%d:%s", schema, table, "Age", entity.Age, createdID)
	idxBytes, err = store.Get(TemporalFC, testColumnFamilySector, generalIndexKeyAge, now)
	require.NoError(t, err)
	assert.NotNil(t, idxBytes)

	generalIndexKeyID := fmt.Sprintf("%s:%s:idx:%s:%s:%s", schema, table, "ID", createdID, createdID)
	idxBytes, err = store.Get(TemporalFC, testColumnFamilySector, generalIndexKeyID, now)
	require.NoError(t, err)
	assert.NotNil(t, idxBytes)

	entity = testEntity{
		ID:       createdID, // Use the same ID to update
		Name:     "ttlTestEntity",
		Age:      20,
		LastName: "Gomez",
		TTL:      5, // 5 seconds, update ttl
	}
	updateTime := time.Now()

	result, err := repo.Update(&entity, updateTime)
	assert.True(t, result)
	require.NoError(t, err)

	time.Sleep(3 * time.Second)
	found, err = repo.FindByField("ID", createdID, time.Now())
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, entity.Name, found.Name)

	// Wait for TTL to expire
	time.Sleep(6 * time.Second)
	afterSleepNow := time.Now()

	// Verify entity is gone from repository
	notFound, err := repo.FindByField("ID", createdID, afterSleepNow)
	require.NoError(t, err)
	assert.Nil(t, notFound)

	// Verify directly from kvStore (TemporalFC) after ttl
	// We use a 'now' that is definitely after expiry for these checks

	mainDataKey = fmt.Sprintf("%s:%s:data:%s", schema, table, createdID)
	dataBytes, err = store.Get(TemporalFC, testColumnFamilySector, mainDataKey, afterSleepNow)
	require.NoError(t, err)
	assert.Nil(t, dataBytes)

	uniqueIndexKey = fmt.Sprintf("%s:%s:idx-u:%s:%s", schema, table, "Name", entity.Name)
	idxBytes, err = store.Get(TemporalFC, testColumnFamilySector, uniqueIndexKey, afterSleepNow)
	require.NoError(t, err)
	assert.Nil(t, idxBytes)

	generalIndexKeyName = fmt.Sprintf("%s:%s:idx:%s:%s:%s", schema, table, "Name", entity.Name, createdID)
	idxBytes, err = store.Get(TemporalFC, testColumnFamilySector, generalIndexKeyName, afterSleepNow)
	require.NoError(t, err)
	assert.Nil(t, idxBytes)

	generalIndexKeyLastName = fmt.Sprintf("%s:%s:idx:%s:%s:%s", schema, table, "LastName", entity.LastName, createdID)
	idxBytes, err = store.Get(TemporalFC, testColumnFamilySector, generalIndexKeyLastName, afterSleepNow)
	require.NoError(t, err)
	assert.Nil(t, idxBytes)

	generalIndexKeyAge = fmt.Sprintf("%s:%s:idx:%s:%d:%s", schema, table, "Age", entity.Age, createdID)
	idxBytes, err = store.Get(TemporalFC, testColumnFamilySector, generalIndexKeyAge, afterSleepNow)
	require.NoError(t, err)
	assert.Nil(t, idxBytes)

	generalIndexKeyID = fmt.Sprintf("%s:%s:idx:%s:%s:%s", schema, table, "ID", createdID, createdID)
	idxBytes, err = store.Get(TemporalFC, testColumnFamilySector, generalIndexKeyID, afterSleepNow)
	require.NoError(t, err)
	assert.Nil(t, idxBytes)

	err = store.CleanExpiredKeys(time.Now())
	require.NoError(t, err)

	// Dump all data and verify that the keys are gone
	dumpX, err := store.DumpAll()
	require.NoError(t, err)

	// The dump should be empty or not contain the test column family
	assert.Empty(t, dumpX)
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
	creationTime := time.Now()
	createdIds, err := repo.BulkCreate(entitiesToCreate, creationTime)
	require.NoError(t, err)
	require.Len(t, createdIds, 3)
	_, idB, idC := createdIds[0], createdIds[1], createdIds[2]

	// 3. Verify initial retrieval
	for i, currentID := range createdIds {
		found, err := repo.FindByField("ID", currentID, time.Now())
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
	updateTime := time.Now()
	updateResults, err := repo.BulkUpdate([]*testEntity{entityB, entityC}, updateTime)
	require.NoError(t, err)
	for _, res := range updateResults {
		assert.True(t, res)
	}

	// 6. Verify entities immediately after update
	nowCheck1 := time.Now()
	foundB, _ := repo.FindByField("ID", idB, nowCheck1)
	require.NotNil(t, foundB)
	assert.Equal(t, entityBUpdateName, foundB.Name)
	foundC, _ := repo.FindByField("ID", idC, nowCheck1)
	require.NotNil(t, foundC)
	assert.Equal(t, entityCUpdateName, foundC.Name)

	// 7. Timing and Expiration Checks
	// Wait 4 seconds (entityA TTL 3s; entityB initial TTL 2s, now 6s; entityC TTL 8s)
	time.Sleep(4 * time.Second)
	nowCheck2 := time.Now()

	// idA (3s TTL) should be GONE

	// idB (orig 2s, now 6s TTL) should EXIST
	foundB, err = repo.FindByField("ID", idB, nowCheck2)
	require.NoError(t, err)
	require.NotNil(t, foundB)
	assert.Equal(t, entityBUpdateName, foundB.Name)

	// idC (8s TTL) should EXIST
	foundC, err = repo.FindByField("ID", idC, nowCheck2)
	require.NoError(t, err)
	require.NotNil(t, foundC)
	assert.Equal(t, entityCUpdateName, foundC.Name)

	// Wait another 3 seconds (total 7 seconds)
	// (entityB 6s TTL; entityC 8s TTL)
	time.Sleep(3 * time.Second)
	nowCheck3 := time.Now()

	// idB (6s TTL) should be GONE
	foundB, err = repo.FindByField("ID", idB, nowCheck3)
	require.NoError(t, err)
	assert.Nil(t, foundB)

	// idC (8s TTL) should EXIST
	foundC, err = repo.FindByField("ID", idC, nowCheck3)
	require.NoError(t, err)
	require.NotNil(t, foundC)
	assert.Equal(t, entityCUpdateName, foundC.Name)

	// Wait another 2 seconds (total 9 seconds)
	// (entityC 8s TTL)
	time.Sleep(2 * time.Second)
	nowCheck4 := time.Now()

	// idC (8s TTL) should be GONE
	foundC, err = repo.FindByField("ID", idC, nowCheck4)
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
	creationTime := time.Now()
	createdIds, err := repo.BulkCreate(entitiesToCreate, creationTime)
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

		found, err := repo.FindByField("ID", createdID, time.Now())
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, originalEntity.Name, found.Name)
	}

	// Wait for the max TTL to expire + a little buffer
	time.Sleep(time.Duration(maxTTL+1) * time.Second)
	afterSleepNow := time.Now()

	schema := "test_schema"
	table := entitiesToCreate[0].TableName() // All entities are of the same type
	// Use a 'now' that is definitely after expiry for these checks
	expiredTimeCheck := creationTime.Add(time.Duration(maxTTL+1) * time.Second)

	for i, createdID := range createdIds {
		originalEntity := entitiesToCreate[i] // Now has ID assigned

		// Verify entity is gone from repository
		notFoundInRepo, err := repo.FindByField("ID", createdID, afterSleepNow)
		require.NoError(t, err)
		assert.Nil(t, notFoundInRepo)

		// Verify directly from kvStore (TemporalFC)
		mainDataKey := fmt.Sprintf("%s:%s:data:%s", schema, table, createdID)
		dataBytes, err := store.Get(TemporalFC, testColumnFamilySector, mainDataKey, expiredTimeCheck)
		require.NoError(t, err)
		assert.Nil(t, dataBytes, "Main data key should be nil for ID %s", createdID)

		uniqueIndexKey := fmt.Sprintf("%s:%s:idx-u:%s:%s", schema, table, "Name", originalEntity.Name)
		idxBytes, err := store.Get(TemporalFC, testColumnFamilySector, uniqueIndexKey, expiredTimeCheck)
		require.NoError(t, err)
		assert.Nil(t, idxBytes, "Unique index key should be nil for Name %s", originalEntity.Name)

		generalIndexKeyName := fmt.Sprintf("%s:%s:idx:%s:%s:%s", schema, table, "Name", originalEntity.Name, createdID)
		idxBytes, err = store.Get(TemporalFC, testColumnFamilySector, generalIndexKeyName, expiredTimeCheck)
		require.NoError(t, err)
		assert.Nil(t, idxBytes, "General name index key should be nil for Name %s, ID %s", originalEntity.Name, createdID)

		generalIndexKeyID := fmt.Sprintf("%s:%s:idx:%s:%s:%s", schema, table, "ID", createdID, createdID)
		idxBytes, err = store.Get(TemporalFC, testColumnFamilySector, generalIndexKeyID, expiredTimeCheck)
		require.NoError(t, err)
		assert.Nil(t, idxBytes, "General ID index key should be nil for ID %s", createdID)
	}
}

func TestRepository_Create_DuplicatePrimaryKey_Pebble(t *testing.T) {
	repo, err := newTestDeterministicRepositoryPebble(t) // Uses DeterministicIDGeneratorFactory
	require.NoError(t, err)

	entity1 := testEntity{ID: "dup-pk-pebble-1", Name: "FirstPebbleEntity"}
	id1, err := repo.Create(&entity1, time.Now())
	require.NoError(t, err)
	assert.Equal(t, "dup-pk-pebble-1", id1)

	// Attempt to create another entity with the same ID
	entity2 := testEntity{ID: "dup-pk-pebble-1", Name: "SecondPebbleEntitySameID"}
	_, err = repo.Create(&entity2, time.Now())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate primary key: ID = dup-pk-pebble-1 already exists")

	// Verify only the first entity is there
	found, err := repo.FindByField("ID", "dup-pk-pebble-1", time.Now())
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, "FirstPebbleEntity", found.Name) // Should be the name of the first entity
}

func TestRepository_BulkCreate_DuplicatePrimaryKey_InDB_Pebble(t *testing.T) {
	repo, err := newTestDeterministicRepositoryPebble(t) // Uses DeterministicIDGeneratorFactory
	require.NoError(t, err)

	// Pre-existing entity
	existingEntity := testEntity{ID: "existing-pebble-pk", Name: "AlreadyInPebbleDB"}
	_, err = repo.Create(&existingEntity, time.Now())
	require.NoError(t, err)

	entitiesToBulkCreate := []*testEntity{
		{ID: "new-pebble-pk-1", Name: "NewPebbleEntity1"},
		{ID: "existing-pebble-pk", Name: "TryToOverwritePebble"}, // This ID conflicts
		{ID: "new-pebble-pk-2", Name: "NewPebbleEntity2"},
	}

	_, err = repo.BulkCreate(entitiesToBulkCreate, time.Now())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate primary key: ID = existing-pebble-pk already exists")

	// Verify that non-conflicting new entities were not created
	foundNew1, err := repo.FindByField("ID", "new-pebble-pk-1", time.Now())
	require.NoError(t, err)
	assert.Nil(t, foundNew1)

	// Verify existing entity is still the original one
	foundExisting, err := repo.FindByField("ID", "existing-pebble-pk", time.Now())
	require.NoError(t, err)
	require.NotNil(t, foundExisting)
	assert.Equal(t, "AlreadyInPebbleDB", foundExisting.Name)
}

func TestRepository_BulkCreate_DuplicatePrimaryKey_InBatch_Pebble(t *testing.T) {
	repo, err := newTestDeterministicRepositoryPebble(t) // Uses DeterministicIDGeneratorFactory
	require.NoError(t, err)

	duplicateIDInBatch := "batch-dup-pebble-pk"
	entities := []*testEntity{
		{ID: "unique-batch-pebble-1", Name: "BatchPebble1"},
		{ID: duplicateIDInBatch, Name: "BatchPebble2"},
		{ID: duplicateIDInBatch, Name: "BatchPebble3"}, // Duplicate ID within the batch
	}

	_, err = repo.BulkCreate(entities, time.Now())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate primary key in input batch: ID = "+duplicateIDInBatch)

	// Verify no entities from the batch were created
	foundUnique, err := repo.FindByField("ID", "unique-batch-pebble-1", time.Now())
	require.NoError(t, err)
	assert.Nil(t, foundUnique)
}

func TestRepository_Create_EmptyProvidedID_Pebble(t *testing.T) {
	repo, err := newTestDeterministicRepositoryPebble(t) // Uses DeterministicIDGeneratorFactory
	require.NoError(t, err)

	entityWithEmptyID := testEntity{
		ID:   "", // Explicitly empty ID
		Name: "EntityWithEmptyIDPebble",
	}

	_, err = repo.Create(&entityWithEmptyID, time.Now())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "primary key field 'ID' cannot be empty when not generated")
}

func TestRepository_Get_NotFound_Pebble(t *testing.T) {
	repo, err := newTestRepositoryPebble(t)
	require.NoError(t, err)
	now := time.Now()

	found, err := repo.FindByField("ID", "5566", now)
	require.NoError(t, err)
	assert.Nil(t, found)
}

func TestPebbleConditionalUniquenessCreate(t *testing.T) {
	t.Run("IgnoreUniqueness", func(t *testing.T) {
		ids := []string{"id1", "id2", "id3", "id4"}
		repo, _ := newTestConditionalUniqueRepoPebble(t, ids)
		now := time.Now()

		entity1 := &ConditionalUniqueEntity{Name: "E1", UniqueValue: "uv1", ShouldIgnoreUniqueness: true}
		_, err := repo.Create(entity1, now)
		require.NoError(t, err, "Create entity1 should succeed")

		entity2 := &ConditionalUniqueEntity{Name: "E2", UniqueValue: "uv1", ShouldIgnoreUniqueness: true}
		_, err = repo.Create(entity2, now)
		require.NoError(t, err, "Create entity2 with same UniqueValue (ignored) should succeed")

		entity3 := &ConditionalUniqueEntity{Name: "E3", UniqueValue: "uv1", ShouldIgnoreUniqueness: false}
		_, err = repo.Create(entity3, now)
		require.NoError(t, err, "Create entity3 with same UniqueValue (enforced, but not previously by E1/E2) should succeed")

		entity4 := &ConditionalUniqueEntity{Name: "E3", UniqueValue: "uv1", ShouldIgnoreUniqueness: false}
		_, err = repo.Create(entity4, now)
		require.Error(t, err)
		require.Contains(t, err.Error(), "duplicate unique field: UniqueValue = uv1")

		// Verify all created
		e1, _ := repo.FindByField("ID", ids[0], now)
		require.NotNil(t, e1)
		assert.Equal(t, "uv1", e1.UniqueValue)

		e2, _ := repo.FindByField("ID", ids[1], now)
		require.NotNil(t, e2)
		assert.Equal(t, "uv1", e2.UniqueValue)

		e3, _ := repo.FindByField("ID", ids[2], now)
		require.NotNil(t, e3)
		assert.Equal(t, "uv1", e3.UniqueValue)
		assert.False(t, e3.ShouldIgnoreUniqueness)
	})

	t.Run("EnforceUniqueness", func(t *testing.T) {
		ids := []string{"id4", "id5", "id6"} // Fresh IDs for this subtest
		repo, _ := newTestConditionalUniqueRepoPebble(t, ids)
		now := time.Now()

		entity4 := &ConditionalUniqueEntity{Name: "E4", UniqueValue: "uv2", ShouldIgnoreUniqueness: false}
		_, err := repo.Create(entity4, now)
		require.NoError(t, err, "Create entity4 should succeed")

		entity5 := &ConditionalUniqueEntity{Name: "E5", UniqueValue: "uv2", ShouldIgnoreUniqueness: false}
		_, err = repo.Create(entity5, now)
		require.Error(t, err, "Create entity5 with same UniqueValue (enforced) should fail")
		assert.Contains(t, err.Error(), "duplicate unique field")

		entity6 := &ConditionalUniqueEntity{Name: "E6", UniqueValue: "uv2", ShouldIgnoreUniqueness: true}
		_, err = repo.Create(entity6, now)
		require.NoError(t, err, "Create entity6 with same UniqueValue (ignored) should succeed")

		e4, _ := repo.FindByField("ID", ids[0], now)
		require.NotNil(t, e4)
		assert.Equal(t, "uv2", e4.UniqueValue)

		e6, _ := repo.FindByField("ID", ids[2], now) // id5 failed, so entity6 gets ids[2] if factory generates sequentially
		require.NotNil(t, e6)
		assert.Equal(t, "uv2", e6.UniqueValue)
		assert.True(t, e6.ShouldIgnoreUniqueness)
	})
}

func TestPebbleConditionalUniquenessUpdate(t *testing.T) {
	t.Run("UpdateWithFlagTrueBypassesConflict", func(t *testing.T) {
		ids := []string{"idA", "idB"}
		repo, _ := newTestConditionalUniqueRepoPebble(t, ids)
		now := time.Now()

		entityA := &ConditionalUniqueEntity{ID: ids[0], Name: "EntityA", UniqueValue: "uva", ShouldIgnoreUniqueness: false}
		_, err := repo.Create(entityA, now)
		require.NoError(t, err)

		entityB := &ConditionalUniqueEntity{ID: ids[1], Name: "EntityB", UniqueValue: "uvb", ShouldIgnoreUniqueness: false}
		_, err = repo.Create(entityB, now)
		require.NoError(t, err)

		// Update entityB to have UniqueValue "uva" (conflicts with A) but with ShouldIgnoreUniqueness = true
		entityB.UniqueValue = "uva"
		entityB.ShouldIgnoreUniqueness = true
		updated, err := repo.Update(entityB, now)
		require.NoError(t, err, "Update entityB should succeed")
		assert.True(t, updated)

		// Verify B is updated
		bUpdated, _ := repo.FindByField("ID", ids[1], now)
		require.NotNil(t, bUpdated)
		assert.Equal(t, "uva", bUpdated.UniqueValue)
		assert.True(t, bUpdated.ShouldIgnoreUniqueness)
	})

	t.Run("UpdateWithFlagFalseHitsConflict", func(t *testing.T) {
		ids := []string{"idC", "idD"}
		repo, _ := newTestConditionalUniqueRepoPebble(t, ids)
		now := time.Now()

		entityC := &ConditionalUniqueEntity{ID: ids[0], Name: "EntityC", UniqueValue: "uvc", ShouldIgnoreUniqueness: false}
		_, err := repo.Create(entityC, now)
		require.NoError(t, err)

		entityD := &ConditionalUniqueEntity{ID: ids[1], Name: "EntityD", UniqueValue: "uvd", ShouldIgnoreUniqueness: false}
		_, err = repo.Create(entityD, now)
		require.NoError(t, err)

		// Attempt to update entityD to have UniqueValue "uvc" (conflicts with C) with ShouldIgnoreUniqueness = false
		entityD.UniqueValue = "uvc"
		entityD.ShouldIgnoreUniqueness = false
		updated, err := repo.Update(entityD, now)
		require.Error(t, err, "Update entityD should fail due to unique constraint")
		assert.False(t, updated)
		assert.Contains(t, err.Error(), "duplicate unique field")
	})

	t.Run("UpdateFlagFromFalseToTrueThenCreateNew", func(t *testing.T) {
		ids := []string{"idE", "idF"}
		repo, _ := newTestConditionalUniqueRepoPebble(t, ids)
		now := time.Now()

		entityE := &ConditionalUniqueEntity{ID: ids[0], Name: "EntityE", UniqueValue: "uve", ShouldIgnoreUniqueness: false}
		_, err := repo.Create(entityE, now)
		require.NoError(t, err)

		// Update entityE to set ShouldIgnoreUniqueness = true
		entityE.ShouldIgnoreUniqueness = true
		updated, err := repo.Update(entityE, now)
		require.NoError(t, err, "Update entityE should succeed")
		assert.True(t, updated)

		entityF := &ConditionalUniqueEntity{ID: ids[1], Name: "EntityF", UniqueValue: "uve", ShouldIgnoreUniqueness: true}
		_, err = repo.Create(entityF, now)
		require.NoError(t, err, "Create entityF should succeed as E is ignoring uniqueness")

		// Verify F
		fCreated, _ := repo.FindByField("ID", ids[1], now)
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
	now := time.Now()

	id, err := repo.Create(&a, now)
	assert.Equal(t, id, "123")
	require.NoError(t, err)
	id, err = repo.Create(&b, now)
	require.NoError(t, err)
	assert.Equal(t, id, "456")

	resA, err := repo.FindByField("ID", "123", now)
	require.NoError(t, err)
	assert.Equal(t, a.Name, resA.Name)

	resB, err := repo.FindByField("ID", "456", now)
	require.NoError(t, err)
	assert.Equal(t, b.Name, resB.Name)
}

func TestRepository_SearchByPatternPaginatedKV_MatchSingle_Pebble(t *testing.T) {
	repo, err := newTestRepositoryPebble(t)
	require.NoError(t, err)
	now := time.Now()

	entity := testEntity{ID: "user:123:name", Name: "Alice"}
	id, err := repo.Create(&entity, now)
	assert.Equal(t, id, "123")
	require.NoError(t, err)

	results, err := repo.Find("Name=Alice", 1000, "", now)
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
	now := time.Now()
	id, err := repo.Create(&entity, now)
	require.NoError(t, err)
	assert.Equal(t, "123", id)

	// Change Name
	entity.Name = "Alice Smith"
	updated, err := repo.Update(&entity, now)
	require.NoError(t, err)
	assert.True(t, updated)

	// Verify updated value
	res, err := repo.FindByField("ID", "123", now)
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Equal(t, "Alice Smith", res.Name)
}

func TestRepository_Update_UniqueIndexChange_Pebble(t *testing.T) {
	repo, err := newTestRepositoryPebble(t)
	require.NoError(t, err)
	now := time.Now()

	entity := testEntity{ID: "123", Name: "Alice"}
	_, err = repo.Create(&entity, now)
	require.NoError(t, err)

	entity.Name = "Bob"
	updated, err := repo.Update(&entity, now)
	require.NoError(t, err)
	assert.True(t, updated)

	// Should no longer be found by old index
	old, err := repo.FindByField("Name", "Alice", now)
	require.NoError(t, err)
	assert.Nil(t, old)

	// Should now be found by new index
	newFound, err := repo.FindByField("Name", "Bob", now)
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
	now := time.Now()

	_, err = repo.Create(&a, now)
	require.NoError(t, err)
	_, err = repo.Create(&b, now)
	require.NoError(t, err)

	// Try to rename Alice to "Bob" → should fail (unique collision)
	a.Name = "Bob"
	updated, err := repo.Update(&a, now)
	assert.Error(t, err)
	assert.False(t, updated)
}

func TestRepository_Update_NotFound_Pebble(t *testing.T) {
	repo, err := newTestRepositoryPebble(t)
	require.NoError(t, err)
	now := time.Now()

	entity := testEntity{ID: "999", Name: "Zoe"}
	ok, err := repo.Update(&entity, now)
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestRepository_Delete_Success_Pebble(t *testing.T) {
	repo, err := newTestRepositoryPebble(t)
	require.NoError(t, err)

	// Crear entidad
	entity := testEntity{ID: "---", Name: "Charlie"}
	now := time.Now()
	id, err := repo.Create(&entity, now)
	require.NoError(t, err)
	assert.Equal(t, "123", id)

	// Asegurar que se puede encontrar
	found, err := repo.FindByField("ID", "123", now)
	require.NoError(t, err)
	require.NotNil(t, found)

	// Eliminar entidad
	deleted, err := repo.Delete("123", now)
	require.NoError(t, err)
	assert.True(t, deleted)

	// Verificar que ya no se encuentra
	found, err = repo.FindByField("ID", "123", now)
	require.NoError(t, err)
	assert.Nil(t, found)

	// Verificar que el índice único también se eliminó
	found, err = repo.FindByField("Name", "Charlie", now)
	require.NoError(t, err)
	assert.Nil(t, found)
}

func TestRepository_Delete_NotFound_Pebble(t *testing.T) {
	repo, err := newTestRepositoryPebble(t)
	require.NoError(t, err)
	now := time.Now()

	deleted, err := repo.Delete("nonexistent-id", now)
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
		_, err := repo.Create(&entity, time.Now())
		require.NoError(t, err)
	}
	now := time.Now()

	// === Filtro simple ===
	targetName := "Name_0003"
	results, err := repo.Find(fmt.Sprintf("Name=%s", targetName), 10, "", now)
	require.NoError(t, err)
	require.Len(t, results.Entities, 1)
	assert.Equal(t, targetName, results.Entities[0].Name)

	firstPage, err := repo.Find("Name=Name_0003 | Name=Name_0004 | Name=Name_0005", 2, "", now)
	require.NoError(t, err)
	assert.Len(t, firstPage.Entities, 2)
	assert.NotEmpty(t, firstPage.Cursor)

	secondPage, err := repo.Find("Name=Name_0003 | Name=Name_0004 | Name=Name_0005", 2, firstPage.Cursor, now)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(secondPage.Entities), 1) // solo hay 3 en total

	orFilter := "Name=Name_0007 | Name=Name_0010"
	orResults, err := repo.Find(orFilter, 10, "", now)
	require.NoError(t, err)
	assert.Len(t, orResults.Entities, 2)
	names := map[string]bool{
		"Name_0007": true,
		"Name_0010": true,
	}
	for _, e := range orResults.Entities {
		assert.True(t, names[e.Name])
	}

	andResults, err := repo.Find("Name=Name_0003 & Name=Name_0004", 10, "", now)
	require.NoError(t, err)
	assert.Empty(t, andResults.Entities)

	quoted, err := repo.Find("Name='Name_0003'", 10, "", now)
	require.NoError(t, err)
	require.Len(t, quoted.Entities, 1)
	assert.Equal(t, "Name_0003", quoted.Entities[0].Name)

	_, err = repo.Find("NameName_0003", 10, "", now)
	assert.Error(t, err)

	entity := testEntity{ID: "abc123", Name: "Zeta_Unique"}
	_, err = repo.Create(&entity, time.Now())
	require.NoError(t, err)

	multiField := fmt.Sprintf("ID=%s & Name=%s", entity.ID, entity.Name)
	multiResults, err := repo.Find(multiField, 10, "", now)
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
		_, err := repo.Create(&entity, time.Now())
		require.NoError(t, err)
		names[name] = true
	}
	now := time.Now()

	var filterParts []string
	for i := 10; i < 60; i++ {
		filterParts = append(filterParts, fmt.Sprintf("Name=Name_%04d", i))
	}
	filter := strings.Join(filterParts, " | ")

	limit := 7
	cursor := ""
	found := make(map[string]bool)

	for {
		page, err := repo.Find(filter, limit, cursor, now)
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
	creationTime := time.Now()
	for _, e := range seed {
		_, err := repo.Create(&e, creationTime)
		require.NoError(t, err)
	}
	now := time.Now()

	t.Run("Equal operator", func(t *testing.T) {
		res, err := repo.Find("Age=30", 10, "", now)
		require.NoError(t, err)
		require.Len(t, res.Entities, 1)
		assert.Equal(t, "Bea", res.Entities[0].Name)
	})

	t.Run("Not equal operator", func(t *testing.T) {
		res, err := repo.Find("Age!=30", 10, "", now)
		require.NoError(t, err)
		assert.Len(t, res.Entities, 4)
		for _, e := range res.Entities {
			assert.NotEqual(t, 30, e.Age)
		}
	})

	t.Run("Greater than operator", func(t *testing.T) {
		res, err := repo.Find("Age>30", 10, "", now)
		require.NoError(t, err)
		assert.Len(t, res.Entities, 2)
		assert.ElementsMatch(t, []string{"Cleo", "Eva"}, []string{res.Entities[0].Name, res.Entities[1].Name})
	})

	t.Run("Greater than or equal", func(t *testing.T) {
		res, err := repo.Find("Age>=30", 10, "", now)
		require.NoError(t, err)
		assert.Len(t, res.Entities, 3)
	})

	t.Run("Less than operator", func(t *testing.T) {
		res, err := repo.Find("Age<30", 10, "", now)
		require.NoError(t, err)
		assert.Len(t, res.Entities, 2)
	})

	t.Run("Less than or equal", func(t *testing.T) {
		res, err := repo.Find("Age<=25", 10, "", now)
		require.NoError(t, err)
		assert.Len(t, res.Entities, 2)
	})

	t.Run("LIKE operator - prefix", func(t *testing.T) {
		res, err := repo.Find("LastName LIKE Z*", 10, "", now)
		require.NoError(t, err)
		require.Len(t, res.Entities, 1)
		assert.Equal(t, "Ana", res.Entities[0].Name)
	})

	t.Run("LIKE operator - suffix", func(t *testing.T) {
		res, err := repo.Find("LastName LIKE *nez", 10, "", now)
		require.NoError(t, err)
		require.Len(t, res.Entities, 2)
		assert.ElementsMatch(t, []string{"Cleo", "Bea"}, []string{res.Entities[0].Name, res.Entities[1].Name})
	})

	t.Run("LIKE operator - contains", func(t *testing.T) {
		res, err := repo.Find("LastName LIKE *ela*", 10, "", now)
		require.NoError(t, err)
		assert.Len(t, res.Entities, 1)
		assert.Equal(t, "Eva", res.Entities[0].Name)
	})

	t.Run("BETWEEN operator", func(t *testing.T) {
		res, err := repo.Find("Age BETWEEN 25 AND 35", 10, "", now)
		require.NoError(t, err)
		assert.Len(t, res.Entities, 3)
		names := []string{res.Entities[0].Name, res.Entities[1].Name, res.Entities[2].Name}
		assert.ElementsMatch(t, []string{"Bea", "Dana", "Eva"}, names)
	})

	t.Run("Combined operators", func(t *testing.T) {
		res, err := repo.Find("Age>=25 & Age<=35", 10, "", now)
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
		_, err := repo.Create(&entity, time.Now())
		require.NoError(t, err)
		names[name] = true
	}
	now := time.Now()

	t.Run("Simple pagination loop with 100 per page", func(t *testing.T) {
		cursor := ""
		found := make(map[string]bool)
		limit := 100

		for {
			res, err := repo.Find("Age>=25", limit, cursor, now)
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
			res, err := repo.Find(filterStr, limit, cursor, now)
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
		res, err := repo.Find("Name=User_0001 & Age=25", 10, "", now)
		require.NoError(t, err)
		if len(res.Entities) == 1 {
			assert.Equal(t, "User_0001", res.Entities[0].Name)
			assert.Equal(t, 25, res.Entities[0].Age)
		} else {
			assert.Empty(t, res.Entities)
		}
	})

	t.Run("LIKE operator with many matches", func(t *testing.T) {
		res, err := repo.Find("LastName LIKE Last_1*", 100, "", now)
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
	creationTime := time.Now()
	for _, e := range seed {
		_, err := repo.Create(&e, creationTime)
		require.NoError(t, err)
	}
	now := time.Now()

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
			res, err := repo.Find(tt.filter, 100, "", now)
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
		now := time.Now()
		ids, err := repo.BulkCreate(entities, now)
		require.NoError(t, err)
		assert.Len(t, ids, 3)

		for _, id := range ids {
			found, err := repo.FindByField("ID", id, now)
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
		now := time.Now()
		_, err = repo.BulkCreate(first, now)
		require.NoError(t, err)

		second := []*testEntity{
			{ID: "---", Name: "UniqueName2"}, // duplicate
			{ID: "---", Name: "UniqueName3"},
		}
		_, err = repo.BulkCreate(second, now)
		assert.Error(t, err)
		require.Contains(t, err.Error(), "duplicate")
	})

	t.Run("Empty list should return empty result", func(t *testing.T) {
		repo, err := newTestRepositoryDefaultIdGeneratorPebble(t)
		require.NoError(t, err)

		ids, err := repo.BulkCreate([]*testEntity{}, time.Now())
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
		now := time.Now()
		ids, err := repo.BulkCreate(bulk, now)
		require.NoError(t, err)
		assert.Len(t, ids, 500)

		// Spot check
		found, err := repo.FindByField("Name", "Bulk_42", now)
		require.NoError(t, err)
		require.NotNil(t, found)
	})

	t.Run("Partially duplicate should fail all", func(t *testing.T) {
		repo, err := newTestRepositoryDefaultIdGeneratorPebble(t)
		require.NoError(t, err)
		now := time.Now()

		_, err = repo.Create(&testEntity{ID: "---", Name: "ConflictName"}, now)

		conflicting := []*testEntity{
			{ID: "---", Name: "NewName1"},
			{ID: "---", Name: "ConflictName"},
			{ID: "---", Name: "NewName2"},
		}
		_, err = repo.BulkCreate(conflicting, now)
		assert.Error(t, err)
		require.Contains(t, err.Error(), "duplicate")
	})
	t.Run("should fail when input slice contains duplicate unique index values", func(t *testing.T) {
		repo, err := newTestRepositoryDefaultIdGeneratorPebble(t)

		batch := []*testEntity{
			{Name: "duplicate-name", Age: 100},
			{Name: "duplicate-name", Age: 200}, // mismo Name
		}

		_, err = repo.BulkCreate(batch, time.Now())
		require.Error(t, err)
		require.Contains(t, err.Error(), "duplicate") // adapta esto según el mensaje real
	})
}

func TestRepository_BulkDelete_Pebble(t *testing.T) {
	t.Run("Delete multiple existing entities", func(t *testing.T) {
		ids := []string{"123", "456", "789"}
		repo, err := newTestRepositorySpesificIdsPebble(t, ids)
		require.NoError(t, err)
		now := time.Now()

		for _, id := range ids {
			entity := testEntity{ID: id, Name: "Name_" + id}
			_, err := repo.Create(&entity, now)
			require.NoError(t, err)
		}

		for _, id := range ids {
			found, err := repo.FindByField("ID", id, now)
			require.NoError(t, err)
			require.NotNil(t, found)
		}

		_, err = repo.BulkDelete(ids, now)
		require.NoError(t, err)

		for _, id := range ids {
			found, err := repo.FindByField("ID", id, now)
			require.NoError(t, err)
			assert.Nil(t, found)
		}
	})

	t.Run("Bulk delete with some non-existing IDs", func(t *testing.T) {
		repo, err := newTestRepositoryPebble(t)
		require.NoError(t, err)
		now := time.Now()

		entity := testEntity{ID: "999", Name: "Alive"}
		_, err = repo.Create(&entity, now)
		require.NoError(t, err)

		// Mixed list: existing and non-existing
		ids := []string{"999", "does_not_exist", "also_missing"}
		_, err = repo.BulkDelete(ids, now)
		require.NoError(t, err)

		// Ensure the one that existed is deleted
		found, err := repo.FindByField("ID", "999", now)
		require.NoError(t, err)
		assert.Nil(t, found)
	})

	t.Run("Empty list should not fail", func(t *testing.T) {
		repo, err := newTestRepositoryPebble(t)
		require.NoError(t, err)
		_, err = repo.BulkDelete([]string{}, time.Now())
		require.NoError(t, err)
	})

	t.Run("Bulk delete should also remove unique indices", func(t *testing.T) {
		ids := []string{"111", "222"}
		repo, err := newTestRepositorySpesificIdsPebble(t, ids)
		require.NoError(t, err)
		now := time.Now()

		a := testEntity{ID: "111", Name: "Alpha"}
		b := testEntity{ID: "222", Name: "Beta"}
		_, err = repo.Create(&a, now)
		require.NoError(t, err)
		_, err = repo.Create(&b, now)
		require.NoError(t, err)

		_, err = repo.BulkDelete([]string{"111", "222"}, now)
		require.NoError(t, err)

		foundA, err := repo.FindByField("Name", "Alpha", now)
		require.NoError(t, err)
		assert.Nil(t, foundA)

		foundB, err := repo.FindByField("Name", "Beta", now)
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
		now := time.Now()
		ids, err := repo.BulkCreate(original, now)
		require.NoError(t, err)
		require.Len(t, ids, 3)

		// Preparamos la actualización
		updated := []*testEntity{
			{ID: ids[0], Name: "UpdatedA"},
			{ID: ids[1], Name: "UpdatedB"},
			{ID: ids[2], Name: "UpdatedC"},
		}
		result, err := repo.BulkUpdate(updated, now)
		require.NoError(t, err)
		assert.Equal(t, []bool{true, true, true}, result)

		for i, id := range ids {
			entity, err := repo.FindByField("ID", id, now)
			require.NoError(t, err)
			assert.Equal(t, updated[i].Name, entity.Name)
		}
	})

	t.Run("Empty input should return empty result", func(t *testing.T) {
		repo, err := newTestRepositoryDefaultIdGeneratorPebble(t)
		require.NoError(t, err)

		result, err := repo.BulkUpdate([]*testEntity{}, time.Now())
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("Should return false for missing records", func(t *testing.T) {
		repo, err := newTestRepositoryDefaultIdGeneratorPebble(t)
		require.NoError(t, err)

		// Insert uno solo
		entity := &testEntity{ID: "---", Name: "UserX"}
		now := time.Now()
		createdID, err := repo.Create(entity, now)
		require.NoError(t, err)

		// Uno existe, otro no
		toUpdate := []*testEntity{
			{ID: createdID, Name: "UpdatedX"},
			{ID: "non-existent-id", Name: "ShouldFail"},
		}
		result, err := repo.BulkUpdate(toUpdate, now)
		require.NoError(t, err)
		assert.Equal(t, []bool{true, false}, result)
	})

	t.Run("Update should not insert new if ID not exists", func(t *testing.T) {
		repo, err := newTestRepositoryDefaultIdGeneratorPebble(t)
		require.NoError(t, err)

		toUpdate := []*testEntity{
			{ID: "ghost-id", Name: "Ghost"},
		}
		now := time.Now()
		result, err := repo.BulkUpdate(toUpdate, now)
		require.NoError(t, err)
		assert.Equal(t, []bool{false}, result)

		found, err := repo.FindByField("ID", "ghost-id", now)
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
		now := time.Now()
		ids, err := repo.BulkCreate(original, now)
		require.NoError(t, err)
		require.Len(t, ids, 500)

		var updated []*testEntity
		for i, id := range ids {
			updated = append(updated, &testEntity{ID: id, Name: fmt.Sprintf("Updated_%d", i)})
		}
		result, err := repo.BulkUpdate(updated, now)
		require.NoError(t, err)
		assert.Len(t, result, 500)
		for i := range result {
			assert.True(t, result[i], "index %d should be true", i)
		}

		// Spot check
		check, err := repo.FindByField("Name", "Updated_42", now)
		require.NoError(t, err)
		assert.NotNil(t, check)
	})

	t.Run("Duplicate IDs in input should update once", func(t *testing.T) {
		repo, err := newTestRepositoryDefaultIdGeneratorPebble(t)
		require.NoError(t, err)
		now := time.Now()

		entity := &testEntity{ID: "---", Name: "Original"}
		id, err := repo.Create(entity, now)
		require.NoError(t, err)

		// Duplicado el mismo ID dos veces, con valores diferentes
		toUpdate := []*testEntity{
			{ID: id, Name: "FirstUpdate"},
			{ID: id, Name: "SecondUpdate"},
		}
		result, err := repo.BulkUpdate(toUpdate, now)
		require.NoError(t, err)
		assert.Equal(t, []bool{true, true}, result)

		final, err := repo.FindByField("ID", id, now)
		require.NoError(t, err)
		assert.Equal(t, "SecondUpdate", final.Name) // el último debe prevalecer
	})

	t.Run("Nil entries should be skipped or fail gracefully", func(t *testing.T) {
		repo, err := newTestRepositoryDefaultIdGeneratorPebble(t)
		require.NoError(t, err)
		now := time.Now()

		entity := &testEntity{ID: "---", Name: "Valid"}
		id, err := repo.Create(entity, now)
		require.NoError(t, err)

		updates := []*testEntity{
			nil,
			{ID: id, Name: "ValidUpdate"},
		}
		result, err := repo.BulkUpdate(updates, now)
		require.NoError(t, err)
		assert.Equal(t, []bool{false, true}, result) // nil => false, válido => true

		final, err := repo.FindByField("ID", id, now)
		require.NoError(t, err)
		assert.Equal(t, "ValidUpdate", final.Name)
	})
}

func TestRepository_PutAndGet_Nested_Pebble(t *testing.T) {
	repo, err := newTestNRepositoryPebble(t)
	require.NoError(t, err)
	entity := UserComplex{ID: "----", Email: "Alice@x.com", Status: "active", Meta: Meta{
		Tag:         "t1",
		ConfigCode:  55,
		Description: "None",
	}}
	now := time.Now()

	id, err := repo.Create(&entity, now)
	require.NoError(t, err)
	assert.Equal(t, id, "123")

	found, err := repo.FindByField("ID", "123", now)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, "123", found.ID)
	assert.Equal(t, entity.Email, found.Email)

	found, err = repo.FindByField("Email", "Alice@x.com", now)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, "123", found.ID)
	assert.Equal(t, entity.Email, found.Email)

	found, err = repo.FindByField("Meta.Tag", "t1", now)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, "123", found.ID)
	assert.Equal(t, entity.Email, found.Email)

	found, err = repo.FindByField("Meta.Tag1", "t1", now)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Unknown field Meta.Tag1")
}

func TestRepository_BulkCreate_Nested_Pebble(t *testing.T) {
	repo, err := newNestedEntityTestPebbleRepositoryPebble(t)
	require.NoError(t, err, "Failed to create repository for NestedEntityTest (Pebble)")

	t.Run("Successful bulk creation with nested structs pebble", func(t *testing.T) {
		entities := []*NestedEntityTest{
			{ID: "---", Data: "EntityP1", Meta: NestedMetaTest{UniqueID: "uniqueP1", OTValue: "ttlP1", Description: "DescP1"}},
			{ID: "---", Data: "EntityP2", Meta: NestedMetaTest{UniqueID: "uniqueP2", OTValue: "ttlP2", Description: "DescP2"}},
			{ID: "---", Data: "EntityP3", Meta: NestedMetaTest{UniqueID: "uniqueP3", OTValue: "ttlP3", Description: "DescP3"}},
		}
		now := time.Now()

		ids, err := repo.BulkCreate(entities, now)
		require.NoError(t, err, "BulkCreate failed for valid nested entities (Pebble)")
		require.Len(t, ids, len(entities), "BulkCreate should return an ID for each entity (Pebble)")

		for i, id := range ids {
			require.NotEmpty(t, id, "Expected a non-empty ID for entity %d (Pebble)", i)
			entities[i].ID = id // Assign returned ID for later checks

			found, err := repo.FindByField("ID", id, now)
			require.NoError(t, err, "FindByField by ID failed for created entity %s (Pebble)", id)
			require.NotNil(t, found, "Should find entity by ID %s (Pebble)", id)
			assert.Equal(t, entities[i].Data, found.Data, "Data field mismatch for entity %s (Pebble)", id)
			assert.Equal(t, entities[i].Meta.UniqueID, found.Meta.UniqueID, "Meta.UniqueID field mismatch for entity %s (Pebble)", id)
			assert.Equal(t, entities[i].Meta.OTValue, found.Meta.OTValue, "Meta.TTLValue field mismatch for entity %s (Pebble)", id)
			assert.Equal(t, entities[i].Meta.Description, found.Meta.Description, "Meta.Description field mismatch for entity %s (Pebble)", id)
		}
		nowVerify := time.Now() // Use a consistent 'now' for these verification reads

		// Verify finding by nested unique index
		foundByNestedUnique, err := repo.FindByField("Meta.UniqueID", "uniqueP2", nowVerify)
		require.NoError(t, err, "FindByField by Meta.UniqueID failed (Pebble)")
		require.NotNil(t, foundByNestedUnique, "Should find entity by Meta.UniqueID 'uniqueP2' (Pebble)")
		assert.Equal(t, entities[1].ID, foundByNestedUnique.ID, "ID mismatch when finding by Meta.UniqueID (Pebble)")
		assert.Equal(t, "EntityP2", foundByNestedUnique.Data)
		assert.Equal(t, "uniqueP2", foundByNestedUnique.Meta.UniqueID)
	})

	t.Run("Bulk creation with duplicate UniqueID within the batch pebble", func(t *testing.T) {
		repoFresh, err := newNestedEntityTestPebbleRepositoryPebble(t)
		require.NoError(t, err)

		entities := []*NestedEntityTest{
			{ID: "---", Data: "EntityPX", Meta: NestedMetaTest{UniqueID: "duplicateKeyInBatchP", OTValue: "ttlPX", Description: "DescPX"}},
			{ID: "---", Data: "EntityPY", Meta: NestedMetaTest{UniqueID: "anotherUniqueInBatchP", OTValue: "ttlPY", Description: "DescPY"}},
			{ID: "---", Data: "EntityPZ", Meta: NestedMetaTest{UniqueID: "duplicateKeyInBatchP", OTValue: "ttlPZ", Description: "DescPZ"}}, // Duplicate UniqueID
		}
		now := time.Now()

		ids, err := repoFresh.BulkCreate(entities, now)
		require.Error(t, err, "BulkCreate should fail due to duplicate UniqueID within the batch (Pebble)")
		assert.Nil(t, ids, "IDs should be nil on batch creation failure (Pebble)")
		require.Contains(t, err.Error(), "duplicate", "Error message should indicate a duplicate key problem (Pebble)")

		// Verify no entities were partially inserted
		found, err := repoFresh.FindByField("Meta.UniqueID", "duplicateKeyInBatchP", now)
		require.NoError(t, err)
		assert.Nil(t, found, "No entity should be found with the conflicting UniqueID if batch failed (Pebble)")

		found, err = repoFresh.FindByField("Meta.UniqueID", "anotherUniqueInBatchP", now)
		require.NoError(t, err)
		assert.Nil(t, found, "No entity should be found with a non-conflicting UniqueID if batch failed (Pebble)")
	})

	t.Run("Bulk creation with UniqueID conflicting with existing data pebble", func(t *testing.T) {
		repoClean, err := newNestedEntityTestPebbleRepositoryPebble(t) // Fresh repo
		require.NoError(t, err)

		// Pre-existing entity
		existingEntity := NestedEntityTest{ID: "---", Data: "ExistingDataP", Meta: NestedMetaTest{UniqueID: "conflictWithExistingP", OTValue: "ttlPE", Description: "DescPE"}}
		now := time.Now()
		_, err = repoClean.Create(&existingEntity, now)
		require.NoError(t, err, "Setup: Failed to create initial entity (Pebble)")

		entitiesToBulkCreate := []*NestedEntityTest{
			{ID: "---", Data: "NewEntityP1", Meta: NestedMetaTest{UniqueID: "newUniqueP1", OTValue: "ttlPN1", Description: "DescPN1"}},
			{ID: "---", Data: "NewEntityP2Conflicting", Meta: NestedMetaTest{UniqueID: "conflictWithExistingP", OTValue: "ttlPN2", Description: "DescPN2"}}, // Conflicts
			{ID: "---", Data: "NewEntityP3", Meta: NestedMetaTest{UniqueID: "newUniqueP3", OTValue: "ttlPN3", Description: "DescPN3"}},
		}

		ids, err := repoClean.BulkCreate(entitiesToBulkCreate, now)
		require.Error(t, err, "BulkCreate should fail due to conflict with existing UniqueID (Pebble)")
		assert.Nil(t, ids, "IDs should be nil on batch creation failure due to existing conflict (Pebble)")
		require.Contains(t, err.Error(), "duplicate", "Error message should indicate a duplicate key problem (Pebble)")

		foundNew, err := repoClean.FindByField("Meta.UniqueID", "newUniqueP1", now)
		require.NoError(t, err)
		assert.Nil(t, foundNew, "Non-conflicting entity from failed batch should not be inserted (Pebble)")

		foundExisting, err := repoClean.FindByField("Meta.UniqueID", "conflictWithExistingP", now)
		require.NoError(t, err)
		require.NotNil(t, foundExisting, "Original entity with the conflicting key should still exist (Pebble)")
		assert.Equal(t, existingEntity.Data, foundExisting.Data)
	})
}

func TestRepository_BulkUpdate_Nested_Pebble(t *testing.T) {
	t.Run("Successful bulk update of nested structs pebble", func(t *testing.T) {
		repo, err := newNestedEntityTestPebbleRepositoryPebble(t) // Uses specific IDs: pnid1, pnid2, ...
		require.NoError(t, err)
		now := time.Now()

		initialEntities := []*NestedEntityTest{
			{ID: "---", Data: "DataOneP", Meta: NestedMetaTest{UniqueID: "uniquePU1", OTValue: "ttlPU1", Description: "DescPU1"}},
			{ID: "---", Data: "DataTwoP", Meta: NestedMetaTest{UniqueID: "uniquePU2", OTValue: "ttlPU2", Description: "DescPU2"}},
		}
		createdIds, err := repo.BulkCreate(initialEntities, now)
		require.NoError(t, err)
		require.Len(t, createdIds, 2)

		// Prepare updates
		updatedEntities := []*NestedEntityTest{
			{ID: createdIds[0], Data: "DataOneUpdatedP", Meta: NestedMetaTest{UniqueID: "uniquePU1_new", OTValue: "ttlPU1_new", Description: "DescPU1_new"}},
			{ID: createdIds[1], Data: "DataTwoUpdatedP", Meta: NestedMetaTest{UniqueID: "uniquePU2_new", OTValue: "ttlPU2_new", Description: "DescPU2_new"}},
		}

		results, err := repo.BulkUpdate(updatedEntities, now)
		require.NoError(t, err, "BulkUpdate failed for valid nested entity updates (Pebble)")
		require.Len(t, results, len(updatedEntities), "BulkUpdate should return a result for each entity (Pebble)")
		for i, success := range results {
			assert.True(t, success, "Expected update for entity ID %s to succeed (Pebble)", updatedEntities[i].ID)
		}

		// Verify updates
		for i, updatedEntity := range updatedEntities {
			found, err := repo.FindByField("ID", updatedEntity.ID, now)
			require.NoError(t, err)
			require.NotNil(t, found)
			assert.Equal(t, updatedEntity.Data, found.Data)
			assert.Equal(t, updatedEntity.Meta.UniqueID, found.Meta.UniqueID)
			assert.Equal(t, updatedEntity.Meta.OTValue, found.Meta.OTValue)
			assert.Equal(t, updatedEntity.Meta.Description, found.Meta.Description)

			oldUniqueValue := initialEntities[i].Meta.UniqueID
			foundByOldUnique, err := repo.FindByField("Meta.UniqueID", oldUniqueValue, now)
			require.NoError(t, err)
			assert.Nil(t, foundByOldUnique, "Entity should not be found by old Meta.UniqueID %s (Pebble)", oldUniqueValue)

			foundByNewUnique, err := repo.FindByField("Meta.UniqueID", updatedEntity.Meta.UniqueID, now)
			require.NoError(t, err)
			require.NotNil(t, foundByNewUnique, "Entity should be found by new Meta.UniqueID %s (Pebble)", updatedEntity.Meta.UniqueID)
			assert.Equal(t, updatedEntity.ID, foundByNewUnique.ID)
		}
	})

	t.Run("Bulk update with UniqueID conflict within the batch pebble", func(t *testing.T) {
		repo, err := newNestedEntityTestPebbleRepositoryPebble(t)
		require.NoError(t, err)

		initialEntities := []*NestedEntityTest{
			{ID: "---", Data: "AlphaP", Meta: NestedMetaTest{UniqueID: "alphaUniqueP", OTValue: "ttlAP", Description: "DescAP"}},
			{ID: "---", Data: "BetaP", Meta: NestedMetaTest{UniqueID: "betaUniqueP", OTValue: "ttlBP", Description: "DescBP"}},
		}
		now := time.Now()
		createdIds, err := repo.BulkCreate(initialEntities, now)
		require.NoError(t, err)
		require.Len(t, createdIds, 2)
		initialEntities[0].ID = createdIds[0]
		initialEntities[1].ID = createdIds[1]

		conflictingUpdates := []*NestedEntityTest{
			{ID: createdIds[0], Data: "AlphaUpdatedP", Meta: NestedMetaTest{UniqueID: "conflictKeyP", OTValue: "ttlAP_new", Description: "DescAP_new"}},
			{ID: createdIds[1], Data: "BetaUpdatedP", Meta: NestedMetaTest{UniqueID: "conflictKeyP", OTValue: "ttlBP_new", Description: "DescBP_new"}},
		}

		_, err = repo.BulkUpdate(conflictingUpdates, now)
		require.Error(t, err, "BulkUpdate should fail due to UniqueID conflict within the batch (Pebble)")
		require.Contains(t, err.Error(), "duplicate", "Error message should indicate a duplicate key problem (Pebble)")

		for _, originalEntity := range initialEntities {
			found, err := repo.FindByField("ID", originalEntity.ID, now)
			require.NoError(t, err)
			require.NotNil(t, found)
			assert.Equal(t, originalEntity.Data, found.Data)
			assert.Equal(t, originalEntity.Meta.UniqueID, found.Meta.UniqueID)
		}
	})

	t.Run("Bulk update with UniqueID conflicting with another existing (untouched) entity pebble", func(t *testing.T) {
		repo, err := newNestedEntityTestPebbleRepositoryPebble(t)
		require.NoError(t, err)

		entities := []*NestedEntityTest{
			{ID: "---", Data: "EntityToUpdateP", Meta: NestedMetaTest{UniqueID: "originalUniqueP1", OTValue: "ttlP1", Description: "DescP1"}},
			{ID: "---", Data: "EntityToConflictWithP", Meta: NestedMetaTest{UniqueID: "existingUniqueP2", OTValue: "ttlP2", Description: "DescP2"}},
		}
		now := time.Now()
		createdIds, err := repo.BulkCreate(entities, now)
		require.NoError(t, err)
		require.Len(t, createdIds, 2)
		entities[0].ID = createdIds[0]
		entities[1].ID = createdIds[1]

		updateAttempt := []*NestedEntityTest{
			{ID: createdIds[0], Data: "EntityToUpdateModifiedP", Meta: NestedMetaTest{UniqueID: "existingUniqueP2", OTValue: "ttlP1_mod", Description: "DescP1_mod"}},
		}

		_, err = repo.BulkUpdate(updateAttempt, now)
		require.Error(t, err, "BulkUpdate should fail due to conflict with another existing entity's UniqueID (Pebble)")
		require.Contains(t, err.Error(), "duplicate", "Error message should indicate a duplicate key problem (Pebble)")

		foundOriginal, err := repo.FindByField("ID", createdIds[0], now)
		require.NoError(t, err)
		require.NotNil(t, foundOriginal)
		assert.Equal(t, "EntityToUpdateP", foundOriginal.Data)
		assert.Equal(t, "originalUniqueP1", foundOriginal.Meta.UniqueID)

		foundUntouched, err := repo.FindByField("ID", createdIds[1], now)
		require.NoError(t, err)
		require.NotNil(t, foundUntouched)
		assert.Equal(t, "EntityToConflictWithP", foundUntouched.Data)
		assert.Equal(t, "existingUniqueP2", foundUntouched.Meta.UniqueID)
	})

	t.Run("Bulk update including non-existent entities pebble", func(t *testing.T) {
		repo, err := newNestedEntityTestPebbleRepositoryPebble(t)
		require.NoError(t, err)
		now := time.Now()

		existingEntity := NestedEntityTest{ID: "---", Data: "RealDataP", Meta: NestedMetaTest{UniqueID: "realUniqueP", OTValue: "ttlRealP", Description: "DescRealP"}}
		createdIds, err := repo.BulkCreate([]*NestedEntityTest{&existingEntity}, now)
		require.NoError(t, err)
		require.Len(t, createdIds, 1)
		existingEntity.ID = createdIds[0]

		updates := []*NestedEntityTest{
			{ID: existingEntity.ID, Data: "RealDataUpdatedP", Meta: NestedMetaTest{UniqueID: "realUniqueUpdatedP", OTValue: "ttlRealUpdatedP", Description: "DescRealUpdatedP"}},
			{ID: "nonExistentIDP1", Data: "PhantomDataP1", Meta: NestedMetaTest{UniqueID: "phantomUniqueP1", OTValue: "ttlPhantomP1", Description: "DescPhantomP1"}},
			{ID: "nonExistentIDP2", Data: "PhantomDataP2", Meta: NestedMetaTest{UniqueID: "phantomUniqueP2", OTValue: "ttlPhantomP2", Description: "DescPhantomP2"}},
		}

		results, err := repo.BulkUpdate(updates, now)
		require.NoError(t, err, "BulkUpdate with non-existent IDs should not error out (Pebble)")
		require.Len(t, results, len(updates))
		assert.True(t, results[0], "Update for existing entity should succeed (Pebble)")
		assert.False(t, results[1], "Update for non-existent entity nonExistentIDP1 should be marked as false (Pebble)")
		assert.False(t, results[2], "Update for non-existent entity nonExistentIDP2 should be marked as false (Pebble)")

		found, err := repo.FindByField("ID", existingEntity.ID, now)
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, "RealDataUpdatedP", found.Data)
		assert.Equal(t, "realUniqueUpdatedP", found.Meta.UniqueID)

		foundPhantom1, err := repo.FindByField("ID", "nonExistentIDP1", now)
		require.NoError(t, err)
		assert.Nil(t, foundPhantom1)
	})
}
func TestRepository_PutAndGet_Deterministic_Id_Generator_Pebble(t *testing.T) {
	repo, err := newTestDeterministicRepositoryPebble(t)
	require.NoError(t, err)
	entity := testEntity{ID: "det-123", Name: "Alice"}

	id, err := repo.Create(&entity, time.Now())
	require.NoError(t, err)
	assert.Equal(t, id, "det-123")

	found, err := repo.FindByField("ID", "det-123", time.Now())
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, "det-123", found.ID)
	assert.Equal(t, entity.Name, found.Name)

	found, err = repo.FindByField("Name", "Alice", time.Now())
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, "det-123", found.ID)
	assert.Equal(t, entity.Name, found.Name)
}

// Helper functions for compound uniqueness tests
func newTestExchangeRepositoryPebble(t *testing.T) (*db.Repository[Exchange], db.KVStore, error) {
	store := newTestPebbleStore(t, []string{DefaultFC, TestFC}, []string{TemporalFC})
	iGF := &db.DefaultIDGeneratorFactory{}
	repository, err := db.NewRepository[Exchange](store, TestFC, testColumnFamilySector, "test_schema", iGF)
	return repository, store, err
}

func TestRepository_UniqueCompound_Pebble_CreateAndFind(t *testing.T) {
	repo, _, err := newTestExchangeRepositoryPebble(t)
	require.NoError(t, err)

	now := time.Now()

	// Test 1: Create two exchanges with same name but different namespaces - should succeed
	exchange1 := &Exchange{
		Name:       "test-exchange",
		Type:       "direct",
		VNamespace: "namespace-1",
		CreatedAt:  now.Format(time.RFC3339),
		UpdatedAt:  now.Format(time.RFC3339),
	}

	exchange2 := &Exchange{
		Name:       "test-exchange", // Same name
		Type:       "topic",
		VNamespace: "namespace-2", // Different namespace
		CreatedAt:  now.Format(time.RFC3339),
		UpdatedAt:  now.Format(time.RFC3339),
	}

	id1, err := repo.Create(exchange1, now)
	require.NoError(t, err)
	assert.NotEmpty(t, id1)

	id2, err := repo.Create(exchange2, now)
	require.NoError(t, err)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2)

	// Test 2: Try to create an exchange with same name and namespace - should fail
	exchange3 := &Exchange{
		Name:       "test-exchange", // Same name
		Type:       "fanout",
		VNamespace: "namespace-1", // Same namespace as exchange1
		CreatedAt:  now.Format(time.RFC3339),
		UpdatedAt:  now.Format(time.RFC3339),
	}

	_, err = repo.Create(exchange3, now)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate unique compound constraint")

	// Test 3: Find by compound fields using Find method
	filter1 := "Name='test-exchange' & VNamespace='namespace-1'"

	result1, err := repo.Find(filter1, 10, "", now)
	require.NoError(t, err)
	require.NotNil(t, result1)
	require.Len(t, result1.Entities, 1)
	found1 := result1.Entities[0]
	assert.Equal(t, id1, found1.ID)
	assert.Equal(t, "test-exchange", found1.Name)
	assert.Equal(t, "namespace-1", found1.VNamespace)
	assert.Equal(t, "direct", found1.Type)

	filter2 := "Name='test-exchange' & VNamespace='namespace-2'"

	result2, err := repo.Find(filter2, 10, "", now)
	require.NoError(t, err)
	require.NotNil(t, result2)
	require.Len(t, result2.Entities, 1)
	found2 := result2.Entities[0]
	assert.Equal(t, id2, found2.ID)
	assert.Equal(t, "test-exchange", found2.Name)
	assert.Equal(t, "namespace-2", found2.VNamespace)
	assert.Equal(t, "topic", found2.Type)

	// Test 4: Find non-existent compound
	filterNotFound := "Name='non-existent' & VNamespace='namespace-1'"

	resultNotFound, err := repo.Find(filterNotFound, 10, "", now)
	require.NoError(t, err)
	require.NotNil(t, resultNotFound)
	require.Len(t, resultNotFound.Entities, 0)
}

func TestRepository_UniqueCompound_Pebble_BulkCreate(t *testing.T) {
	repo, _, err := newTestExchangeRepositoryPebble(t)
	require.NoError(t, err)

	now := time.Now()

	// Test 1: Bulk create with valid compound uniqueness
	exchanges := []*Exchange{
		{
			Name:       "exchange-1",
			Type:       "direct",
			VNamespace: "namespace-1",
			CreatedAt:  now.Format(time.RFC3339),
			UpdatedAt:  now.Format(time.RFC3339),
		},
		{
			Name:       "exchange-1", // Same name, different namespace
			Type:       "topic",
			VNamespace: "namespace-2",
			CreatedAt:  now.Format(time.RFC3339),
			UpdatedAt:  now.Format(time.RFC3339),
		},
		{
			Name:       "exchange-2", // Different name, same namespace as first
			Type:       "fanout",
			VNamespace: "namespace-1",
			CreatedAt:  now.Format(time.RFC3339),
			UpdatedAt:  now.Format(time.RFC3339),
		},
	}

	ids, err := repo.BulkCreate(exchanges, now)
	require.NoError(t, err)
	assert.Len(t, ids, 3)

	// Verify all exchanges can be found
	for i, id := range ids {
		filter := fmt.Sprintf("Name='%s' & VNamespace='%s'", exchanges[i].Name, exchanges[i].VNamespace)

		result, err := repo.Find(filter, 10, "", now)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Len(t, result.Entities, 1)
		found := result.Entities[0]
		assert.Equal(t, id, found.ID)
		assert.Equal(t, exchanges[i].Name, found.Name)
		assert.Equal(t, exchanges[i].VNamespace, found.VNamespace)
		assert.Equal(t, exchanges[i].Type, found.Type)
	}

	// Test 2: Bulk create with duplicate compound constraint in batch - should fail
	duplicateExchanges := []*Exchange{
		{
			Name:       "new-exchange",
			Type:       "direct",
			VNamespace: "new-namespace",
		},
		{
			Name:       "new-exchange", // Same name and namespace - should fail
			Type:       "topic",
			VNamespace: "new-namespace",
		},
	}

	_, err = repo.BulkCreate(duplicateExchanges, now)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate unique compound constraint in input batch")
}

func TestRepository_UniqueCompound_Pebble_ValidationErrors(t *testing.T) {
	repo, _, err := newTestExchangeRepositoryPebble(t)
	require.NoError(t, err)

	now := time.Now()

	// Test that non-compound fields fall back to normal query behavior
	t.Run("Non-compound field query", func(t *testing.T) {
		// Create an exchange to test with
		exchange := &Exchange{
			Name:       "test-exchange",
			Type:       "direct",
			VNamespace: "namespace-1",
		}

		id, err := repo.Create(exchange, now)
		require.NoError(t, err)

		// Query by a single field (should work normally)
		result, err := repo.Find("Name='test-exchange'", 10, "", now)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Len(t, result.Entities, 1)
		assert.Equal(t, id, result.Entities[0].ID)
	})
}

func TestRepository_UniqueCompound_Pebble_Delete_DatabaseCleanup(t *testing.T) {
	repo, store, err := newTestExchangeRepositoryPebble(t)
	require.NoError(t, err)

	now := time.Now()

	// Test: Create exchanges with compound unique constraints, then delete them
	// and verify the database is completely clean (no orphaned compound indexes)

	// Create multiple exchanges with compound uniqueness
	exchanges := []*Exchange{
		{
			Name:       "exchange-1",
			Type:       "direct",
			VNamespace: "namespace-1",
			CreatedAt:  "2023-01-01T00:00:00Z",
			UpdatedAt:  "2023-01-01T00:00:00Z",
		},
		{
			Name:       "exchange-1", // Same name, different namespace
			Type:       "topic",
			VNamespace: "namespace-2",
			CreatedAt:  "2023-01-01T00:00:00Z",
			UpdatedAt:  "2023-01-01T00:00:00Z",
		},
		{
			Name:       "exchange-2",
			Type:       "fanout",
			VNamespace: "namespace-1",
			CreatedAt:  "2023-01-01T00:00:00Z",
			UpdatedAt:  "2023-01-01T00:00:00Z",
		},
	}

	var ids []string
	for _, exchange := range exchanges {
		id, err := repo.Create(exchange, now)
		require.NoError(t, err)
		ids = append(ids, id)
	}

	// Verify all exchanges can be found by compound fields
	for i, id := range ids {
		filter := fmt.Sprintf("Name='%s' & VNamespace='%s'", exchanges[i].Name, exchanges[i].VNamespace)
		result, err := repo.Find(filter, 10, "", now)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Len(t, result.Entities, 1)
		assert.Equal(t, id, result.Entities[0].ID)
	}

	// Verify compound index keys exist before deletion
	schema := "test_schema"
	table := "exchanges"

	for i, id := range ids {
		// Check that compound index keys exist
		compoundIdxKey := fmt.Sprintf("%s:%s:idx-uc:0:Name:%s|VNamespace:%s", schema, table, exchanges[i].Name, exchanges[i].VNamespace)

		// Verify the compound index key exists
		compoundBytes, err := store.Get(TestFC, testColumnFamilySector, compoundIdxKey, now)
		require.NoError(t, err)
		require.NotNil(t, compoundBytes)
		assert.Equal(t, id, string(compoundBytes))

		// Check that data key exists
		dataKey := fmt.Sprintf("%s:%s:data:%s", schema, table, id)
		dataBytes, err := store.Get(TestFC, testColumnFamilySector, dataKey, now)
		require.NoError(t, err)
		require.NotNil(t, dataBytes)

		// Check that regular index keys exist (for Name field)
		nameIdxKey := fmt.Sprintf("%s:%s:idx:Name:%s:%s", schema, table, exchanges[i].Name, id)
		nameBytes, err := store.Get(TestFC, testColumnFamilySector, nameIdxKey, now)
		require.NoError(t, err)
		require.NotNil(t, nameBytes)
		assert.Equal(t, id, string(nameBytes))

		// Check that VNamespace index keys exist
		vnameIdxKey := fmt.Sprintf("%s:%s:idx:VNamespace:%s:%s", schema, table, exchanges[i].VNamespace, id)
		vnameBytes, err := store.Get(TestFC, testColumnFamilySector, vnameIdxKey, now)
		require.NoError(t, err)
		require.NotNil(t, vnameBytes)
		assert.Equal(t, id, string(vnameBytes))

		// Check that Type index keys exist
		typeIdxKey := fmt.Sprintf("%s:%s:idx:Type:%s:%s", schema, table, exchanges[i].Type, id)
		typeBytes, err := store.Get(TestFC, testColumnFamilySector, typeIdxKey, now)
		require.NoError(t, err)
		require.NotNil(t, typeBytes)
		assert.Equal(t, id, string(typeBytes))
	}

	// Now delete all exchanges
	for _, id := range ids {
		deleted, err := repo.Delete(id, now)
		require.NoError(t, err)
		require.True(t, deleted)
	}

	// Verify all exchanges are gone
	for i := range exchanges {
		filter := fmt.Sprintf("Name='%s' & VNamespace='%s'", exchanges[i].Name, exchanges[i].VNamespace)
		result, err := repo.Find(filter, 10, "", now)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Len(t, result.Entities, 0)
	}

	// Verify all keys related to our test data are completely removed
	for i, id := range ids {
		// Check that compound index keys are gone
		compoundIdxKey := fmt.Sprintf("%s:%s:idx-uc:0:Name:%s|VNamespace:%s", schema, table, exchanges[i].Name, exchanges[i].VNamespace)
		compoundBytes, err := store.Get(TestFC, testColumnFamilySector, compoundIdxKey, now)
		require.NoError(t, err)
		assert.Nil(t, compoundBytes, "Compound index key should be deleted: %s", compoundIdxKey)

		// Check that data keys are gone
		dataKey := fmt.Sprintf("%s:%s:data:%s", schema, table, id)
		dataBytes, err := store.Get(TestFC, testColumnFamilySector, dataKey, now)
		require.NoError(t, err)
		assert.Nil(t, dataBytes, "Data key should be deleted: %s", dataKey)

		// Check that regular index keys are gone
		nameIdxKey := fmt.Sprintf("%s:%s:idx:Name:%s:%s", schema, table, exchanges[i].Name, id)
		nameBytes, err := store.Get(TestFC, testColumnFamilySector, nameIdxKey, now)
		require.NoError(t, err)
		assert.Nil(t, nameBytes, "Name index key should be deleted: %s", nameIdxKey)

		// Check that VNamespace index keys are gone
		vnameIdxKey := fmt.Sprintf("%s:%s:idx:VNamespace:%s:%s", schema, table, exchanges[i].VNamespace, id)
		vnameBytes, err := store.Get(TestFC, testColumnFamilySector, vnameIdxKey, now)
		require.NoError(t, err)
		assert.Nil(t, vnameBytes, "VNamespace index key should be deleted: %s", vnameIdxKey)

		// Check that Type index keys are gone
		typeIdxKey := fmt.Sprintf("%s:%s:idx:Type:%s:%s", schema, table, exchanges[i].Type, id)
		typeBytes, err := store.Get(TestFC, testColumnFamilySector, typeIdxKey, now)
		require.NoError(t, err)
		assert.Nil(t, typeBytes, "Type index key should be deleted: %s", typeIdxKey)
	}

	// Get final dump and verify database is clean for our test data
	finalDump, err := store.DumpAll()
	require.NoError(t, err)

	// The dump should not contain any references to our test data
	if finalDump != nil {
		dumpStr := fmt.Sprintf("%v", finalDump)
		assert.NotContains(t, dumpStr, "exchange-1", "Dump should not contain exchange-1 references")
		assert.NotContains(t, dumpStr, "exchange-2", "Dump should not contain exchange-2 references")
		assert.NotContains(t, dumpStr, "namespace-1", "Dump should not contain namespace-1 references")
		assert.NotContains(t, dumpStr, "namespace-2", "Dump should not contain namespace-2 references")
	}

	t.Logf("Database cleanup verification completed successfully")
}

func TestRepository_UniqueCompound_Pebble_Update_NoStaleIndexes(t *testing.T) {
	repo, store, err := newTestExchangeRepositoryPebble(t)
	require.NoError(t, err)

	now := time.Now()
	schema := "test_schema"
	table := "exchanges"

	// Test: Create an exchange with compound unique constraint, then update it
	// and verify no stale compound indexes remain in the database

	exchange := &Exchange{
		Name:       "test-exchange",
		Type:       "direct",
		VNamespace: "namespace-1",
		CreatedAt:  "2023-01-01T00:00:00Z",
		UpdatedAt:  "2023-01-01T00:00:00Z",
	}

	// Create the exchange
	id, err := repo.Create(exchange, now)
	require.NoError(t, err)
	require.NotEmpty(t, id)

	// Verify initial compound index exists
	initialCompoundIdxKey := fmt.Sprintf("%s:%s:idx-uc:0:Name:%s|VNamespace:%s", schema, table, exchange.Name, exchange.VNamespace)
	compoundBytes, err := store.Get(TestFC, testColumnFamilySector, initialCompoundIdxKey, now)
	require.NoError(t, err)
	require.NotNil(t, compoundBytes)
	assert.Equal(t, id, string(compoundBytes))

	// Verify we can find it using compound constraint
	filter := fmt.Sprintf("Name='%s' & VNamespace='%s'", exchange.Name, exchange.VNamespace)
	result, err := repo.Find(filter, 10, "", now)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Entities, 1)
	assert.Equal(t, id, result.Entities[0].ID)

	// Update the exchange - change the VNamespace (part of compound constraint)
	updatedExchange := &Exchange{
		ID:         id,
		Name:       "test-exchange", // Same name
		Type:       "topic",         // Different type
		VNamespace: "namespace-2",   // Different namespace - this changes compound constraint
		CreatedAt:  "2023-01-01T00:00:00Z",
		UpdatedAt:  "2023-01-02T00:00:00Z",
	}

	updated, err := repo.Update(updatedExchange, now)
	require.NoError(t, err)
	require.True(t, updated)

	// Verify old compound index is gone
	oldCompoundBytes, err := store.Get(TestFC, testColumnFamilySector, initialCompoundIdxKey, now)
	require.NoError(t, err)
	assert.Nil(t, oldCompoundBytes, "Old compound index should be deleted: %s", initialCompoundIdxKey)

	// Verify new compound index exists
	newCompoundIdxKey := fmt.Sprintf("%s:%s:idx-uc:0:Name:%s|VNamespace:%s", schema, table, updatedExchange.Name, updatedExchange.VNamespace)
	newCompoundBytes, err := store.Get(TestFC, testColumnFamilySector, newCompoundIdxKey, now)
	require.NoError(t, err)
	require.NotNil(t, newCompoundBytes)
	assert.Equal(t, id, string(newCompoundBytes))

	// Verify we can find it using the new compound constraint
	newFilter := fmt.Sprintf("Name='%s' & VNamespace='%s'", updatedExchange.Name, updatedExchange.VNamespace)
	newResult, err := repo.Find(newFilter, 10, "", now)
	require.NoError(t, err)
	require.NotNil(t, newResult)
	require.Len(t, newResult.Entities, 1)
	assert.Equal(t, id, newResult.Entities[0].ID)
	assert.Equal(t, "namespace-2", newResult.Entities[0].VNamespace)
	assert.Equal(t, "topic", string(newResult.Entities[0].Type))

	// Verify we cannot find it using the old compound constraint
	oldResult, err := repo.Find(filter, 10, "", now)
	require.NoError(t, err)
	require.NotNil(t, oldResult)
	require.Len(t, oldResult.Entities, 0)

	// Verify database dump doesn't contain the old compound index
	dump, err := store.DumpAll()
	require.NoError(t, err)
	if dump != nil {
		dumpStr := fmt.Sprintf("%v", dump)
		assert.NotContains(t, dumpStr, "Name:test-exchange|VNamespace:namespace-1", "Dump should not contain old compound index")
		assert.Contains(t, dumpStr, "Name:test-exchange|VNamespace:namespace-2", "Dump should contain new compound index")
	}

	t.Logf("Update compound index cleanup verification completed successfully")
}

func TestRepository_UniqueCompound_Pebble_Update_BothFieldsChange(t *testing.T) {
	repo, store, err := newTestExchangeRepositoryPebble(t)
	require.NoError(t, err)

	now := time.Now()
	schema := "test_schema"
	table := "exchanges"

	// Test: Update both fields of a compound constraint and verify proper cleanup

	exchange := &Exchange{
		Name:       "original-exchange",
		Type:       "direct",
		VNamespace: "original-namespace",
		CreatedAt:  "2023-01-01T00:00:00Z",
		UpdatedAt:  "2023-01-01T00:00:00Z",
	}

	// Create the exchange
	id, err := repo.Create(exchange, now)
	require.NoError(t, err)

	// Verify initial compound index
	initialCompoundIdxKey := fmt.Sprintf("%s:%s:idx-uc:0:Name:%s|VNamespace:%s", schema, table, exchange.Name, exchange.VNamespace)
	compoundBytes, err := store.Get(TestFC, testColumnFamilySector, initialCompoundIdxKey, now)
	require.NoError(t, err)
	require.NotNil(t, compoundBytes)

	// Update BOTH fields that are part of the compound constraint
	updatedExchange := &Exchange{
		ID:         id,
		Name:       "updated-exchange", // Changed
		Type:       "fanout",
		VNamespace: "updated-namespace", // Changed
		CreatedAt:  "2023-01-01T00:00:00Z",
		UpdatedAt:  "2023-01-02T00:00:00Z",
	}

	updated, err := repo.Update(updatedExchange, now)
	require.NoError(t, err)
	require.True(t, updated)

	// Verify old compound index is completely gone
	oldCompoundBytes, err := store.Get(TestFC, testColumnFamilySector, initialCompoundIdxKey, now)
	require.NoError(t, err)
	assert.Nil(t, oldCompoundBytes, "Old compound index should be deleted")

	// Verify new compound index exists
	newCompoundIdxKey := fmt.Sprintf("%s:%s:idx-uc:0:Name:%s|VNamespace:%s", schema, table, updatedExchange.Name, updatedExchange.VNamespace)
	newCompoundBytes, err := store.Get(TestFC, testColumnFamilySector, newCompoundIdxKey, now)
	require.NoError(t, err)
	require.NotNil(t, newCompoundBytes)
	assert.Equal(t, id, string(newCompoundBytes))

	// Verify search behavior
	oldFilter := fmt.Sprintf("Name='%s' & VNamespace='%s'", exchange.Name, exchange.VNamespace)
	oldResult, err := repo.Find(oldFilter, 10, "", now)
	require.NoError(t, err)
	require.Len(t, oldResult.Entities, 0)

	newFilter := fmt.Sprintf("Name='%s' & VNamespace='%s'", updatedExchange.Name, updatedExchange.VNamespace)
	newResult, err := repo.Find(newFilter, 10, "", now)
	require.NoError(t, err)
	require.Len(t, newResult.Entities, 1)
	assert.Equal(t, id, newResult.Entities[0].ID)

	t.Logf("Both fields compound update verification completed successfully")
}

func TestRepository_UniqueCompound_Pebble_Update_ConstraintViolation(t *testing.T) {
	repo, _, err := newTestExchangeRepositoryPebble(t)
	require.NoError(t, err)

	now := time.Now()

	// Test: Verify that updating to values that would violate compound constraint fails

	// Create first exchange
	exchange1 := &Exchange{
		Name:       "exchange-1",
		Type:       "direct",
		VNamespace: "namespace-1",
		CreatedAt:  "2023-01-01T00:00:00Z",
		UpdatedAt:  "2023-01-01T00:00:00Z",
	}
	id1, err := repo.Create(exchange1, now)
	require.NoError(t, err)

	// Create second exchange
	exchange2 := &Exchange{
		Name:       "exchange-2",
		Type:       "topic",
		VNamespace: "namespace-2",
		CreatedAt:  "2023-01-01T00:00:00Z",
		UpdatedAt:  "2023-01-01T00:00:00Z",
	}
	id2, err := repo.Create(exchange2, now)
	require.NoError(t, err)

	// Try to update exchange2 to have the same Name+VNamespace as exchange1 - should fail
	conflictingUpdate := &Exchange{
		ID:         id2,
		Name:       "exchange-1", // Same as exchange1
		Type:       "fanout",
		VNamespace: "namespace-1", // Same as exchange1
		CreatedAt:  "2023-01-01T00:00:00Z",
		UpdatedAt:  "2023-01-02T00:00:00Z",
	}

	updated, err := repo.Update(conflictingUpdate, now)
	require.Error(t, err)
	require.False(t, updated)
	assert.Contains(t, err.Error(), "duplicate compound unique constraint")

	// Verify original exchanges remain unchanged
	result1, err := repo.Find(fmt.Sprintf("Name='%s' & VNamespace='%s'", exchange1.Name, exchange1.VNamespace), 10, "", now)
	require.NoError(t, err)
	require.Len(t, result1.Entities, 1)
	assert.Equal(t, id1, result1.Entities[0].ID)

	result2, err := repo.Find(fmt.Sprintf("Name='%s' & VNamespace='%s'", exchange2.Name, exchange2.VNamespace), 10, "", now)
	require.NoError(t, err)
	require.Len(t, result2.Entities, 1)
	assert.Equal(t, id2, result2.Entities[0].ID)

	t.Logf("Compound constraint violation test completed successfully")
}

// MARK: Data-Only Field Tests for Pebble

func TestDataOnlyFields_PebbleValidations(t *testing.T) {
	store := newTestPebbleStore(t, []string{DefaultFC, TestFC}, nil)

	t.Run("Valid DataOnlyEntity", func(t *testing.T) {
		repo, err := db.NewRepository[DataOnlyEntity](store, TestFC, testColumnFamilySector, "test_data_only", &db.DefaultIDGeneratorFactory{})
		require.NoError(t, err)
		assert.NotNil(t, repo)
	})

	t.Run("InvalidDataOnlyUniqueEntity - data-only fields cannot be unique", func(t *testing.T) {
		_, err := db.NewRepository[InvalidDataOnlyUniqueEntity](store, TestFC, testColumnFamilySector, "test_invalid_unique", &db.DefaultIDGeneratorFactory{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "field 'Config' cannot be both data-only and unique")
	})

	t.Run("InvalidDataOnlyCompoundEntity - data-only fields cannot be in compound constraints", func(t *testing.T) {
		_, err := db.NewRepository[InvalidDataOnlyCompoundEntity](store, TestFC, testColumnFamilySector, "test_invalid_compound", &db.DefaultIDGeneratorFactory{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "field 'Config' cannot be both data-only and part of compound uniqueness")
	})
}

func TestDataOnlyFields_PebbleCreateAndRead(t *testing.T) {
	store := newTestPebbleStore(t, []string{DefaultFC, TestFC}, nil)

	repo, err := db.NewRepository[DataOnlyEntity](store, TestFC, testColumnFamilySector, "test_data_only", &db.DefaultIDGeneratorFactory{})
	require.NoError(t, err)

	now := time.Now()
	entity := &DataOnlyEntity{
		Name:        "test-entity",
		SearchField: "searchable-value",
		Config: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
		Metadata: map[int]int{
			1: 10,
			2: 20,
		},
		Tags:       []string{"tag1", "tag2", "tag3"},
		Statistics: map[string]interface{}{"count": 42, "rate": 3.14},
		CreatedAt:  now,
	}

	// Create entity
	id, err := repo.Create(entity, now)
	require.NoError(t, err)
	assert.NotEmpty(t, id)
	assert.Equal(t, id, entity.ID)

	// Read entity back using FindByField with ID
	readEntity, err := repo.FindByField("ID", id, now)
	require.NoError(t, err)
	assert.Equal(t, entity.ID, readEntity.ID)
	assert.Equal(t, entity.Name, readEntity.Name)
	assert.Equal(t, entity.SearchField, readEntity.SearchField)
	assert.Equal(t, entity.Config, readEntity.Config)
	assert.Equal(t, entity.Metadata, readEntity.Metadata)
	assert.Equal(t, entity.Tags, readEntity.Tags)

	// Verify we can search by normal fields
	result, err := repo.Find("Name='test-entity'", 10, "", now)
	require.NoError(t, err)
	assert.Len(t, result.Entities, 1)
	assert.Equal(t, entity.ID, result.Entities[0].ID)

	// Verify we can search by SearchField
	result2, err := repo.Find("SearchField='searchable-value'", 10, "", now)
	require.NoError(t, err)
	assert.Len(t, result2.Entities, 1)
	assert.Equal(t, entity.ID, result2.Entities[0].ID)
}

func TestVirtualFields_PebbleCreateAndRead(t *testing.T) {
	store := newTestPebbleStore(t, []string{DefaultFC, TestFC}, nil)

	repo, err := db.NewRepository[VirtualFieldEntity](store, TestFC, testColumnFamilySector, "test_virtual", &db.DefaultIDGeneratorFactory{})
	require.NoError(t, err)

	now := time.Now()
	entity := &VirtualFieldEntity{
		Name:        "test-entity",
		SearchField: "searchable-value",
		VirtualData: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
		TempCache:   []string{"cache1", "cache2"},
		RuntimeInfo: map[string]interface{}{"count": 42, "rate": 3.14},
		CreatedAt:   now,
	}

	// Create entity
	id, err := repo.Create(entity, now)
	require.NoError(t, err)
	assert.NotEmpty(t, id)
	assert.Equal(t, id, entity.ID)

	// Read entity back using FindByField with ID
	readEntity, err := repo.FindByField("ID", id, now)
	require.NoError(t, err)
	assert.Equal(t, entity.ID, readEntity.ID)
	assert.Equal(t, entity.Name, readEntity.Name)
	assert.Equal(t, entity.SearchField, readEntity.SearchField)
	assert.Equal(t, entity.CreatedAt.Unix(), readEntity.CreatedAt.Unix())

	// Virtual fields should NOT be persisted - they should be empty/nil when read back
	assert.Nil(t, readEntity.VirtualData, "VirtualData should not be persisted")
	assert.Nil(t, readEntity.TempCache, "TempCache should not be persisted")
	assert.Nil(t, readEntity.RuntimeInfo, "RuntimeInfo should not be persisted")

	// Verify we can search by normal fields
	result, err := repo.Find("Name='test-entity'", 10, "", now)
	require.NoError(t, err)
	assert.Len(t, result.Entities, 1)
	assert.Equal(t, entity.ID, result.Entities[0].ID)

	// Verify we can search by SearchField
	result2, err := repo.Find("SearchField='searchable-value'", 10, "", now)
	require.NoError(t, err)
	assert.Len(t, result2.Entities, 1)
	assert.Equal(t, entity.ID, result2.Entities[0].ID)

	// Verify we CANNOT search by virtual fields - should return error
	_, err = repo.Find("VirtualData='anything'", 10, "", now)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "virtual")

	_, err = repo.Find("TempCache='anything'", 10, "", now)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "virtual")

	_, err = repo.Find("RuntimeInfo='anything'", 10, "", now)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "virtual")
}

func TestVirtualFields_ValidationErrors_Pebble(t *testing.T) {
	store := newTestPebbleStore(t, []string{DefaultFC, TestFC}, nil)

	// Test virtual field cannot be unique
	_, err := db.NewRepository[InvalidVirtualUniqueEntity](store, TestFC, testColumnFamilySector, "test_virtual_unique", &db.DefaultIDGeneratorFactory{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be both virtual and unique")

	// Test virtual field cannot be data-only
	_, err = db.NewRepository[InvalidVirtualDataOnlyEntity](store, TestFC, testColumnFamilySector, "test_virtual_data_only", &db.DefaultIDGeneratorFactory{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be both virtual and data-only")
}

func TestDataOnlyFields_PebbleQueryRestrictions(t *testing.T) {
	store := newTestPebbleStore(t, []string{DefaultFC, TestFC}, nil)

	repo, err := db.NewRepository[DataOnlyEntity](store, TestFC, testColumnFamilySector, "test_data_only", &db.DefaultIDGeneratorFactory{})
	require.NoError(t, err)

	now := time.Now()

	t.Run("Query on data-only Config field should fail", func(t *testing.T) {
		_, err := repo.Find("Config='value1'", 10, "", now)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Field 'Config' is marked as data-only and cannot be used in queries")
	})

	t.Run("Query on data-only Metadata field should fail", func(t *testing.T) {
		_, err := repo.Find("Metadata=10", 10, "", now)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Field 'Metadata' is marked as data-only and cannot be used in queries")
	})

	t.Run("Query on data-only Tags field should fail", func(t *testing.T) {
		_, err := repo.Find("Tags='tag1'", 10, "", now)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Field 'Tags' is marked as data-only and cannot be used in queries")
	})

	t.Run("Query on data-only Statistics field should fail", func(t *testing.T) {
		_, err := repo.Find("Statistics=42", 10, "", now)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Field 'Statistics' is marked as data-only and cannot be used in queries")
	})
}

func TestDataOnlyFields_PebbleUpdate(t *testing.T) {
	store := newTestPebbleStore(t, []string{DefaultFC, TestFC}, nil)

	repo, err := db.NewRepository[DataOnlyEntity](store, TestFC, testColumnFamilySector, "test_data_only", &db.DefaultIDGeneratorFactory{})
	require.NoError(t, err)

	now := time.Now()

	// Create initial entity
	originalEntity := &DataOnlyEntity{
		Name:        "original-name",
		SearchField: "original-search",
		Config: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
		Metadata: map[int]int{
			1: 10,
			2: 20,
		},
		Tags:       []string{"tag1", "tag2"},
		Statistics: map[string]interface{}{"count": 42},
		CreatedAt:  now,
	}

	id, err := repo.Create(originalEntity, now)
	require.NoError(t, err)

	// Update entity with new data-only field values
	updatedEntity := &DataOnlyEntity{
		ID:          id,
		Name:        "updated-name",
		SearchField: "updated-search",
		Config: map[string]string{
			"key1": "updated-value1",
			"key3": "value3",
		},
		Metadata: map[int]int{
			1: 15,
			3: 30,
		},
		Tags:       []string{"tag1", "tag3", "tag4"},
		Statistics: map[string]interface{}{"count": 84, "rate": 2.71},
		CreatedAt:  now,
	}

	// Update entity
	updated, err := repo.Update(updatedEntity, now)
	require.NoError(t, err)
	assert.True(t, updated)

	// Verify the update was successful
	readEntity, err := repo.FindByField("ID", id, now)
	require.NoError(t, err)
	assert.Equal(t, updatedEntity.Name, readEntity.Name)
	assert.Equal(t, updatedEntity.SearchField, readEntity.SearchField)
	assert.Equal(t, updatedEntity.Config, readEntity.Config)
	assert.Equal(t, updatedEntity.Metadata, readEntity.Metadata)
	assert.Equal(t, updatedEntity.Tags, readEntity.Tags)

	// Verify we can still search by the updated searchable fields
	result, err := repo.Find("Name='updated-name'", 10, "", now)
	require.NoError(t, err)
	assert.Len(t, result.Entities, 1)
	assert.Equal(t, id, result.Entities[0].ID)
}
