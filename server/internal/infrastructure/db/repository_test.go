package db_test

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNewRepositoryConditionalUniquenessValidations(t *testing.T) {
	mockStore := new(MockKVStore) // Does not hit DB for these validations
	iGF := NewTestIDGeneratorFactory([]string{})

	t.Run("Valid ConditionalUniqueEntity", func(t *testing.T) {
		_, err := db.NewRepository[ConditionalUniqueEntity](mockStore, "cf_valid", testColumnFamilySector, "test_valid", iGF)
		require.NoError(t, err)
	})

	t.Run("InvalidConditionalEntityBadRef - NonExistentFlag", func(t *testing.T) {
		_, err := db.NewRepository[InvalidConditionalEntityBadRef](mockStore, "cf_bad_ref", testColumnFamilySector, "test_bad_ref", iGF)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "field 'UniqueValue' tagged with 'ignore-is-true:NonExistentFlag', but referenced field 'NonExistentFlag' does not exist in struct InvalidConditionalEntityBadRef")
	})

	t.Run("InvalidConditionalEntityBadType - NonBoolFlag", func(t *testing.T) {
		_, err := db.NewRepository[InvalidConditionalEntityBadType](mockStore, "cf_bad_type", testColumnFamilySector, "test_bad_type", iGF)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "field 'UniqueValue' tagged with 'ignore-is-true:NonBoolFlag', but referenced field 'NonBoolFlag' must be of type 'bool', found 'int'")
	})

	// Test for createFieldDefinition error: ignore-is-true with empty field name

	t.Run("InvalidConditionalEmptyField - ignore-is-true with empty field", func(t *testing.T) {
		_, err := db.NewRepository[InvalidConditionalEmptyField](mockStore, "cf_empty", testColumnFamilySector, "test_empty", iGF)
		require.Error(t, err)
		// This error comes from createFieldDefinition
		assert.Contains(t, err.Error(), "error extracting fields from struct InvalidConditionalEmptyField: invalid ignore-is-true format for field 'UniqueValue': ignore-is-true:")
	})
}

func TestRepository_Create_Success(t *testing.T) {
	mockStore := new(MockKVStore)

	user := User{
		ID:   "123",
		Name: "Alice",
	}

	iGF := NewTestIDGeneratorFactory([]string{"123"})

	repo, err := db.NewRepository[User](mockStore, "cf1", testColumnFamilySector, "admin", iGF)
	assert.NoError(t, err)

	data, _ := json.Marshal(user)
	dataKey := "admin:users:data:123"
	nameFieldKey := "admin:users:idx:Name:Alice:123"
	uNameFieldKey := "admin:users:idx-u:Name:Alice"
	indexKey := "admin:users:idx:ID:123:123"

	mockStore.On("Exists", "cf1", testColumnFamilySector, uNameFieldKey, mock.Anything).Return(false, nil)
	mockStore.On("Exists", "cf1", testColumnFamilySector, "admin:users:data:123", mock.Anything).Return(false, nil)
	now := time.Now()
	batch := db.NewWriteBatch()
	batch.Put("cf1", testColumnFamilySector, indexKey, []byte("123"), now)
	batch.Put("cf1", testColumnFamilySector, nameFieldKey, []byte("123"), now)
	batch.Put("cf1", testColumnFamilySector, uNameFieldKey, []byte("123"), now)
	batch.Put("cf1", testColumnFamilySector, dataKey, data, now)

	mockStore.On("Write", mock.MatchedBy(func(b *db.WriteBatch) bool {
		return true
	}), mock.Anything).Return(nil)

	id, err := repo.Create(&user, now)
	assert.NoError(t, err)
	assert.Equal(t, id, "123")

	mockStore.AssertExpectations(t)
}

func TestRepository_Create_DuplicatePrimaryKey_InDB(t *testing.T) {
	mockStore := new(MockKVStore)
	iGF := &db.DeterministicIDGeneratorFactory{} // Allows providing ID in entity

	repo, err := db.NewRepository[User](mockStore, "cf1", testColumnFamilySector, "admin", iGF)
	require.NoError(t, err)

	existingUserID := "existing-id-123"
	user := User{
		ID:   existingUserID, // Provide ID directly
		Name: "Alice",
	}

	// Mock that the primary key (data key) already exists
	dataKey := "admin:users:data:" + existingUserID
	mockStore.On("Exists", "cf1", testColumnFamilySector, dataKey, mock.Anything).Return(true, nil)
	// No unique checks for "Name" should be hit if PK check fails first
	// No "Write" should be called

	_, err = repo.Create(&user, time.Now())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate primary key: ID = "+existingUserID+" already exists")

	mockStore.AssertExpectations(t)
}

func TestRepository_BulkCreate_DuplicatePrimaryKey_InDB(t *testing.T) {
	mockStore := new(MockKVStore)
	iGF := &db.DeterministicIDGeneratorFactory{} // Allows providing IDs in entities

	repo, err := db.NewRepository[User](mockStore, "cf1", testColumnFamilySector, "admin", iGF)
	require.NoError(t, err)

	existingUserID := "existing-bulk-id-456"
	users := []*User{
		{ID: "new-bulk-id-1", Name: "UserNew1"},
		{ID: existingUserID, Name: "UserExistingID"}, // This ID already exists in DB
		{ID: "new-bulk-id-2", Name: "UserNew2"},
	}

	// Mock that the primary key for the second user already exists
	pkDataKeyExisting := "admin:users:data:" + existingUserID
	// For the first user (new-bulk-id-1), PK check should pass
	pkDataKeyNew1 := "admin:users:data:new-bulk-id-1"
	mockStore.On("Exists", "cf1", testColumnFamilySector, pkDataKeyNew1, mock.Anything).Return(false, nil).Once()
	// For the second user (existingUserID), PK check should fail
	//mockStore.On("Exists", "cf1", testColumnFamilySector, pkDataKeyExisting).Return(true, nil).Once()

	mockStore.On("Exists", "cf1", testColumnFamilySector, pkDataKeyExisting, mock.Anything).Return(true, nil)
	// No further Exists or Write calls should happen for subsequent users or the batch itself.

	_, err = repo.BulkCreate(users, time.Now())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate primary key: ID = "+existingUserID+" already exists")

	mockStore.AssertExpectations(t)
}

func TestRepository_BulkCreate_DuplicatePrimaryKey_InBatch(t *testing.T) {
	mockStore := new(MockKVStore)
	iGF := &db.DeterministicIDGeneratorFactory{} // Allows providing IDs in entities

	repo, err := db.NewRepository[User](mockStore, "cf1", testColumnFamilySector, "admin", iGF)
	require.NoError(t, err)

	duplicateIDInBatch := "dup-batch-id-789"
	users := []*User{
		{ID: "unique-batch-id-1", Name: "BatchUser1"},
		{ID: duplicateIDInBatch, Name: "BatchUser2"},
		{ID: duplicateIDInBatch, Name: "BatchUser3"}, // Duplicate ID within the same batch
	}

	// Mock that the primary key for the first user (unique-batch-id-1) does not exist
	pkDataKeyUnique1 := "admin:users:data:unique-batch-id-1"
	// Mock that the primary key for the second user (duplicateIDInBatch) does not exist (for the first occurrence)
	pkDataKeyDup := "admin:users:data:" + duplicateIDInBatch

	// The third user will cause the "duplicate primary key in input batch" error before DB check.
	mockStore.On("Exists", "cf1", testColumnFamilySector, pkDataKeyUnique1, mock.Anything).Return(false, nil)
	mockStore.On("Exists", "cf1", testColumnFamilySector, pkDataKeyDup, mock.Anything).Return(false, nil)

	_, err = repo.BulkCreate(users, time.Now())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate primary key in input batch: ID = "+duplicateIDInBatch)

	mockStore.AssertExpectations(t)
}

func TestRepository_Create_EmptyProvidedID(t *testing.T) {
	mockStore := new(MockKVStore)
	iGF := &db.DeterministicIDGeneratorFactory{} // Using this factory means ID must be provided

	repo, err := db.NewRepository[User](mockStore, "cf1", testColumnFamilySector, "admin", iGF)
	require.NoError(t, err)

	userWithEmptyID := User{
		ID:   "", // Explicitly empty ID
		Name: "UserWithEmptyID",
	}

	// No DB calls should be made if the ID is empty.
	_, err = repo.Create(&userWithEmptyID, time.Now())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "primary key field 'ID' cannot be empty when not generated")

	mockStore.AssertExpectations(t) // Should be no expectations set if code fails before DB interaction
}

func TestRepository_Create_DuplicateUnique(t *testing.T) {
	mockStore := new(MockKVStore)

	user := User{
		ID:   "----",
		Name: "Alice",
	}

	iGF := NewTestIDGeneratorFactory([]string{"123"})

	repo, err := db.NewRepository[User](mockStore, "cf1", testColumnFamilySector, "admin", iGF)
	assert.NoError(t, err)

	uIndexKey := "admin:users:idx-u:Name:Alice"
	dataKey := "admin:users:data:123"

	// Mock for checking if unique field exists (should return true indicating duplicate)
	mockStore.On("Exists", "cf1", testColumnFamilySector, uIndexKey, mock.Anything).Return(true, nil)
	mockStore.On("Exists", "cf1", testColumnFamilySector, dataKey, mock.Anything).Return(false, nil)

	// Create should fail due to duplicate unique field
	id, err := repo.Create(&user, time.Now())
	assert.Error(t, err)
	assert.Empty(t, id)
	assert.Contains(t, err.Error(), "duplicate unique field: Name = Alice")

	mockStore.AssertExpectations(t)
}

