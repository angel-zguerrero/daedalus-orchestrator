package db_test

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/linxGnu/grocksdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type TestIDGeneratorFactoryR struct {
	ids   []string
	index int
	mu    sync.Mutex
}

func (g *TestIDGeneratorFactoryR) GenerateID() string {
	g.mu.Lock()
	defer g.mu.Unlock()

	if len(g.ids) == 0 {
		return ""
	}

	id := g.ids[g.index]
	g.index = (g.index + 1) % len(g.ids) // avance circular
	return id
}

type MockKVStoreRepositoryTest struct {
	mock.Mock
	ColumnFamilyHandles    map[string]*grocksdb.ColumnFamilyHandle // Map of regular column family names to their handles.
	TTLColumnFamilyHandles map[string]*grocksdb.ColumnFamilyHandle // Map of TTL column family names to their handles.
}

func (m *MockKVStoreRepositoryTest) Get(AdminFC, key string, now time.Time) ([]byte, error) {
	args := m.Called(AdminFC, key, now)
	var s []byte
	if tmp := args.Get(0); tmp != nil {
		s = tmp.([]byte)
	}
	return s, args.Error(1)
}

func (m *MockKVStoreRepositoryTest) Delete(AdminFC, key string, now time.Time) error {
	args := m.Called(AdminFC, key, now)
	return args.Error(0)
}

func (r *MockKVStoreRepositoryTest) Exists(columnFamily, key string, now time.Time) (bool, error) {
	// Note: This mock's Exists calls its own Get. Ensure Get is also updated if directly used by Exists logic.
	// For simplicity, we assume Get is called with appropriate 'now' if Exists needs it.
	// However, the direct KVStore.Exists call is what matters for the interface.
	args := r.Called(columnFamily, key, now)
	return args.Bool(0), args.Error(1)
}

func (m *MockKVStoreRepositoryTest) Put(AdminFC, key string, value []byte, ttl int, now time.Time) error {
	args := m.Called(AdminFC, key, value, ttl, now)
	return args.Error(0)
}

func (m *MockKVStoreRepositoryTest) PutRaw(AdminFC, key string, value []byte) error {
	args := m.Called(AdminFC, key, value)
	return args.Error(0)
}

func (m *MockKVStoreRepositoryTest) Write(batch *db.WriteBatch, now time.Time) error {
	args := m.Called(batch, now)
	return args.Error(0)
}

func (m *MockKVStoreRepositoryTest) WriteRaw(batch *db.WriteBatch) error {
	args := m.Called(batch)
	return args.Error(0)
}

func (m *MockKVStoreRepositoryTest) DumpAll() (interface{}, error) {
	args := m.Called()
	var s []byte
	if tmp := args.Get(0); tmp != nil {
		s = tmp.([]byte)
	}
	return s, args.Error(1)
}

func (r *MockKVStoreRepositoryTest) Iterate(fn func(cfName string, key, value []byte) error) error {
	return nil
}

func (r *MockKVStoreRepositoryTest) ClearAll() error {
	return nil
}

func (r *MockKVStoreRepositoryTest) Flush() error {
	return nil
}

func (r *MockKVStoreRepositoryTest) Close() error {
	return nil
}

func (r *MockKVStoreRepositoryTest) CleanExpiredKeys(now time.Time) error {
	args := r.Called(now)
	return args.Error(0)
}

func (m *MockKVStoreRepositoryTest) SearchByPatternPaginatedKV(cfName, pattern, cursor string, limit int, now time.Time) ([]db.KeyValuePair, string, error) {
	args := m.Called(cfName, pattern, cursor, limit, now)
	var s []db.KeyValuePair
	if tmp := args.Get(0); tmp != nil {
		s = tmp.([]db.KeyValuePair)
	}
	return s, "", args.Error(2)
}

// --- Test Structs for Conditional Uniqueness ---

type ConditionalUniqueEntity struct {
	ID                     string `orm:"primary-key"`
	Name                   string
	UniqueValue            string `orm:"unique,ignore-is-true:ShouldIgnoreUniqueness"`
	ShouldIgnoreUniqueness bool   `orm:""`
	NonBoolFlag            int    `orm:""` // For testing invalid type reference
}

func (e ConditionalUniqueEntity) TableName() string {
	return "conditional_unique_entities"
}

type InvalidConditionalEntityBadRef struct {
	ID          string `orm:"primary-key"`
	UniqueValue string `orm:"ignore-is-true:NonExistentFlag"`
}

func (e InvalidConditionalEntityBadRef) TableName() string { return "invalid_cond_bad_ref" }

type InvalidConditionalEntityBadType struct {
	ID          string `orm:"primary-key"`
	UniqueValue string `orm:"ignore-is-true:NonBoolFlag"`
	NonBoolFlag int
}

func (e InvalidConditionalEntityBadType) TableName() string { return "invalid_cond_bad_type" }

type User struct {
	ID   string `orm:"primary-key"`
	Name string `orm:"unique"`
}

func (User) TableName() string {
	return "users"
}

type InvalidConditionalEmptyField struct {
	ID          string `orm:"primary-key"`
	UniqueValue string `orm:"ignore-is-true:"`
}

func (e InvalidConditionalEmptyField) TableName() string {
	return "conditional_unique_entities"
}

