package db_test

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

type User struct {
	ID   string `orm:"primaryKey"`
	Name string `orm:"unique"`
}

func (User) TableName() string {
	return "users"
}

func TestRepository_Create_Success(t *testing.T) {
	mockStore := new(MockKVStore)

	user := User{
		ID:   "123",
		Name: "Alice",
	}

	repo, err := db.NewRepository[User](mockStore, "cf1", "admin")
	assert.NoError(t, err)

	data, _ := json.Marshal(user)
	dataKey := "admin:users:data:123"
	indexKey := "admin:users:idx:Name:Alice:123"

	mockStore.On("Get", "cf1", indexKey).Return(nil, nil)
	mockStore.On("Put", "cf1", indexKey, []byte("123")).Return(nil)
	mockStore.On("Put", "cf1", dataKey, data).Return(nil)

	err = repo.Create(user)
	assert.NoError(t, err)

	mockStore.AssertExpectations(t)
}

func TestRepository_Create_DuplicateUnique(t *testing.T) {
	mockStore := new(MockKVStore)

	user := User{
		ID:   "123",
		Name: "Alice",
	}

	repo, err := db.NewRepository[User](mockStore, "cf1", "admin")
	assert.NoError(t, err)

	indexKey := "admin:users:idx:Name:Alice:123"
	mockStore.On("Get", "cf1", indexKey).Return([]byte("123"), nil)

	err = repo.Create(user)
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
	mockStore := new(MockKVStore)

	_, err := db.NewRepository[NoPrimary](mockStore, "cf1", "admin")
	assert.EqualError(t, err, "no primaryKey field defined in model nopk")

}

func TestRepository_FindByField_Success(t *testing.T) {
	mockStore := new(MockKVStore)

	repo, err := db.NewRepository[User](mockStore, "cf1", "admin")
	assert.NoError(t, err)

	indexKey := "admin:users:idx:Name:Alice:*"
	dataKey := "admin:users:data:123"
	user := User{ID: "123", Name: "Alice"}
	data, _ := json.Marshal(user)

	mockStore.On("SearchByPatternPaginatedKV", "cf1", indexKey, "", 1).
		Return([]db.KeyValuePair{{Value: []byte("123")}}, "", nil)
	mockStore.On("Get", "cf1", dataKey).Return(data, nil)

	result, err := repo.FindByField("Name", "Alice")
	assert.NoError(t, err)
	assert.Equal(t, &user, result)

	mockStore.AssertExpectations(t)
}

type Invalid struct {
	Name string `orm:"unique"`
}

func (Invalid) TableName() string {
	return "invalid"
}

func TestNewRepository_MissingPrimaryKey(t *testing.T) {
	mockStore := new(MockKVStore)

	_, err := db.NewRepository[Invalid](mockStore, "cf1", "admin")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no primaryKey field defined")
}