func TestRepository_Create_MissingPrimaryKeyValue(t *testing.T) {
	mockStore := new(MockKVStore)

	iGF := NewTestIDGeneratorFactory([]string{"123"})
	_, err := db.NewRepository[NoPrimary](mockStore, "cf1", testColumnFamilySector, "admin", iGF)
	assert.EqualError(t, err, "struct NoPrimary must have a string field named 'ID' with `orm:\"primary-key\"`")

}

func TestRepository_FindByField_Success(t *testing.T) {
	mockStore := new(MockKVStore)
	iGF := NewTestIDGeneratorFactory([]string{"123"})
	repo, err := db.NewRepository[User](mockStore, "cf1", testColumnFamilySector, "admin", iGF)
	assert.NoError(t, err)

	uIndexKey := "admin:users:idx-u:Name:Alice"
	dataKey := "admin:users:data:123"
	user := User{ID: "123", Name: "Alice"}
	data, _ := json.Marshal(user)

	mockStore.On("Get", "cf1", testColumnFamilySector, uIndexKey, mock.Anything).Return([]byte("123"), nil)
	mockStore.On("Get", "cf1", testColumnFamilySector, dataKey, mock.Anything).Return(data, nil)

	result, err := repo.FindByField("Name", "Alice", time.Now())
	assert.NoError(t, err)
	assert.Equal(t, &user, result)

	mockStore.AssertExpectations(t)
}

func TestRepository_FindByField_Unknown_Field_Name(t *testing.T) {
	mockStore := new(MockKVStore)
	iGF := NewTestIDGeneratorFactory([]string{"123"})
	repo, err := db.NewRepository[User](mockStore, "cf1", testColumnFamilySector, "admin", iGF)
	assert.NoError(t, err)

	_, err = repo.FindByField("x", "Alice", time.Now())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Unknown field x")
}

func TestRepository_Find_AND_Unknown_Field_Name(t *testing.T) {
	mockStore := new(MockKVStore)
	iGF := NewTestIDGeneratorFactory([]string{"123"})
	repo, err := db.NewRepository[User](mockStore, "cf1", testColumnFamilySector, "admin", iGF)
	assert.NoError(t, err)

	mockStore.On("SearchByPatternPaginatedKV", "cf1", testColumnFamilySector, "admin:users:idx:ID:123:*", "", 1000, mock.Anything).
		Return([]db.KeyValuePair{{Value: []byte("123")}}, "", nil)

	_, err = repo.Find("ID=123&X=Alice", 1000, "", time.Now())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Unknown field X")
	mockStore.AssertExpectations(t)
}

func TestRepository_FindByField_Invalid_Use_For_TTL_Query(t *testing.T) {
	mockStore := new(MockKVStore)
	iGF := NewTestIDGeneratorFactory([]string{"123"})
	repo, err := db.NewRepository[TempEntity](mockStore, "cf1", testColumnFamilySector, "admin", iGF)
	assert.NoError(t, err)

	_, err = repo.FindByField("TTL", "111", time.Now())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "TTL columns are not supported in query operations")
}

func TestRepository_Find_AND_Invalid_Use_For_TTL_Query(t *testing.T) {
	mockStore := new(MockKVStore)
	iGF := NewTestIDGeneratorFactory([]string{"123"})
	repo, err := db.NewRepository[TempEntity](mockStore, "cf1", testColumnFamilySector, "admin", iGF)
	assert.NoError(t, err)

	mockStore.On("SearchByPatternPaginatedKV", "cf1", testColumnFamilySector, "admin:temporal_entities:idx:ID:123:*", "", 1000, mock.Anything).
		Return([]db.KeyValuePair{{Value: []byte("123")}}, "", nil)

	_, err = repo.Find("ID=123&TTL=22", 1000, "", time.Now())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "TTL columns are not supported in query operation")
	mockStore.AssertExpectations(t)
}

func TestRepository_FindByField_Invalid_TTL_Field_name(t *testing.T) {
	mockStore := new(MockKVStore)
	iGF := NewTestIDGeneratorFactory([]string{"123"})
	_, err := db.NewRepository[InvalidTempEntity](mockStore, "cf1", testColumnFamilySector, "admin", iGF)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error extracting fields from struct InvalidTempEntity: ttl can only be defined on top-level 'TTL' field, found on 'ttl'")
}

func TestNewRepository_MissingPrimaryKey(t *testing.T) {
	mockStore := new(MockKVStore)
	iGF := NewTestIDGeneratorFactory([]string{"123"})

	_, err := db.NewRepository[Invalid](mockStore, "cf1", testColumnFamilySector, "admin", iGF)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "struct Invalid must have a string field named 'ID' with `orm:\"primary-key\"`")
}

func TestRepository_Find_AND(t *testing.T) {
	mockStore := new(MockKVStore)
	iGF := NewTestIDGeneratorFactory([]string{"123"})
	repo, err := db.NewRepository[User](mockStore, "cf1", testColumnFamilySector, "admin", iGF)
	assert.NoError(t, err)

	user := User{ID: "123", Name: "Alice"}
	data, _ := json.Marshal(user)

	mockStore.On("SearchByPatternPaginatedKV", "cf1", testColumnFamilySector, "admin:users:idx:ID:123:*", "", 1000, mock.Anything).
		Return([]db.KeyValuePair{{Value: []byte("123")}}, "", nil)
	mockStore.On("SearchByPatternPaginatedKV", "cf1", testColumnFamilySector, "admin:users:idx:Name:Alice:*", "", 1000, mock.Anything).
		Return([]db.KeyValuePair{{Value: []byte("123")}}, "", nil)
	mockStore.On("Get", "cf1", testColumnFamilySector, "admin:users:data:123", mock.Anything).
		Return(data, nil)

	result, err := repo.Find("ID=123&Name=Alice", 1000, "", time.Now())
	assert.NoError(t, err)
	assert.Len(t, result.Entities, 1)
	assert.Equal(t, &user, &result.Entities[0])
}

func TestRepository_Find_OR(t *testing.T) {
	mockStore := new(MockKVStore)
	iGF := NewTestIDGeneratorFactory([]string{"123"})
	repo, err := db.NewRepository[User](mockStore, "cf1", testColumnFamilySector, "admin", iGF)
	assert.NoError(t, err)

	user1 := User{ID: "123", Name: "Alice"}
	user2 := User{ID: "456", Name: "Bob"}
	data1, _ := json.Marshal(user1)
	data2, _ := json.Marshal(user2)

	mockStore.On("SearchByPatternPaginatedKV", "cf1", testColumnFamilySector, "admin:users:idx:Name:Alice:*", "", 1000, mock.Anything).
		Return([]db.KeyValuePair{{Value: []byte("123")}}, "", nil)
	mockStore.On("SearchByPatternPaginatedKV", "cf1", testColumnFamilySector, "admin:users:idx:Name:Bob:*", "", 1000, mock.Anything).
		Return([]db.KeyValuePair{{Value: []byte("456")}}, "", nil)
	mockStore.On("Get", "cf1", testColumnFamilySector, "admin:users:data:123", mock.Anything).
		Return(data1, nil)
	mockStore.On("Get", "cf1", testColumnFamilySector, "admin:users:data:456", mock.Anything).
		Return(data2, nil)

	result, err := repo.Find("Name=Alice|Name=Bob", 1000, "", time.Now())
	assert.NoError(t, err)
	assert.Len(t, result.Entities, 2)
}

func TestRepository_Find_SpecialCharacters(t *testing.T) {
	mockStore := new(MockKVStore)
	iGF := NewTestIDGeneratorFactory([]string{"123"})
	repo, err := db.NewRepository[User](mockStore, "cf1", testColumnFamilySector, "admin", iGF)
	assert.NoError(t, err)

	user := User{ID: "999", Name: "foo:bar"}
	data, _ := json.Marshal(user)

	mockStore.On("SearchByPatternPaginatedKV", "cf1", testColumnFamilySector, "admin:users:idx:Name:foo:bar:*", "", 1000, mock.Anything).
		Return([]db.KeyValuePair{{Value: []byte("999")}}, "", nil)
	mockStore.On("Get", "cf1", testColumnFamilySector, "admin:users:data:999", mock.Anything).
		Return(data, nil)

	result, err := repo.Find("Name=foo:bar", 1000, "", time.Now())
	assert.NoError(t, err)
	assert.Len(t, result.Entities, 1)
	assert.Equal(t, &user, &result.Entities[0])
}

func TestRepository_Find_NoMatch(t *testing.T) {
	mockStore := new(MockKVStore)
	iGF := NewTestIDGeneratorFactory([]string{"123"})
	repo, err := db.NewRepository[User](mockStore, "cf1", testColumnFamilySector, "admin", iGF)
	assert.NoError(t, err)

	mockStore.On("SearchByPatternPaginatedKV", "cf1", testColumnFamilySector, "admin:users:idx:Name:Ghost:*", "", 1000, mock.Anything).
		Return(nil, "", nil)

	result, err := repo.Find("Name=Ghost", 1000, "", time.Now())
	assert.NoError(t, err)
	assert.Len(t, result.Entities, 0)
}