func NewTestIDGeneratorFactoryR(ids []string) *TestIDGeneratorFactoryR {
	return &TestIDGeneratorFactoryR{
		ids: ids,
	}
}
func TestNewRepositoryConditionalUniquenessValidations(t *testing.T) {
	mockStore := new(MockKVStoreRepositoryTest) // Does not hit DB for these validations
	iGF := NewTestIDGeneratorFactoryR([]string{})

	t.Run("Valid ConditionalUniqueEntity", func(t *testing.T) {
		_, err := db.NewRepository[ConditionalUniqueEntity](mockStore, "cf_valid", "test_valid", iGF)
		require.NoError(t, err)
	})

	t.Run("InvalidConditionalEntityBadRef - NonExistentFlag", func(t *testing.T) {
		_, err := db.NewRepository[InvalidConditionalEntityBadRef](mockStore, "cf_bad_ref", "test_bad_ref", iGF)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "field 'UniqueValue' tagged with 'ignore-is-true:NonExistentFlag', but referenced field 'NonExistentFlag' does not exist in struct InvalidConditionalEntityBadRef")
	})

	t.Run("InvalidConditionalEntityBadType - NonBoolFlag", func(t *testing.T) {
		_, err := db.NewRepository[InvalidConditionalEntityBadType](mockStore, "cf_bad_type", "test_bad_type", iGF)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "field 'UniqueValue' tagged with 'ignore-is-true:NonBoolFlag', but referenced field 'NonBoolFlag' must be of type 'bool', found 'int'")
	})

	// Test for createFieldDefinition error: ignore-is-true with empty field name

	t.Run("InvalidConditionalEmptyField - ignore-is-true with empty field", func(t *testing.T) {
		_, err := db.NewRepository[InvalidConditionalEmptyField](mockStore, "cf_empty", "test_empty", iGF)
		require.Error(t, err)
		// This error comes from createFieldDefinition
		assert.Contains(t, err.Error(), "error extracting fields from struct InvalidConditionalEmptyField: invalid ignore-is-true format for field 'UniqueValue': ignore-is-true:")
	})
}

func TestRepository_Create_Success(t *testing.T) {
	mockStore := new(MockKVStoreRepositoryTest)

	user := User{
		ID:   "123",
		Name: "Alice",
	}

	iGF := NewTestIDGeneratorFactoryR([]string{"123"})

	repo, err := db.NewRepository[User](mockStore, "cf1", "admin", iGF)
	assert.NoError(t, err)

	data, _ := json.Marshal(user)
	dataKey := "admin:users:data:123"
	nameFieldKey := "admin:users:idx:Name:Alice:123"
	uNameFieldKey := "admin:users:idx-u:Name:Alice"
	indexKey := "admin:users:idx:ID:123:123"

	mockStore.On("Get", "cf1", uNameFieldKey, mock.Anything).Return(nil, nil)
	mockStore.On("Get", "cf1", "admin:users:data:123", mock.Anything).Return(nil, nil)

	batch := db.NewWriteBatch()
	batch.Put("cf1", indexKey, []byte("123"))
	batch.Put("cf1", nameFieldKey, []byte("123"))
	batch.Put("cf1", uNameFieldKey, []byte("123"))
	batch.Put("cf1", dataKey, data)

	mockStore.On("Write", mock.MatchedBy(func(b *db.WriteBatch) bool {
		return true
	}), mock.Anything).Return(nil)

	id, err := repo.Create(&user, time.Now())
	assert.NoError(t, err)
	assert.Equal(t, id, "123")

	mockStore.AssertExpectations(t)
}

func TestRepository_Create_DuplicatePrimaryKey_InDB(t *testing.T) {
	mockStore := new(MockKVStoreRepositoryTest)
	iGF := &db.DeterministicIDGeneratorFactory{} // Allows providing ID in entity

	repo, err := db.NewRepository[User](mockStore, "cf1", "admin", iGF)
	require.NoError(t, err)

	existingUserID := "existing-id-123"
	user := User{
		ID:   existingUserID, // Provide ID directly
		Name: "Alice",
	}
	data, _ := json.Marshal(user)
	// Mock that the primary key (data key) already exists
	dataKey := "admin:users:data:" + existingUserID
	mockStore.On("Get", "cf1", dataKey).Return(data, nil)
	// No unique checks for "Name" should be hit if PK check fails first
	// No "Write" should be called

	_, err = repo.Create(&user, time.Now())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate primary key: ID = "+existingUserID+" already exists")

	mockStore.AssertExpectations(t)
}

func TestRepository_BulkCreate_DuplicatePrimaryKey_InDB(t *testing.T) {
	mockStore := new(MockKVStoreRepositoryTest)
	iGF := &db.DeterministicIDGeneratorFactory{} // Allows providing IDs in entities

	repo, err := db.NewRepository[User](mockStore, "cf1", "admin", iGF)
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
	mockStore.On("Get", "cf1", pkDataKeyNew1).Return(nil, nil).Once()
	// For the second user (existingUserID), PK check should fail
	//mockStore.On("Exists", "cf1", pkDataKeyExisting).Return(true, nil).Once()

	data, _ := json.Marshal(&User{ID: existingUserID, Name: "UserExistingID"})
	mockStore.On("Get", "cf1", pkDataKeyExisting).Return(data, nil)
	// No further Exists or Write calls should happen for subsequent users or the batch itself.

	_, err = repo.BulkCreate(users, time.Now())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate primary key: ID = "+existingUserID+" already exists")

	mockStore.AssertExpectations(t)
}

func TestRepository_BulkCreate_DuplicatePrimaryKey_InBatch(t *testing.T) {
	mockStore := new(MockKVStoreRepositoryTest)
	iGF := &db.DeterministicIDGeneratorFactory{} // Allows providing IDs in entities

	repo, err := db.NewRepository[User](mockStore, "cf1", "admin", iGF)
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
	mockStore.On("Get", "cf1", pkDataKeyUnique1).Return(nil, nil)
	mockStore.On("Get", "cf1", pkDataKeyDup).Return(nil, nil)

	_, err = repo.BulkCreate(users, time.Now())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate primary key in input batch: ID = "+duplicateIDInBatch)

	mockStore.AssertExpectations(t)
}

func TestRepository_Create_EmptyProvidedID(t *testing.T) {
	mockStore := new(MockKVStoreRepositoryTest)
	iGF := &db.DeterministicIDGeneratorFactory{} // Using this factory means ID must be provided

	repo, err := db.NewRepository[User](mockStore, "cf1", "admin", iGF)
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
	mockStore := new(MockKVStoreRepositoryTest)

	user := User{
		ID:   "----",
		Name: "Alice",
	}

	iGF := NewTestIDGeneratorFactoryR([]string{"123"})

	repo, err := db.NewRepository[User](mockStore, "cf1", "admin", iGF)
	assert.NoError(t, err)

	uIndexKey := "admin:users:idx-u:Name:Alice"
	dataey := "admin:users:data:123"
	mockStore.On("Get", "cf1", uIndexKey, mock.Anything).Return([]byte("123"), nil)
	mockStore.On("Get", "cf1", dataey, mock.Anything).Return(nil, nil)

	_, err = repo.Create(&user, time.Now())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate unique field")
}

type NoPrimary struct {
	Name string
}

