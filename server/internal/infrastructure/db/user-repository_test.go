package db_test

import (
	"encoding/json"
	"errors"
	"sync"
	"time"

	"testing"

	"github.com/linxGnu/grocksdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/shared/models"

	"golang.org/x/crypto/bcrypt"
)

type MockKVStore struct {
	mock.Mock
	ColumnFamilyHandles    map[string]*grocksdb.ColumnFamilyHandle // Map of regular column family names to their handles.
	TTLColumnFamilyHandles map[string]*grocksdb.ColumnFamilyHandle // Map of TTL column family names to their handles.
}

func (m *MockKVStore) Get(AdminFC, key string, now time.Time) ([]byte, error) {
	args := m.Called(AdminFC, key, now)
	var s []byte
	if tmp := args.Get(0); tmp != nil {
		s = tmp.([]byte)
	}
	return s, args.Error(1)
}

func (m *MockKVStore) Delete(AdminFC, key string, now time.Time) error {
	args := m.Called(AdminFC, key, now)
	return args.Error(0)
}

func (r *MockKVStore) Exists(columnFamily, key string, now time.Time) (bool, error) {
	// This mock Exists calls its own Get.
	// For tests that mock Exists directly: args := r.Called(columnFamily, key, now) ...
	// For tests that rely on this passthrough:
	data, err := r.Get(columnFamily, key, now) // Pass now here
	if err != nil {
		return false, err
	}
	return data != nil, nil
}

func (m *MockKVStore) Put(AdminFC, key string, value []byte, ttl int, now time.Time) error {
	args := m.Called(AdminFC, key, value, ttl, now)
	return args.Error(0)
}

func (m *MockKVStore) PutRaw(AdminFC, key string, value []byte) error {
	args := m.Called(AdminFC, key, value)
	return args.Error(0)
}