func TestRepository_Update_Success(t *testing.T) {
	mockStore := new(MockKVStore)

	iGF := NewTestIDGeneratorFactory([]string{"123"})
	repo, err := db.NewRepository[User](mockStore, "cf1", testColumnFamilySector, "admin", iGF)
	assert.NoError(t, err)

	originalUser := User{ID: "123", Name: "Alice"}
	updatedUser := User{ID: "123", Name: "Bob"}

	oldIndexKey := "admin:users:idx:Name:Alice:123"
	newIndexKey := "admin:users:idx:Name:Bob:123"

	oldUIndexKey := "admin:users:idx-u:Name:Alice"
	newUIndexKey := "admin:users:idx-u:Name:Bob"

	dataKey := "admin:users:data:123"

	originalData, _ := json.Marshal(originalUser)
	updatedData, _ := json.Marshal(updatedUser)

	// FindByField (which calls Get) will be called with now.
	// The Get for unique check will also be called with now.
	mockStore.On("Get", "cf1", testColumnFamilySector, dataKey, mock.Anything).Return(originalData, nil)
	mockStore.On("Get", "cf1", testColumnFamilySector, newUIndexKey, mock.Anything).Return(nil, nil)
	now := time.Now()
	batch := db.NewWriteBatch()
	batch.Delete("cf1", testColumnFamilySector, oldUIndexKey, now)
	batch.Put("cf1", testColumnFamilySector, newUIndexKey, []byte("123"), now)
	batch.Delete("cf1", testColumnFamilySector, oldIndexKey, now)

	batch.Put("cf1", testColumnFamilySector, newIndexKey, []byte("123"), now)
	batch.Put("cf1", testColumnFamilySector, dataKey, updatedData, now)

	mockStore.On("Write", mock.MatchedBy(func(b *db.WriteBatch) bool {
		return true
	}), mock.Anything).Return(nil)

	changed, err := repo.Update(&updatedUser, time.Now())
	assert.NoError(t, err)
	assert.Equal(t, changed, true)

	mockStore.AssertExpectations(t)
}

func TestRepository_Update_Nonexistent(t *testing.T) {
	mockStore := new(MockKVStore)

	iGF := NewTestIDGeneratorFactory([]string{"123"})
	repo, err := db.NewRepository[User](mockStore, "cf1", testColumnFamilySector, "admin", iGF)
	assert.NoError(t, err)

	user := User{ID: "999", Name: "Ghost"}
	dataKey := "admin:users:data:999"

	mockStore.On("Get", "cf1", testColumnFamilySector, dataKey, mock.Anything).Return(nil, nil) // For FindByField

	changed, err := repo.Update(&user, time.Now())
	assert.NoError(t, err)
	assert.Equal(t, changed, false)
}
func TestRepository_Delete_Success(t *testing.T) {
	mockStore := new(MockKVStore)

	iGF := NewTestIDGeneratorFactory([]string{"123"})
	repo, err := db.NewRepository[User](mockStore, "cf1", testColumnFamilySector, "admin", iGF)
	assert.NoError(t, err)

	user := User{ID: "123", Name: "Alice"}
	dataKey := "admin:users:data:123"
	indexKey := "admin:users:idx:Name:Alice:123"
	uIndexKey := "admin:users:idx-u:Name:Alice"
	pkIndexKey := "admin:users:idx:ID:123:123"
	data, _ := json.Marshal(user)

	mockStore.On("Get", "cf1", testColumnFamilySector, dataKey, mock.Anything).Return(data, nil) // For FindByField

	batch := db.NewWriteBatch()
	now := time.Now()
	batch.Delete("cf1", testColumnFamilySector, indexKey, now)
	batch.Delete("cf1", testColumnFamilySector, pkIndexKey, now)
	batch.Delete("cf1", testColumnFamilySector, uIndexKey, now)
	batch.Delete("cf1", testColumnFamilySector, dataKey, now)

	mockStore.On("Write", mock.MatchedBy(func(b *db.WriteBatch) bool {
		return true
	}), mock.Anything).Return(nil)

	deleted, err := repo.Delete("123", time.Now())
	assert.Equal(t, deleted, true)
	assert.NoError(t, err)

	mockStore.AssertExpectations(t)
}

func TestRepository_Delete_NotFound(t *testing.T) {
	mockStore := new(MockKVStore)

	iGF := NewTestIDGeneratorFactory([]string{"123"})
	repo, err := db.NewRepository[User](mockStore, "cf1", testColumnFamilySector, "admin", iGF)
	assert.NoError(t, err)

	dataKey := "admin:users:data:123"
	mockStore.On("Get", "cf1", testColumnFamilySector, dataKey, mock.Anything).Return(nil, nil) // For FindByField

	deleted, err := repo.Delete("123", time.Now())
	assert.Equal(t, deleted, false)
	assert.NoError(t, err)
	mockStore.AssertExpectations(t)
}
func TestRepository_Delete_CorruptedData(t *testing.T) {
	mockStore := new(MockKVStore)

	iGF := NewTestIDGeneratorFactory([]string{"123"})
	repo, err := db.NewRepository[User](mockStore, "cf1", testColumnFamilySector, "admin", iGF)
	assert.NoError(t, err)

	dataKey := "admin:users:data:123"

	mockStore.On("Get", "cf1", testColumnFamilySector, dataKey, mock.Anything).Return([]byte("not a valid json"), nil) // For FindByField

	deleted, err := repo.Delete("123", time.Now())
	assert.Equal(t, deleted, false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid character")
	mockStore.AssertExpectations(t)
}

func TestRepository_BulkCreate_Success(t *testing.T) {
	mockStore := new(MockKVStore)
	iGF := NewTestIDGeneratorFactory([]string{"id1", "id2", "id3"})
	repo, err := db.NewRepository[User](mockStore, "cf1", testColumnFamilySector, "admin", iGF)
	assert.NoError(t, err)

	users := []*User{
		{Name: "Alice"},
		{Name: "Bob"},
		{Name: "Charlie"},
	}

	batch := db.NewWriteBatch()
	for i, u := range users {
		id := iGF.ids[i]
		u.ID = id
		d, _ := json.Marshal(u)

		dataKey := "admin:users:data:" + id
		nameIdx := "admin:users:idx:Name:" + u.Name + ":" + id
		uIdx := "admin:users:idx-u:Name:" + u.Name
		pkIdx := "admin:users:idx:ID:" + id + ":" + id

		mockStore.On("Exists", "cf1", testColumnFamilySector, uIdx, mock.Anything).Return(false, nil)
		mockStore.On("Exists", "cf1", testColumnFamilySector, dataKey, mock.Anything).Return(false, nil)
		now := time.Now()
		batch.Put("cf1", testColumnFamilySector, dataKey, d, now)
		batch.Put("cf1", testColumnFamilySector, nameIdx, []byte(id), now)
		batch.Put("cf1", testColumnFamilySector, uIdx, []byte(id), now)
		batch.Put("cf1", testColumnFamilySector, pkIdx, []byte(id), now)
	}

	mockStore.On("Write", mock.MatchedBy(func(b *db.WriteBatch) bool {
		return true // ya validamos el contenido arriba
	}), mock.Anything).Return(nil)

	ids, err := repo.BulkCreate(users, time.Now())
	assert.NoError(t, err)
	assert.Equal(t, []string{"id1", "id2", "id3"}, ids)

	mockStore.AssertExpectations(t)
}

func TestRepository_BulkCreate_DuplicateUnique(t *testing.T) {
	mockStore := new(MockKVStore)
	iGF := NewTestIDGeneratorFactory([]string{"id1", "id2"})
	repo, err := db.NewRepository[User](mockStore, "cf1", testColumnFamilySector, "admin", iGF)
	assert.NoError(t, err)

	users := []*User{
		{Name: "Alice"}, // New user
		{Name: "Bob"},   // Duplicate user that already exists
	}

	// Mock for Alice (new user)
	mockStore.On("Exists", "cf1", testColumnFamilySector, "admin:users:idx-u:Name:Alice", mock.Anything).Return(false, nil).Once()
	mockStore.On("Exists", "cf1", testColumnFamilySector, "admin:users:data:id1", mock.Anything).Return(false, nil).Once()

	// Mock for Bob (existing user) - should cause error
	mockStore.On("Exists", "cf1", testColumnFamilySector, "admin:users:idx-u:Name:Bob", mock.Anything).Return(true, nil).Once()
	mockStore.On("Exists", "cf1", testColumnFamilySector, "admin:users:data:id2", mock.Anything).Return(false, nil).Once()

	// BulkCreate should fail due to duplicate unique field
	ids, err := repo.BulkCreate(users, time.Now())
	assert.Error(t, err)
	assert.Nil(t, ids)
	assert.Contains(t, err.Error(), "duplicate unique field: Name = Bob")

	mockStore.AssertExpectations(t)
}

func TestRepository_BulkCreate_WriteError(t *testing.T) {
	mockStore := new(MockKVStore)
	iGF := NewTestIDGeneratorFactory([]string{"id1"})
	repo, err := db.NewRepository[User](mockStore, "cf1", testColumnFamilySector, "admin", iGF)
	assert.NoError(t, err)

	users := []*User{
		{Name: "Alice"},
	}

	mockStore.On("Exists", "cf1", testColumnFamilySector, "admin:users:idx-u:Name:Alice", mock.Anything).Return(false, nil)
	mockStore.On("Exists", "cf1", testColumnFamilySector, "admin:users:data:id1", mock.Anything).Return(false, nil)
	mockStore.On("Write", mock.Anything, mock.Anything).Return(errors.New("write failed"))

	_, err = repo.BulkCreate(users, time.Now())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "write failed")
}
func TestRepository_BulkDelete_Success(t *testing.T) {
	mockStore := new(MockKVStore)
	iGF := NewTestIDGeneratorFactory([]string{"id1", "id2"})
	repo, err := db.NewRepository[User](mockStore, "cf1", testColumnFamilySector, "admin", iGF)
	assert.NoError(t, err)

	users := []*User{
		{ID: "id1", Name: "Alice"},
		{ID: "id2", Name: "Bob"},
	}

	for _, u := range users {
		data, _ := json.Marshal(u)
		// This Get is part of the FindByField call within BulkDelete
		mockStore.On("Get", "cf1", testColumnFamilySector, "admin:users:data:"+u.ID, mock.Anything).Return(data, nil)
	}
	now := time.Now()
	batch := db.NewWriteBatch()
	batch.Delete("cf1", testColumnFamilySector, "admin:users:idx:Name:Alice:id1", now)
	batch.Delete("cf1", testColumnFamilySector, "admin:users:idx:Name:Bob:id2", now)
	batch.Delete("cf1", testColumnFamilySector, "admin:users:idx:ID:id1:id1", now)
	batch.Delete("cf1", testColumnFamilySector, "admin:users:idx:ID:id2:id2", now)
	batch.Delete("cf1", testColumnFamilySector, "admin:users:idx-u:Name:Alice", now)
	batch.Delete("cf1", testColumnFamilySector, "admin:users:idx-u:Name:Bob", now)
	batch.Delete("cf1", testColumnFamilySector, "admin:users:data:id1", now)
	batch.Delete("cf1", testColumnFamilySector, "admin:users:data:id2", now)

	mockStore.On("Write", mock.MatchedBy(func(b *db.WriteBatch) bool {
		return true
	}), mock.Anything).Return(nil)

	deleted, err := repo.BulkDelete([]string{"id1", "id2"}, time.Now())
	assert.NoError(t, err)
	require.Len(t, deleted, 2)
	assert.True(t, deleted[0])
	assert.True(t, deleted[1])
	mockStore.AssertExpectations(t)
}

