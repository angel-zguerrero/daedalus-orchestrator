package db_test

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type User struct {
	ID   string `orm:"primary-key"`
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

	batch := db.NewWriteBatch()
	batch.Put("cf1", indexKey, []byte("123"))
	batch.Put("cf1", nameFieldKey, []byte("123"))
	batch.Put("cf1", uNameFieldKey, []byte("123"))
	batch.Put("cf1", dataKey, data)

	mockStore.On("Write", mock.MatchedBy(func(b *db.WriteBatch) bool {
		return true
	})).Return(nil)

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
	assert.EqualError(t, err, "struct NoPrimary must have a string field named 'ID' with `orm:\"primary-key\"`")

}

func TestRepository_FindByField_Success(t *testing.T) {
	mockStore := new(MockKVStore)
	iGF := NewTestIDGeneratorFactory([]string{"123"})
	repo, err := db.NewRepository[User](mockStore, "cf1", "admin", iGF)
	assert.NoError(t, err)

	uIndexKey := "admin:users:idx-u:Name:Alice"
	dataKey := "admin:users:data:123"
	user := User{ID: "123", Name: "Alice"}
	data, _ := json.Marshal(user)

	mockStore.On("Get", "cf1", uIndexKey).Return([]byte("123"), nil)
	mockStore.On("Get", "cf1", dataKey).Return(data, nil)

	result, err := repo.FindByField("Name", "Alice")
	assert.NoError(t, err)
	assert.Equal(t, &user, result)

	mockStore.AssertExpectations(t)
}

func TestRepository_FindByField_Unknown_Field_Name(t *testing.T) {
	mockStore := new(MockKVStore)
	iGF := NewTestIDGeneratorFactory([]string{"123"})
	repo, err := db.NewRepository[User](mockStore, "cf1", "admin", iGF)
	assert.NoError(t, err)

	_, err = repo.FindByField("x", "Alice")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Unknown field x")
}

func TestRepository_Find_AND_Unknown_Field_Name(t *testing.T) {
	mockStore := new(MockKVStore)
	iGF := NewTestIDGeneratorFactory([]string{"123"})
	repo, err := db.NewRepository[User](mockStore, "cf1", "admin", iGF)
	assert.NoError(t, err)

	mockStore.On("SearchByPatternPaginatedKV", "cf1", "admin:users:idx:ID:123:*", "", 1000).
		Return([]db.KeyValuePair{{Value: []byte("123")}}, "", nil)

	_, err = repo.Find("ID=123&X=Alice", 1000, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Unknown field X")
	mockStore.AssertExpectations(t)
}

type TempEntity struct {
	ID    string `orm:"primary-key"`
	Token string `orm:"unique"`
	TTL   int    `orm:"ttl"`
}

func (TempEntity) TableName() string {
	return "temporal_entities"
}

func TestRepository_FindByField_Invalid_Use_For_TTL_Query(t *testing.T) {
	mockStore := new(MockKVStore)
	iGF := NewTestIDGeneratorFactory([]string{"123"})
	repo, err := db.NewRepository[TempEntity](mockStore, "cf1", "admin", iGF)
	assert.NoError(t, err)

	_, err = repo.FindByField("TTL", "111")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "TTL columns are not supported in query operations")
}

func TestRepository_Find_AND_Invalid_Use_For_TTL_Query(t *testing.T) {
	mockStore := new(MockKVStore)
	iGF := NewTestIDGeneratorFactory([]string{"123"})
	repo, err := db.NewRepository[TempEntity](mockStore, "cf1", "admin", iGF)
	assert.NoError(t, err)

	mockStore.On("SearchByPatternPaginatedKV", "cf1", "admin:temporal_entities:idx:ID:123:*", "", 1000).
		Return([]db.KeyValuePair{{Value: []byte("123")}}, "", nil)

	_, err = repo.Find("ID=123&TTL=22", 1000, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "TTL columns are not supported in query operation")
	mockStore.AssertExpectations(t)
}

type InvalidTempEntity struct {
	ID    string `orm:"primary-key"`
	Token string `orm:"unique"`
	ttl   int    `orm:"ttl"`
}

func (InvalidTempEntity) TableName() string {
	return "invalid_temporal_entities"
}

