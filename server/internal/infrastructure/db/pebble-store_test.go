package db_test

import (
	"fmt"
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

func TestPebbleStore_PutAndGet(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := db.CreatePebbleStore(tmpDir, []string{DefaultFC, TestFC}, nil)
	require.NoError(t, err)
	defer store.Close()

	key := "key"
	value := []byte("value")
	now := time.Now()

	err = store.Put(TestFC, testColumnFamilySector, key, value, 0, now)
	require.NoError(t, err)

	result, err := store.Get(TestFC, testColumnFamilySector, key, now)
	require.NoError(t, err)
	assert.Equal(t, value, result)
}

func TestPebbleStore_Get_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := db.CreatePebbleStore(tmpDir, []string{DefaultFC, TestFC}, nil)
	require.NoError(t, err)
	defer store.Close()
	now := time.Now()

	result, err := store.Get(TestFC, testColumnFamilySector, "nonexistent", now)
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestPebbleStore_WriteBatch(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := db.CreatePebbleStore(tmpDir, []string{DefaultFC, TestFC}, nil)
	require.NoError(t, err)
	defer store.Close()
	now := time.Now()

	batch := db.NewWriteBatch()
	batch.Put(TestFC, testColumnFamilySector, "a", []byte("valueA"), now)
	batch.Put(TestFC, testColumnFamilySector, "b", []byte("valueB"), now)

	err = store.Write(batch)
	require.NoError(t, err)

	resultA, err := store.Get(TestFC, testColumnFamilySector, "a", now)
	require.NoError(t, err)
	assert.Equal(t, []byte("valueA"), resultA)

	resultB, err := store.Get(TestFC, testColumnFamilySector, "b", now)
	require.NoError(t, err)
	assert.Equal(t, []byte("valueB"), resultB)
}

func TestPebbleStore_SearchByPatternPaginatedKV_MatchSingle(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := db.CreatePebbleStore(tmpDir, []string{DefaultFC, TestFC}, nil)
	require.NoError(t, err)
	defer store.Close()
	now := time.Now()

	require.NoError(t, store.Put(TestFC, testColumnFamilySector, "user:123:name", []byte("Alice"), 0, now))

	results, next, err := store.SearchByPatternPaginatedKV(TestFC, testColumnFamilySector, "user:123:*", "", 10, now)
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
	now := time.Now()

	require.NoError(t, store.Put(TestFC, testColumnFamilySector, "user:1", []byte("a"), 0, now))
	require.NoError(t, store.Put(TestFC, testColumnFamilySector, "user:2", []byte("b"), 0, now))
	require.NoError(t, store.Put(TestFC, testColumnFamilySector, "user:3", []byte("c"), 0, now))

	var all []db.KeyValuePair
	cursor := ""
	for {
		page, next, err := store.SearchByPatternPaginatedKV(TestFC, testColumnFamilySector, "user:*", cursor, 2, now)
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
	now := time.Now()

	require.NoError(t, store.Put(TestFC, testColumnFamilySector, "product:1", []byte("item"), 0, now))

	results, next, err := store.SearchByPatternPaginatedKV(TestFC, testColumnFamilySector, "user:*", "", 10, now)
	require.NoError(t, err)
	require.Empty(t, results)
	require.Equal(t, "", next)
}

func TestPebbleStore_SearchByPatternPaginatedKV_InvalidColumnFamily(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := db.CreatePebbleStore(tmpDir, []string{DefaultFC}, nil)
	require.NoError(t, err)
	defer store.Close()
	now := time.Now()

	_, _, err = store.SearchByPatternPaginatedKV("nonexistent", testColumnFamilySector, "pattern:*", "", 10, now)
	require.Error(t, err)
	require.Contains(t, err.Error(), "column family")
}

func TestPebbleStore_Delete_ExistingKey(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := db.CreatePebbleStore(tmpDir, []string{DefaultFC, TestFC}, nil)
	require.NoError(t, err)
	defer store.Close()
	now := time.Now()

	key := "delete-key"
	value := []byte("to-delete")

	require.NoError(t, store.Put(TestFC, testColumnFamilySector, key, value, 0, now))
	require.NoError(t, store.Delete(TestFC, testColumnFamilySector, key, now))

	result, err := store.Get(TestFC, testColumnFamilySector, key, now)
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestPebbleStore_Delete_NonExistentKey(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := db.CreatePebbleStore(tmpDir, []string{DefaultFC, TestFC}, nil)
	require.NoError(t, err)
	defer store.Close()
	now := time.Now()

	err = store.Delete(TestFC, testColumnFamilySector, "nonexistent", now)
	assert.NoError(t, err)
}

func TestPebbleStore_Delete_InvalidColumnFamily(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := db.CreatePebbleStore(tmpDir, []string{DefaultFC}, nil)
	require.NoError(t, err)
	defer store.Close()
	now := time.Now()

	err = store.Delete("nonexistent_cf", testColumnFamilySector, "key", now)
	require.Error(t, err)
	require.Contains(t, err.Error(), "column family")
}

func TestPebbleStore_Delete_TTLColumnFamily(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := db.CreatePebbleStore(tmpDir, []string{}, []string{TestFC})
	require.NoError(t, err)
	defer store.Close()
	now := time.Now()

	key := "ttl-key"
	value := []byte("ttl-value")

	require.NoError(t, store.Put(TestFC, testColumnFamilySector, key, value, 0, now))
	require.NoError(t, store.Delete(TestFC, testColumnFamilySector, key, now))

	result, err := store.Get(TestFC, testColumnFamilySector, key, now)
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

func TestPebbleStore_ColumnFamilyOperations(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := db.CreatePebbleStore(tmpDir, []string{DefaultFC}, nil)
	require.NoError(t, err)
	defer store.Close()

	cfName := "new_cf"
	cfNameTTL := "new_cf_ttl"

	// 1. ExistsColumnFamily - initially not found
	exists, isTTL, err := store.ExistsColumnFamily(cfName)
	require.NoError(t, err)
	assert.False(t, exists)
	assert.False(t, isTTL)

	// 2. CreateColumnFamily - non-TTL
	err = store.CreateColumnFamily(cfName, false)
	require.NoError(t, err)

	// 3. ExistsColumnFamily - non-TTL should exist
	exists, isTTL, err = store.ExistsColumnFamily(cfName)
	require.NoError(t, err)
	assert.True(t, exists)
	assert.False(t, isTTL)

	// 4. CreateColumnFamily - TTL
	err = store.CreateColumnFamily(cfNameTTL, true)
	require.NoError(t, err)

	// 5. ExistsColumnFamily - TTL should exist
	exists, isTTL, err = store.ExistsColumnFamily(cfNameTTL)
	require.NoError(t, err)
	assert.True(t, exists)
	assert.True(t, isTTL)

	// 6. CreateColumnFamily - already exists
	err = store.CreateColumnFamily(cfName, false)
	require.Error(t, err) // Should error

	// 7. Put/Get in new non-TTL CF
	key1, val1 := "key1", []byte("val1")
	err = store.Put(cfName, testColumnFamilySector, key1, val1, 0, time.Now())
	require.NoError(t, err)
	retVal1, err := store.Get(cfName, testColumnFamilySector, key1, time.Now())
	require.NoError(t, err)
	assert.Equal(t, val1, retVal1)

	// 8. Put/Get in new TTL CF
	key2, val2 := "key2", []byte("val2")
	err = store.Put(cfNameTTL, testColumnFamilySector, key2, val2, 3600, time.Now())
	require.NoError(t, err)
	retVal2, err := store.Get(cfNameTTL, testColumnFamilySector, key2, time.Now())
	require.NoError(t, err)
	assert.Equal(t, val2, retVal2)

	// 9. DeleteColumnFamily - non-TTL
	err = store.DeleteColumnFamily(cfName)
	require.NoError(t, err)

	// 10. ExistsColumnFamily - non-TTL should not exist
	exists, isTTL, err = store.ExistsColumnFamily(cfName)
	require.NoError(t, err)
	assert.False(t, exists)
	assert.False(t, isTTL)

	// 11. Get from deleted non-TTL CF should fail or return not found (implementation specific, check for error)
	_, err = store.Get(cfName, testColumnFamilySector, key1, time.Now())
	require.Error(t, err) // Expect an error as CF is gone

	// 12. DeleteColumnFamily - TTL
	err = store.DeleteColumnFamily(cfNameTTL)
	require.NoError(t, err)

	// 13. ExistsColumnFamily - TTL should not exist
	exists, isTTL, err = store.ExistsColumnFamily(cfNameTTL)
	require.NoError(t, err)
	assert.False(t, exists)
	assert.False(t, isTTL)

	// 14. DeleteColumnFamily - already deleted / not found
	err = store.DeleteColumnFamily(cfName)
	require.Error(t, err) // Should error
}

func TestPebbleStore_CleanExpiredKeys(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := db.CreatePebbleStore(tmpDir, []string{}, []string{TestFC})
	require.NoError(t, err)
	defer store.Close()

	key := "my-ttl-key"
	value := []byte("some data")
	now := time.Now()

	// Put a key with a 1-second TTL
	err = store.Put(TestFC, testColumnFamilySector, key, value, 1, now)
	require.NoError(t, err)

	// Wait for the key to expire
	time.Sleep(2 * time.Second)

	// Clean expired keys
	err = store.CleanExpiredKeys(time.Now())
	require.NoError(t, err)

	// Dump all data and verify that the keys are gone
	dump, err := store.DumpAll()
	require.NoError(t, err)

	// The dump should be empty or not contain the test column family
	assert.Empty(t, dump)
}
func TestPebbleStore_CleanExpiredKeysWithBatch(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := db.CreatePebbleStore(tmpDir, []string{"non_ttl_cf"}, []string{TestFC})
	require.NoError(t, err)
	defer store.Close()

	now := time.Now()
	expiredKey := "expired-key"
	validTTLKey := "valid-ttl-key"
	nonTTLKey := "non-ttl-key"

	batch := db.NewWriteBatch()
	batch.PutTTl(TestFC, testColumnFamilySector, expiredKey, []byte("will expire"), 1, now)     // 1-second TTL
	batch.PutTTl(TestFC, testColumnFamilySector, validTTLKey, []byte("should remain"), 30, now) // 30-second TTL
	batch.Put("non_ttl_cf", testColumnFamilySector, nonTTLKey, []byte("no ttl"), now)           // No TTL
	err = store.Write(batch)
	require.NoError(t, err)

	// Wait for the short TTL to expire
	time.Sleep(2 * time.Second)

	// Clean expired keys
	err = store.CleanExpiredKeys(time.Now())
	require.NoError(t, err)

	// Dump all data and verify
	dumpX, err := store.DumpAll()
	require.NoError(t, err)
	dump := dumpX.(map[string]map[string][]byte)
	// Check that the expired key is gone
	fullColumnFamily := TestFC + ":test-sector"
	_, cfExists := dump[fullColumnFamily]

	if cfExists {
		_, keyExists := dump[fullColumnFamily][expiredKey]
		assert.False(t, keyExists, "Expired key should have been removed")
	}

	// Check that the valid TTL key is still present
	assert.NotNil(t, dump[fullColumnFamily][validTTLKey], "Valid TTL key should be present")

	// Check that the non-TTL key is still present
	assert.NotNil(t, dump["non_ttl_cf:test-sector"][nonTTLKey], "Non-TTL key should be present")
}
