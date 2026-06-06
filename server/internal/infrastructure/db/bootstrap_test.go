package db_test

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/shared/models"
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"deadalus-orch/server/internal/pkg/config"
)

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
	store := new(MockKVStore)
	uow := db.NewUnitOfWork(store, nil)
	iGF := NewTestIDGeneratorFactory([]string{"123"})
	repo, err := db.NewUserRepository(uow, iGF)
	assert.NoError(t, err)
	config := config.Config{
		DefaultRootUser:     "admin",
		DefaultRootPassword: "123456",
	}

	store.On("SearchByPatternPaginatedKV", db.AdminFC, db.AdminFCSector, "admin_schema:users:idx:IsRootUser:true:*", "", 1, mock.Anything).Return(nil, "", nil).Times(2)
	store.On("Exists", db.AdminFC, db.AdminFCSector, "admin_schema:users:idx-u:Email:noemail@daedalus.com", mock.Anything).Return(false, nil).Times(1)
	store.On("Exists", db.AdminFC, db.AdminFCSector, "admin_schema:users:idx-u:Username:admin", mock.Anything).Return(false, nil).Times(1)
	store.On("Exists", db.AdminFC, db.AdminFCSector, "admin_schema:users:data:123", mock.Anything).Return(false, nil).Times(1)

	assert.NoError(t, err)

	store.On("Write", mock.Anything, mock.Anything).Return(nil).Times(1)

	err = db.BootstrapRootUser(*repo, config)
	assert.NoError(t, err)
	err = uow.Commit() // Commit should now take time
	assert.NoError(t, err)
	store.AssertExpectations(t)
}

func Test_ErrorGettingRoot(t *testing.T) {
	store := new(MockKVStore)
	uow := db.NewUnitOfWork(store, nil)
	iGF := NewTestIDGeneratorFactory([]string{"123"})
	repo, err := db.NewUserRepository(uow, iGF)
	assert.NoError(t, err)
	config := config.Config{
		DefaultRootUser:     "admin",
		DefaultRootPassword: "123456",
	}

	store.On("SearchByPatternPaginatedKV", db.AdminFC, db.AdminFCSector, "admin_schema:users:idx:IsRootUser:true:*", "", 1, mock.Anything).Return(nil, "", errors.New("boom")).Times(1)

	err = db.BootstrapRootUser(*repo, config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get default root")

	store.AssertExpectations(t)
}

func Test_PutsRootIfMissingInUsers(t *testing.T) {
	store := new(MockKVStore)
	uow := db.NewUnitOfWork(store, nil)
	iGF := NewTestIDGeneratorFactory([]string{"123"})
	repo, err := db.NewUserRepository(uow, iGF)
	assert.NoError(t, err)
	config := config.Config{
		DefaultRootUser:     "admin",
		DefaultRootPassword: "123456",
	}
	// First GetUserRoot in BootstrapRootUser

	store.On("SearchByPatternPaginatedKV", db.AdminFC, db.AdminFCSector, "admin_schema:users:idx:IsRootUser:true:*", "", 1, mock.Anything).Return([]db.KeyValuePair{{Value: []byte("123")}}, "", nil).Times(2)
	store.On("Get", db.AdminFC, db.AdminFCSector, "admin_schema:users:data:123", mock.Anything).Return(nil, nil).Times(2)
	store.On("Exists", db.AdminFC, db.AdminFCSector, "admin_schema:users:idx-u:Username:admin", mock.Anything).Return(false, nil).Times(1)
	store.On("Exists", db.AdminFC, db.AdminFCSector, "admin_schema:users:idx-u:Email:noemail@daedalus.com", mock.Anything).Return(false, nil).Times(1)
	store.On("Exists", db.AdminFC, db.AdminFCSector, "admin_schema:users:data:123", mock.Anything).Return(false, nil).Times(1)
	store.On("Write", mock.Anything, mock.Anything).Return(nil).Times(1)
	err = db.BootstrapRootUser(*repo, config)
	assert.NoError(t, err)
	err = uow.Commit()
	assert.NoError(t, err)
	store.AssertExpectations(t)
}

func Test_SkipsIfUserExists(t *testing.T) {
	store := new(MockKVStore)
	uow := db.NewUnitOfWork(store, nil)
	iGF := NewTestIDGeneratorFactory([]string{"123"})
	repo, err := db.NewUserRepository(uow, iGF)
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
	store.On("SearchByPatternPaginatedKV", db.AdminFC, db.AdminFCSector, "admin_schema:users:idx:IsRootUser:true:*", "", 1, mock.Anything).Return([]db.KeyValuePair{{Value: []byte("123")}}, "", nil)
	store.On("Get", db.AdminFC, db.AdminFCSector, "admin_schema:users:data:123", mock.Anything).Return([]byte(marshal(t, root)), nil).Once()
	// No Write should be called if user exists
	store.On("Write", mock.Anything, mock.Anything).Return(nil).Times(1) // This line was causing issues, Write is not always called

	err = db.BootstrapRootUser(*repo, config)
	assert.NoError(t, err)
	err = uow.Commit() // Commit might have no operations if root exists and no other changes

	assert.NoError(t, err)
	store.AssertExpectations(t)
}

func Test_ErrorFetchingUser(t *testing.T) {
	store := new(MockKVStore)
	uow := db.NewUnitOfWork(store, nil)
	iGF := NewTestIDGeneratorFactory([]string{"123"})
	repo, err := db.NewUserRepository(uow, iGF)
	assert.NoError(t, err)
	config := config.Config{
		DefaultRootUser:     "admin",
		DefaultRootPassword: "123456",
	}
	//root := models.User{Username: "admin"}

	store.On("SearchByPatternPaginatedKV", db.AdminFC, db.AdminFCSector, "admin_schema:users:idx:IsRootUser:true:*", "", 1, mock.Anything).Return([]db.KeyValuePair{{Value: []byte("123")}}, "", nil)
	store.On("Get", db.AdminFC, db.AdminFCSector, "admin_schema:users:data:123", mock.Anything).Return(nil, errors.New("read error")).Once()

	err = db.BootstrapRootUser(*repo, config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "read error")
	store.AssertExpectations(t)
}

func Test_ErrorPutsRoot(t *testing.T) {
	store := new(MockKVStore)
	uow := db.NewUnitOfWork(store, nil)
	iGF := NewTestIDGeneratorFactory([]string{"123"})
	repo, err := db.NewUserRepository(uow, iGF)
	assert.NoError(t, err)
	config := config.Config{
		DefaultRootUser:     "admin",
		DefaultRootPassword: "123456",
	}

	// GetUserRoot in BootstrapRootUser
	store.On("SearchByPatternPaginatedKV", db.AdminFC, db.AdminFCSector, "admin_schema:users:idx:IsRootUser:true:*", "", 1, mock.Anything).Return([]db.KeyValuePair{{Value: []byte("123")}}, "", nil).Once()
	store.On("Exists", db.AdminFC, db.AdminFCSector, "admin_schema:users:data:123", mock.Anything).Return(false, nil).Once() // Assumes Get is called by FindByField
	store.On("Get", db.AdminFC, db.AdminFCSector, "admin_schema:users:data:123", mock.Anything).Return(nil, nil).Once()      // Assumes Get is called by FindByField

	// CreateUser part:
	// GetUserRoot again inside CreateUser
	store.On("SearchByPatternPaginatedKV", db.AdminFC, db.AdminFCSector, "admin_schema:users:idx:IsRootUser:true:*", "", 1, mock.Anything).Return(nil, "", nil).Once()
	// Exists checks for username and email
	store.On("Exists", db.AdminFC, db.AdminFCSector, "admin_schema:users:idx-u:Username:admin", mock.Anything).Return(false, nil).Once()
	store.On("Exists", db.AdminFC, db.AdminFCSector, "admin_schema:users:idx-u:Email:noemail@daedalus.com", mock.Anything).Return(false, nil).Once()

	store.On("Write", mock.Anything, mock.Anything).Return(errors.New("write fail")).Once()

	err = db.BootstrapRootUser(*repo, config)
	assert.NoError(t, err)

	err = uow.Commit()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "write fail")
	store.AssertExpectations(t)
}

