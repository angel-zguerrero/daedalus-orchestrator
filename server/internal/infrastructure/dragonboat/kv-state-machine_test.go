package dragonboat_test

import (
	"testing"
	"time"

	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/infrastructure/dragonboat"

	"github.com/stretchr/testify/assert"
)

func TestOne(t *testing.T) {
	dbConn1, err := db.InitDB("dragonboat_test_1", db.DefaultPathProvider{})
	assert.NoError(t, err)

	defer dbConn1.Close()

	rocksdbStore1 := &db.RocksdbStore{DB: dbConn1}

	dragonboat.Init(rocksdbStore1, 1, 1, "3435")
	//-----

	dbConn2, err := db.InitDB("dragonboat_test_2", db.DefaultPathProvider{})
	assert.NoError(t, err)

	defer dbConn2.Close()

	rocksdbStore2 := &db.RocksdbStore{DB: dbConn2}

	dragonboat.Init(rocksdbStore2, 1, 2, "3436")

	time.Sleep(240 * time.Second)
}