func (m *MockKVStore) Write(batch *db.WriteBatch, now time.Time) error {
	args := m.Called(batch, now)
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

func (r *MockKVStore) CleanExpiredKeys(now time.Time) error {
	args := r.Called(now)
	return args.Error(0)
}

func (m *MockKVStore) SearchByPatternPaginatedKV(cfName, pattern, cursor string, limit int, now time.Time) ([]db.KeyValuePair, string, error) {
	args := m.Called(cfName, pattern, cursor, limit, now)
	var s []db.KeyValuePair
	if tmp := args.Get(0); tmp != nil {
		s = tmp.([]db.KeyValuePair)
	}
	return s, "", args.Error(2)
}

type TestIDGeneratorFactoryRepository struct {
	ids   []string
	index int
	mu    sync.Mutex
}

func (g *TestIDGeneratorFactoryRepository) GenerateID() string {
	g.mu.Lock()
	defer g.mu.Unlock()

	if len(g.ids) == 0 {
		return ""
	}

	id := g.ids[g.index]
	g.index = (g.index + 1) % len(g.ids) // avance circular
	return id
}

func NewTestIDGeneratorFactoryRepository(ids []string) *TestIDGeneratorFactoryRepository {
	return &TestIDGeneratorFactoryRepository{
		ids: ids,
	}
}

func TestPutUser_Success(t *testing.T) {
	mockStore := new(MockKVStore)
	uow := db.NewUnitOfWork(mockStore, nil)
	iGF := NewTestIDGeneratorFactoryRepository([]string{"123"})
	repo, err := db.NewUserRepository(uow, iGF)
	assert.NoError(t, err)

	user := models.CreateUser{Username: "foo", Email: "foo@mail.com", Password: "1234"}

	mockStore.On("SearchByPatternPaginatedKV", db.AdminFC, "admin_schema:users:idx:IsRootUser:true:*", "", 1, mock.Anything).Return(nil, "", nil)
	mockStore.On("Get", db.AdminFC, "admin_schema:users:idx-u:Username:foo", mock.Anything).Return(nil, nil).Times(1)
	mockStore.On("Get", db.AdminFC, "admin_schema:users:idx-u:Email:foo@mail.com", mock.Anything).Return(nil, nil).Times(1)
	mockStore.On("Get", db.AdminFC, "admin_schema:users:data:123", mock.Anything).Return(nil, nil).Times(1)
	mockStore.On("Write", mock.Anything, mock.Anything).Return(nil).Times(1)

	id, err := repo.CreateUser(user)
	assert.NoError(t, err)
	err = uow.Commit(time.Now())
	assert.NotNil(t, id)
	assert.NoError(t, err)
	mockStore.AssertExpectations(t)
}

func TestGetUser_Success(t *testing.T) {
	mockStore := new(MockKVStore)
	uow := db.NewUnitOfWork(mockStore, nil)
	iGF := NewTestIDGeneratorFactoryRepository([]string{"123"})
	repo, err := db.NewUserRepository(uow, iGF)

	u := models.User{Username: "foo", Email: "bar"}
	data, _ := json.Marshal(u)

	mockStore.On("Get", db.AdminFC, "admin_schema:users:idx-u:Username:foo", mock.Anything).Return([]byte("123"), nil)
	mockStore.On("Get", db.AdminFC, "admin_schema:users:data:123", mock.Anything).Return(data, nil)

	user, err := repo.GetUserByUsername("foo")
	assert.NoError(t, err)
	assert.Equal(t, "foo", user.Username)
	mockStore.AssertExpectations(t)
}

func TestGetUser_NotFound(t *testing.T) {
	mockStore := new(MockKVStore)
	uow := db.NewUnitOfWork(mockStore, nil)
	iGF := NewTestIDGeneratorFactoryRepository([]string{"123"})
	repo, err := db.NewUserRepository(uow, iGF)
	mockStore.On("Get", db.AdminFC, "admin_schema:users:idx-u:Username:bar", mock.Anything).Return(nil, nil)

	user, err := repo.GetUserByUsername("bar")
	assert.NoError(t, err)
	assert.Nil(t, user)
	mockStore.AssertExpectations(t)
}

func TestGetUser_ErrorOnGet(t *testing.T) {
	mockStore := new(MockKVStore)
	uow := db.NewUnitOfWork(mockStore, nil)
	iGF := NewTestIDGeneratorFactoryRepository([]string{"123"})
	repo, err := db.NewUserRepository(uow, iGF)

	mockStore.On("Get", db.AdminFC, "admin_schema:users:idx-u:Username:x", mock.Anything).Return(nil, errors.New("get failed"))
	user, err := repo.GetUserByUsername("x")
	assert.Error(t, err)
	assert.Nil(t, user)
	mockStore.AssertExpectations(t)
}

func TestGetUser_UnmarshalError(t *testing.T) {
	mockStore := new(MockKVStore)
	uow := db.NewUnitOfWork(mockStore, nil)
	iGF := NewTestIDGeneratorFactoryRepository([]string{"123"})
	repo, err := db.NewUserRepository(uow, iGF)

	mockStore.On("Get", db.AdminFC, "admin_schema:users:idx-u:Username:x", mock.Anything).Return([]byte("123"), nil)
	mockStore.On("Get", db.AdminFC, "admin_schema:users:data:123", mock.Anything).Return([]byte("invalid-json"), nil)

	user, err := repo.GetUserByUsername("x")
	assert.Error(t, err)
	assert.Empty(t, user)
	mockStore.AssertExpectations(t)
}

func TestDeleteUser_Success(t *testing.T) {
	mockStore := new(MockKVStore)
	uow := db.NewUnitOfWork(mockStore, nil)
	iGF := NewTestIDGeneratorFactoryRepository([]string{"123"})
	repo, err := db.NewUserRepository(uow, iGF)
	root := models.User{Username: "other", ID: "123"}
	rootData, _ := json.Marshal(root)

	mockStore.On("Get", db.AdminFC, "admin_schema:users:idx-u:Username:bob", mock.Anything).Return([]byte("123"), nil)
	mockStore.On("Get", db.AdminFC, "admin_schema:users:data:123", mock.Anything).Return(rootData, nil)

	result, err := repo.DeleteUser("bob")
	assert.Equal(t, true, result)
	assert.NoError(t, err)
	mockStore.AssertExpectations(t)
}

func TestDeleteUser_CannotDeleteRoot(t *testing.T) {
	mockStore := new(MockKVStore)
	uow := db.NewUnitOfWork(mockStore, nil)
	iGF := NewTestIDGeneratorFactoryRepository([]string{"123"})
	repo, err := db.NewUserRepository(uow, iGF)
	root := models.User{Username: "admin", ID: "123", IsRootUser: true}
	rootData, _ := json.Marshal(root)
	mockStore.On("Get", db.AdminFC, "admin_schema:users:idx-u:Username:admin", mock.Anything).Return([]byte("123"), nil)
	mockStore.On("Get", db.AdminFC, "admin_schema:users:data:123", mock.Anything).Return(rootData, nil)
	_, err = repo.DeleteUser("admin")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot delete root user")
	mockStore.AssertExpectations(t)
}

func TestDeleteUser_GetError(t *testing.T) {
	mockStore := new(MockKVStore)
	uow := db.NewUnitOfWork(mockStore, nil)
	iGF := NewTestIDGeneratorFactoryRepository([]string{"123"})
	repo, err := db.NewUserRepository(uow, iGF)
	mockStore.On("Get", db.AdminFC, mock.Anything, mock.Anything).Return(nil, errors.New("get failed"))

	_, err = repo.DeleteUser("someone")
	assert.Error(t, err)
	mockStore.AssertExpectations(t)
}

func TestDeleteUser_UnmarshalRootError(t *testing.T) {
	mockStore := new(MockKVStore)
	uow := db.NewUnitOfWork(mockStore, nil)
	iGF := NewTestIDGeneratorFactoryRepository([]string{"123"})
	repo, err := db.NewUserRepository(uow, iGF)
	mockStore.On("Get", db.AdminFC, "admin_schema:users:idx-u:Username:x", mock.Anything).Return([]byte("123"), nil)
	mockStore.On("Get", db.AdminFC, "admin_schema:users:data:123", mock.Anything).Return([]byte("invalid-json"), nil)
	_, err = repo.DeleteUser("x")
	assert.Error(t, err)
	mockStore.AssertExpectations(t)
}

func TestDeleteUser_WriteError(t *testing.T) {
	mockStore := new(MockKVStore)
	uow := db.NewUnitOfWork(mockStore, nil)
	iGF := NewTestIDGeneratorFactoryRepository([]string{"123"})
	repo, err := db.NewUserRepository(uow, iGF)
	root := models.User{Username: "user", ID: "123"}
	rootData, _ := json.Marshal(root)
	mockStore.On("Get", db.AdminFC, "admin_schema:users:idx-u:Username:user", mock.Anything).Return([]byte("123"), nil)
	mockStore.On("Get", db.AdminFC, "admin_schema:users:data:123", mock.Anything).Return(rootData, nil)

	mockStore.On("Write", mock.Anything, mock.Anything).Return(errors.New("write failed"))

	_, err = repo.DeleteUser("user")
	assert.NoError(t, err)
	err = uow.Commit(time.Now())
	assert.Error(t, err)
	mockStore.AssertExpectations(t)
}
func TestPutUser_KVStorePutError(t *testing.T) {
	mockStore := new(MockKVStore)
	uow := db.NewUnitOfWork(mockStore, nil)
	iGF := NewTestIDGeneratorFactoryRepository([]string{"123"})
	repo, err := db.NewUserRepository(uow, iGF)
	userInput := models.CreateUser{
		Username: "testuser",
		Password: "password123",
		Email:    "test@example.com",
	}

	mockStore.On("SearchByPatternPaginatedKV", db.AdminFC, "admin_schema:users:idx:IsRootUser:true:*", "", 1, mock.Anything).Return(nil, "", nil)
	mockStore.On("Get", db.AdminFC, "admin_schema:users:idx-u:Username:testuser", mock.Anything).Return(nil, nil).Times(1)
	mockStore.On("Get", db.AdminFC, "admin_schema:users:idx-u:Email:test@example.com", mock.Anything).Return(nil, nil).Times(1)
	mockStore.On("Get", db.AdminFC, "admin_schema:users:data:123", mock.Anything).Return(nil, nil).Times(1)
	mockStore.On("Write", mock.Anything, mock.Anything).Return(errors.New("kv put failed")).Times(1)

	_, err = repo.CreateUser(userInput)

	assert.NoError(t, err)
	err = uow.Commit(time.Now())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "kv put failed")
	mockStore.AssertExpectations(t)
}

