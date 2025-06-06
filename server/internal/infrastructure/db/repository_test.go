package db_test

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"encoding/json"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

type TestIDGeneratorFactory struct {
	ids   []string
	index int
	mu    sync.Mutex
}

func NewTestIDGeneratorFactory(ids []string) *TestIDGeneratorFactory {
	return &TestIDGeneratorFactory{
		ids: ids,
	}
}

func (g *TestIDGeneratorFactory) GenerateID() string {
	g.mu.Lock()
	defer g.mu.Unlock()

	if len(g.ids) == 0 {
		return ""
	}

	id := g.ids[g.index]
	g.index = (g.index + 1) % len(g.ids) // avance circular
	return id
}

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

	iGF := NewTestIDGeneratorFactory([]string{"123"})

	repo, err := db.NewRepository[User](mockStore, "cf1", "admin", iGF)
	assert.NoError(t, err)

	data, _ := json.Marshal(user)
	dataKey := "admin:users:data:123"
	nameFieldKey := "admin:users:idx:Name:Alice:123"
	uNameFieldKey := "admin:users:idx-u:Name:Alice"
	indexKey := "admin:users:idx:ID:123:123"

	mockStore.On("Get", "cf1", uNameFieldKey).Return(nil, nil)
	mockStore.On("Put", "cf1", indexKey, []byte("123")).Return(nil)
	mockStore.On("Put", "cf1", nameFieldKey, []byte("123")).Return(nil)
	mockStore.On("Put", "cf1", dataKey, data).Return(nil)
	mockStore.On("Put", "cf1", uNameFieldKey, []byte("123")).Return(nil)

	id, err := repo.Create(&user)
	assert.NoError(t, err)
	assert.Equal(t, id, "123")

	mockStore.AssertExpectations(t)
}

func TestRepository_Create_DuplicateUnique(t *testing.T) {
	mockStore := new(MockKVStore)

	user := User{
		ID:   "----",
		Name: "Alice",
	}

	iGF := NewTestIDGeneratorFactory([]string{"123"})

	repo, err := db.NewRepository[User](mockStore, "cf1", "admin", iGF)
	assert.NoError(t, err)

	uIndexKey := "admin:users:idx-u:Name:Alice"
	mockStore.On("Get", "cf1", uIndexKey).Return([]byte("123"), nil)

	_, err = repo.Create(&user)
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

	iGF := NewTestIDGeneratorFactory([]string{"123"})
	_, err := db.NewRepository[NoPrimary](mockStore, "cf1", "admin", iGF)
	assert.EqualError(t, err, "struct NoPrimary must have a string field named 'ID' with `orm:primaryKey`")

}

func TestRepository_FindByField_Success(t *testing.T) {
	mockStore := new(MockKVStore)
	iGF := NewTestIDGeneratorFactory([]string{"123"})
	repo, err := db.NewRepository[User](mockStore, "cf1", "admin", iGF)
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
	iGF := NewTestIDGeneratorFactory([]string{"123"})

	_, err := db.NewRepository[Invalid](mockStore, "cf1", "admin", iGF)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "struct Invalid must have a string field named 'ID' with `orm:primaryKey`")
}

func TestRepository_Find_AND(t *testing.T) {
	mockStore := new(MockKVStore)
	iGF := NewTestIDGeneratorFactory([]string{"123"})
	repo, err := db.NewRepository[User](mockStore, "cf1", "admin", iGF)
	assert.NoError(t, err)

	user := User{ID: "123", Name: "Alice"}
	data, _ := json.Marshal(user)

	mockStore.On("SearchByPatternPaginatedKV", "cf1", "admin:users:idx:ID:123:*", "", 1000).
		Return([]db.KeyValuePair{{Value: []byte("123")}}, "", nil)
	mockStore.On("SearchByPatternPaginatedKV", "cf1", "admin:users:idx:Name:Alice:*", "", 1000).
		Return([]db.KeyValuePair{{Value: []byte("123")}}, "", nil)
	mockStore.On("Get", "cf1", "admin:users:data:123").
		Return(data, nil)

	result, err := repo.Find("ID=123&Name=Alice", 1000, "")
	assert.NoError(t, err)
	assert.Len(t, result.Entities, 1)
	assert.Equal(t, &user, &result.Entities[0])
}