func TestRepository_BulkDelete_Partial(t *testing.T) {
	mockStore := new(MockKVStore)
	iGF := NewTestIDGeneratorFactory([]string{"id1", "id2"})
	repo, err := db.NewRepository[User](mockStore, "cf1", testColumnFamilySector, "admin", iGF)
	assert.NoError(t, err)

	user := &User{ID: "id1", Name: "Alice"}
	data, _ := json.Marshal(user)
	// These Gets are part of FindByField calls
	mockStore.On("Get", "cf1", testColumnFamilySector, "admin:users:data:id1", mock.Anything).Return(data, nil)
	mockStore.On("Get", "cf1", testColumnFamilySector, "admin:users:data:id2", mock.Anything).Return(nil, nil)
	now := time.Now()
	batch := db.NewWriteBatch()
	batch.Delete("cf1", testColumnFamilySector, "admin:users:idx:Name:Alice:id1", now)
	batch.Delete("cf1", testColumnFamilySector, "admin:users:idx:ID:id1:id1", now)
	batch.Delete("cf1", testColumnFamilySector, "admin:users:idx-u:Name:Alice", now)
	batch.Delete("cf1", testColumnFamilySector, "admin:users:data:id1", now)

	mockStore.On("Write", mock.MatchedBy(func(b *db.WriteBatch) bool {
		return true
	}), mock.Anything).Return(nil)

	deleted, err := repo.BulkDelete([]string{"id1", "id2"}, time.Now())
	assert.NoError(t, err)
	require.Len(t, deleted, 2)
	assert.True(t, deleted[0])
	assert.False(t, deleted[1])
	mockStore.AssertExpectations(t)
}

func TestRepository_BulkDelete_InvalidData(t *testing.T) {
	mockStore := new(MockKVStore)
	iGF := NewTestIDGeneratorFactory([]string{"id1"})
	repo, err := db.NewRepository[User](mockStore, "cf1", testColumnFamilySector, "admin", iGF)
	assert.NoError(t, err)

	mockStore.On("Get", "cf1", testColumnFamilySector, "admin:users:data:id1", mock.Anything).Return([]byte("invalid json"), nil) // For FindByField

	deleted, err := repo.BulkDelete([]string{"id1"}, time.Now())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid character")

	require.Len(t, deleted, 0)

	mockStore.AssertExpectations(t)
}
func TestRepository_BulkUpdate_Success(t *testing.T) {
	mockStore := new(MockKVStore)
	iGF := NewTestIDGeneratorFactory([]string{"id1", "id2"})
	repo, err := db.NewRepository[User](mockStore, "cf1", testColumnFamilySector, "admin", iGF)
	require.NoError(t, err)

	original1 := User{ID: "id1", Name: "Alice"}
	updated1 := User{ID: "id1", Name: "AliceUpdated"}

	original2 := User{ID: "id2", Name: "Bob"}
	updated2 := User{ID: "id2", Name: "BobUpdated"}

	// Serializar original data para mockear GET
	originalData1, _ := json.Marshal(original1)
	originalData2, _ := json.Marshal(original2)

	// Se mockea GET para los datos originales (called by FindByField)
	mockStore.On("Get", "cf1", testColumnFamilySector, "admin:users:data:id1", mock.Anything).Return(originalData1, nil)
	mockStore.On("Get", "cf1", testColumnFamilySector, "admin:users:data:id2", mock.Anything).Return(originalData2, nil)

	// Para las claves únicas nuevas, simular que no existen (para no dar error de duplicados)
	mockStore.On("Get", "cf1", testColumnFamilySector, "admin:users:idx-u:Name:AliceUpdated", mock.Anything).Return(nil, nil)
	mockStore.On("Get", "cf1", testColumnFamilySector, "admin:users:idx-u:Name:BobUpdated", mock.Anything).Return(nil, nil)

	mockStore.On("Write", mock.Anything, mock.Anything).Return(nil)

	results, err := repo.BulkUpdate([]*User{&updated1, &updated2}, time.Now())
	assert.NoError(t, err)
	assert.Len(t, results, 2)
	assert.True(t, results[0])
	assert.True(t, results[1])

	mockStore.AssertExpectations(t)
}

func TestRepository_BulkUpdate_SomeNonexistent(t *testing.T) {
	mockStore := new(MockKVStore)
	iGF := NewTestIDGeneratorFactory([]string{"id1", "id2"})
	repo, err := db.NewRepository[User](mockStore, "cf1", testColumnFamilySector, "admin", iGF)
	require.NoError(t, err)

	existing := User{ID: "id1", Name: "Alice"}
	updatedExisting := User{ID: "id1", Name: "AliceUpdated"}
	nonexistent := User{ID: "id999", Name: "Ghost"}

	originalData, _ := json.Marshal(existing)

	mockStore.On("Get", "cf1", testColumnFamilySector, "admin:users:data:id1", mock.Anything).Return(originalData, nil)
	mockStore.On("Get", "cf1", testColumnFamilySector, "admin:users:data:id999", mock.Anything).Return(nil, nil) // no existe

	mockStore.On("Get", "cf1", testColumnFamilySector, "admin:users:idx-u:Name:AliceUpdated", mock.Anything).Return(nil, nil)

	mockStore.On("Write", mock.Anything, mock.Anything).Return(nil)

	results, err := repo.BulkUpdate([]*User{&updatedExisting, &nonexistent}, time.Now())
	assert.NoError(t, err)
	assert.Len(t, results, 2)
	assert.True(t, results[0])  // id1 actualizado
	assert.False(t, results[1]) // id999 no existe, no actualizado

	mockStore.AssertExpectations(t)
}

func TestRepository_BulkUpdate_DuplicateUniqueWithinBatch(t *testing.T) {
	mockStore := new(MockKVStore)
	iGF := NewTestIDGeneratorFactory([]string{"id1", "id2"})
	repo, err := db.NewRepository[User](mockStore, "cf1", testColumnFamilySector, "admin", iGF)
	require.NoError(t, err)

	original1 := User{ID: "id1", Name: "Alice"}
	original2 := User{ID: "id2", Name: "Bob"}

	updated1 := User{ID: "id1", Name: "SameName"} // Cambia a SameName
	updated2 := User{ID: "id2", Name: "SameName"} // Mismo nombre -> duplicado interno

	originalData1, _ := json.Marshal(original1)
	originalData2, _ := json.Marshal(original2)

	mockStore.On("Get", "cf1", testColumnFamilySector, "admin:users:data:id1", mock.Anything).Return(originalData1, nil)
	mockStore.On("Get", "cf1", testColumnFamilySector, "admin:users:data:id2", mock.Anything).Return(originalData2, nil)

	// El repositorio debería detectar el duplicado dentro del batch sin llamar al store para el índice uName

	results, err := repo.BulkUpdate([]*User{&updated1, &updated2}, time.Now())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate unique field")
	assert.Nil(t, results)
}

func TestRepository_BulkUpdate_DuplicateUniqueExisting(t *testing.T) {
	mockStore := new(MockKVStore)
	iGF := NewTestIDGeneratorFactory([]string{"id1", "id2"})
	repo, err := db.NewRepository[User](mockStore, "cf1", testColumnFamilySector, "admin", iGF)
	require.NoError(t, err)

	original1 := User{ID: "id1", Name: "Alice"}
	updated1 := User{ID: "id1", Name: "Bob"} // Cambia a "Bob"

	originalData1, _ := json.Marshal(original1)

	mockStore.On("Get", "cf1", testColumnFamilySector, "admin:users:data:id1", mock.Anything).Return(originalData1, nil)
	// Simulamos que el índice único ya apunta a otro ID distinto al que actualizamos
	mockStore.On("Get", "cf1", testColumnFamilySector, "admin:users:idx-u:Name:Bob", mock.Anything).Return([]byte("otherID"), nil)

	results, err := repo.BulkUpdate([]*User{&updated1}, time.Now())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate unique field")
	assert.Nil(t, results)
}

