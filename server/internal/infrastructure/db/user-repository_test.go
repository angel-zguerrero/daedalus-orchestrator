package db_test

import (
	"encoding/json"
	"errors"
	"fmt"

	"testing"

	"github.com/linxGnu/grocksdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/shared/constants"
	"deadalus-orch/shared/models"
)

type MockKVStore struct {
	mock.Mock
}

func (m *MockKVStore) Get(AdminFC, key string) ([]byte, error) {
	args := m.Called(AdminFC, key)
	var s []byte
	if tmp := args.Get(0); tmp != nil {
		s = tmp.([]byte)
	}
	return s, args.Error(1)
}

func (m *MockKVStore) Put(AdminFC, key string, value []byte) error {
	args := m.Called(AdminFC, key, value)
	return args.Error(0)
}

func (m *MockKVStore) Write(batch interface{}) error {
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

func TestPutUser_Success(t *testing.T) {
	mockStore := new(MockKVStore)

	user := models.CreateUser{Username: "foo", Email: "foo@mail.com", Password: "1234"}

	mockStore.On("Put", db.AdminFC, "user:foo", mock.Anything).Return(nil)

	err := db.PutUser(mockStore, user)
	assert.NoError(t, err)
	mockStore.AssertExpectations(t)
}

func TestGetUser_Success(t *testing.T) {
	mockStore := new(MockKVStore)

	u := models.User{Username: "foo", Email: "bar"}
	data, _ := json.Marshal(u)

	mockStore.On("Get", db.AdminFC, "user:foo").Return(data, nil)

	user, err := db.GetUser(mockStore, "foo")
	assert.NoError(t, err)
	assert.Equal(t, "foo", user.Username)
}

func TestGetUser_NotFound(t *testing.T) {
	mockStore := new(MockKVStore)
	mockStore.On("Get", db.AdminFC, "user:bar").Return(nil, nil)

	user, err := db.GetUser(mockStore, "bar")
	assert.NoError(t, err)
	assert.Nil(t, user)
}

func TestGetUser_ErrorOnGet(t *testing.T) {
	mockStore := new(MockKVStore)
	mockStore.On("Get", db.AdminFC, "user:x").Return(nil, errors.New("get failed"))

	user, err := db.GetUser(mockStore, "x")
	assert.Error(t, err)
	assert.Nil(t, user)
}

func TestGetUser_UnmarshalError(t *testing.T) {
	mockStore := new(MockKVStore)

	mockStore.On("Get", db.AdminFC, "user:x").Return([]byte("invalid-json"), nil)

	user, err := db.GetUser(mockStore, "x")
	assert.Error(t, err)
	assert.Nil(t, user)
}

func TestPutDefaultRootUserRoot_Success(t *testing.T) {
	mockStore := new(MockKVStore)
	mockStore.On("Write", mock.Anything).Return(nil)

	input := models.CreateUser{Username: "admin", Email: "root@mail.com", Password: "pass"}
	err := db.PutDefaultRootUserRoot(mockStore, input)
	assert.NoError(t, err)
}

func TestPutDefaultRootUserRoot_WriteError(t *testing.T) {
	mockStore := new(MockKVStore)
	mockStore.On("Write", mock.Anything).Return(errors.New("write failed"))

	input := models.CreateUser{Username: "admin", Email: "x", Password: "x"}
	err := db.PutDefaultRootUserRoot(mockStore, input)
	assert.Error(t, err)
}

func TestGetDefaultRootUserRoot_UnmarshalError(t *testing.T) {
	mockStore := new(MockKVStore)
	mockStore.On("Get", db.AdminFC, constants.DefaultRootUserRootKey).Return([]byte("bad-json"), nil)

	root, err := db.GetDefaultRootUserRoot(mockStore)
	assert.Error(t, err)
	assert.Nil(t, root)
}

func TestDeleteUser_Success(t *testing.T) {
	mockStore := new(MockKVStore)
	root := models.User{Username: "other"}
	rootData, _ := json.Marshal(root)

	mockStore.On("Get", db.AdminFC, constants.DefaultRootUserRootKey).Return(rootData, nil)
	mockStore.On("Write", mock.Anything).Return(nil)

	err := db.DeleteUser(mockStore, "bob")
	assert.NoError(t, err)
}

func TestDeleteUser_CannotDeleteRoot(t *testing.T) {
	mockStore := new(MockKVStore)
	root := models.User{Username: "admin"}
	rootData, _ := json.Marshal(root)
	mockStore.On("Get", db.AdminFC, constants.DefaultRootUserRootKey).Return(rootData, nil)

	err := db.DeleteUser(mockStore, "admin")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot delete root user")
}

func TestDeleteUser_GetError(t *testing.T) {
	mockStore := new(MockKVStore)
	mockStore.On("Get", db.AdminFC, mock.Anything).Return(nil, errors.New("get failed"))

	err := db.DeleteUser(mockStore, "someone")
	assert.Error(t, err)
}

func TestDeleteUser_UnmarshalRootError(t *testing.T) {
	mockStore := new(MockKVStore)
	mockStore.On("Get", db.AdminFC, constants.DefaultRootUserRootKey).Return([]byte("bad"), nil)

	err := db.DeleteUser(mockStore, "x")
	assert.Error(t, err)
}

func TestDeleteUser_WriteError(t *testing.T) {
	mockStore := new(MockKVStore)
	root := models.User{Username: "root"}
	rootData, _ := json.Marshal(root)
	mockStore.On("Get", db.AdminFC, constants.DefaultRootUserRootKey).Return(rootData, nil)
	mockStore.On("Write", mock.Anything).Return(errors.New("write failed"))

	err := db.DeleteUser(mockStore, "user")
	assert.Error(t, err)
}
func TestPutUser_KVStorePutError(t *testing.T) {
	store := new(MockKVStore)
	userInput := models.CreateUser{
		Username: "testuser",
		Password: "password123",
		Email:    "test@example.com",
	}
	userKey := fmt.Sprintf("user:%s", userInput.Username)

	store.On("Put", db.AdminFC, userKey, mock.AnythingOfType("[]uint8")).Return(errors.New("kv put failed")).Once()

	err := db.PutUser(store, userInput)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "kv put failed")
	store.AssertExpectations(t)
}

