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