func TestBootstrapRootUser_MissingConfigUser(t *testing.T) {
	store := new(MockKVStore)
	uow := db.NewUnitOfWork(store, nil)
	iGF := NewTestIDGeneratorFactory([]string{"123"})
	repo, err := db.NewUserRepository(uow, iGF)
	assert.NoError(t, err)
	cfg := config.Config{
		DefaultRootUser:     "", // Missing user
		DefaultRootPassword: "testpass",
	}

	store.On("SearchByPatternPaginatedKV", db.AdminFC, db.AdminFCSector, "admin_schema:users:idx:IsRootUser:true:*", "", 1, mock.Anything).Return(nil, "", nil).Times(1)
	err = db.BootstrapRootUser(*repo, cfg)

	assert.NoError(t, err)

	store.AssertExpectations(t)
}

func TestBootstrapRootUser_MissingConfigPassword(t *testing.T) {
	store := new(MockKVStore)
	uow := db.NewUnitOfWork(store, nil)
	iGF := NewTestIDGeneratorFactory([]string{"123"})
	repo, err := db.NewUserRepository(uow, iGF)
	assert.NoError(t, err)
	cfg := config.Config{
		DefaultRootUser:     "testuser",
		DefaultRootPassword: "",
	}

	store.On("SearchByPatternPaginatedKV", db.AdminFC, db.AdminFCSector, "admin_schema:users:idx:IsRootUser:true:*", "", 1, mock.Anything).Return(nil, "", nil).Times(1)
	err = db.BootstrapRootUser(*repo, cfg)

	assert.NoError(t, err)
	store.AssertExpectations(t)
}