func TestGetDefaultRootUserRoot_Success(t *testing.T) {
	store := new(MockKVStore)
	expectedUser := models.CreateUser{
		Username: "admin",
		Password: "securepassword",
		Email:    "admin@daedalus.com",
	}
	jsonData, err := json.Marshal(expectedUser)
	require.NoError(t, err)

	store.On("Get", db.AdminFC, constants.DefaultRootUserRootKey).Return(jsonData, nil).Once()

	user, err := db.GetDefaultRootUserRoot(store)

	assert.NoError(t, err)
	require.NotNil(t, user)
	assert.Equal(t, expectedUser.Username, user.Username)
	assert.Equal(t, expectedUser.Password, user.Password)
	assert.Equal(t, expectedUser.Email, user.Email)
	store.AssertExpectations(t)
}

func TestGetDefaultRootUserRoot_NotFound(t *testing.T) {
	store := new(MockKVStore)
	store.On("Get", db.AdminFC, constants.DefaultRootUserRootKey).Return(nil, nil).Once()

	user, err := db.GetDefaultRootUserRoot(store)

	assert.NoError(t, err)
	assert.Nil(t, user)
	store.AssertExpectations(t)
}

func TestGetDefaultRootUserRoot_KVStoreGetError(t *testing.T) {
	store := new(MockKVStore)
	store.On("Get", db.AdminFC, constants.DefaultRootUserRootKey).Return(nil, errors.New("kv get failed")).Once()

	user, err := db.GetDefaultRootUserRoot(store)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "kv get failed")
	assert.Nil(t, user)
	store.AssertExpectations(t)
}

func TestDeleteUser_NoRootUserDefined(t *testing.T) {
	store := new(MockKVStore)
	usernameToDelete := "someuser"

	store.On("Get", db.AdminFC, constants.DefaultRootUserRootKey).Return(nil, nil).Once()

	store.On("Write", mock.AnythingOfType("*grocksdb.WriteBatch")).Run(func(args mock.Arguments) {
		batch := args.Get(0).(*grocksdb.WriteBatch)
		assert.Equal(t, 1, batch.Count(), "WriteBatch should have one operation (the delete)")

	}).Return(nil).Once()

	err := db.DeleteUser(store, usernameToDelete)

	assert.NoError(t, err)
	store.AssertExpectations(t)
}
