package db

import (
	"encoding/json"
	"errors"

	"testing"

	"github.com/linxGnu/grocksdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"deadalus-orch/shared/constants"
	"deadalus-orch/shared/models"
)

type MockSlice struct {
	mock.Mock
	data   []byte
	exists bool
}

func (m *MockSlice) Data() []byte {
	return m.data
}
func (m *MockSlice) Free()        {}
func (m *MockSlice) Exists() bool { return m.exists }

type MockKVStore struct {
	mock.Mock
}

func (m *MockKVStore) Get(ro *grocksdb.ReadOptions, key []byte) (Slice, error) {
	args := m.Called(ro, key)
	var s Slice
	if tmp := args.Get(0); tmp != nil {
		s = tmp.(Slice)
	}
	return s, args.Error(1)
}

func (m *MockKVStore) Put(wo *grocksdb.WriteOptions, key, value []byte) error {
	args := m.Called(wo, key, value)
	return args.Error(0)
}

func (m *MockKVStore) Write(wo *grocksdb.WriteOptions, batch *grocksdb.WriteBatch) error {
	args := m.Called(wo, batch)
	return args.Error(0)
}

func TestPutUser_Success(t *testing.T) {
	mockStore := new(MockKVStore)

	user := models.CreateUser{Username: "foo", Email: "foo@mail.com", Password: "1234"}

	mockStore.On("Put", mock.Anything, []byte("user:foo"), mock.Anything).Return(nil)

	err := PutUser(mockStore, user)
	assert.NoError(t, err)
	mockStore.AssertExpectations(t)
}

func TestGetUser_Success(t *testing.T) {
	mockStore := new(MockKVStore)

	u := models.User{Username: "foo", Email: "bar"}
	data, _ := json.Marshal(u)
	mockSlice := &MockSlice{data: data, exists: true}

	mockStore.On("Get", mock.Anything, []byte("user:foo")).Return(mockSlice, nil)

	user, err := GetUser(mockStore, "foo")
	assert.NoError(t, err)
	assert.Equal(t, "foo", user.Username)
}

func TestGetUser_NotFound(t *testing.T) {
	mockStore := new(MockKVStore)
	mockSlice := &MockSlice{exists: false}
	mockStore.On("Get", mock.Anything, []byte("user:bar")).Return(mockSlice, nil)

	user, err := GetUser(mockStore, "bar")
	assert.NoError(t, err)
	assert.Nil(t, user)
}

func TestGetUser_ErrorOnGet(t *testing.T) {
	mockStore := new(MockKVStore)
	mockStore.On("Get", mock.Anything, []byte("user:x")).Return(nil, errors.New("get failed"))

	user, err := GetUser(mockStore, "x")
	assert.Error(t, err)
	assert.Nil(t, user)
}

func TestGetUser_UnmarshalError(t *testing.T) {
	mockStore := new(MockKVStore)
	mockSlice := &MockSlice{data: []byte("invalid-json"), exists: true}
	mockStore.On("Get", mock.Anything, []byte("user:x")).Return(mockSlice, nil)

	user, err := GetUser(mockStore, "x")
	assert.Error(t, err)
	assert.Nil(t, user)
}

func TestPutDefaultRootUserRoot_Success(t *testing.T) {
	mockStore := new(MockKVStore)
	mockStore.On("Write", mock.Anything, mock.Anything).Return(nil)

	input := models.CreateUser{Username: "admin", Email: "root@mail.com", Password: "pass"}
	err := PutDefaultRootUserRoot(mockStore, input)
	assert.NoError(t, err)
}

func TestPutDefaultRootUserRoot_WriteError(t *testing.T) {
	mockStore := new(MockKVStore)
	mockStore.On("Write", mock.Anything, mock.Anything).Return(errors.New("write failed"))

	input := models.CreateUser{Username: "admin", Email: "x", Password: "x"}
	err := PutDefaultRootUserRoot(mockStore, input)
	assert.Error(t, err)
}

func TestGetDefaultRootUserRoot_UnmarshalError(t *testing.T) {
	mockStore := new(MockKVStore)
	mockSlice := &MockSlice{data: []byte("bad-json"), exists: true}
	mockStore.On("Get", mock.Anything, []byte(constants.DefaultRootUserRootKey)).Return(mockSlice, nil)

	root, err := GetDefaultRootUserRoot(mockStore)
	assert.Error(t, err)
	assert.Nil(t, root)
}

func TestDeleteUser_Success(t *testing.T) {
	mockStore := new(MockKVStore)
	root := models.User{Username: "other"}
	rootData, _ := json.Marshal(root)
	mockSlice := &MockSlice{data: rootData, exists: true}
	mockStore.On("Get", mock.Anything, []byte(constants.DefaultRootUserRootKey)).Return(mockSlice, nil)
	mockStore.On("Write", mock.Anything, mock.Anything).Return(nil)

	err := DeleteUser(mockStore, "bob")
	assert.NoError(t, err)
}

func TestDeleteUser_CannotDeleteRoot(t *testing.T) {
	mockStore := new(MockKVStore)
	root := models.User{Username: "admin"}
	rootData, _ := json.Marshal(root)
	mockSlice := &MockSlice{data: rootData, exists: true}
	mockStore.On("Get", mock.Anything, []byte(constants.DefaultRootUserRootKey)).Return(mockSlice, nil)

	err := DeleteUser(mockStore, "admin")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot delete root user")
}

func TestDeleteUser_GetError(t *testing.T) {
	mockStore := new(MockKVStore)
	mockStore.On("Get", mock.Anything, mock.Anything).Return(nil, errors.New("get failed"))

	err := DeleteUser(mockStore, "someone")
	assert.Error(t, err)
}

func TestDeleteUser_UnmarshalRootError(t *testing.T) {
	mockStore := new(MockKVStore)
	mockSlice := &MockSlice{data: []byte("bad"), exists: true}
	mockStore.On("Get", mock.Anything, []byte(constants.DefaultRootUserRootKey)).Return(mockSlice, nil)

	err := DeleteUser(mockStore, "x")
	assert.Error(t, err)
}

func TestDeleteUser_WriteError(t *testing.T) {
	mockStore := new(MockKVStore)
	root := models.User{Username: "root"}
	rootData, _ := json.Marshal(root)
	mockSlice := &MockSlice{data: rootData, exists: true}
	mockStore.On("Get", mock.Anything, []byte(constants.DefaultRootUserRootKey)).Return(mockSlice, nil)
	mockStore.On("Write", mock.Anything, mock.Anything).Return(errors.New("write failed"))

	err := DeleteUser(mockStore, "user")
	assert.Error(t, err)
}
