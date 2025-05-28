package db_test

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/shared/constants"
	"deadalus-orch/shared/models"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/linxGnu/grocksdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	// "golang.org/x/crypto/bcrypt" // Not strictly needed for these new tests, but often used with users
)

// MockKVStore is a mock type for the KVStore interface
type MockKVStore struct {
	mock.Mock
}

func (m *MockKVStore) Get(cf string, key []byte) ([]byte, error) {
	args := m.Called(cf, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockKVStore) Put(cf string, key []byte, value []byte) error {
	args := m.Called(cf, key, value)
	return args.Error(0)
}

func (m *MockKVStore) Delete(cf string, key []byte) error {
	args := m.Called(cf, key)
	return args.Error(0)
}

func (m *MockKVStore) Write(batch *grocksdb.WriteBatch) error {
	args := m.Called(batch)
	return args.Error(0)
}

func (m *MockKVStore) NewIterator(cfName string) *grocksdb.Iterator {
	args := m.Called(cfName)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*grocksdb.Iterator)
}

func (m *MockKVStore) Close() {}

// --- New Test Cases ---

func TestPutUser_MarshalError(t *testing.T) {
	store := new(MockKVStore)
	// Create a user input that will cause json.Marshal to fail.
	// A channel is a common way to do this.
	userInput := models.CreateUser{
		Username: "testuser",
		Password: "password",
		Email:    (chan int)(nil), // This will cause Marshal to fail
	}

	err := db.PutUser(store, userInput)

	assert.Error(t, err)
	// Check if the error is a json.MarshalerError or contains relevant text
	// Note: The exact error type might vary, so checking for content is safer.
	assert.Contains(t, err.Error(), "json: unsupported type: chan int")
	store.AssertNotCalled(t, "Put", mock.Anything, mock.Anything, mock.Anything)
}

func TestPutUser_KVStorePutError(t *testing.T) {
	store := new(MockKVStore)
	userInput := models.CreateUser{
		Username: "testuser",
		Password: "password123",
		Email:    "test@example.com",
	}
	userKey := fmt.Sprintf("user:%s", userInput.Username)

	// Mock the Put operation to fail
	store.On("Put", db.AdminFC, []byte(userKey), mock.AnythingOfType("[]uint8")).Return(errors.New("kv put failed")).Once()

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

	store.On("Get", db.AdminFC, []byte(constants.DefaultRootUserRootKey)).Return(jsonData, nil).Once()

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
	store.On("Get", db.AdminFC, []byte(constants.DefaultRootUserRootKey)).Return(nil, nil).Once()

	user, err := db.GetDefaultRootUserRoot(store)

	assert.NoError(t, err)
	assert.Nil(t, user)
	store.AssertExpectations(t)
}

func TestGetDefaultRootUserRoot_KVStoreGetError(t *testing.T) {
	store := new(MockKVStore)
	store.On("Get", db.AdminFC, []byte(constants.DefaultRootUserRootKey)).Return(nil, errors.New("kv get failed")).Once()

	user, err := db.GetDefaultRootUserRoot(store)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "kv get failed")
	assert.Nil(t, user)
	store.AssertExpectations(t)
}

func TestPutDefaultRootUserRoot_MarshalErrorInput(t *testing.T) {
	store := new(MockKVStore)
	// Create a user input that will cause json.Marshal to fail for the input itself.
	userInput := models.CreateUser{
		Username: "rootadmin",
		Password: "password",
		Email:    (chan bool)(nil), // This will cause Marshal(input) to fail
	}

	err := db.PutDefaultRootUserRoot(store, userInput)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "json: unsupported type: chan bool")
	store.AssertNotCalled(t, "Write", mock.Anything) // Write should not be called if marshalling input fails
}

func TestDeleteUser_NoRootUserDefined(t *testing.T) {
	store := new(MockKVStore)
	usernameToDelete := "someuser"
	userKey := []byte(fmt.Sprintf("user:%s", usernameToDelete))

	// Simulate no root user defined
	store.On("Get", db.AdminFC, []byte(constants.DefaultRootUserRootKey)).Return(nil, nil).Once()

	// Expect a Write operation with a WriteBatch
	store.On("Write", mock.AnythingOfType("*grocksdb.WriteBatch")).Run(func(args mock.Arguments) {
		batch := args.Get(0).(*grocksdb.WriteBatch)
		// Optionally, inspect the batch here if needed, e.g., count operations
		assert.Equal(t, 1, batch.Count(), "WriteBatch should have one operation (the delete)")
		// A more thorough check would be to iterate the batch and check the key, but it's complex.
	}).Return(nil).Once()

	err := db.DeleteUser(store, usernameToDelete)

	assert.NoError(t, err)
	store.AssertExpectations(t)
}