func TestRepository_FindByField_Invalid_TTL_Field_name(t *testing.T) {
	mockStore := new(MockKVStore)
	iGF := NewTestIDGeneratorFactory([]string{"123"})
	_, err := db.NewRepository[InvalidTempEntity](mockStore, "cf1", "admin", iGF)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error extracting fields from struct InvalidTempEntity: ttl can only be defined on top-level 'TTL' field, found on 'ttl'")
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
	assert.Contains(t, err.Error(), "struct Invalid must have a string field named 'ID' with `orm:\"primary-key\"`")
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

	oldUIndexKey := "admin:users:idx-u:Name:Alice"
	newUIndexKey := "admin:users:idx-u:Name:Bob"

	dataKey := "admin:users:data:123"

	originalData, _ := json.Marshal(originalUser)
	updatedData, _ := json.Marshal(updatedUser)

	mockStore.On("Get", "cf1", newUIndexKey).Return([]byte{}, nil)
	mockStore.On("Get", "cf1", dataKey).Return(originalData, nil)

	batch := db.NewWriteBatch()
	batch.Delete("cf1", oldUIndexKey)
	batch.Put("cf1", newUIndexKey, []byte("123"))
	batch.Delete("cf1", oldIndexKey)

	batch.Put("cf1", newIndexKey, []byte("123"))
	batch.Put("cf1", dataKey, updatedData)

	mockStore.On("Write", mock.MatchedBy(func(b *db.WriteBatch) bool {
		return true
	})).Return(nil)

	changed, err := repo.Update(&updatedUser)
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

	changed, err := repo.Update(&user)
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
	uIndexKey := "admin:users:idx-u:Name:Alice"
	pkIndexKey := "admin:users:idx:ID:123:123"
	data, _ := json.Marshal(user)

	mockStore.On("Get", "cf1", dataKey).Return(data, nil)

	batch := db.NewWriteBatch()

	batch.Delete("cf1", indexKey)
	batch.Delete("cf1", pkIndexKey)
	batch.Delete("cf1", uIndexKey)
	batch.Delete("cf1", dataKey)

	mockStore.On("Write", mock.MatchedBy(func(b *db.WriteBatch) bool {
		return true
	})).Return(nil)

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

	dataKey := "admin:users:data:123"
	mockStore.On("Get", "cf1", dataKey).Return(nil, nil)

	deleted, err := repo.Delete("123")
	assert.Equal(t, deleted, false)
	assert.NoError(t, err)
	mockStore.AssertExpectations(t)
}
func TestRepository_Delete_CorruptedData(t *testing.T) {
	mockStore := new(MockKVStore)

	iGF := NewTestIDGeneratorFactory([]string{"123"})
	repo, err := db.NewRepository[User](mockStore, "cf1", "admin", iGF)
	assert.NoError(t, err)

	dataKey := "admin:users:data:123"

	mockStore.On("Get", "cf1", dataKey).Return([]byte("not a valid json"), nil)

	deleted, err := repo.Delete("123")
	assert.Equal(t, deleted, false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid character")
	mockStore.AssertExpectations(t)
}

func TestRepository_BulkCreate_Success(t *testing.T) {
	mockStore := new(MockKVStore)
	iGF := NewTestIDGeneratorFactory([]string{"id1", "id2", "id3"})
	repo, err := db.NewRepository[User](mockStore, "cf1", "admin", iGF)
	assert.NoError(t, err)

	users := []*User{
		{Name: "Alice"},
		{Name: "Bob"},
		{Name: "Charlie"},
	}

	batch := db.NewWriteBatch()
	for i, u := range users {
		id := iGF.ids[i]
		u.ID = id
		d, _ := json.Marshal(u)

		dataKey := "admin:users:data:" + id
		nameIdx := "admin:users:idx:Name:" + u.Name + ":" + id
		uIdx := "admin:users:idx-u:Name:" + u.Name
		pkIdx := "admin:users:idx:ID:" + id + ":" + id

		mockStore.On("Get", "cf1", uIdx).Return(nil, nil)

		batch.Put("cf1", dataKey, d)
		batch.Put("cf1", nameIdx, []byte(id))
		batch.Put("cf1", uIdx, []byte(id))
		batch.Put("cf1", pkIdx, []byte(id))
	}

	mockStore.On("Write", mock.MatchedBy(func(b *db.WriteBatch) bool {
		return true // ya validamos el contenido arriba
	})).Return(nil)

	ids, err := repo.BulkCreate(users)
	assert.NoError(t, err)
	assert.Equal(t, []string{"id1", "id2", "id3"}, ids)

	mockStore.AssertExpectations(t)
}

func TestRepository_BulkCreate_DuplicateUnique(t *testing.T) {
	mockStore := new(MockKVStore)
	iGF := NewTestIDGeneratorFactory([]string{"id1", "id2"})
	repo, err := db.NewRepository[User](mockStore, "cf1", "admin", iGF)
	assert.NoError(t, err)

	users := []*User{
		{Name: "Alice"},
		{Name: "Alice"}, // duplicado intencional
	}

	mockStore.On("Get", "cf1", "admin:users:idx-u:Name:Alice").Return(nil, nil).Once()
	mockStore.On("Get", "cf1", "admin:users:idx-u:Name:Alice").Return([]byte("id1"), nil).Once()

	_, err = repo.BulkCreate(users)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate unique field")
}

func TestRepository_BulkCreate_WriteError(t *testing.T) {
	mockStore := new(MockKVStore)
	iGF := NewTestIDGeneratorFactory([]string{"id1"})
	repo, err := db.NewRepository[User](mockStore, "cf1", "admin", iGF)
	assert.NoError(t, err)

	users := []*User{
		{Name: "Alice"},
	}

	mockStore.On("Get", "cf1", "admin:users:idx-u:Name:Alice").Return(nil, nil)
	mockStore.On("Write", mock.Anything).Return(errors.New("write failed"))

	_, err = repo.BulkCreate(users)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "write failed")
}
func TestRepository_BulkDelete_Success(t *testing.T) {
	mockStore := new(MockKVStore)
	iGF := NewTestIDGeneratorFactory([]string{"id1", "id2"})
	repo, err := db.NewRepository[User](mockStore, "cf1", "admin", iGF)
	assert.NoError(t, err)

	users := []*User{
		{ID: "id1", Name: "Alice"},
		{ID: "id2", Name: "Bob"},
	}

	for _, u := range users {
		data, _ := json.Marshal(u)
		mockStore.On("Get", "cf1", "admin:users:data:"+u.ID).Return(data, nil)
	}

	batch := db.NewWriteBatch()
	batch.Delete("cf1", "admin:users:idx:Name:Alice:id1")
	batch.Delete("cf1", "admin:users:idx:Name:Bob:id2")
	batch.Delete("cf1", "admin:users:idx:ID:id1:id1")
	batch.Delete("cf1", "admin:users:idx:ID:id2:id2")
	batch.Delete("cf1", "admin:users:idx-u:Name:Alice")
	batch.Delete("cf1", "admin:users:idx-u:Name:Bob")
	batch.Delete("cf1", "admin:users:data:id1")
	batch.Delete("cf1", "admin:users:data:id2")

	mockStore.On("Write", mock.MatchedBy(func(b *db.WriteBatch) bool {
		return true
	})).Return(nil)

	deleted, err := repo.BulkDelete([]string{"id1", "id2"})
	assert.NoError(t, err)
	require.Len(t, deleted, 2)
	assert.True(t, deleted[0])
	assert.True(t, deleted[1])
	mockStore.AssertExpectations(t)
}