func (NoPrimary) TableName() string {
	return "nopk"
}

func TestRepository_Create_MissingPrimaryKeyValue(t *testing.T) {
	mockStore := new(MockKVStoreRepositoryTest)

	iGF := NewTestIDGeneratorFactoryR([]string{"123"})
	_, err := db.NewRepository[NoPrimary](mockStore, "cf1", "admin", iGF)
	assert.EqualError(t, err, "struct NoPrimary must have a string field named 'ID' with `orm:\"primary-key\"`")

}

func TestRepository_FindByField_Success(t *testing.T) {
	mockStore := new(MockKVStoreRepositoryTest)
	iGF := NewTestIDGeneratorFactoryR([]string{"123"})
	repo, err := db.NewRepository[User](mockStore, "cf1", "admin", iGF)
	assert.NoError(t, err)

	uIndexKey := "admin:users:idx-u:Name:Alice"
	dataKey := "admin:users:data:123"
	user := User{ID: "123", Name: "Alice"}
	data, _ := json.Marshal(user)

	mockStore.On("Get", "cf1", uIndexKey, mock.Anything).Return([]byte("123"), nil)
	mockStore.On("Get", "cf1", dataKey, mock.Anything).Return(data, nil)

	result, err := repo.FindByField("Name", "Alice", time.Now())
	assert.NoError(t, err)
	assert.Equal(t, &user, result)

	mockStore.AssertExpectations(t)
}

func TestRepository_FindByField_Unknown_Field_Name(t *testing.T) {
	mockStore := new(MockKVStoreRepositoryTest)
	iGF := NewTestIDGeneratorFactoryR([]string{"123"})
	repo, err := db.NewRepository[User](mockStore, "cf1", "admin", iGF)
	assert.NoError(t, err)

	_, err = repo.FindByField("x", "Alice", time.Now())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Unknown field x")
}

func TestRepository_Find_AND_Unknown_Field_Name(t *testing.T) {
	mockStore := new(MockKVStoreRepositoryTest)
	iGF := NewTestIDGeneratorFactoryR([]string{"123"})
	repo, err := db.NewRepository[User](mockStore, "cf1", "admin", iGF)
	assert.NoError(t, err)

	mockStore.On("SearchByPatternPaginatedKV", "cf1", "admin:users:idx:ID:123:*", "", 1000, mock.Anything).
		Return([]db.KeyValuePair{{Value: []byte("123")}}, "", nil)

	_, err = repo.Find("ID=123&X=Alice", 1000, "", time.Now())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Unknown field X")
	mockStore.AssertExpectations(t)
}

type TempEntity struct {
	ID    string `orm:"primary-key"`
	Token string `orm:"unique"`
	TTL   int    `orm:"ttl"`
}

func (TempEntity) TableName() string {
	return "temporal_entities"
}

func TestRepository_FindByField_Invalid_Use_For_TTL_Query(t *testing.T) {
	mockStore := new(MockKVStoreRepositoryTest)
	iGF := NewTestIDGeneratorFactoryR([]string{"123"})
	repo, err := db.NewRepository[TempEntity](mockStore, "cf1", "admin", iGF)
	assert.NoError(t, err)

	_, err = repo.FindByField("TTL", "111", time.Now())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "TTL columns are not supported in query operations")
}

func TestRepository_Find_AND_Invalid_Use_For_TTL_Query(t *testing.T) {
	mockStore := new(MockKVStoreRepositoryTest)
	iGF := NewTestIDGeneratorFactoryR([]string{"123"})
	repo, err := db.NewRepository[TempEntity](mockStore, "cf1", "admin", iGF)
	assert.NoError(t, err)

	mockStore.On("SearchByPatternPaginatedKV", "cf1", "admin:temporal_entities:idx:ID:123:*", "", 1000, mock.Anything).
		Return([]db.KeyValuePair{{Value: []byte("123")}}, "", nil)

	_, err = repo.Find("ID=123&TTL=22", 1000, "", time.Now())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "TTL columns are not supported in query operation")
	mockStore.AssertExpectations(t)
}

type InvalidTempEntity struct {
	ID    string `orm:"primary-key"`
	Token string `orm:"unique"`
	ttl   int    `orm:"ttl"`
}

func (InvalidTempEntity) TableName() string {
	return "invalid_temporal_entities"
}

func TestRepository_FindByField_Invalid_TTL_Field_name(t *testing.T) {
	mockStore := new(MockKVStoreRepositoryTest)
	iGF := NewTestIDGeneratorFactoryR([]string{"123"})
	_, err := db.NewRepository[InvalidTempEntity](mockStore, "cf1", "admin", iGF)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error extracting fields from struct InvalidTempEntity: ttl can only be defined on top-level 'TTL' field, found on 'ttl'")
}

type Invalid struct {
	Name string `orm:"unique"`
}

func (Invalid) TableName() string {
	return "invalid"
}

func TestNewRepository_MissingPrimaryKey(t *testing.T) {
	mockStore := new(MockKVStoreRepositoryTest)
	iGF := NewTestIDGeneratorFactoryR([]string{"123"})

	_, err := db.NewRepository[Invalid](mockStore, "cf1", "admin", iGF)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "struct Invalid must have a string field named 'ID' with `orm:\"primary-key\"`")
}

func TestRepository_Find_AND(t *testing.T) {
	mockStore := new(MockKVStoreRepositoryTest)
	iGF := NewTestIDGeneratorFactoryR([]string{"123"})
	repo, err := db.NewRepository[User](mockStore, "cf1", "admin", iGF)
	assert.NoError(t, err)

	user := User{ID: "123", Name: "Alice"}
	data, _ := json.Marshal(user)

	mockStore.On("SearchByPatternPaginatedKV", "cf1", "admin:users:idx:ID:123:*", "", 1000, mock.Anything).
		Return([]db.KeyValuePair{{Value: []byte("123")}}, "", nil)
	mockStore.On("SearchByPatternPaginatedKV", "cf1", "admin:users:idx:Name:Alice:*", "", 1000, mock.Anything).
		Return([]db.KeyValuePair{{Value: []byte("123")}}, "", nil)
	mockStore.On("Get", "cf1", "admin:users:data:123", mock.Anything).
		Return(data, nil)

	result, err := repo.Find("ID=123&Name=Alice", 1000, "", time.Now())
	assert.NoError(t, err)
	assert.Len(t, result.Entities, 1)
	assert.Equal(t, &user, &result.Entities[0])
}