func TestRepository_BulkUpdate_WriteError(t *testing.T) {
	mockStore := new(MockKVStore)
	iGF := NewTestIDGeneratorFactory([]string{"id1"})
	repo, err := db.NewRepository[User](mockStore, "cf1", testColumnFamilySector, "admin", iGF)
	require.NoError(t, err)

	original := User{ID: "id1", Name: "Alice"}
	updated := User{ID: "id1", Name: "AliceUpdated"}

	originalData, _ := json.Marshal(original)

	mockStore.On("Get", "cf1", testColumnFamilySector, "admin:users:data:id1", mock.Anything).Return(originalData, nil)
	mockStore.On("Get", "cf1", testColumnFamilySector, "admin:users:idx-u:Name:AliceUpdated", mock.Anything).Return(nil, nil)

	mockStore.On("Write", mock.Anything, mock.Anything).Return(errors.New("write failed"))

	results, err := repo.BulkUpdate([]*User{&updated}, time.Now())
	assert.Error(t, err)
	assert.Nil(t, results)
}

func TestRepository_BulkUpdate_InvalidData(t *testing.T) {
	mockStore := new(MockKVStore)
	iGF := NewTestIDGeneratorFactory([]string{"id1"})
	repo, err := db.NewRepository[User](mockStore, "cf1", testColumnFamilySector, "admin", iGF)
	require.NoError(t, err)

	// Simula que data original almacenada está corrupta (no json válido)
	mockStore.On("Get", "cf1", testColumnFamilySector, "admin:users:data:id1", mock.Anything).Return([]byte("not json"), nil)

	updated := User{ID: "id1", Name: "AliceUpdated"}

	results, err := repo.BulkUpdate([]*User{&updated}, time.Now())
	assert.Error(t, err)
	assert.Nil(t, results)
}

func TestRepository_BulkUpdate_EmptyInput(t *testing.T) {
	mockStore := new(MockKVStore)
	iGF := NewTestIDGeneratorFactory([]string{})
	repo, err := db.NewRepository[User](mockStore, "cf1", testColumnFamilySector, "admin", iGF)
	require.NoError(t, err)

	results, err := repo.BulkUpdate([]*User{}, time.Now())
	assert.NoError(t, err)
	assert.NotNil(t, results)
	assert.Len(t, results, 0)
}

func TestRepository_Create_NestedSuccess_UserComplexEmbedded(t *testing.T) {
	mockStore := new(MockKVStore)
	entityID := "uce123"
	iGF := NewTestIDGeneratorFactory([]string{entityID})

	repo, err := db.NewRepository[UserComplexEmbedded](mockStore, "cf_embed", testColumnFamilySector, "test_sch_embed", iGF)
	require.NoError(t, err)

	user := UserComplexEmbedded{
		Email:  "embedded@example.com",
		Status: "pending",
		MetaForEmbed: MetaForEmbed{ // Fields from MetaForEmbed should be top-level
			Tag:         "embeddedTag1",
			ConfigCode:  303,
			Description: "Embedded Desc",
		},
	}

	// Mock for unique checks (Email and embedded Tag)
	// These Gets are effectively Exists checks
	mockStore.On("Exists", "cf_embed", testColumnFamilySector, "test_sch_embed:users_complex_embedded:idx-u:Email:embedded@example.com", mock.Anything).Return(false, nil).Once()
	// Assuming embedded field 'Tag' becomes a top-level field name 'Tag'
	mockStore.On("Exists", "cf_embed", testColumnFamilySector, "test_sch_embed:users_complex_embedded:idx-u:Tag:embeddedTag1", mock.Anything).Return(false, nil).Once()
	mockStore.On("Exists", "cf_embed", testColumnFamilySector, "test_sch_embed:users_complex_embedded:data:uce123", mock.Anything).Return(false, nil).Once()

	mockStore.On("Write", mock.MatchedBy(func(batch *db.WriteBatch) bool {
		return batch.Count() > 0
	}), mock.Anything).Return(nil).Once()

	createdID, err := repo.Create(&user, time.Now())
	require.NoError(t, err)
	assert.Equal(t, entityID, createdID)
	assert.Equal(t, entityID, user.ID)

	// Verify some key names based on embedding (Tag, ConfigCode should be top-level)
	// In a real test, capture the batch passed to Write and assert its contents.
	// For example, one of the calls to Put in the batch should be for:
	// "test_sch_embed:users_complex_embedded:idx:Tag:embeddedTag1:" + entityID

	mockStore.AssertExpectations(t)
}

func TestRepository_Create_NestedDuplicateUnique_UserComplex(t *testing.T) {
	mockStore := new(MockKVStore)
	entityID := "uc456"
	iGF := NewTestIDGeneratorFactory([]string{entityID})

	repo, err := db.NewRepository[UserComplex](mockStore, "cf_complex", testColumnFamilySector, "test_sch", iGF)
	require.NoError(t, err)

	user := UserComplex{
		Email:  "new@example.com", // Assume this is unique for now
		Status: "active",
		Meta: Meta{
			Tag:         "existingTag", // This tag will cause a duplicate match
			ConfigCode:  102,
			Description: "Desc",
		},
	}

	// Mock for unique checks.
	// Email may or may not be checked before Meta.Tag depending on map iteration order,
	// so it is marked as optional with Maybe().
	mockStore.On("Exists", "cf_complex", testColumnFamilySector, "test_sch:users_complex:idx-u:Email:new@example.com", mock.Anything).Return(false, nil).Maybe()
	mockStore.On("Exists", "cf_complex", testColumnFamilySector, "test_sch:users_complex:data:uc456", mock.Anything).Return(false, nil)
	// Simulate Meta.Tag being a duplicate
	mockStore.On("Exists", "cf_complex", testColumnFamilySector, "test_sch:users_complex:idx-u:Meta.Tag:existingTag", mock.Anything).Return(true, nil)

	// Create should fail due to duplicate nested unique field
	id, err := repo.Create(&user, time.Now())
	assert.Error(t, err)
	assert.Empty(t, id)
	assert.Contains(t, err.Error(), "duplicate unique field: Meta.Tag = existingTag")

	mockStore.AssertExpectations(t)
}

func TestRepository_FindByField_NestedSuccess_UserComplex(t *testing.T) {
	mockStore := new(MockKVStore)
	entityID := "uc789"
	iGF := NewTestIDGeneratorFactory([]string{}) // Not used for FindByField directly for ID generation

	repo, err := db.NewRepository[UserComplex](mockStore, "cf_complex", testColumnFamilySector, "test_sch", iGF)
	require.NoError(t, err)

	expectedUser := UserComplex{
		ID:     entityID,
		Email:  "findme@example.com",
		Status: "found",
		Meta: Meta{
			Tag:         "findThisTag",
			ConfigCode:  103,
			Description: "Found Description",
		},
	}
	jsonData, err := json.Marshal(expectedUser)
	require.NoError(t, err)

	uniqueIdxKey := "test_sch:users_complex:idx-u:Meta.Tag:findThisTag"
	dataKey := "test_sch:users_complex:data:" + entityID

	mockStore.On("Get", "cf_complex", testColumnFamilySector, uniqueIdxKey, mock.Anything).Return([]byte(entityID), nil).Once()
	mockStore.On("Get", "cf_complex", testColumnFamilySector, dataKey, mock.Anything).Return(jsonData, nil).Once()

	foundUser, err := repo.FindByField("Meta.Tag", "findThisTag", time.Now())
	require.NoError(t, err)
	require.NotNil(t, foundUser)
	assert.Equal(t, expectedUser, *foundUser)

	mockStore.AssertExpectations(t)
}

func TestRepository_Find_NestedCondition_UserComplex(t *testing.T) {
	mockStore := new(MockKVStore)
	entityID := "uc101"
	iGF := NewTestIDGeneratorFactory([]string{})

	repo, err := db.NewRepository[UserComplex](mockStore, "cf_complex", testColumnFamilySector, "test_sch", iGF)
	require.NoError(t, err)

	user := UserComplex{
		ID:     entityID,
		Email:  "filter@example.com",
		Status: "active",
		Meta: Meta{
			Tag:         "filterTag",
			ConfigCode:  404,
			Description: "Filtered item",
		},
	}
	jsonData, _ := json.Marshal(user)

	// Mock KVStore SearchByPatternPaginatedKV calls
	// For "Meta.Tag = 'filterTag'"
	mockStore.On("SearchByPatternPaginatedKV", "cf_complex", testColumnFamilySector, "test_sch:users_complex:idx:Meta.Tag:filterTag:*", "", 1000, mock.Anything).
		Return([]db.KeyValuePair{{Key: "...", Value: []byte(entityID)}}, "", nil).Once()
	// For "Email = 'filter@example.com'"
	mockStore.On("SearchByPatternPaginatedKV", "cf_complex", testColumnFamilySector, "test_sch:users_complex:idx:Email:filter@example.com:*", "", 1000, mock.Anything).
		Return([]db.KeyValuePair{{Key: "...", Value: []byte(entityID)}}, "", nil).Once()

	// Mock Get for the data key
	mockStore.On("Get", "cf_complex", testColumnFamilySector, "test_sch:users_complex:data:"+entityID, mock.Anything).Return(jsonData, nil).Once()

	result, err := repo.Find("Meta.Tag = 'filterTag' & Email = 'filter@example.com'", 1000, "", time.Now())
	require.NoError(t, err)
	require.Len(t, result.Entities, 1)
	assert.Equal(t, user, result.Entities[0])

	mockStore.AssertExpectations(t)
}

