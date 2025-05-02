package db_test

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/shared/models"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/linxGnu/grocksdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"

	constants "deadalus-orch/shared/constants"

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
	config := config.Config{
		DefaultRootUser:     "admin",
		DefaultRootPassword: "123456",
	}

	store.On("Get", db.AdminFC, constants.DefaultRootUserRootKey).Return(nil, nil).Times(1)

	input := models.CreateUser{
		Username: config.DefaultRootUser,
		Email:    "noemail@daedalus.com",
		Password: config.DefaultRootPassword,
	}
	defaultUserData, err := json.Marshal(input)
	assert.NoError(t, err)

	batch := grocksdb.NewWriteBatch()
	defer batch.Destroy()
	batch.Put([]byte(constants.DefaultRootUserRootKey), defaultUserData)
	userKey := fmt.Sprintf("user:%s", input.Username)

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	assert.NoError(t, err)

	user := models.User{
		Username:     input.Username,
		Email:        input.Email,
		PasswordHash: string(hash),
	}

	userData, err := json.Marshal(user)
	assert.NoError(t, err)

	batch.Put([]byte(userKey), userData)

	store.On("Write", batch).Return(nil).Times(1)

	err = db.BootstrapRootUser(store, config)
	assert.NoError(t, err)
	store.AssertExpectations(t)
}

func Test_ErrorGettingRoot(t *testing.T) {
	store := new(MockKVStore)
	config := config.Config{
		DefaultRootUser:     "admin",
		DefaultRootPassword: "123456",
	}

	store.On("Get", db.AdminFC, constants.DefaultRootUserRootKey).Return(nil, errors.New("boom")).Once()

	err := db.BootstrapRootUser(store, config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get default root")
	store.AssertExpectations(t)
}

func Test_PutsRootIfMissingInUsers(t *testing.T) {
	store := new(MockKVStore)
	config := config.Config{
		DefaultRootUser:     "admin",
		DefaultRootPassword: "123456",
	}
	root := models.CreateUser{
		Username: "admin",
		Password: "123456",
		Email:    "x@x.com",
	}

	store.On("Get", db.AdminFC, constants.DefaultRootUserRootKey).Return([]byte(marshal(t, root)), nil).Once()
	store.On("Get", db.AdminFC, "user:admin").Return(nil, nil).Once()
	store.On("Put", db.AdminFC, "user:admin", mock.AnythingOfType("[]uint8")).Return(nil).Once()

	err := db.BootstrapRootUser(store, config)
	assert.NoError(t, err)
	store.AssertExpectations(t)
}

func Test_SkipsIfUserExists(t *testing.T) {
	store := new(MockKVStore)
	config := config.Config{
		DefaultRootUser:     "admin",
		DefaultRootPassword: "123456",
	}

	root := models.User{
		Username:     "admin",
		PasswordHash: "hash",
		Email:        "x@x.com",
	}

	store.On("Get", db.AdminFC, constants.DefaultRootUserRootKey).Return([]byte(marshal(t, root)), nil).Once()
	store.On("Get", db.AdminFC, "user:admin").Return([]byte(marshal(t, root)), nil).Once()

	err := db.BootstrapRootUser(store, config)
	assert.NoError(t, err)
	store.AssertExpectations(t)
}

func Test_ErrorFetchingUser(t *testing.T) {
	store := new(MockKVStore)
	config := config.Config{
		DefaultRootUser:     "admin",
		DefaultRootPassword: "123456",
	}
	root := models.User{Username: "admin"}

	store.On("Get", db.AdminFC, constants.DefaultRootUserRootKey).Return([]byte(marshal(t, root)), nil).Once()
	store.On("Get", db.AdminFC, "user:admin").Return(nil, errors.New("read error")).Once()

	err := db.BootstrapRootUser(store, config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "read error")
	store.AssertExpectations(t)
}

func Test_ErrorPutsRoot(t *testing.T) {
	store := new(MockKVStore)
	config := config.Config{
		DefaultRootUser:     "admin",
		DefaultRootPassword: "123456",
	}

	store.On("Get", db.AdminFC, constants.DefaultRootUserRootKey).Return(nil, nil).Times(1)
	store.On("Write", mock.Anything).Return(errors.New("write fail")).Once()

	err := db.BootstrapRootUser(store, config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "write fail")
	store.AssertExpectations(t)
}