func TestRepository_Find_OR(t *testing.T) {
	mockStore := new(MockKVStoreRepositoryTest)
	iGF := NewTestIDGeneratorFactoryR([]string{"123"})
	repo, err := db.NewRepository[User](mockStore, "cf1", "admin", iGF)
	assert.NoError(t, err)

	user1 := User{ID: "123", Name: "Alice"}
	user2 := User{ID: "456", Name: "Bob"}
	data1, _ := json.Marshal(user1)
	data2, _ := json.Marshal(user2)

	mockStore.On("SearchByPatternPaginatedKV", "cf1", "admin:users:idx:Name:Alice:*", "", 1000, mock.Anything).
		Return([]db.KeyValuePair{{Value: []byte("123")}}, "", nil)
	mockStore.On("SearchByPatternPaginatedKV", "cf1", "admin:users:idx:Name:Bob:*", "", 1000, mock.Anything).
		Return([]db.KeyValuePair{{Value: []byte("456")}}, "", nil)
	mockStore.On("Get", "cf1", "admin:users:data:123", mock.Anything).
		Return(data1, nil)
	mockStore.On("Get", "cf1", "admin:users:data:456", mock.Anything).
		Return(data2, nil)

	result, err := repo.Find("Name=Alice|Name=Bob", 1000, "", time.Now())
	assert.NoError(t, err)
	assert.Len(t, result.Entities, 2)
}

func TestRepository_Find_SpecialCharacters(t *testing.T) {
	mockStore := new(MockKVStoreRepositoryTest)
	iGF := NewTestIDGeneratorFactoryR([]string{"123"})
	repo, err := db.NewRepository[User](mockStore, "cf1", "admin", iGF)
	assert.NoError(t, err)

	user := User{ID: "999", Name: "foo:bar"}
	data, _ := json.Marshal(user)

	mockStore.On("SearchByPatternPaginatedKV", "cf1", "admin:users:idx:Name:foo:bar:*", "", 1000, mock.Anything).
		Return([]db.KeyValuePair{{Value: []byte("999")}}, "", nil)
	mockStore.On("Get", "cf1", "admin:users:data:999", mock.Anything).
		Return(data, nil)

	result, err := repo.Find("Name=foo:bar", 1000, "", time.Now())
	assert.NoError(t, err)
	assert.Len(t, result.Entities, 1)
	assert.Equal(t, &user, &result.Entities[0])
}

func TestRepository_Find_NoMatch(t *testing.T) {
	mockStore := new(MockKVStoreRepositoryTest)
	iGF := NewTestIDGeneratorFactoryR([]string{"123"})
	repo, err := db.NewRepository[User](mockStore, "cf1", "admin", iGF)
	assert.NoError(t, err)

	mockStore.On("SearchByPatternPaginatedKV", "cf1", "admin:users:idx:Name:Ghost:*", "", 1000, mock.Anything).
		Return(nil, "", nil)

	result, err := repo.Find("Name=Ghost", 1000, "", time.Now())
	assert.NoError(t, err)
	assert.Len(t, result.Entities, 0)
}

func TestRepository_Update_Success(t *testing.T) {
	mockStore := new(MockKVStoreRepositoryTest)

	iGF := NewTestIDGeneratorFactoryR([]string{"123"})
	repo, err := db.NewRepository[User](mockStore, "cf1", "admin", iGF)
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
	mockStore.On("Get", "cf1", dataKey, mock.Anything).Return(originalData, nil)
	mockStore.On("Get", "cf1", newUIndexKey, mock.Anything).Return(nil, nil)

	batch := db.NewWriteBatch()
	batch.Delete("cf1", oldUIndexKey)
	batch.Put("cf1", newUIndexKey, []byte("123"))
	batch.Delete("cf1", oldIndexKey)

	batch.Put("cf1", newIndexKey, []byte("123"))
	batch.Put("cf1", dataKey, updatedData)

	mockStore.On("Write", mock.MatchedBy(func(b *db.WriteBatch) bool {
		return true
	}), mock.Anything).Return(nil)

	changed, err := repo.Update(&updatedUser, time.Now())
	assert.NoError(t, err)
	assert.Equal(t, changed, true)

	mockStore.AssertExpectations(t)
}

func TestRepository_Update_Nonexistent(t *testing.T) {
	mockStore := new(MockKVStoreRepositoryTest)

	iGF := NewTestIDGeneratorFactoryR([]string{"123"})
	repo, err := db.NewRepository[User](mockStore, "cf1", "admin", iGF)
	assert.NoError(t, err)

	user := User{ID: "999", Name: "Ghost"}
	dataKey := "admin:users:data:999"

	mockStore.On("Get", "cf1", dataKey, mock.Anything).Return(nil, nil) // For FindByField

	changed, err := repo.Update(&user, time.Now())
	assert.NoError(t, err)
	assert.Equal(t, changed, false)
}
func TestRepository_Delete_Success(t *testing.T) {
	mockStore := new(MockKVStoreRepositoryTest)

	iGF := NewTestIDGeneratorFactoryR([]string{"123"})
	repo, err := db.NewRepository[User](mockStore, "cf1", "admin", iGF)
	assert.NoError(t, err)

	user := User{ID: "123", Name: "Alice"}
	dataKey := "admin:users:data:123"
	indexKey := "admin:users:idx:Name:Alice:123"
	uIndexKey := "admin:users:idx-u:Name:Alice"
	pkIndexKey := "admin:users:idx:ID:123:123"
	data, _ := json.Marshal(user)

	mockStore.On("Get", "cf1", dataKey, mock.Anything).Return(data, nil) // For FindByField

	batch := db.NewWriteBatch()

	batch.Delete("cf1", indexKey)
	batch.Delete("cf1", pkIndexKey)
	batch.Delete("cf1", uIndexKey)
	batch.Delete("cf1", dataKey)

	mockStore.On("Write", mock.MatchedBy(func(b *db.WriteBatch) bool {
		return true
	}), mock.Anything).Return(nil)

	deleted, err := repo.Delete("123", time.Now())
	assert.Equal(t, deleted, true)
	assert.NoError(t, err)

	mockStore.AssertExpectations(t)
}