func TestRepository_Update_NestedField_UserComplex(t *testing.T) {
	mockStore := new(MockKVStore)
	entityID := "ucUpdate1"
	iGF := NewTestIDGeneratorFactory([]string{})
	repo, err := db.NewRepository[UserComplex](mockStore, "cf_complex", testColumnFamilySector, "test_sch", iGF)
	require.NoError(t, err)

	originalUser := UserComplex{
		ID:     entityID,
		Email:  "update@example.com",
		Status: "stale",
		Meta: Meta{
			Tag:         "oldTag",
			ConfigCode:  500,
			Description: "Old Description",
		},
	}
	updatedUser := UserComplex{
		ID:     entityID,             // ID must be the same
		Email:  "update@example.com", // Email not changed
		Status: "current",            // Top-level field changed
		Meta: Meta{
			Tag:         "newTag",          // Nested unique field changed
			ConfigCode:  501,               // Nested non-unique field changed
			Description: "New Description", // Nested non-unique field changed
		},
	}

	originalData, _ := json.Marshal(originalUser)
	// updatedData, _ := json.Marshal(updatedUser) // repo.Update internally marshals the modified 'current'

	// 1. FindByField (ID) to get current entity
	mockStore.On("Get", "cf_complex", testColumnFamilySector, "test_sch:users_complex:data:"+entityID, mock.Anything).Return(originalData, nil).Once()

	// 2. Unique check for new Meta.Tag value
	mockStore.On("Get", "cf_complex", testColumnFamilySector, "test_sch:users_complex:idx-u:Meta.Tag:newTag", mock.Anything).Return(nil, nil).Once()
	// (No other unique fields changed in this test case for Meta)

	// 3. Write batch operations (simplified, actual batch content is complex)
	mockStore.On("Write", mock.MatchedBy(func(batch *db.WriteBatch) bool {
		// A real test would capture and inspect the batch.
		// Count > 0 implies deletes and puts for changed fields' indexes and the main data.
		return batch.Count() > 0
	}), mock.Anything).Return(nil).Once()

	changed, err := repo.Update(&updatedUser, time.Now())
	require.NoError(t, err)
	assert.True(t, changed)

	mockStore.AssertExpectations(t)
}

func TestRepository_Update_NestedField_DuplicateUnique_UserComplex(t *testing.T) {
	mockStore := new(MockKVStore)
	entityID := "ucUpdateDup1"
	iGF := NewTestIDGeneratorFactory([]string{})
	repo, err := db.NewRepository[UserComplex](mockStore, "cf_complex", testColumnFamilySector, "test_sch", iGF)
	require.NoError(t, err)

	originalUser := UserComplex{
		ID:    entityID,
		Email: "updatedup@example.com",
		Meta:  Meta{Tag: "originalTag"},
	}
	updatedUser := UserComplex{
		ID:    entityID,
		Email: "updatedup@example.com",
		Meta:  Meta{Tag: "conflictingTag"}, // This tag will conflict
	}

	originalData, _ := json.Marshal(originalUser)

	// 1. FindByField (ID)
	mockStore.On("Get", "cf_complex", testColumnFamilySector, "test_sch:users_complex:data:"+entityID, mock.Anything).Return(originalData, nil).Once()

	// 2. Unique check for new Meta.Tag shows it exists for another ID
	mockStore.On("Get", "cf_complex", testColumnFamilySector, "test_sch:users_complex:idx-u:Meta.Tag:conflictingTag", mock.Anything).Return([]byte("anotherEntityID"), nil).Once()

	changed, err := repo.Update(&updatedUser, time.Now())
	require.Error(t, err)
	assert.False(t, changed)
	assert.Contains(t, err.Error(), "duplicate unique field: Meta.Tag = conflictingTag")

	mockStore.AssertExpectations(t)
}

func TestRepository_Delete_WithNestedFields_UserComplex(t *testing.T) {
	mockStore := new(MockKVStore)
	entityID := "ucDelete1"
	iGF := NewTestIDGeneratorFactory([]string{})
	repo, err := db.NewRepository[UserComplex](mockStore, "cf_complex", testColumnFamilySector, "test_sch", iGF)
	require.NoError(t, err)

	userToDelete := UserComplex{
		ID:     entityID,
		Email:  "delete@example.com",
		Status: "marked_for_deletion",
		Meta: Meta{
			Tag:         "deleteTag",
			ConfigCode:  999,
			Description: "To be deleted",
		},
		Extra: &Meta{Tag: "delExtra"},
	}
	jsonData, _ := json.Marshal(userToDelete)

	// 1. FindByField (ID) to get entity before deleting its indexes
	mockStore.On("Get", "cf_complex", testColumnFamilySector, "test_sch:users_complex:data:"+entityID, mock.Anything).Return(jsonData, nil).Once()

	// 2. Write batch for deletions
	mockStore.On("Write", mock.MatchedBy(func(batch *db.WriteBatch) bool {
		// Expect many deletes: data, all regular indexes, all unique indexes
		return batch.Count() > 0 // Simplified check
	}), mock.Anything).Return(nil).Once()

	deleted, err := repo.Delete(entityID, time.Now())
	require.NoError(t, err)
	assert.True(t, deleted)

	mockStore.AssertExpectations(t)
}

// Test for UserComplexEmbedded with anonymous MetaForEmbed
// Focus on Create to check field name generation for embedded fields.
func TestRepository_Create_UserComplexEmbedded_FieldNames(t *testing.T) {
	mockStore := new(MockKVStore)
	entityID := "uceFieldTest"
	iGF := NewTestIDGeneratorFactory([]string{entityID})

	repo, err := db.NewRepository[UserComplexEmbedded](mockStore, "cf_embed_fn", testColumnFamilySector, "test_sch_fn", iGF)
	require.NoError(t, err)

	user := UserComplexEmbedded{
		Email:  "embedfn@example.com",
		Status: "active_embed_fn",
		MetaForEmbed: MetaForEmbed{
			Tag:         "embedFnTag", // Expect 'Tag' as field name
			ConfigCode:  777,          // Expect 'ConfigCode'
			Description: "Desc for embed FN",
		},
	}

	// Mock unique checks. If extractFieldsRecursively makes embedded fields top-level,
	// then "Tag" should be the unique field name, not "MetaForEmbed.Tag".
	mockStore.On("Exists", "cf_embed_fn", testColumnFamilySector, "test_sch_fn:users_complex_embedded:idx-u:Email:embedfn@example.com", mock.Anything).Return(false, nil).Once()
	mockStore.On("Exists", "cf_embed_fn", testColumnFamilySector, "test_sch_fn:users_complex_embedded:idx-u:Tag:embedFnTag", mock.Anything).Return(false, nil).Once() // Key check
	mockStore.On("Exists", "cf_embed_fn", testColumnFamilySector, "test_sch_fn:users_complex_embedded:data:uceFieldTest", mock.Anything).Return(false, nil).Once()    // Key check

	// Mock the Write call
	// In a real test, capture the batch and verify specific index keys, e.g.:
	// - test_sch_fn:users_complex_embedded:idx:Tag:embedFnTag:<id>
	// - test_sch_fn:users_complex_embedded:idx:ConfigCode:777:<id>
	mockStore.On("Write", mock.MatchedBy(func(batch *db.WriteBatch) bool {
		// For now, a simple count check.
		// A more thorough test would capture the batch and verify its contents.
		// Example: Iterate batch.Ops, check for specific key patterns.
		// For this test, ensuring the unique checks above are for "Tag" (not "MetaForEmbed.Tag")
		// and that the code doesn't error out is the primary goal given the complexity of full batch verification here.
		return batch.Count() > 0
	}), mock.Anything).Return(nil).Once()

	createdID, err := repo.Create(&user, time.Now())
	require.NoError(t, err)
	assert.Equal(t, entityID, createdID)

	mockStore.AssertExpectations(t)
}
func TestRepository_Create_Success_Deterministic_Generator(t *testing.T) {
	mockStore := new(MockKVStore)

	user := User{
		ID:   "det-123",
		Name: "Alice",
	}

	iGF := &db.DeterministicIDGeneratorFactory{}

	repo, err := db.NewRepository[User](mockStore, "cf1", testColumnFamilySector, "admin", iGF)
	assert.NoError(t, err)

	data, _ := json.Marshal(user)
	dataKey := "admin:users:data:det-123"
	nameFieldKey := "admin:users:idx:Name:Alice:det-123"
	uNameFieldKey := "admin:users:idx-u:Name:Alice"
	indexKey := "admin:users:idx:ID:det-123:det-123"

	mockStore.On("Exists", "cf1", testColumnFamilySector, uNameFieldKey, mock.Anything).Return(false, nil)
	mockStore.On("Exists", "cf1", testColumnFamilySector, dataKey, mock.Anything).Return(false, nil)
	now := time.Now()
	batch := db.NewWriteBatch()
	batch.Put("cf1", testColumnFamilySector, indexKey, []byte("det-123"), now)
	batch.Put("cf1", testColumnFamilySector, nameFieldKey, []byte("det-123"), now)
	batch.Put("cf1", testColumnFamilySector, uNameFieldKey, []byte("det-123"), now)
	batch.Put("cf1", testColumnFamilySector, dataKey, data, now)

	mockStore.On("Write", mock.MatchedBy(func(b *db.WriteBatch) bool {
		return true
	}), mock.Anything).Return(nil)

	id, err := repo.Create(&user, time.Now())
	assert.NoError(t, err)
	assert.Equal(t, id, "det-123")

	mockStore.AssertExpectations(t)
}

