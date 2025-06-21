package db_test

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/shared/models"
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/linxGnu/grocksdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"deadalus-orch/server/internal/pkg/config"
)

type MockKVStoreBootstrap struct {
	mock.Mock
	ColumnFamilyHandles    map[string]*grocksdb.ColumnFamilyHandle // Map of regular column family names to their handles.
	TTLColumnFamilyHandles map[string]*grocksdb.ColumnFamilyHandle // Map of TTL column family names to their handles.
}

func (m *MockKVStoreBootstrap) Get(AdminFC, key string, now time.Time) ([]byte, error) {
	args := m.Called(AdminFC, key, now)
	var s []byte
	if tmp := args.Get(0); tmp != nil {
		s = tmp.([]byte)
	}
	return s, args.Error(1)
}

func (m *MockKVStoreBootstrap) Delete(AdminFC, key string, now time.Time) error {
	args := m.Called(AdminFC, key, now)
	return args.Error(0)
}

func (r *MockKVStoreBootstrap) Exists(columnFamily, key string, now time.Time) (bool, error) {
	// This mock Exists calls its own Get.
	data, err := r.Get(columnFamily, key, now) // Pass now here
	if err != nil {
		return false, err
	}
	return data != nil, nil
	// If directly mocking Exists:
	// args := r.Called(columnFamily, key, now)
	// return args.Bool(0), args.Error(1)
}

func (m *MockKVStoreBootstrap) Put(AdminFC, key string, value []byte, ttl int, now time.Time) error {
	args := m.Called(AdminFC, key, value, ttl, now)
	return args.Error(0)
}

func (m *MockKVStoreBootstrap) PutRaw(AdminFC, key string, value []byte) error {
	args := m.Called(AdminFC, key, value)
	return args.Error(0)
}

func (m *MockKVStoreBootstrap) Write(batch *db.WriteBatch, now time.Time) error {
	args := m.Called(batch, now)
	return args.Error(0)
}

func (m *MockKVStoreBootstrap) WriteRaw(batch *db.WriteBatch) error {
	args := m.Called(batch)
	return args.Error(0)
}

func (m *MockKVStoreBootstrap) DumpAll() (interface{}, error) {
	args := m.Called()
	var s []byte
	if tmp := args.Get(0); tmp != nil {
		s = tmp.([]byte)
	}
	return s, args.Error(1)
}

func (r *MockKVStoreBootstrap) Iterate(fn func(cfName string, key, value []byte) error) error {
	return nil
}

func (r *MockKVStoreBootstrap) ClearAll() error {
	return nil
}

func (r *MockKVStoreBootstrap) Flush() error {
	return nil
}

func (r *MockKVStoreBootstrap) Close() error {
	return nil
}

func (r *MockKVStoreBootstrap) CleanExpiredKeys(now time.Time) error {
	args := r.Called(now)
	return args.Error(0)
}

func (m *MockKVStoreBootstrap) SearchByPatternPaginatedKV(cfName, pattern, cursor string, limit int, now time.Time) ([]db.KeyValuePair, string, error) {
	args := m.Called(cfName, pattern, cursor, limit, now)
	var s []db.KeyValuePair
	if tmp := args.Get(0); tmp != nil {
		s = tmp.([]db.KeyValuePair)
	}
	return s, "", args.Error(2)
}

func TestMain(m *testing.M) {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	code := m.Run()
	os.Exit(code)
}

func marshal(t *testing.T, v interface{}) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	return data
}

func Test_CreatesRootIfMissing(t *testing.T) {
	store := new(MockKVStoreBootstrap)
	uow := db.NewUnitOfWork(store)
	repo, err := db.NewUserRepository(uow)
	assert.NoError(t, err)
	config := config.Config{
		DefaultRootUser:     "admin",
		DefaultRootPassword: "123456",
	}

	// GetUserRoot (calls FindByField, which calls SearchByPatternPaginatedKV or Get)
	// First call to GetUserRoot (is root missing?)
	store.On("SearchByPatternPaginatedKV", db.AdminFC, "admin_schema:users:idx:IsRootUser:true:*", "", 1, mock.AnythingOfType("time.Time")).Return(nil, "", nil).Once()
	// CreateUser part:
	// GetUserRoot again inside CreateUser
	store.On("SearchByPatternPaginatedKV", db.AdminFC, "admin_schema:users:idx:IsRootUser:true:*", "", 1, mock.AnythingOfType("time.Time")).Return(nil, "", nil).Once()
	// Exists checks for username and email (these are simplified, actual repo uses Get for unique index)
	store.On("Exists", db.AdminFC, "admin_schema:users:idx-u:Username:admin", mock.AnythingOfType("time.Time")).Return(false, nil).Once()
	store.On("Exists", db.AdminFC, "admin_schema:users:idx-u:Email:noemail@daedalus.com", mock.AnythingOfType("time.Time")).Return(false, nil).Once()
	// Write for CreateUser
	store.On("Write", mock.Anything, mock.AnythingOfType("time.Time")).Return(nil).Once()

	err = db.BootstrapRootUser(*repo, config)
	assert.NoError(t, err)
	err = uow.Commit(time.Now()) // Commit should now take time
	assert.NoError(t, err)
	store.AssertExpectations(t)
}