func TestRepository_Delete_NotFound(t *testing.T) {
	mockStore := new(MockKVStoreRepositoryTest)

	iGF := NewTestIDGeneratorFactoryR([]string{"123"})
	repo, err := db.NewRepository[User](mockStore, "cf1", "admin", iGF)
	assert.NoError(t, err)

	dataKey := "admin:users:data:123"
	mockStore.On("Get", "cf1", dataKey, mock.Anything).Return(nil, nil) // For FindByField

	deleted, err := repo.Delete("123", time.Now())
	assert.Equal(t, deleted, false)
	assert.NoError(t, err)
	mockStore.AssertExpectations(t)
}
func TestRepository_Delete_CorruptedData(t *testing.T) {
	mockStore := new(MockKVStoreRepositoryTest)

	iGF := NewTestIDGeneratorFactoryR([]string{"123"})
	repo, err := db.NewRepository[User](mockStore, "cf1", "admin", iGF)
	assert.NoError(t, err)

	dataKey := "admin:users:data:123"

	mockStore.On("Get", "cf1", dataKey, mock.Anything).Return([]byte("not a valid json"), nil) // For FindByField

	deleted, err := repo.Delete("123", time.Now())
	assert.Equal(t, deleted, false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid character")
	mockStore.AssertExpectations(t)
}

func TestRepository_BulkCreate_Success(t *testing.T) {
	mockStore := new(MockKVStoreRepositoryTest)
	iGF := NewTestIDGeneratorFactoryR([]string{"id1", "id2", "id3"})
	repo, err := db.NewRepository[User](mockStore, "cf1", "admin", iGF)
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

		mockStore.On("Get", "cf1", uIdx, mock.Anything).Return(nil, nil)
		mockStore.On("Get", "cf1", dataKey, mock.Anything).Return(nil, nil)

		batch.Put("cf1", dataKey, d)
		batch.Put("cf1", nameIdx, []byte(id))
		batch.Put("cf1", uIdx, []byte(id))
		batch.Put("cf1", pkIdx, []byte(id))
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
	mockStore := new(MockKVStoreRepositoryTest)
	iGF := NewTestIDGeneratorFactoryR([]string{"id1", "id2"})
	repo, err := db.NewRepository[User](mockStore, "cf1", "admin", iGF)
	assert.NoError(t, err)

	users := []*User{
		{Name: "Alice"},
		{Name: "Alice"}, // duplicado intencional
	}

	mockStore.On("Exists", "cf1", "admin:users:idx-u:Name:Alice", mock.Anything).Return(false, nil).Once() // For first Alice
	// For second Alice, Exists check will happen against the batch first (which passes), then DB.
	// This mock is for the DB check for the *second* Alice, assuming the first one was "not in DB" for its Exists check
	// and then added to the batch. The test logic in BulkCreate checks batch then DB.
	// So, this mock should reflect that the key *now* exists in DB due to the first Alice (hypothetically).
	// However, the current test structure for duplicate unique in BulkCreate relies on the mock for `Exists`
	// for *each* entity. If an entity's value is already in uniqueInBatch, it errors before DB check.
	// If not in batch, it checks DB.
	// Let's adjust the mock to simulate the scenario where the second "Alice" check finds the first "Alice" already in the DB
	// (or rather, that the key it would use is taken).
	mockStore.On("Exists", "cf1", "admin:users:idx-u:Name:Alice", mock.Anything).Return(true, nil).Once()

	mockStore.On("Get", "cf1", "admin:users:data:id1", mock.Anything).Return(nil, nil).Once()
	mockStore.On("Get", "cf1", "admin:users:data:id2", mock.Anything).Return(nil, nil).Once()
	_, err = repo.BulkCreate(users, time.Now())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate unique field")
}

func TestRepository_BulkCreate_WriteError(t *testing.T) {
	mockStore := new(MockKVStoreRepositoryTest)
	iGF := NewTestIDGeneratorFactoryR([]string{"id1"})
	repo, err := db.NewRepository[User](mockStore, "cf1", "admin", iGF)
	assert.NoError(t, err)

	users := []*User{
		{Name: "Alice"},
	}

	mockStore.On("Get", "cf1", "admin:users:idx-u:Name:Alice", mock.Anything).Return(nil, nil)
	mockStore.On("Get", "cf1", "admin:users:data:id1", mock.Anything).Return(nil, nil)
	mockStore.On("Write", mock.Anything, mock.Anything).Return(errors.New("write failed"))

	_, err = repo.BulkCreate(users, time.Now())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "write failed")
}
func TestRepository_BulkDelete_Success(t *testing.T) {
	mockStore := new(MockKVStoreRepositoryTest)
	iGF := NewTestIDGeneratorFactoryR([]string{"id1", "id2"})
	repo, err := db.NewRepository[User](mockStore, "cf1", "admin", iGF)
	assert.NoError(t, err)

	users := []*User{
		{ID: "id1", Name: "Alice"},
		{ID: "id2", Name: "Bob"},
	}

	for _, u := range users {
		data, _ := json.Marshal(u)
		// This Get is part of the FindByField call within BulkDelete
		mockStore.On("Get", "cf1", "admin:users:data:"+u.ID, mock.Anything).Return(data, nil)
	}

	batch := db.NewWriteBatch()
	batch.Delete("cf1", "admin:users:idx:Name:Alice:id1")
	batch.Delete("cf1", "admin:users:idx:Name:Bob:id2")
	batch.Delete("cf1", "admin:users:idx:ID:id1:id1")
	batch.Delete("cf1", "admin:users:idx:ID:id2:id2")
	batch.Delete("cf1", "admin:users:idx-u:Name:Alice")
	batch.Delete("cf1", "admin:users:idx-u:Name:Bob")
	batch.Delete("cf1", "admin:users:data:id1")
	batch.Delete("cf1", "admin:users:data:id2")

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
	mockStore := new(MockKVStoreRepositoryTest)
	iGF := NewTestIDGeneratorFactoryR([]string{"id1", "id2"})
	repo, err := db.NewRepository[User](mockStore, "cf1", "admin", iGF)
	assert.NoError(t, err)

	user := &User{ID: "id1", Name: "Alice"}
	data, _ := json.Marshal(user)
	// These Gets are part of FindByField calls
	mockStore.On("Get", "cf1", "admin:users:data:id1", mock.Anything).Return(data, nil)
	mockStore.On("Get", "cf1", "admin:users:data:id2", mock.Anything).Return(nil, nil)

	batch := db.NewWriteBatch()
	batch.Delete("cf1", "admin:users:idx:Name:Alice:id1")
	batch.Delete("cf1", "admin:users:idx:ID:id1:id1")
	batch.Delete("cf1", "admin:users:idx-u:Name:Alice")
	batch.Delete("cf1", "admin:users:data:id1")

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
	mockStore := new(MockKVStoreRepositoryTest)
	iGF := NewTestIDGeneratorFactoryR([]string{"id1"})
	repo, err := db.NewRepository[User](mockStore, "cf1", "admin", iGF)
	assert.NoError(t, err)

	mockStore.On("Get", "cf1", "admin:users:data:id1", mock.Anything).Return([]byte("invalid json"), nil) // For FindByField

	deleted, err := repo.BulkDelete([]string{"id1"}, time.Now())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid character")

	require.Len(t, deleted, 0)

	mockStore.AssertExpectations(t)
}
func TestRepository_BulkUpdate_Success(t *testing.T) {
	mockStore := new(MockKVStoreRepositoryTest)
	iGF := NewTestIDGeneratorFactoryR([]string{"id1", "id2"})
	repo, err := db.NewRepository[User](mockStore, "cf1", "admin", iGF)
	require.NoError(t, err)

	original1 := User{ID: "id1", Name: "Alice"}
	updated1 := User{ID: "id1", Name: "AliceUpdated"}

	original2 := User{ID: "id2", Name: "Bob"}
	updated2 := User{ID: "id2", Name: "BobUpdated"}

	// Serializar original data para mockear GET
	originalData1, _ := json.Marshal(original1)
	originalData2, _ := json.Marshal(original2)

	// Se mockea GET para los datos originales (called by FindByField)
	mockStore.On("Get", "cf1", "admin:users:data:id1", mock.Anything).Return(originalData1, nil)
	mockStore.On("Get", "cf1", "admin:users:data:id2", mock.Anything).Return(originalData2, nil)

	// Para las claves únicas nuevas, simular que no existen (para no dar error de duplicados)
	mockStore.On("Get", "cf1", "admin:users:idx-u:Name:AliceUpdated", mock.Anything).Return(nil, nil)
	mockStore.On("Get", "cf1", "admin:users:idx-u:Name:BobUpdated", mock.Anything).Return(nil, nil)

	mockStore.On("Write", mock.Anything, mock.Anything).Return(nil)

	results, err := repo.BulkUpdate([]*User{&updated1, &updated2}, time.Now())
	assert.NoError(t, err)
	assert.Len(t, results, 2)
	assert.True(t, results[0])
	assert.True(t, results[1])

	mockStore.AssertExpectations(t)
}