func TestNewRepositoryUniqueCompoundValidations(t *testing.T) {
	mockStore := new(MockKVStore) // Does not hit DB for these validations
	iGF := NewTestIDGeneratorFactory([]string{})

	t.Run("Valid Exchange with compound uniqueness", func(t *testing.T) {
		repo, err := db.NewRepository[Exchange](mockStore, "cf_valid", testColumnFamilySector, "test_valid", iGF)
		require.NoError(t, err)

		// Verify that the compound groups are properly configured
		expectedGroups := map[int][]string{
			0: {"Name", "VNamespace"}, // Should be sorted
		}
		for idx, expectedFields := range expectedGroups {
			actualFields, exists := repo.GetTableDefinition().UniqueCompoundGroups[idx]
			require.True(t, exists, "Compound group %d should exist", idx)
			assert.Equal(t, expectedFields, actualFields, "Compound group %d fields mismatch", idx)
		}
	})

	t.Run("Valid UserWithCompoundUnique with multiple compound constraints", func(t *testing.T) {
		repo, err := db.NewRepository[UserWithCompoundUnique](mockStore, "cf_valid", testColumnFamilySector, "test_valid", iGF)
		require.NoError(t, err)

		// Verify that the compound groups are properly configured
		expectedGroups := map[int][]string{
			0: {"Domain", "Email"},    // Should be sorted
			1: {"Tenant", "Username"}, // Should be sorted
		}
		for idx, expectedFields := range expectedGroups {
			actualFields, exists := repo.GetTableDefinition().UniqueCompoundGroups[idx]
			require.True(t, exists, "Compound group %d should exist", idx)
			assert.Equal(t, expectedFields, actualFields, "Compound group %d fields mismatch", idx)
		}
	})

	t.Run("Invalid compound constraint - single field", func(t *testing.T) {
		_, err := db.NewRepository[InvalidCompoundSingle](mockStore, "cf_invalid", testColumnFamilySector, "test_invalid", iGF)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unique-compound:0 must have at least 2 fields, found only 1 field(s): [Name]")
	})
}

func TestRepository_CreateExchangeWithCompoundUniqueness_Success(t *testing.T) {
	mockStore := new(MockKVStore)
	iGF := NewTestIDGeneratorFactory([]string{"exchange-123"})

	repo, err := db.NewRepository[Exchange](mockStore, "cf1", testColumnFamilySector, "admin", iGF)
	require.NoError(t, err)

	exchange := &Exchange{
		ID:         "",
		Name:       "test-exchange",
		Type:       "direct",
		VNamespace: "namespace-1",
		CreatedAt:  "2023-01-01T00:00:00Z",
		UpdatedAt:  "2023-01-01T00:00:00Z",
	}

	now := time.Now()

	// Mock expectations for existence checks
	dataKey := "admin:exchanges:data:exchange-123"
	compoundIdxKey := "admin:exchanges:idx-uc:0:Name:test-exchange|VNamespace:namespace-1"

	// Check primary key doesn't exist
	mockStore.On("Exists", "cf1", testColumnFamilySector, dataKey, mock.Anything).Return(false, nil)
	// Check compound uniqueness doesn't exist
	mockStore.On("Exists", "cf1", testColumnFamilySector, compoundIdxKey, mock.Anything).Return(false, nil)

	// Mock the batch write
	mockStore.On("Write", mock.MatchedBy(func(b *db.WriteBatch) bool {
		return true
	}), mock.Anything).Return(nil)

	id, err := repo.Create(exchange, now)
	require.NoError(t, err)
	assert.Equal(t, "exchange-123", id)

	mockStore.AssertExpectations(t)
}

func TestRepository_CreateExchangeWithCompoundUniqueness_DuplicateCompound(t *testing.T) {
	mockStore := new(MockKVStore)
	iGF := NewTestIDGeneratorFactory([]string{"exchange-123"})

	repo, err := db.NewRepository[Exchange](mockStore, "cf1", testColumnFamilySector, "admin", iGF)
	require.NoError(t, err)

	exchange := &Exchange{
		ID:         "",
		Name:       "test-exchange",
		Type:       "direct",
		VNamespace: "namespace-1",
		CreatedAt:  "2023-01-01T00:00:00Z",
		UpdatedAt:  "2023-01-01T00:00:00Z",
	}

	now := time.Now()

	// Mock expectations - primary key doesn't exist but compound key does
	dataKey := "admin:exchanges:data:exchange-123"
	compoundIdxKey := "admin:exchanges:idx-uc:0:Name:test-exchange|VNamespace:namespace-1"

	mockStore.On("Exists", "cf1", testColumnFamilySector, dataKey, mock.Anything).Return(false, nil)
	mockStore.On("Exists", "cf1", testColumnFamilySector, compoundIdxKey, mock.Anything).Return(true, nil)

	// Create should fail due to duplicate compound unique constraint
	id, err := repo.Create(exchange, now)
	assert.Error(t, err)
	assert.Empty(t, id)
	assert.Contains(t, err.Error(), "duplicate unique compound constraint")

	mockStore.AssertExpectations(t)
}

func TestRepository_BulkCreateWithCompoundUniqueness_Success(t *testing.T) {
	mockStore := new(MockKVStore)
	iGF := NewTestIDGeneratorFactory([]string{"exchange-1", "exchange-2"})

	repo, err := db.NewRepository[Exchange](mockStore, "cf1", testColumnFamilySector, "admin", iGF)
	require.NoError(t, err)

	exchanges := []*Exchange{
		{
			Name:       "exchange-1",
			Type:       "direct",
			VNamespace: "namespace-1",
		},
		{
			Name:       "exchange-1", // Same name but different namespace - should be allowed
			Type:       "topic",
			VNamespace: "namespace-2",
		},
	}

	now := time.Now()

	// Mock existence checks for primary keys
	mockStore.On("Exists", "cf1", testColumnFamilySector, "admin:exchanges:data:exchange-1", mock.Anything).Return(false, nil)
	mockStore.On("Exists", "cf1", testColumnFamilySector, "admin:exchanges:data:exchange-2", mock.Anything).Return(false, nil)

	// Mock existence checks for compound constraints
	mockStore.On("Exists", "cf1", testColumnFamilySector, "admin:exchanges:idx-uc:0:Name:exchange-1|VNamespace:namespace-1", mock.Anything).Return(false, nil)
	mockStore.On("Exists", "cf1", testColumnFamilySector, "admin:exchanges:idx-uc:0:Name:exchange-1|VNamespace:namespace-2", mock.Anything).Return(false, nil)

	// Mock batch write
	mockStore.On("Write", mock.MatchedBy(func(b *db.WriteBatch) bool {
		return true
	}), mock.Anything).Return(nil)

	ids, err := repo.BulkCreate(exchanges, now)
	require.NoError(t, err)
	assert.Equal(t, []string{"exchange-1", "exchange-2"}, ids)

	mockStore.AssertExpectations(t)
}

func TestRepository_BulkCreateWithCompoundUniqueness_DuplicateInBatch(t *testing.T) {
	mockStore := new(MockKVStore)
	iGF := NewTestIDGeneratorFactory([]string{"exchange-1", "exchange-2"})

	repo, err := db.NewRepository[Exchange](mockStore, "cf1", testColumnFamilySector, "admin", iGF)
	require.NoError(t, err)

	exchanges := []*Exchange{
		{
			Name:       "exchange-1",
			Type:       "direct",
			VNamespace: "namespace-1",
		},
		{
			Name:       "exchange-1", // Same name and same namespace - should fail
			Type:       "topic",
			VNamespace: "namespace-1",
		},
	}

	now := time.Now()

	// Mock existence checks for primary keys
	mockStore.On("Exists", "cf1", testColumnFamilySector, "admin:exchanges:data:exchange-1", mock.Anything).Return(false, nil)
	mockStore.On("Exists", "cf1", testColumnFamilySector, "admin:exchanges:data:exchange-2", mock.Anything).Return(false, nil)

	_, err = repo.BulkCreate(exchanges, now)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate unique compound constraint in input batch")

	mockStore.AssertExpectations(t)
}

func TestRepository_Find_AutoDetectCompoundConstraint(t *testing.T) {
	mockStore := new(MockKVStore)
	iGF := NewTestIDGeneratorFactory([]string{})

	repo, err := db.NewRepository[Exchange](mockStore, "cf1", testColumnFamilySector, "admin", iGF)
	require.NoError(t, err)

	now := time.Now()

	// Test compound query using Find method with automatic detection
	filter := "Name='test-exchange' & VNamespace='namespace-1'"

	expectedExchange := &Exchange{
		ID:         "exchange-123",
		Name:       "test-exchange",
		Type:       "direct",
		VNamespace: "namespace-1",
		CreatedAt:  "2023-01-01T00:00:00Z",
		UpdatedAt:  "2023-01-01T00:00:00Z",
	}

	// Mock compound index lookup
	compoundIdxKey := "admin:exchanges:idx-uc:0:Name:test-exchange|VNamespace:namespace-1"
	mockStore.On("Get", "cf1", testColumnFamilySector, compoundIdxKey, mock.Anything).Return([]byte("exchange-123"), nil)

	// Mock data lookup
	dataKey := "admin:exchanges:data:exchange-123"
	exchangeData, _ := json.Marshal(expectedExchange)
	mockStore.On("Get", "cf1", testColumnFamilySector, dataKey, mock.Anything).Return(exchangeData, nil)

	result, err := repo.Find(filter, 10, "", now)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Entities, 1)
	assert.Equal(t, expectedExchange.ID, result.Entities[0].ID)
	assert.Equal(t, expectedExchange.Name, result.Entities[0].Name)
	assert.Equal(t, expectedExchange.VNamespace, result.Entities[0].VNamespace)

	mockStore.AssertExpectations(t)
}