func Test_ErrorGettingRoot(t *testing.T) {
	store := new(MockKVStoreBootstrap)
	uow := db.NewUnitOfWork(store)
	repo, err := db.NewUserRepository(uow)
	assert.NoError(t, err)
	config := config.Config{
		DefaultRootUser:     "admin",
		DefaultRootPassword: "123456",
	}

	store.On("SearchByPatternPaginatedKV", db.AdminFC, "admin_schema:users:idx:IsRootUser:true:*", "", 1, mock.AnythingOfType("time.Time")).Return(nil, "", errors.New("boom")).Times(1)

	err = db.BootstrapRootUser(*repo, config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get default root")

	store.AssertExpectations(t)
}

func Test_PutsRootIfMissingInUsers(t *testing.T) {
	store := new(MockKVStoreBootstrap)
	uow := db.NewUnitOfWork(store)
	repo, err := db.NewUserRepository(uow)
	assert.NoError(t, err)
	config := config.Config{
		DefaultRootUser:     "admin",
		DefaultRootPassword: "123456",
	}
	// First GetUserRoot in BootstrapRootUser
	store.On("SearchByPatternPaginatedKV", db.AdminFC, "admin_schema:users:idx:IsRootUser:true:*", "", 1, mock.AnythingOfType("time.Time")).Return([]db.KeyValuePair{{Value: []byte("123")}}, "", nil).Once()
	store.On("Get", db.AdminFC, "admin_schema:users:data:123", mock.AnythingOfType("time.Time")).Return(nil, nil).Once() // Assumes Get is called by FindByField

	// CreateUser part:
	// GetUserRoot again inside CreateUser
	store.On("SearchByPatternPaginatedKV", db.AdminFC, "admin_schema:users:idx:IsRootUser:true:*", "", 1, mock.AnythingOfType("time.Time")).Return(nil, "", nil).Once() // No root user found this time for CreateUser's internal check
	// Exists checks for username and email
	store.On("Exists", db.AdminFC, "admin_schema:users:idx-u:Username:admin", mock.AnythingOfType("time.Time")).Return(false, nil).Once()
	store.On("Exists", db.AdminFC, "admin_schema:users:idx-u:Email:noemail@daedalus.com", mock.AnythingOfType("time.Time")).Return(false, nil).Once()
	// Write for CreateUser
	store.On("Write", mock.Anything, mock.AnythingOfType("time.Time")).Return(nil).Once()


	err = db.BootstrapRootUser(*repo, config)
	assert.NoError(t, err)
	err = uow.Commit(time.Now())
	assert.NoError(t, err)
	store.AssertExpectations(t)
}

func Test_SkipsIfUserExists(t *testing.T) {
	store := new(MockKVStoreBootstrap)
	uow := db.NewUnitOfWork(store)
	repo, err := db.NewUserRepository(uow)
	assert.NoError(t, err)
	config := config.Config{
		DefaultRootUser:     "admin",
		DefaultRootPassword: "123456",
	}

	root := models.User{
		Username:     "admin",
		PasswordHash: "hash",
		Email:        "x@x.com",
	}

	// GetUserRoot in BootstrapRootUser
	store.On("SearchByPatternPaginatedKV", db.AdminFC, "admin_schema:users:idx:IsRootUser:true:*", "", 1, mock.AnythingOfType("time.Time")).Return([]db.KeyValuePair{{Value: []byte("123")}}, "", nil)
	store.On("Get", db.AdminFC, "admin_schema:users:data:123", mock.AnythingOfType("time.Time")).Return([]byte(marshal(t, root)), nil).Once()
	// No Write should be called if user exists
	// store.On("Write", mock.Anything, mock.AnythingOfType("time.Time")).Return(nil).Times(1) // This line was causing issues, Write is not always called

	err = db.BootstrapRootUser(*repo, config)
	assert.NoError(t, err)
	err = uow.Commit(time.Now()) // Commit might have no operations if root exists and no other changes

	assert.NoError(t, err)
	store.AssertExpectations(t)
}

func Test_ErrorFetchingUser(t *testing.T) {
	store := new(MockKVStoreBootstrap)
	uow := db.NewUnitOfWork(store)
	repo, err := db.NewUserRepository(uow)
	assert.NoError(t, err)
	config := config.Config{
		DefaultRootUser:     "admin",
		DefaultRootPassword: "123456",
	}
	//root := models.User{Username: "admin"}

	store.On("SearchByPatternPaginatedKV", db.AdminFC, "admin_schema:users:idx:IsRootUser:true:*", "", 1, mock.AnythingOfType("time.Time")).Return([]db.KeyValuePair{{Value: []byte("123")}}, "", nil)
	store.On("Get", db.AdminFC, "admin_schema:users:data:123", mock.AnythingOfType("time.Time")).Return(nil, errors.New("read error")).Once()

	err = db.BootstrapRootUser(*repo, config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "read error")
	store.AssertExpectations(t)
}

func Test_ErrorPutsRoot(t *testing.T) {
	store := new(MockKVStoreBootstrap)
	uow := db.NewUnitOfWork(store)
	repo, err := db.NewUserRepository(uow)
	assert.NoError(t, err)
	config := config.Config{
		DefaultRootUser:     "admin",
		DefaultRootPassword: "123456",
	}

	// GetUserRoot in BootstrapRootUser
	store.On("SearchByPatternPaginatedKV", db.AdminFC, "admin_schema:users:idx:IsRootUser:true:*", "", 1, mock.AnythingOfType("time.Time")).Return([]db.KeyValuePair{{Value: []byte("123")}}, "", nil).Once()
	store.On("Get", db.AdminFC, "admin_schema:users:data:123", mock.AnythingOfType("time.Time")).Return(nil, nil).Once() // Assumes Get is called by FindByField

	// CreateUser part:
	// GetUserRoot again inside CreateUser
	store.On("SearchByPatternPaginatedKV", db.AdminFC, "admin_schema:users:idx:IsRootUser:true:*", "", 1, mock.AnythingOfType("time.Time")).Return(nil, "", nil).Once()
	// Exists checks for username and email
	store.On("Exists", db.AdminFC, "admin_schema:users:idx-u:Username:admin", mock.AnythingOfType("time.Time")).Return(false, nil).Once()
	store.On("Exists", db.AdminFC, "admin_schema:users:idx-u:Email:noemail@daedalus.com", mock.AnythingOfType("time.Time")).Return(false, nil).Once()

	store.On("Write", mock.Anything, mock.AnythingOfType("time.Time")).Return(errors.New("write fail")).Once()

	err = db.BootstrapRootUser(*repo, config)
	assert.NoError(t, err)

	err = uow.Commit(time.Now())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "write fail")
	store.AssertExpectations(t)
}