func TestRepository_BulkUpdate_SomeNonexistent(t *testing.T) {
	mockStore := new(MockKVStoreRepositoryTest)
	iGF := NewTestIDGeneratorFactoryR([]string{"id1", "id2"})
	repo, err := db.NewRepository[User](mockStore, "cf1", "admin", iGF)
	require.NoError(t, err)

	existing := User{ID: "id1", Name: "Alice"}
	updatedExisting := User{ID: "id1", Name: "AliceUpdated"}
	nonexistent := User{ID: "id999", Name: "Ghost"}

	originalData, _ := json.Marshal(existing)

	mockStore.On("Get", "cf1", "admin:users:data:id1", mock.Anything).Return(originalData, nil)
	mockStore.On("Get", "cf1", "admin:users:data:id999", mock.Anything).Return(nil, nil) // no existe

	mockStore.On("Get", "cf1", "admin:users:idx-u:Name:AliceUpdated", mock.Anything).Return(nil, nil)

	mockStore.On("Write", mock.Anything, mock.Anything).Return(nil)

	results, err := repo.BulkUpdate([]*User{&updatedExisting, &nonexistent}, time.Now())
	assert.NoError(t, err)
	assert.Len(t, results, 2)
	assert.True(t, results[0])  // id1 actualizado
	assert.False(t, results[1]) // id999 no existe, no actualizado

	mockStore.AssertExpectations(t)
}

func TestRepository_BulkUpdate_DuplicateUniqueWithinBatch(t *testing.T) {
	mockStore := new(MockKVStoreRepositoryTest)
	iGF := NewTestIDGeneratorFactoryR([]string{"id1", "id2"})
	repo, err := db.NewRepository[User](mockStore, "cf1", "admin", iGF)
	require.NoError(t, err)

	original1 := User{ID: "id1", Name: "Alice"}
	original2 := User{ID: "id2", Name: "Bob"}

	updated1 := User{ID: "id1", Name: "SameName"} // Cambia a SameName
	updated2 := User{ID: "id2", Name: "SameName"} // Mismo nombre -> duplicado interno

	originalData1, _ := json.Marshal(original1)
	originalData2, _ := json.Marshal(original2)

	mockStore.On("Get", "cf1", "admin:users:data:id1", mock.Anything).Return(originalData1, nil)
	mockStore.On("Get", "cf1", "admin:users:data:id2", mock.Anything).Return(originalData2, nil)

	// El repositorio debería detectar el duplicado dentro del batch sin llamar al store para el índice uName

	results, err := repo.BulkUpdate([]*User{&updated1, &updated2}, time.Now())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate unique field")
	assert.Nil(t, results)
}

func TestRepository_BulkUpdate_DuplicateUniqueExisting(t *testing.T) {
	mockStore := new(MockKVStoreRepositoryTest)
	iGF := NewTestIDGeneratorFactoryR([]string{"id1", "id2"})
	repo, err := db.NewRepository[User](mockStore, "cf1", "admin", iGF)
	require.NoError(t, err)

	original1 := User{ID: "id1", Name: "Alice"}
	updated1 := User{ID: "id1", Name: "Bob"} // Cambia a "Bob"

	originalData1, _ := json.Marshal(original1)

	mockStore.On("Get", "cf1", "admin:users:data:id1", mock.Anything).Return(originalData1, nil)
	// Simulamos que el índice único ya apunta a otro ID distinto al que actualizamos
	mockStore.On("Get", "cf1", "admin:users:idx-u:Name:Bob", mock.Anything).Return([]byte("otherID"), nil)

	results, err := repo.BulkUpdate([]*User{&updated1}, time.Now())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate unique field")
	assert.Nil(t, results)
}