func TestRepository_Find_AutoDetectCompoundConstraint_NotFound(t *testing.T) {
	mockStore := new(MockKVStore)
	iGF := NewTestIDGeneratorFactory([]string{})

	repo, err := db.NewRepository[Exchange](mockStore, "cf1", testColumnFamilySector, "admin", iGF)
	require.NoError(t, err)

	now := time.Now()

	// Test compound query that returns no results
	filter := "Name='nonexistent' & VNamespace='namespace-1'"

	// Mock compound index lookup returning nil (not found)
	compoundIdxKey := "admin:exchanges:idx-uc:0:Name:nonexistent|VNamespace:namespace-1"
	mockStore.On("Get", "cf1", testColumnFamilySector, compoundIdxKey, mock.Anything).Return(nil, nil)

	result, err := repo.Find(filter, 10, "", now)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Entities, 0)

	mockStore.AssertExpectations(t)
}

func TestRepository_Find_FallbackToNormalQuery(t *testing.T) {
	mockStore := new(MockKVStore)
	iGF := NewTestIDGeneratorFactory([]string{})

	repo, err := db.NewRepository[Exchange](mockStore, "cf1", testColumnFamilySector, "admin", iGF)
	require.NoError(t, err)

	now := time.Now()

	// Test query that doesn't match compound constraint (should fall back to normal processing)
	filter := "Name='test-exchange' & Type='direct'"

	// Mock normal query processing
	mockStore.On("SearchByPatternPaginatedKV", "cf1", testColumnFamilySector, "admin:exchanges:idx:Name:test-exchange:*", "", 10, mock.Anything).
		Return([]db.KeyValuePair{{Value: []byte("exchange-123")}}, "", nil)
	mockStore.On("SearchByPatternPaginatedKV", "cf1", testColumnFamilySector, "admin:exchanges:idx:Type:direct:*", "", 10, mock.Anything).
		Return([]db.KeyValuePair{{Value: []byte("exchange-123")}}, "", nil)

	expectedExchange := &Exchange{
		ID:         "exchange-123",
		Name:       "test-exchange",
		Type:       "direct",
		VNamespace: "namespace-1",
		CreatedAt:  "2023-01-01T00:00:00Z",
		UpdatedAt:  "2023-01-01T00:00:00Z",
	}

	dataKey := "admin:exchanges:data:exchange-123"
	exchangeData, _ := json.Marshal(expectedExchange)
	mockStore.On("Get", "cf1", testColumnFamilySector, dataKey, mock.Anything).Return(exchangeData, nil)

	result, err := repo.Find(filter, 10, "", now)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Entities, 1)
	assert.Equal(t, expectedExchange.ID, result.Entities[0].ID)

	mockStore.AssertExpectations(t)
}

// MARK: Data-Only Field Tests

func TestNewRepositoryDataOnlyValidations(t *testing.T) {
	mockStore := new(MockKVStore)
	iGF := NewTestIDGeneratorFactory([]string{})

	t.Run("Valid DataOnlyEntity", func(t *testing.T) {
		repo, err := db.NewRepository[DataOnlyEntity](mockStore, "cf_data_only", testColumnFamilySector, "test_data_only", iGF)
		require.NoError(t, err)
		assert.NotNil(t, repo)
	})

	t.Run("InvalidDataOnlyUniqueEntity - data-only fields cannot be unique", func(t *testing.T) {
		_, err := db.NewRepository[InvalidDataOnlyUniqueEntity](mockStore, "cf_invalid_unique", testColumnFamilySector, "test_invalid_unique", iGF)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "field 'Config' cannot be both data-only and unique")
	})

	t.Run("InvalidDataOnlyCompoundEntity - data-only fields cannot be in compound constraints", func(t *testing.T) {
		_, err := db.NewRepository[InvalidDataOnlyCompoundEntity](mockStore, "cf_invalid_compound", testColumnFamilySector, "test_invalid_compound", iGF)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "field 'Config' cannot be both data-only and part of compound uniqueness")
	})
}

func TestDataOnlyFields_CreateAndRead(t *testing.T) {
	mockStore := new(MockKVStore)
	iGF := NewTestIDGeneratorFactory([]string{"entity-123"})

	repo, err := db.NewRepository[DataOnlyEntity](mockStore, "cf_data_only", testColumnFamilySector, "test_data_only", iGF)
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

	// Mock expectations for create operation
	// Should create indices for Name and SearchField but NOT for data-only fields
	mockStore.On("Exists", "cf_data_only", testColumnFamilySector, "test_data_only:data_only_entities:data:entity-123", mock.Anything).Return(false, nil)
	mockStore.On("Exists", "cf_data_only", testColumnFamilySector, "test_data_only:data_only_entities:idx-u:Name:test-entity", mock.Anything).Return(false, nil)

	// Mock Write for data and indices (should NOT include data-only fields in indices)
	mockStore.On("Write", mock.AnythingOfType("*db.WriteBatch"), mock.Anything).Return(nil)

	// Create entity
	id, err := repo.Create(entity, now)
	require.NoError(t, err)
	assert.Equal(t, "entity-123", id)
	assert.Equal(t, "entity-123", entity.ID)

	// Mock expectations for read operation
	entityData, _ := json.Marshal(entity)
	mockStore.On("Get", "cf_data_only", testColumnFamilySector, "test_data_only:data_only_entities:data:entity-123", mock.Anything).Return(entityData, nil)

	// Read entity back using FindByField with ID
	readEntity, err := repo.FindByField("ID", "entity-123", now)
	require.NoError(t, err)
	assert.Equal(t, entity.ID, readEntity.ID)
	assert.Equal(t, entity.Name, readEntity.Name)
	assert.Equal(t, entity.SearchField, readEntity.SearchField)
	assert.Equal(t, entity.Config, readEntity.Config)
	assert.Equal(t, entity.Metadata, readEntity.Metadata)
	assert.Equal(t, entity.Tags, readEntity.Tags)

	mockStore.AssertExpectations(t)
}

func TestDataOnlyFields_QueryRestrictions(t *testing.T) {
	mockStore := new(MockKVStore)
	iGF := NewTestIDGeneratorFactory([]string{})

	repo, err := db.NewRepository[DataOnlyEntity](mockStore, "cf_data_only", testColumnFamilySector, "test_data_only", iGF)
	require.NoError(t, err)

	now := time.Now()

	t.Run("Query on normal field should work", func(t *testing.T) {
		// Mock for searchable field query
		mockStore.On("SearchByPatternPaginatedKV", "cf_data_only", testColumnFamilySector, "test_data_only:data_only_entities:idx:SearchField:searchable-value:*", "", 10, mock.Anything).
			Return([]db.KeyValuePair{{Value: []byte("entity-123")}}, "", nil)

		entityData, _ := json.Marshal(&DataOnlyEntity{ID: "entity-123", SearchField: "searchable-value"})
		mockStore.On("Get", "cf_data_only", testColumnFamilySector, "test_data_only:data_only_entities:data:entity-123", mock.Anything).Return(entityData, nil)

		result, err := repo.Find("SearchField='searchable-value'", 10, "", now)
		require.NoError(t, err)
		assert.Len(t, result.Entities, 1)
	})

	t.Run("Query on data-only field should fail", func(t *testing.T) {
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

	mockStore.AssertExpectations(t)
}

func TestDataOnlyFields_UpdatePreservesData(t *testing.T) {
	mockStore := new(MockKVStore)
	iGF := NewTestIDGeneratorFactory([]string{})

	repo, err := db.NewRepository[DataOnlyEntity](mockStore, "cf_data_only", testColumnFamilySector, "test_data_only", iGF)
	require.NoError(t, err)

	now := time.Now()
	originalEntity := &DataOnlyEntity{
		ID:          "entity-123",
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

	updatedEntity := &DataOnlyEntity{
		ID:          "entity-123",
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

	// Mock expectations for update operation
	originalData, _ := json.Marshal(originalEntity)
	mockStore.On("Get", "cf_data_only", testColumnFamilySector, "test_data_only:data_only_entities:data:entity-123", mock.Anything).Return(originalData, nil)

	// Mock unique constraint check for updated name (index existence check)
	mockStore.On("Get", "cf_data_only", testColumnFamilySector, "test_data_only:data_only_entities:idx-u:Name:updated-name", mock.Anything).Return(nil, nil)

	// Mock Write for update (should update indices for searchable fields but not data-only fields)
	mockStore.On("Write", mock.AnythingOfType("*db.WriteBatch"), mock.Anything).Return(nil)

	// Update entity
	updated, err := repo.Update(updatedEntity, now)
	require.NoError(t, err)
	assert.True(t, updated)

	mockStore.AssertExpectations(t)
}
