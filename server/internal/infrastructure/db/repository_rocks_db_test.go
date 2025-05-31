package db_test

import (
	"testing"

	"github.com/linxGnu/grocksdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"deadalus-orch/server/internal/infrastructure/db"
)

type testEntity struct {
	ID   string `orm:"primaryKey"`
	Name string `orm:"unique"`
}

func (testEntity) TableName() string {
	return "users"
}

func newRocksdbStore(t *testing.T) *db.RocksdbStore {
	tmpDir := t.TempDir()
	opts := grocksdb.NewDefaultOptions()
	opts.SetCreateIfMissing(true)
	opts.SetCreateIfMissingColumnFamilies(true)
	goOp := grocksdb.NewDefaultOptions()

	rocks, cfHs, err := grocksdb.OpenDbColumnFamilies(opts, tmpDir, []string{DefaultFC, TestFC}, []*grocksdb.Options{goOp, goOp})
	require.NoError(t, err)
	t.Cleanup(func() { rocks.Close() })

	columnFamilyNames, err := grocksdb.ListColumnFamilies(opts, tmpDir)
	require.NoError(t, err)

	cfMap := make(map[string]*grocksdb.ColumnFamilyHandle, len(columnFamilyNames))
	for i, name := range columnFamilyNames {
		cfMap[name] = cfHs[i]
	}

	return &db.RocksdbStore{
		DB:                     rocks,
		ColumnFamilyHandles:    cfMap,
		TTLColumnFamilyHandles: make(map[string]*grocksdb.ColumnFamilyHandle),
	}
}

func newTestRepository(t *testing.T) (*db.Repository[testEntity], error) {
	store := newRocksdbStore(t)
	iGF := NewTestIDGeneratorFactory([]string{"123", "456"})
	return db.NewRepository[testEntity](store, TestFC, "test_schema", iGF)
}

func TestRepository_PutAndGet(t *testing.T) {
	repo, err := newTestRepository(t)
	require.NoError(t, err)
	entity := testEntity{ID: "----", Name: "Alice"}

	id, err := repo.Create(&entity)
	require.NoError(t, err)
	assert.Equal(t, id, "123")

	found, err := repo.FindByField("ID", "123")
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, "123", found.ID)
	assert.Equal(t, entity.Name, found.Name)

	found, err = repo.FindByField("Name", "Alice")
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, "123", found.ID)
	assert.Equal(t, entity.Name, found.Name)
}

func TestRepository_Get_NotFound(t *testing.T) {
	repo, err := newTestRepository(t)
	require.NoError(t, err)

	found, err := repo.FindByField("ID", "5566")
	require.NoError(t, err)
	assert.Nil(t, found)
}

func TestRepository_WriteBatch(t *testing.T) {
	repo, err := newTestRepository(t)
	require.NoError(t, err)

	a := testEntity{ID: "---", Name: "Alpha"}
	b := testEntity{ID: "---", Name: "Beta"}

	id, err := repo.Create(&a)
	assert.Equal(t, id, "123")
	require.NoError(t, err)
	id, err = repo.Create(&b)
	require.NoError(t, err)
	assert.Equal(t, id, "456")

	resA, err := repo.FindByField("ID", "123")
	require.NoError(t, err)
	assert.Equal(t, a.Name, resA.Name)

	resB, err := repo.FindByField("ID", "456")
	require.NoError(t, err)
	assert.Equal(t, b.Name, resB.Name)
}

func TestRepository_SearchByPatternPaginatedKV_MatchSingle(t *testing.T) {
	repo, err := newTestRepository(t)
	require.NoError(t, err)

	entity := testEntity{ID: "user:123:name", Name: "Alice"}
	id, err := repo.Create(&entity)
	assert.Equal(t, id, "123")
	require.NoError(t, err)

	results, err := repo.Find("Name=Alice")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, entity.ID, results[0].ID)
	assert.Equal(t, entity.Name, results[0].Name)
}