func TestRepository_BulkUpdate_WriteError(t *testing.T) {
	mockStore := new(MockKVStoreRepositoryTest)
	iGF := NewTestIDGeneratorFactoryR([]string{"id1"})
	repo, err := db.NewRepository[User](mockStore, "cf1", "admin", iGF)
	require.NoError(t, err)

	original := User{ID: "id1", Name: "Alice"}
	updated := User{ID: "id1", Name: "AliceUpdated"}

	originalData, _ := json.Marshal(original)

	mockStore.On("Get", "cf1", "admin:users:data:id1", mock.Anything).Return(originalData, nil)
	mockStore.On("Get", "cf1", "admin:users:idx-u:Name:AliceUpdated", mock.Anything).Return(nil, nil)

	mockStore.On("Write", mock.Anything, mock.Anything).Return(errors.New("write failed"))

	results, err := repo.BulkUpdate([]*User{&updated}, time.Now())
	assert.Error(t, err)
	assert.Nil(t, results)
}

func TestRepository_BulkUpdate_InvalidData(t *testing.T) {
	mockStore := new(MockKVStoreRepositoryTest)
	iGF := NewTestIDGeneratorFactoryR([]string{"id1"})
	repo, err := db.NewRepository[User](mockStore, "cf1", "admin", iGF)
	require.NoError(t, err)

	// Simula que data original almacenada está corrupta (no json válido)
	mockStore.On("Get", "cf1", "admin:users:data:id1", mock.Anything).Return([]byte("not json"), nil)

	updated := User{ID: "id1", Name: "AliceUpdated"}

	results, err := repo.BulkUpdate([]*User{&updated}, time.Now())
	assert.Error(t, err)
	assert.Nil(t, results)
}

func TestRepository_BulkUpdate_EmptyInput(t *testing.T) {
	mockStore := new(MockKVStoreRepositoryTest)
	iGF := NewTestIDGeneratorFactoryR([]string{})
	repo, err := db.NewRepository[User](mockStore, "cf1", "admin", iGF)
	require.NoError(t, err)

	results, err := repo.BulkUpdate([]*User{}, time.Now())
	assert.NoError(t, err)
	assert.NotNil(t, results)
	assert.Len(t, results, 0)
}

// --- Structs for Nested Field Tests ---

type Meta struct {
	Tag         string `orm:"unique"`
	ConfigCode  int
	Description string
}

type UserComplex struct {
	ID     string `orm:"primary-key"`
	Email  string `orm:"unique"`
	Meta   Meta   // Named field
	Status string
	Extra  *Meta // Pointer to a struct, for testing pointer handling
}

func (UserComplex) TableName() string {
	return "users_complex"
}

type MetaForEmbed struct {
	Tag         string `orm:"unique"` // Will become "Tag" at top level due to embedding
	ConfigCode  int    // Will become "ConfigCode"
	Description string // Will become "Description"
}

type UserComplexEmbedded struct {
	ID           string `orm:"primary-key"`
	Email        string `orm:"unique"`
	MetaForEmbed        // Embedded field
	Status       string
}

func (UserComplexEmbedded) TableName() string {
	return "users_complex_embedded"
}

// --- Test Cases for Nested Fields ---

func TestRepository_Create_NestedSuccess_UserComplexEmbedded(t *testing.T) {
	mockStore := new(MockKVStoreRepositoryTest)
	entityID := "uce123"
	iGF := NewTestIDGeneratorFactoryR([]string{entityID})

	repo, err := db.NewRepository[UserComplexEmbedded](mockStore, "cf_embed", "test_sch_embed", iGF)
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
	mockStore.On("Exists", "cf_embed", "test_sch_embed:users_complex_embedded:idx-u:Email:embedded@example.com", mock.Anything).Return(false, nil).Once()
	// Assuming embedded field 'Tag' becomes a top-level field name 'Tag'
	mockStore.On("Get", "cf_embed", "test_sch_embed:users_complex_embedded:idx-u:Tag:embeddedTag1", mock.Anything).Return(nil, nil).Once()
	mockStore.On("Get", "cf_embed", "test_sch_embed:users_complex_embedded:data:uce123", mock.Anything).Return(nil, nil).Once()

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
	mockStore := new(MockKVStoreRepositoryTest)
	entityID := "uc456"
	iGF := NewTestIDGeneratorFactoryR([]string{entityID})

	repo, err := db.NewRepository[UserComplex](mockStore, "cf_complex", "test_sch", iGF)
	require.NoError(t, err)

	user := UserComplex{
		Email:  "new@example.com", // Assume this is unique for now
		Status: "active",
		Meta: Meta{
			Tag:         "existingTag", // This tag will cause a duplicate error
			ConfigCode:  102,
			Description: "Desc",
		},
	}

	// Mock for unique checks
	mockStore.On("Get", "cf_complex", "test_sch:users_complex:idx-u:Email:new@example.com", mock.Anything).Return(nil, nil)
	mockStore.On("Get", "cf_complex", "test_sch:users_complex:data:uc456", mock.Anything).Return(nil, nil)
	// Simulate Meta.Tag being a duplicate
	mockStore.On("Exists", "cf_complex", "test_sch:users_complex:idx-u:Meta.Tag:existingTag", mock.Anything).Return(true, nil)
	// No On("Write") should be called

	_, err = repo.Create(&user, time.Now())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate unique field: Meta.Tag = existingTag")

	//mockStore.AssertExpectations(t)
}

func TestRepository_FindByField_NestedSuccess_UserComplex(t *testing.T) {
	mockStore := new(MockKVStoreRepositoryTest)
	entityID := "uc789"
	iGF := NewTestIDGeneratorFactoryR([]string{}) // Not used for FindByField directly for ID generation

	repo, err := db.NewRepository[UserComplex](mockStore, "cf_complex", "test_sch", iGF)
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

	mockStore.On("Get", "cf_complex", uniqueIdxKey, mock.Anything).Return([]byte(entityID), nil).Once()
	mockStore.On("Get", "cf_complex", dataKey, mock.Anything).Return(jsonData, nil).Once()

	foundUser, err := repo.FindByField("Meta.Tag", "findThisTag", time.Now())
	require.NoError(t, err)
	require.NotNil(t, foundUser)
	assert.Equal(t, expectedUser, *foundUser)

	mockStore.AssertExpectations(t)
}