func TestRepository_BulkDelete_Partial(t *testing.T) {
	mockStore := new(MockKVStore)
	iGF := NewTestIDGeneratorFactory([]string{"id1", "id2"})
	repo, err := db.NewRepository[User](mockStore, "cf1", "admin", iGF)
	assert.NoError(t, err)

	user := &User{ID: "id1", Name: "Alice"}
	data, _ := json.Marshal(user)
	mockStore.On("Get", "cf1", "admin:users:data:id1").Return(data, nil)
	mockStore.On("Get", "cf1", "admin:users:data:id2").Return(nil, nil)

	batch := db.NewWriteBatch()
	batch.Delete("cf1", "admin:users:idx:Name:Alice:id1")
	batch.Delete("cf1", "admin:users:idx:ID:id1:id1")
	batch.Delete("cf1", "admin:users:idx-u:Name:Alice")
	batch.Delete("cf1", "admin:users:data:id1")

	mockStore.On("Write", mock.MatchedBy(func(b *db.WriteBatch) bool {
		return true
	})).Return(nil)

	deleted, err := repo.BulkDelete([]string{"id1", "id2"})
	assert.NoError(t, err)
	require.Len(t, deleted, 2)
	assert.True(t, deleted[0])
	assert.False(t, deleted[1])
	mockStore.AssertExpectations(t)
}