func TestLoginUser(t *testing.T) {
	mockStore := new(MockKVStore)
	uow := db.NewUnitOfWork(mockStore, nil)
	iGF := NewTestIDGeneratorFactoryRepository([]string{"123"})
	repo, err := db.NewUserRepository(uow, iGF)
	assert.NoError(t, err)

	userEmail := "test@example.com"
	userUsername := "testuser"
	correctPassword := "password123"
	incorrectPassword := "wrongpassword"
	userID := "user123"

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(correctPassword), bcrypt.DefaultCost)

	user := models.User{
		ID:           userID,
		Username:     userUsername,
		Email:        userEmail,
		PasswordHash: string(hashedPassword),
	}
	userData, _ := json.Marshal(user)

	t.Run("LoginWithEmail_CorrectPassword_Success", func(t *testing.T) {
		mockStore.On("Get", db.AdminFC, "admin_schema:users:idx-u:Email:"+userEmail, mock.Anything).Return([]byte(userID), nil).Once()
		mockStore.On("Get", db.AdminFC, "admin_schema:users:data:"+userID, mock.Anything).Return(userData, nil).Once()

		found, err := repo.Login(userEmail, correctPassword)
		assert.NoError(t, err)
		assert.True(t, found)
		mockStore.AssertExpectations(t)
	})

	t.Run("LoginWithUsername_CorrectPassword_Success", func(t *testing.T) {
		// Mock email lookup to return nil, then username lookup to return the user
		mockStore.On("Get", db.AdminFC, "admin_schema:users:idx-u:Email:"+userUsername, mock.Anything).Return(nil, nil).Once()
		mockStore.On("Get", db.AdminFC, "admin_schema:users:idx-u:Username:"+userUsername, mock.Anything).Return([]byte(userID), nil).Once()
		mockStore.On("Get", db.AdminFC, "admin_schema:users:data:"+userID, mock.Anything).Return(userData, nil).Once()

		found, err := repo.Login(userUsername, correctPassword)
		assert.NoError(t, err)
		assert.True(t, found)
		mockStore.AssertExpectations(t)
	})

	t.Run("LoginWithEmail_IncorrectPassword_Failure", func(t *testing.T) {
		mockStore.On("Get", db.AdminFC, "admin_schema:users:idx-u:Email:"+userEmail, mock.Anything).Return([]byte(userID), nil).Once()
		mockStore.On("Get", db.AdminFC, "admin_schema:users:data:"+userID, mock.Anything).Return(userData, nil).Once()

		found, err := repo.Login(userEmail, incorrectPassword)
		assert.NoError(t, err) // bcrypt.ErrMismatchedHashAndPassword is not an "error" for Login logic, it's a valid outcome
		assert.False(t, found)
		mockStore.AssertExpectations(t)
	})

	t.Run("LoginWithUsername_IncorrectPassword_Failure", func(t *testing.T) {
		mockStore.On("Get", db.AdminFC, "admin_schema:users:idx-u:Email:"+userUsername, mock.Anything).Return(nil, nil).Once()
		mockStore.On("Get", db.AdminFC, "admin_schema:users:idx-u:Username:"+userUsername, mock.Anything).Return([]byte(userID), nil).Once()
		mockStore.On("Get", db.AdminFC, "admin_schema:users:data:"+userID, mock.Anything).Return(userData, nil).Once()

		found, err := repo.Login(userUsername, incorrectPassword)
		assert.NoError(t, err) // bcrypt.ErrMismatchedHashAndPassword is not an "error" for Login logic
		assert.False(t, found)
		mockStore.AssertExpectations(t)
	})

	t.Run("Login_UserNotFound_Failure", func(t *testing.T) {
		unknownIdentifier := "unknown@example.com"
		mockStore.On("Get", db.AdminFC, "admin_schema:users:idx-u:Email:"+unknownIdentifier, mock.Anything).Return(nil, nil).Once()
		mockStore.On("Get", db.AdminFC, "admin_schema:users:idx-u:Username:"+unknownIdentifier, mock.Anything).Return(nil, nil).Once()

		found, err := repo.Login(unknownIdentifier, "anypassword")
		assert.NoError(t, err)
		assert.False(t, found)
		mockStore.AssertExpectations(t)
	})

	t.Run("Login_ErrorOnEmailLookup", func(t *testing.T) {
		errorIdentifier := "error@example.com"
		expectedError := errors.New("db error on email lookup")
		mockStore.On("Get", db.AdminFC, "admin_schema:users:idx-u:Email:"+errorIdentifier, mock.Anything).Return(nil, expectedError).Once()

		found, err := repo.Login(errorIdentifier, "anypassword")
		assert.Error(t, err)
		assert.Equal(t, expectedError, err)
		assert.False(t, found)
		mockStore.AssertExpectations(t)
	})

	t.Run("Login_ErrorOnUsernameLookup", func(t *testing.T) {
		errorIdentifier := "erroruser"
		expectedError := errors.New("db error on username lookup")
		mockStore.On("Get", db.AdminFC, "admin_schema:users:idx-u:Email:"+errorIdentifier, mock.Anything).Return(nil, nil).Once() // Email lookup fine, returns nil
		mockStore.On("Get", db.AdminFC, "admin_schema:users:idx-u:Username:"+errorIdentifier, mock.Anything).Return(nil, expectedError).Once()

		found, err := repo.Login(errorIdentifier, "anypassword")
		assert.Error(t, err)
		assert.Equal(t, expectedError, err)
		assert.False(t, found)
		mockStore.AssertExpectations(t)
	})

	// Test case for bcrypt error other than ErrMismatchedHashAndPassword (though hard to simulate without specific bcrypt internals)
	// For now, the existing error handling in Login method should cover general errors from bcrypt.CompareHashAndPassword
}