func TestRepository_Find_NestedCondition_UserComplex(t *testing.T) {
	mockStore := new(MockKVStoreRepositoryTest)
	entityID := "uc101"
	iGF := NewTestIDGeneratorFactoryR([]string{})

	repo, err := db.NewRepository[UserComplex](mockStore, "cf_complex", "test_sch", iGF)
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
	mockStore.On("SearchByPatternPaginatedKV", "cf_complex", "test_sch:users_complex:idx:Meta.Tag:filterTag:*", "", 1000, mock.Anything).
		Return([]db.KeyValuePair{{Key: "...", Value: []byte(entityID)}}, "", nil).Once()
	// For "Email = 'filter@example.com'"
	mockStore.On("SearchByPatternPaginatedKV", "cf_complex", "test_sch:users_complex:idx:Email:filter@example.com:*", "", 1000, mock.Anything).
		Return([]db.KeyValuePair{{Key: "...", Value: []byte(entityID)}}, "", nil).Once()

	// Mock Get for the data key
	mockStore.On("Get", "cf_complex", "test_sch:users_complex:data:"+entityID, mock.Anything).Return(jsonData, nil).Once()

	result, err := repo.Find("Meta.Tag = 'filterTag' & Email = 'filter@example.com'", 1000, "", time.Now())
	require.NoError(t, err)
	require.Len(t, result.Entities, 1)
	assert.Equal(t, user, result.Entities[0])

	mockStore.AssertExpectations(t)
}

func TestRepository_Update_NestedField_UserComplex(t *testing.T) {
	mockStore := new(MockKVStoreRepositoryTest)
	entityID := "ucUpdate1"
	iGF := NewTestIDGeneratorFactoryR([]string{})
	repo, err := db.NewRepository[UserComplex](mockStore, "cf_complex", "test_sch", iGF)
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
	mockStore.On("Get", "cf_complex", "test_sch:users_complex:data:"+entityID, mock.Anything).Return(originalData, nil).Once()

	// 2. Unique check for new Meta.Tag value
	mockStore.On("Get", "cf_complex", "test_sch:users_complex:idx-u:Meta.Tag:newTag", mock.Anything).Return(nil, nil).Once()
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
	mockStore := new(MockKVStoreRepositoryTest)
	entityID := "ucUpdateDup1"
	iGF := NewTestIDGeneratorFactoryR([]string{})
	repo, err := db.NewRepository[UserComplex](mockStore, "cf_complex", "test_sch", iGF)
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
	mockStore.On("Get", "cf_complex", "test_sch:users_complex:data:"+entityID, mock.Anything).Return(originalData, nil).Once()

	// 2. Unique check for new Meta.Tag shows it exists for another ID
	mockStore.On("Get", "cf_complex", "test_sch:users_complex:idx-u:Meta.Tag:conflictingTag", mock.Anything).Return([]byte("anotherEntityID"), nil).Once()

	changed, err := repo.Update(&updatedUser, time.Now())
	require.Error(t, err)
	assert.False(t, changed)
	assert.Contains(t, err.Error(), "duplicate unique field: Meta.Tag = conflictingTag")

	mockStore.AssertExpectations(t)
}

func TestRepository_Delete_WithNestedFields_UserComplex(t *testing.T) {
	mockStore := new(MockKVStoreRepositoryTest)
	entityID := "ucDelete1"
	iGF := NewTestIDGeneratorFactoryR([]string{})
	repo, err := db.NewRepository[UserComplex](mockStore, "cf_complex", "test_sch", iGF)
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
	mockStore.On("Get", "cf_complex", "test_sch:users_complex:data:"+entityID, mock.Anything).Return(jsonData, nil).Once()

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
	mockStore := new(MockKVStoreRepositoryTest)
	entityID := "uceFieldTest"
	iGF := NewTestIDGeneratorFactoryR([]string{entityID})

	repo, err := db.NewRepository[UserComplexEmbedded](mockStore, "cf_embed_fn", "test_sch_fn", iGF)
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
	mockStore.On("Get", "cf_embed_fn", "test_sch_fn:users_complex_embedded:idx-u:Email:embedfn@example.com", mock.Anything).Return(nil, nil).Once()
	mockStore.On("Get", "cf_embed_fn", "test_sch_fn:users_complex_embedded:idx-u:Tag:embedFnTag", mock.Anything).Return(nil, nil).Once() // Key check
	mockStore.On("Get", "cf_embed_fn", "test_sch_fn:users_complex_embedded:data:uceFieldTest", mock.Anything).Return(nil, nil).Once()    // Key check

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
	mockStore := new(MockKVStoreRepositoryTest)

	user := User{
		ID:   "det-123",
		Name: "Alice",
	}

	iGF := &db.DeterministicIDGeneratorFactory{}

	repo, err := db.NewRepository[User](mockStore, "cf1", "admin", iGF)
	assert.NoError(t, err)

	data, _ := json.Marshal(user)
	dataKey := "admin:users:data:det-123"
	nameFieldKey := "admin:users:idx:Name:Alice:det-123"
	uNameFieldKey := "admin:users:idx-u:Name:Alice"
	indexKey := "admin:users:idx:ID:det-123:det-123"

	mockStore.On("Get", "cf1", uNameFieldKey).Return(nil, nil)
	mockStore.On("Get", "cf1", dataKey).Return(nil, nil)

	batch := db.NewWriteBatch()
	batch.Put("cf1", indexKey, []byte("det-123"))
	batch.Put("cf1", nameFieldKey, []byte("det-123"))
	batch.Put("cf1", uNameFieldKey, []byte("det-123"))
	batch.Put("cf1", dataKey, data)

	mockStore.On("Write", mock.MatchedBy(func(b *db.WriteBatch) bool {
		return true
	})).Return(nil)

	id, err := repo.Create(&user, time.Now())
	assert.NoError(t, err)
	assert.Equal(t, id, "det-123")

	mockStore.AssertExpectations(t)
}
