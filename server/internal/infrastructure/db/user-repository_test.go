package db_test

import (
	"encoding/json"
	"errors"

	"testing"

	"github.com/linxGnu/grocksdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/shared/models"
)

type MockKVStore struct {
	mock.Mock
	ColumnFamilyHandles    map[string]*grocksdb.ColumnFamilyHandle // Map of regular column family names to their handles.
	TTLColumnFamilyHandles map[string]*grocksdb.ColumnFamilyHandle // Map of TTL column family names to their handles.
}

func (m *MockKVStore) Get(AdminFC, key string) ([]byte, error) {
	args := m.Called(AdminFC, key)
	var s []byte
	if tmp := args.Get(0); tmp != nil {
		s = tmp.([]byte)
	}
	return s, args.Error(1)
}

func (m *MockKVStore) Delete(AdminFC, key string) error {
	args := m.Called(AdminFC, key)
	return args.Error(0)
}

func (r *MockKVStore) Exists(columnFamily, key string) (bool, error) {
	val, err := r.Get(columnFamily, key)
	if err != nil {
		return false, err
	}
	return val != nil, nil
}

func (m *MockKVStore) Put(AdminFC, key string, value []byte, ttl int) error {
	args := m.Called(AdminFC, key, value, ttl)
	return args.Error(0)
}

func (m *MockKVStore) PutRaw(AdminFC, key string, value []byte) error {
	args := m.Called(AdminFC, key, value)
	return args.Error(0)
}

func (m *MockKVStore) Write(batch *db.WriteBatch) error {
	args := m.Called(batch)
	return args.Error(0)
}

func (m *MockKVStore) WriteRaw(batch *db.WriteBatch) error {
	args := m.Called(batch)
	return args.Error(0)
}

func (m *MockKVStore) DumpAll() (interface{}, error) {
	args := m.Called()
	var s []byte
	if tmp := args.Get(0); tmp != nil {
		s = tmp.([]byte)
	}
	return s, args.Error(1)
}

func (r *MockKVStore) Iterate(fn func(cfName string, key, value []byte) error) error {
	return nil
}

func (r *MockKVStore) ClearAll() error {
	return nil
}

func (r *MockKVStore) Flush() error {
	return nil
}

func (r *MockKVStore) Close() error {
	return nil
}

func (r *MockKVStore) CleanExpiredKeys() error {
	return nil
}

func (m *MockKVStore) SearchByPatternPaginatedKV(cfName, pattern, cursor string, limit int) ([]db.KeyValuePair, string, error) {
	args := m.Called(cfName, pattern, cursor, limit)
	var s []db.KeyValuePair
	if tmp := args.Get(0); tmp != nil {
		s = tmp.([]db.KeyValuePair)
	}
	return s, "", args.Error(2)
}

func TestPutUser_Success(t *testing.T) {
	mockStore := new(MockKVStore)
	uow := db.NewUnitOfWork(mockStore)
	repo, err := db.NewUserRepository(uow)
	assert.NoError(t, err)

	user := models.CreateUser{Username: "foo", Email: "foo@mail.com", Password: "1234"}

	mockStore.On("SearchByPatternPaginatedKV", db.AdminFC, "admin_schema:users:idx:IsRootUser:true:*", "", 1).Return(nil, "", nil)
	mockStore.On("Get", db.AdminFC, "admin_schema:users:idx-u:Username:foo").Return(nil, nil).Times(1)
	mockStore.On("Get", db.AdminFC, "admin_schema:users:idx-u:Email:foo@mail.com").Return(nil, nil).Times(1)
	mockStore.On("Write", mock.Anything).Return(nil).Times(1)

	id, err := repo.CreateUser(user)
	assert.NoError(t, err)
	err = uow.Commit()
	assert.NotNil(t, id)
	assert.NoError(t, err)
	mockStore.AssertExpectations(t)
}

func TestGetUser_Success(t *testing.T) {
	mockStore := new(MockKVStore)
	uow := db.NewUnitOfWork(mockStore)
	repo, err := db.NewUserRepository(uow)

	u := models.User{Username: "foo", Email: "bar"}
	data, _ := json.Marshal(u)

	mockStore.On("Get", db.AdminFC, "admin_schema:users:idx-u:Username:foo").Return([]byte("123"), nil)
	mockStore.On("Get", db.AdminFC, "admin_schema:users:data:123").Return(data, nil)

	user, err := repo.GetUserByUsername("foo")
	assert.NoError(t, err)
	assert.Equal(t, "foo", user.Username)
	mockStore.AssertExpectations(t)
}

func TestGetUser_NotFound(t *testing.T) {
	mockStore := new(MockKVStore)
	uow := db.NewUnitOfWork(mockStore)
	repo, err := db.NewUserRepository(uow)
	mockStore.On("Get", db.AdminFC, "admin_schema:users:idx-u:Username:bar").Return(nil, nil)

	user, err := repo.GetUserByUsername("bar")
	assert.NoError(t, err)
	assert.Nil(t, user)
	mockStore.AssertExpectations(t)
}