func TestRepository_Find_OR(t *testing.T) {
	mockStore := new(MockKVStore)
	iGF := NewTestIDGeneratorFactory([]string{"123"})
	repo, err := db.NewRepository[User](mockStore, "cf1", "admin", iGF)
	assert.NoError(t, err)

	user1 := User{ID: "123", Name: "Alice"}
	user2 := User{ID: "456", Name: "Bob"}
	data1, _ := json.Marshal(user1)
	data2, _ := json.Marshal(user2)

	mockStore.On("SearchByPatternPaginatedKV", "cf1", "admin:users:idx:Name:Alice:*", "", 1000).
		Return([]db.KeyValuePair{{Value: []byte("123")}}, "", nil)
	mockStore.On("SearchByPatternPaginatedKV", "cf1", "admin:users:idx:Name:Bob:*", "", 1000).
		Return([]db.KeyValuePair{{Value: []byte("456")}}, "", nil)
	mockStore.On("Get", "cf1", "admin:users:data:123").
		Return(data1, nil)
	mockStore.On("Get", "cf1", "admin:users:data:456").
		Return(data2, nil)

	result, err := repo.Find("Name=Alice|Name=Bob", 1000, "")
	assert.NoError(t, err)
	assert.Len(t, result.Entities, 2)
}

func TestRepository_Find_SpecialCharacters(t *testing.T) {
	mockStore := new(MockKVStore)
	iGF := NewTestIDGeneratorFactory([]string{"123"})
	repo, err := db.NewRepository[User](mockStore, "cf1", "admin", iGF)
	assert.NoError(t, err)

	user := User{ID: "999", Name: "foo:bar"}
	data, _ := json.Marshal(user)

	mockStore.On("SearchByPatternPaginatedKV", "cf1", "admin:users:idx:Name:foo:bar:*", "", 1000).
		Return([]db.KeyValuePair{{Value: []byte("999")}}, "", nil)
	mockStore.On("Get", "cf1", "admin:users:data:999").
		Return(data, nil)

	result, err := repo.Find("Name=foo:bar", 1000, "")
	assert.NoError(t, err)
	assert.Len(t, result.Entities, 1)
	assert.Equal(t, &user, &result.Entities[0])
}

func TestRepository_Find_NoMatch(t *testing.T) {
	mockStore := new(MockKVStore)
	iGF := NewTestIDGeneratorFactory([]string{"123"})
	repo, err := db.NewRepository[User](mockStore, "cf1", "admin", iGF)
	assert.NoError(t, err)

	mockStore.On("SearchByPatternPaginatedKV", "cf1", "admin:users:idx:Name:Ghost:*", "", 1000).
		Return(nil, "", nil)

	result, err := repo.Find("Name=Ghost", 1000, "")
	assert.NoError(t, err)
	assert.Len(t, result.Entities, 0)
}

func TestRepository_Update_Success(t *testing.T) {
	mockStore := new(MockKVStore)

	iGF := NewTestIDGeneratorFactory([]string{"123"})
	repo, err := db.NewRepository[User](mockStore, "cf1", "admin", iGF)
	assert.NoError(t, err)

	originalUser := User{ID: "123", Name: "Alice"}
	updatedUser := User{ID: "123", Name: "Bob"}

	oldIndexKey := "admin:users:idx:Name:Alice:123"
	newIndexKey := "admin:users:idx:Name:Bob:123"
	dataKey := "admin:users:data:123"

	originalData, _ := json.Marshal(originalUser)
	updatedData, _ := json.Marshal(updatedUser)

	mockStore.On("SearchByPatternPaginatedKV", "cf1", "admin:users:idx:ID:123:*", "", 1).
		Return([]db.KeyValuePair{{Value: []byte("123")}}, "", nil)

	mockStore.On("SearchByPatternPaginatedKV", "cf1", "admin:users:idx:Name:Bob:*", "", 1).
		Return([]db.KeyValuePair{}, "", nil)

	mockStore.On("Get", "cf1", dataKey).Return(originalData, nil)

	mockStore.On("Delete", "cf1", oldIndexKey).Return(nil)

	mockStore.On("Put", "cf1", newIndexKey, []byte("123")).Return(nil)

	mockStore.On("Put", "cf1", dataKey, updatedData).Return(nil)

	changed, err := repo.Update("123", &updatedUser)
	assert.NoError(t, err)
	assert.Equal(t, changed, true)

	mockStore.AssertExpectations(t)
}