func TestBootstrapRootUser_MissingConfigUser(t *testing.T) {
	store := new(MockKVStoreBootstrap)
	uow := db.NewUnitOfWork(store)
	repo, err := db.NewUserRepository(uow)
	assert.NoError(t, err)
	cfg := config.Config{
		DefaultRootUser:     "", // Missing user
		DefaultRootPassword: "testpass",
	}

	store.On("SearchByPatternPaginatedKV", db.AdminFC, "admin_schema:users:idx:IsRootUser:true:*", "", 1, mock.AnythingOfType("time.Time")).Return(nil, "", nil).Times(1)
	err = db.BootstrapRootUser(*repo, cfg)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing default root user/password")

	store.AssertExpectations(t)
}

func TestBootstrapRootUser_MissingConfigPassword(t *testing.T) {
	store := new(MockKVStoreBootstrap)
	uow := db.NewUnitOfWork(store)
	repo, err := db.NewUserRepository(uow)
	assert.NoError(t, err)
	cfg := config.Config{
		DefaultRootUser:     "testuser",
		DefaultRootPassword: "",
	}

	store.On("SearchByPatternPaginatedKV", db.AdminFC, "admin_schema:users:idx:IsRootUser:true:*", "", 1, mock.AnythingOfType("time.Time")).Return(nil, "", nil).Times(1)
	err = db.BootstrapRootUser(*repo, cfg)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing default root user/password")
	store.AssertExpectations(t)
}