func TestGetUser_ErrorOnGet(t *testing.T) {
	mockStore := new(MockKVStore)
	uow := db.NewUnitOfWork(mockStore)
	repo, err := db.NewUserRepository(uow)

	mockStore.On("Get", db.AdminFC, "admin_schema:users:idx-u:Username:x").Return(nil, errors.New("get failed"))
	user, err := repo.GetUserByUsername("x")
	assert.Error(t, err)
	assert.Nil(t, user)
	mockStore.AssertExpectations(t)
}

func TestGetUser_UnmarshalError(t *testing.T) {
	mockStore := new(MockKVStore)
	uow := db.NewUnitOfWork(mockStore)
	repo, err := db.NewUserRepository(uow)

	mockStore.On("Get", db.AdminFC, "admin_schema:users:idx-u:Username:x").Return([]byte("123"), nil)
	mockStore.On("Get", db.AdminFC, "admin_schema:users:data:123").Return([]byte("invalid-json"), nil)

	user, err := repo.GetUserByUsername("x")
	assert.Error(t, err)
	assert.Empty(t, user)
	mockStore.AssertExpectations(t)
}

func TestDeleteUser_Success(t *testing.T) {
	mockStore := new(MockKVStore)
	uow := db.NewUnitOfWork(mockStore)
	repo, err := db.NewUserRepository(uow)
	root := models.User{Username: "other", ID: "123"}
	rootData, _ := json.Marshal(root)

	mockStore.On("Get", db.AdminFC, "admin_schema:users:idx-u:Username:bob").Return([]byte("123"), nil)
	mockStore.On("Get", db.AdminFC, "admin_schema:users:data:123").Return(rootData, nil)

	result, err := repo.DeleteUser("bob")
	assert.Equal(t, true, result)
	assert.NoError(t, err)
	mockStore.AssertExpectations(t)
}

func TestDeleteUser_CannotDeleteRoot(t *testing.T) {
	mockStore := new(MockKVStore)
	uow := db.NewUnitOfWork(mockStore)
	repo, err := db.NewUserRepository(uow)
	root := models.User{Username: "admin", ID: "123", IsRootUser: true}
	rootData, _ := json.Marshal(root)
	mockStore.On("Get", db.AdminFC, "admin_schema:users:idx-u:Username:admin").Return([]byte("123"), nil)
	mockStore.On("Get", db.AdminFC, "admin_schema:users:data:123").Return(rootData, nil)
	_, err = repo.DeleteUser("admin")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot delete root user")
	mockStore.AssertExpectations(t)
}

func TestDeleteUser_GetError(t *testing.T) {
	mockStore := new(MockKVStore)
	uow := db.NewUnitOfWork(mockStore)
	repo, err := db.NewUserRepository(uow)
	mockStore.On("Get", db.AdminFC, mock.Anything).Return(nil, errors.New("get failed"))

	_, err = repo.DeleteUser("someone")
	assert.Error(t, err)
	mockStore.AssertExpectations(t)
}

func TestDeleteUser_UnmarshalRootError(t *testing.T) {
	mockStore := new(MockKVStore)
	uow := db.NewUnitOfWork(mockStore)
	repo, err := db.NewUserRepository(uow)
	mockStore.On("Get", db.AdminFC, "admin_schema:users:idx-u:Username:x").Return([]byte("123"), nil)
	mockStore.On("Get", db.AdminFC, "admin_schema:users:data:123").Return([]byte("invalid-json"), nil)
	_, err = repo.DeleteUser("x")
	assert.Error(t, err)
	mockStore.AssertExpectations(t)
}

func TestDeleteUser_WriteError(t *testing.T) {
	mockStore := new(MockKVStore)
	uow := db.NewUnitOfWork(mockStore)
	repo, err := db.NewUserRepository(uow)
	root := models.User{Username: "user", ID: "123"}
	rootData, _ := json.Marshal(root)
	mockStore.On("Get", db.AdminFC, "admin_schema:users:idx-u:Username:user").Return([]byte("123"), nil)
	mockStore.On("Get", db.AdminFC, "admin_schema:users:data:123").Return(rootData, nil)

	mockStore.On("Write", mock.Anything).Return(errors.New("write failed"))

	_, err = repo.DeleteUser("user")
	assert.NoError(t, err)
	err = uow.Commit()
	assert.Error(t, err)
	mockStore.AssertExpectations(t)
}
func TestPutUser_KVStorePutError(t *testing.T) {
	mockStore := new(MockKVStore)
	uow := db.NewUnitOfWork(mockStore)
	repo, err := db.NewUserRepository(uow)
	userInput := models.CreateUser{
		Username: "testuser",
		Password: "password123",
		Email:    "test@example.com",
	}

	mockStore.On("SearchByPatternPaginatedKV", db.AdminFC, "admin_schema:users:idx:IsRootUser:true:*", "", 1).Return(nil, "", nil)
	mockStore.On("Get", db.AdminFC, "admin_schema:users:idx-u:Username:testuser").Return(nil, nil).Times(1)
	mockStore.On("Get", db.AdminFC, "admin_schema:users:idx-u:Email:test@example.com").Return(nil, nil).Times(1)
	mockStore.On("Write", mock.Anything).Return(errors.New("kv put failed")).Times(1)

	_, err = repo.CreateUser(userInput)

	assert.NoError(t, err)
	err = uow.Commit()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "kv put failed")
	mockStore.AssertExpectations(t)
}