func TestRepository_BulkDelete_InvalidData(t *testing.T) {
	mockStore := new(MockKVStore)
	iGF := NewTestIDGeneratorFactory([]string{"id1"})
	repo, err := db.NewRepository[User](mockStore, "cf1", "admin", iGF)
	assert.NoError(t, err)

	mockStore.On("Get", "cf1", "admin:users:data:id1").Return([]byte("invalid json"), nil)

	deleted, err := repo.BulkDelete([]string{"id1"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid character")

	require.Len(t, deleted, 0)

	mockStore.AssertExpectations(t)
}
func TestRepository_BulkUpdate_Success(t *testing.T) {
	mockStore := new(MockKVStore)
	iGF := NewTestIDGeneratorFactory([]string{"id1", "id2"})
	repo, err := db.NewRepository[User](mockStore, "cf1", "admin", iGF)
	require.NoError(t, err)

	original1 := User{ID: "id1", Name: "Alice"}
	updated1 := User{ID: "id1", Name: "AliceUpdated"}

	original2 := User{ID: "id2", Name: "Bob"}
	updated2 := User{ID: "id2", Name: "BobUpdated"}

	// Serializar original data para mockear GET
	originalData1, _ := json.Marshal(original1)
	originalData2, _ := json.Marshal(original2)

	// Se mockea GET para los datos originales
	mockStore.On("Get", "cf1", "admin:users:data:id1").Return(originalData1, nil)
	mockStore.On("Get", "cf1", "admin:users:data:id2").Return(originalData2, nil)

	// Para las claves únicas nuevas, simular que no existen (para no dar error de duplicados)
	mockStore.On("Get", "cf1", "admin:users:idx-u:Name:AliceUpdated").Return(nil, nil)
	mockStore.On("Get", "cf1", "admin:users:idx-u:Name:BobUpdated").Return(nil, nil)

	mockStore.On("Write", mock.Anything).Return(nil)

	results, err := repo.BulkUpdate([]*User{&updated1, &updated2})
	assert.NoError(t, err)
	assert.Len(t, results, 2)
	assert.True(t, results[0])
	assert.True(t, results[1])

	mockStore.AssertExpectations(t)
}

func TestRepository_BulkUpdate_SomeNonexistent(t *testing.T) {
	mockStore := new(MockKVStore)
	iGF := NewTestIDGeneratorFactory([]string{"id1", "id2"})
	repo, err := db.NewRepository[User](mockStore, "cf1", "admin", iGF)
	require.NoError(t, err)

	existing := User{ID: "id1", Name: "Alice"}
	updatedExisting := User{ID: "id1", Name: "AliceUpdated"}
	nonexistent := User{ID: "id999", Name: "Ghost"}

	originalData, _ := json.Marshal(existing)

	mockStore.On("Get", "cf1", "admin:users:data:id1").Return(originalData, nil)
	mockStore.On("Get", "cf1", "admin:users:data:id999").Return(nil, nil) // no existe

	mockStore.On("Get", "cf1", "admin:users:idx-u:Name:AliceUpdated").Return(nil, nil)

	mockStore.On("Write", mock.Anything).Return(nil)

	results, err := repo.BulkUpdate([]*User{&updatedExisting, &nonexistent})
	assert.NoError(t, err)
	assert.Len(t, results, 2)
	assert.True(t, results[0])  // id1 actualizado
	assert.False(t, results[1]) // id999 no existe, no actualizado

	mockStore.AssertExpectations(t)
}

func TestRepository_BulkUpdate_DuplicateUniqueWithinBatch(t *testing.T) {
	mockStore := new(MockKVStore)
	iGF := NewTestIDGeneratorFactory([]string{"id1", "id2"})
	repo, err := db.NewRepository[User](mockStore, "cf1", "admin", iGF)
	require.NoError(t, err)

	original1 := User{ID: "id1", Name: "Alice"}
	original2 := User{ID: "id2", Name: "Bob"}

	updated1 := User{ID: "id1", Name: "SameName"} // Cambia a SameName
	updated2 := User{ID: "id2", Name: "SameName"} // Mismo nombre -> duplicado interno

	originalData1, _ := json.Marshal(original1)
	originalData2, _ := json.Marshal(original2)

	mockStore.On("Get", "cf1", "admin:users:data:id1").Return(originalData1, nil)
	mockStore.On("Get", "cf1", "admin:users:data:id2").Return(originalData2, nil)

	// El repositorio debería detectar el duplicado dentro del batch sin llamar al store para el índice uName

	results, err := repo.BulkUpdate([]*User{&updated1, &updated2})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate unique field")
	assert.Nil(t, results)
}

func TestRepository_BulkUpdate_DuplicateUniqueExisting(t *testing.T) {
	mockStore := new(MockKVStore)
	iGF := NewTestIDGeneratorFactory([]string{"id1", "id2"})
	repo, err := db.NewRepository[User](mockStore, "cf1", "admin", iGF)
	require.NoError(t, err)

	original1 := User{ID: "id1", Name: "Alice"}
	updated1 := User{ID: "id1", Name: "Bob"} // Cambia a "Bob"

	originalData1, _ := json.Marshal(original1)

	mockStore.On("Get", "cf1", "admin:users:data:id1").Return(originalData1, nil)
	// Simulamos que el índice único ya apunta a otro ID distinto al que actualizamos
	mockStore.On("Get", "cf1", "admin:users:idx-u:Name:Bob").Return([]byte("otherID"), nil)

	results, err := repo.BulkUpdate([]*User{&updated1})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate unique field")
	assert.Nil(t, results)
}

func TestRepository_BulkUpdate_WriteError(t *testing.T) {
	mockStore := new(MockKVStore)
	iGF := NewTestIDGeneratorFactory([]string{"id1"})
	repo, err := db.NewRepository[User](mockStore, "cf1", "admin", iGF)
	require.NoError(t, err)

	original := User{ID: "id1", Name: "Alice"}
	updated := User{ID: "id1", Name: "AliceUpdated"}

	originalData, _ := json.Marshal(original)

	mockStore.On("Get", "cf1", "admin:users:data:id1").Return(originalData, nil)
	mockStore.On("Get", "cf1", "admin:users:idx-u:Name:AliceUpdated").Return(nil, nil)

	mockStore.On("Write", mock.Anything).Return(errors.New("write failed"))

	results, err := repo.BulkUpdate([]*User{&updated})
	assert.Error(t, err)
	assert.Nil(t, results)
}

func TestRepository_BulkUpdate_InvalidData(t *testing.T) {
	mockStore := new(MockKVStore)
	iGF := NewTestIDGeneratorFactory([]string{"id1"})
	repo, err := db.NewRepository[User](mockStore, "cf1", "admin", iGF)
	require.NoError(t, err)

	// Simula que data original almacenada está corrupta (no json válido)
	mockStore.On("Get", "cf1", "admin:users:data:id1").Return([]byte("not json"), nil)

	updated := User{ID: "id1", Name: "AliceUpdated"}

	results, err := repo.BulkUpdate([]*User{&updated})
	assert.Error(t, err)
	assert.Nil(t, results)
}

func TestRepository_BulkUpdate_EmptyInput(t *testing.T) {
	mockStore := new(MockKVStore)
	iGF := NewTestIDGeneratorFactory([]string{})
	repo, err := db.NewRepository[User](mockStore, "cf1", "admin", iGF)
	require.NoError(t, err)

	results, err := repo.BulkUpdate([]*User{})
	assert.NoError(t, err)
	assert.NotNil(t, results)
	assert.Len(t, results, 0)
}

// --- Structs for Nested Field Tests ---

type Meta struct {
	Tag         string `orm:"unique"`
	ConfigCode  int
	Description string
}

type UserComplex struct {
	ID     string `orm:"primary-key"`
	Email  string `orm:"unique"`
	Meta   Meta   // Named field
	Status string
	Extra  *Meta // Pointer to a struct, for testing pointer handling
}

func (UserComplex) TableName() string {
	return "users_complex"
}

type MetaForEmbed struct {
	Tag         string `orm:"unique"` // Will become "Tag" at top level due to embedding
	ConfigCode  int    // Will become "ConfigCode"
	Description string // Will become "Description"
}

type UserComplexEmbedded struct {
	ID           string `orm:"primary-key"`
	Email        string `orm:"unique"`
	MetaForEmbed        // Embedded field
	Status       string
}

func (UserComplexEmbedded) TableName() string {
	return "users_complex_embedded"
}

// --- Test Cases for Nested Fields ---

func TestRepository_Create_NestedSuccess_UserComplexEmbedded(t *testing.T) {
	mockStore := new(MockKVStore)
	entityID := "uce123"
	iGF := NewTestIDGeneratorFactory([]string{entityID})

	repo, err := db.NewRepository[UserComplexEmbedded](mockStore, "cf_embed", "test_sch_embed", iGF)
	require.NoError(t, err)

	user := UserComplexEmbedded{
		Email:  "embedded@example.com",
		Status: "pending",
		MetaForEmbed: MetaForEmbed{ // Fields from MetaForEmbed should be top-level
			Tag:         "embeddedTag1",
			ConfigCode:  303,
			Description: "Embedded Desc",
		},
	}

	// Mock for unique checks (Email and embedded Tag)
	mockStore.On("Get", "cf_embed", "test_sch_embed:users_complex_embedded:idx-u:Email:embedded@example.com").Return(nil, nil).Once()
	// Assuming embedded field 'Tag' becomes a top-level field name 'Tag'
	mockStore.On("Get", "cf_embed", "test_sch_embed:users_complex_embedded:idx-u:Tag:embeddedTag1").Return(nil, nil).Once()

	mockStore.On("Write", mock.MatchedBy(func(batch *db.WriteBatch) bool {
		return batch.Count() > 0
	})).Return(nil).Once()

	createdID, err := repo.Create(&user)
	require.NoError(t, err)
	assert.Equal(t, entityID, createdID)
	assert.Equal(t, entityID, user.ID)

	// Verify some key names based on embedding (Tag, ConfigCode should be top-level)
	// In a real test, capture the batch passed to Write and assert its contents.
	// For example, one of the calls to Put in the batch should be for:
	// "test_sch_embed:users_complex_embedded:idx:Tag:embeddedTag1:" + entityID

	mockStore.AssertExpectations(t)
}

func TestRepository_Create_NestedDuplicateUnique_UserComplex(t *testing.T) {
	mockStore := new(MockKVStore)
	entityID := "uc456"
	iGF := NewTestIDGeneratorFactory([]string{entityID})

	repo, err := db.NewRepository[UserComplex](mockStore, "cf_complex", "test_sch", iGF)
	require.NoError(t, err)

	user := UserComplex{
		Email:  "new@example.com", // Assume this is unique for now
		Status: "active",
		Meta: Meta{
			Tag:         "existingTag", // This tag will cause a duplicate error
			ConfigCode:  102,
			Description: "Desc",
		},
	}

	// Mock for unique checks
	mockStore.On("Get", "cf_complex", "test_sch:users_complex:idx-u:Email:new@example.com").Return(nil, nil)
	// Simulate Meta.Tag being a duplicate
	mockStore.On("Get", "cf_complex", "test_sch:users_complex:idx-u:Meta.Tag:existingTag").Return([]byte("anotherID"), nil).Once()
	// No On("Write") should be called

	_, err = repo.Create(&user)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate unique field: Meta.Tag = existingTag")

	mockStore.AssertExpectations(t)
}

func TestRepository_FindByField_NestedSuccess_UserComplex(t *testing.T) {
	mockStore := new(MockKVStore)
	entityID := "uc789"
	iGF := NewTestIDGeneratorFactory([]string{}) // Not used for FindByField directly for ID generation

	repo, err := db.NewRepository[UserComplex](mockStore, "cf_complex", "test_sch", iGF)
	require.NoError(t, err)

	expectedUser := UserComplex{
		ID:     entityID,
		Email:  "findme@example.com",
		Status: "found",
		Meta: Meta{
			Tag:         "findThisTag",
			ConfigCode:  103,
			Description: "Found Description",
		},
	}
	jsonData, err := json.Marshal(expectedUser)
	require.NoError(t, err)

	uniqueIdxKey := "test_sch:users_complex:idx-u:Meta.Tag:findThisTag"
	dataKey := "test_sch:users_complex:data:" + entityID

	mockStore.On("Get", "cf_complex", uniqueIdxKey).Return([]byte(entityID), nil).Once()
	mockStore.On("Get", "cf_complex", dataKey).Return(jsonData, nil).Once()

	foundUser, err := repo.FindByField("Meta.Tag", "findThisTag")
	require.NoError(t, err)
	require.NotNil(t, foundUser)
	assert.Equal(t, expectedUser, *foundUser)

	mockStore.AssertExpectations(t)
}

func TestRepository_Find_NestedCondition_UserComplex(t *testing.T) {
	mockStore := new(MockKVStore)
	entityID := "uc101"
	iGF := NewTestIDGeneratorFactory([]string{})

	repo, err := db.NewRepository[UserComplex](mockStore, "cf_complex", "test_sch", iGF)
	require.NoError(t, err)

	user := UserComplex{
		ID:     entityID,
		Email:  "filter@example.com",
		Status: "active",
		Meta: Meta{
			Tag:         "filterTag",
			ConfigCode:  404,
			Description: "Filtered item",
		},
	}
	jsonData, _ := json.Marshal(user)

	// Mock KVStore SearchByPatternPaginatedKV calls
	// For "Meta.Tag = 'filterTag'"
	mockStore.On("SearchByPatternPaginatedKV", "cf_complex", "test_sch:users_complex:idx:Meta.Tag:filterTag:*", "", 1000).
		Return([]db.KeyValuePair{{Key: "...", Value: []byte(entityID)}}, "", nil).Once()
	// For "Email = 'filter@example.com'"
	mockStore.On("SearchByPatternPaginatedKV", "cf_complex", "test_sch:users_complex:idx:Email:filter@example.com:*", "", 1000).
		Return([]db.KeyValuePair{{Key: "...", Value: []byte(entityID)}}, "", nil).Once()

	// Mock Get for the data key
	mockStore.On("Get", "cf_complex", "test_sch:users_complex:data:"+entityID).Return(jsonData, nil).Once()

	result, err := repo.Find("Meta.Tag = 'filterTag' & Email = 'filter@example.com'", 1000, "")
	require.NoError(t, err)
	require.Len(t, result.Entities, 1)
	assert.Equal(t, user, result.Entities[0])

	mockStore.AssertExpectations(t)
}

func TestRepository_Update_NestedField_UserComplex(t *testing.T) {
	mockStore := new(MockKVStore)
	entityID := "ucUpdate1"
	iGF := NewTestIDGeneratorFactory([]string{})
	repo, err := db.NewRepository[UserComplex](mockStore, "cf_complex", "test_sch", iGF)
	require.NoError(t, err)

	originalUser := UserComplex{
		ID:     entityID,
		Email:  "update@example.com",
		Status: "stale",
		Meta: Meta{
			Tag:         "oldTag",
			ConfigCode:  500,
			Description: "Old Description",
		},
	}
	updatedUser := UserComplex{
		ID:     entityID,             // ID must be the same
		Email:  "update@example.com", // Email not changed
		Status: "current",            // Top-level field changed
		Meta: Meta{
			Tag:         "newTag",          // Nested unique field changed
			ConfigCode:  501,               // Nested non-unique field changed
			Description: "New Description", // Nested non-unique field changed
		},
	}

	originalData, _ := json.Marshal(originalUser)
	// updatedData, _ := json.Marshal(updatedUser) // repo.Update internally marshals the modified 'current'

	// 1. FindByField (ID) to get current entity
	mockStore.On("Get", "cf_complex", "test_sch:users_complex:data:"+entityID).Return(originalData, nil).Once()

	// 2. Unique check for new Meta.Tag value
	mockStore.On("Get", "cf_complex", "test_sch:users_complex:idx-u:Meta.Tag:newTag").Return(nil, nil).Once()
	// (No other unique fields changed in this test case for Meta)

	// 3. Write batch operations (simplified, actual batch content is complex)
	mockStore.On("Write", mock.MatchedBy(func(batch *db.WriteBatch) bool {
		// A real test would capture and inspect the batch.
		// Count > 0 implies deletes and puts for changed fields' indexes and the main data.
		return batch.Count() > 0
	})).Return(nil).Once()

	changed, err := repo.Update(&updatedUser)
	require.NoError(t, err)
	assert.True(t, changed)

	mockStore.AssertExpectations(t)
}

func TestRepository_Update_NestedField_DuplicateUnique_UserComplex(t *testing.T) {
	mockStore := new(MockKVStore)
	entityID := "ucUpdateDup1"
	iGF := NewTestIDGeneratorFactory([]string{})
	repo, err := db.NewRepository[UserComplex](mockStore, "cf_complex", "test_sch", iGF)
	require.NoError(t, err)

	originalUser := UserComplex{
		ID:    entityID,
		Email: "updatedup@example.com",
		Meta:  Meta{Tag: "originalTag"},
	}
	updatedUser := UserComplex{
		ID:    entityID,
		Email: "updatedup@example.com",
		Meta:  Meta{Tag: "conflictingTag"}, // This tag will conflict
	}

	originalData, _ := json.Marshal(originalUser)

	// 1. FindByField (ID)
	mockStore.On("Get", "cf_complex", "test_sch:users_complex:data:"+entityID).Return(originalData, nil).Once()

	// 2. Unique check for new Meta.Tag shows it exists for another ID
	mockStore.On("Get", "cf_complex", "test_sch:users_complex:idx-u:Meta.Tag:conflictingTag").Return([]byte("anotherEntityID"), nil).Once()

	changed, err := repo.Update(&updatedUser)
	require.Error(t, err)
	assert.False(t, changed)
	assert.Contains(t, err.Error(), "duplicate unique field: Meta.Tag = conflictingTag")

	mockStore.AssertExpectations(t)
}

func TestRepository_Delete_WithNestedFields_UserComplex(t *testing.T) {
	mockStore := new(MockKVStore)
	entityID := "ucDelete1"
	iGF := NewTestIDGeneratorFactory([]string{})
	repo, err := db.NewRepository[UserComplex](mockStore, "cf_complex", "test_sch", iGF)
	require.NoError(t, err)

	userToDelete := UserComplex{
		ID:     entityID,
		Email:  "delete@example.com",
		Status: "marked_for_deletion",
		Meta: Meta{
			Tag:         "deleteTag",
			ConfigCode:  999,
			Description: "To be deleted",
		},
		Extra: &Meta{Tag: "delExtra"},
	}
	jsonData, _ := json.Marshal(userToDelete)

	// 1. FindByField (ID) to get entity before deleting its indexes
	mockStore.On("Get", "cf_complex", "test_sch:users_complex:data:"+entityID).Return(jsonData, nil).Once()

	// 2. Write batch for deletions
	mockStore.On("Write", mock.MatchedBy(func(batch *db.WriteBatch) bool {
		// Expect many deletes: data, all regular indexes, all unique indexes
		return batch.Count() > 0 // Simplified check
	})).Return(nil).Once()

	deleted, err := repo.Delete(entityID)
	require.NoError(t, err)
	assert.True(t, deleted)

	mockStore.AssertExpectations(t)
}

// Test for UserComplexEmbedded with anonymous MetaForEmbed
// Focus on Create to check field name generation for embedded fields.
func TestRepository_Create_UserComplexEmbedded_FieldNames(t *testing.T) {
	mockStore := new(MockKVStore)
	entityID := "uceFieldTest"
	iGF := NewTestIDGeneratorFactory([]string{entityID})

	repo, err := db.NewRepository[UserComplexEmbedded](mockStore, "cf_embed_fn", "test_sch_fn", iGF)
	require.NoError(t, err)

	user := UserComplexEmbedded{
		Email:  "embedfn@example.com",
		Status: "active_embed_fn",
		MetaForEmbed: MetaForEmbed{
			Tag:         "embedFnTag", // Expect 'Tag' as field name
			ConfigCode:  777,          // Expect 'ConfigCode'
			Description: "Desc for embed FN",
		},
	}

	// Mock unique checks. If extractFieldsRecursively makes embedded fields top-level,
	// then "Tag" should be the unique field name, not "MetaForEmbed.Tag".
	mockStore.On("Get", "cf_embed_fn", "test_sch_fn:users_complex_embedded:idx-u:Email:embedfn@example.com").Return(nil, nil).Once()
	mockStore.On("Get", "cf_embed_fn", "test_sch_fn:users_complex_embedded:idx-u:Tag:embedFnTag").Return(nil, nil).Once() // Key check

	// Mock the Write call
	// In a real test, capture the batch and verify specific index keys, e.g.:
	// - test_sch_fn:users_complex_embedded:idx:Tag:embedFnTag:<id>
	// - test_sch_fn:users_complex_embedded:idx:ConfigCode:777:<id>
	mockStore.On("Write", mock.MatchedBy(func(batch *db.WriteBatch) bool {
		// For now, a simple count check.
		// A more thorough test would capture the batch and verify its contents.
		// Example: Iterate batch.Ops, check for specific key patterns.
		// For this test, ensuring the unique checks above are for "Tag" (not "MetaForEmbed.Tag")
		// and that the code doesn't error out is the primary goal given the complexity of full batch verification here.
		return batch.Count() > 0
	})).Return(nil).Once()

	createdID, err := repo.Create(&user)
	require.NoError(t, err)
	assert.Equal(t, entityID, createdID)

	mockStore.AssertExpectations(t)
}