func TestRepository_Update_Nonexistent(t *testing.T) {
	mockStore := new(MockKVStore)

	iGF := NewTestIDGeneratorFactory([]string{"123"})
	repo, err := db.NewRepository[User](mockStore, "cf1", "admin", iGF)
	assert.NoError(t, err)

	user := User{ID: "999", Name: "Ghost"}
	dataKey := "admin:users:data:999"

	mockStore.On("Get", "cf1", dataKey).Return(nil, nil)
	mockStore.On("SearchByPatternPaginatedKV", "cf1", "admin:users:idx:ID:999:*", "", 1).
		Return([]db.KeyValuePair{}, "", nil)

	changed, err := repo.Update("999", &user)
	assert.NoError(t, err)
	assert.Equal(t, changed, false)
}
func TestRepository_Delete_Success(t *testing.T) {
	mockStore := new(MockKVStore)

	iGF := NewTestIDGeneratorFactory([]string{"123"})
	repo, err := db.NewRepository[User](mockStore, "cf1", "admin", iGF)
	assert.NoError(t, err)

	user := User{ID: "123", Name: "Alice"}
	dataKey := "admin:users:data:123"
	indexKey := "admin:users:idx:Name:Alice:123"
	pkIndexKey := "admin:users:idx:ID:123:123"
	data, _ := json.Marshal(user)

	mockStore.On("SearchByPatternPaginatedKV", "cf1", "admin:users:idx:ID:123:*", "", 1).
		Return([]db.KeyValuePair{{Value: []byte("123")}}, "", nil)

	mockStore.On("Get", "cf1", dataKey).Return(data, nil)

	mockStore.On("Delete", "cf1", indexKey).Return(nil)
	mockStore.On("Delete", "cf1", pkIndexKey).Return(nil)
	mockStore.On("Delete", "cf1", dataKey).Return(nil)

	deleted, err := repo.Delete("123")
	assert.Equal(t, deleted, true)
	assert.NoError(t, err)

	mockStore.AssertExpectations(t)
}

func TestRepository_Delete_NotFound(t *testing.T) {
	mockStore := new(MockKVStore)

	iGF := NewTestIDGeneratorFactory([]string{"123"})
	repo, err := db.NewRepository[User](mockStore, "cf1", "admin", iGF)
	assert.NoError(t, err)

	mockStore.On("SearchByPatternPaginatedKV", "cf1", "admin:users:idx:ID:123:*", "", 1).
		Return([]db.KeyValuePair{}, "", nil)

	deleted, err := repo.Delete("123")
	assert.Equal(t, deleted, false)
	assert.NoError(t, err)
}
func TestRepository_Delete_CorruptedData(t *testing.T) {
	mockStore := new(MockKVStore)

	iGF := NewTestIDGeneratorFactory([]string{"123"})
	repo, err := db.NewRepository[User](mockStore, "cf1", "admin", iGF)
	assert.NoError(t, err)

	dataKey := "admin:users:data:123"
	mockStore.On("SearchByPatternPaginatedKV", "cf1", "admin:users:idx:ID:123:*", "", 1).
		Return([]db.KeyValuePair{{Value: []byte("123")}}, "", nil)

	mockStore.On("Get", "cf1", dataKey).Return([]byte("not a valid json"), nil)

	deleted, err := repo.Delete("123")
	assert.Equal(t, deleted, false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid character")
}
