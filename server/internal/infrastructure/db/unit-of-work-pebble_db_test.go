//go:build rocksdb
// +build rocksdb

package db_test

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"strconv"
	"testing"
	"time"

	"github.com/linxGnu/grocksdb"
	"github.com/stretchr/testify/require"
)

func newRocksdbStoreForUOWPebble(t *testing.T) *db.RocksdbStore {
	tmpDir := t.TempDir()
	opts := grocksdb.NewDefaultOptions()
	opts.SetCreateIfMissing(true)
	opts.SetCreateIfMissingColumnFamilies(true)
	goOp := grocksdb.NewDefaultOptions()

	rocks, cfHs, err := grocksdb.OpenDbColumnFamilies(opts, tmpDir, []string{UOWDefaultFC, UOWTestFC, UOWTemporalFC}, []*grocksdb.Options{goOp, goOp, goOp})
	require.NoError(t, err)
	t.Cleanup(func() { rocks.Close() })

	columnFamilyNames, err := grocksdb.ListColumnFamilies(opts, tmpDir)
	require.NoError(t, err)

	cfMap := make(map[string]*grocksdb.ColumnFamilyHandle, len(columnFamilyNames)-1)
	for i, name := range columnFamilyNames {
		if name != UOWTemporalFC {
			cfMap[name] = cfHs[i]
		}
	}

	ttlCFMap := make(map[string]*grocksdb.ColumnFamilyHandle, len(columnFamilyNames)-2)
	for i, name := range columnFamilyNames {
		if name == UOWTemporalFC {
			ttlCFMap[name] = cfHs[i]
		}
	}

	return &db.RocksdbStore{
		DB:                     rocks,
		ColumnFamilyHandles:    cfMap,
		TTLColumnFamilyHandles: ttlCFMap,
	}
}

func newTestUOWDefaultIdGeneratorPebble(t *testing.T) (*db.UnitOfWork, db.KVStore, *db.Repository[TestCar], *db.Repository[TestCarFixOrder], *db.Repository[TestNotification], error) {
	store := newRocksdbStoreForUOWPebble(t)
	uow := db.NewUnitOfWork(store, nil)

	carRepo, err := db.GetRepository[TestCar](uow, UOWTestFC, testColumnFamilySelector, "test_schema", &db.DefaultIDGeneratorFactory{})
	if err != nil {
		t.Fatalf("failed to create car repo: %v", err)
	}
	testCarFixOrderRepo, err := db.GetRepository[TestCarFixOrder](uow, UOWTestFC, testColumnFamilySelector, "test_schema", &db.DefaultIDGeneratorFactory{})
	if err != nil {
		t.Fatalf("failed to create fix order repo: %v", err)
	}
	testNotificationRepo, err := db.GetRepository[TestNotification](uow, UOWTemporalFC, testColumnFamilySelector, "test_schema", &db.DefaultIDGeneratorFactory{})
	if err != nil {
		t.Fatalf("failed to create notification repo: %v", err)
	}

	return uow, store, carRepo, testCarFixOrderRepo, testNotificationRepo, nil
}

func TestPebbleUnitOfWork_CreateAndCommit(t *testing.T) {
	uow, store, carRepo, fixRepo, notifRepo, err := newTestUOWDefaultIdGeneratorPebble(t)
	if err != nil {
		t.Fatal(err)
	}

	car := &TestCar{
		LicensePlate: "ABC123",
		Name:         "Toyota",
		Model:        "Corolla",
		Performace:   1.6,
		Year:         2020,
	}

	order := &TestCarFixOrder{
		Code:         "ORD-001",
		LicensePlate: "ABC123",
		Amount:       123.45,
	}

	notif := &TestNotification{
		Content: "Test TTL Message",
		TTL:     1, // 1 second TTL
	}
	now := time.Now()

	_, err = carRepo.Create(car, now)
	if err != nil {
		t.Fatalf("failed to create car: %v", err)
	}
	_, err = fixRepo.Create(order, now)
	if err != nil {
		t.Fatalf("failed to create fix order: %v", err)
	}
	_, err = notifRepo.Create(notif, now)
	if err != nil {
		t.Fatalf("failed to create notification: %v", err)
	}

	err = uow.Commit()
	if err != nil {
		t.Fatalf("commit failed: %v", err)
	}

	queryNow := time.Now()
	savedCar, err := carRepo.FindByField("LicensePlate", "ABC123", queryNow)
	if err != nil || savedCar == nil {
		t.Fatalf("car not found after commit: %v", err)
	}

	savedOrder, err := fixRepo.FindByField("Code", "ORD-001", queryNow)
	if err != nil || savedOrder == nil {
		t.Fatalf("fix order not found after commit: %v", err)
	}

	savedNotif, err := notifRepo.FindByField("ID", notif.ID, queryNow)
	if err != nil || savedNotif == nil {
		t.Fatalf("notification not found after commit: %v", err)
	}

	t.Logf("Created car: %+v", savedCar)
	t.Logf("Created fix order: %+v", savedOrder)
	t.Logf("Created notification: %+v", savedNotif)

	t.Log("Waiting TTL expiration...")
	time.Sleep(2 * time.Second)
	checkTime := time.Now()

	uow2 := db.NewUnitOfWork(store, nil)
	nRepo, _ := db.GetRepository[TestNotification](uow2, UOWTemporalFC, testColumnFamilySelector, "test_schema", &db.DefaultIDGeneratorFactory{})
	expired, err := nRepo.FindByField("ID", notif.ID, checkTime)
	if err != nil {
		t.Fatalf("error checking expired ttl entity: %v", err)
	}
	if expired != nil {
		t.Errorf("TTL entity still exists after expiration: %+v", expired)
	}
}
func TestPebbleUnitOfWork_TTLBulkCreateAndExpire(t *testing.T) {
	uow, store, _, _, notifRepo, err := newTestUOWDefaultIdGeneratorPebble(t)
	require.NoError(t, err)

	var notifs []*TestNotification
	for i := 0; i < 1000; i++ {
		n := &TestNotification{
			Content: "Notify " + strconv.Itoa(i),
			TTL:     1,
		}
		notifs = append(notifs, n)
	}
	now := time.Now()

	_, err = notifRepo.BulkCreate(notifs, now)
	require.NoError(t, err)
	require.NoError(t, uow.Commit())

	t.Log("Waiting TTL expiration for bulk notifications...")
	time.Sleep(2 * time.Second)
	checkTime := time.Now()

	uow2 := db.NewUnitOfWork(store, nil)
	nRepo, _ := db.GetRepository[TestNotification](uow2, UOWTemporalFC, testColumnFamilySelector, "test_schema", &db.DefaultIDGeneratorFactory{})

	for _, notif := range notifs {
		found, err := nRepo.FindByField("ID", notif.ID, checkTime)
		require.NoError(t, err)
		require.Nil(t, found, "expected notification with ID %s to be expired", notif.ID)
	}
}
func TestPebbleUnitOfWork_MassiveMixedEntities(t *testing.T) {
	uow, store, carRepo, fixRepo, notifRepo, err := newTestUOWDefaultIdGeneratorPebble(t)
	require.NoError(t, err)

	var cars []*TestCar
	var orders []*TestCarFixOrder
	var notifs []*TestNotification

	for i := 0; i < 1000; i++ {
		license := "XYZ" + strconv.Itoa(i)
		cars = append(cars, &TestCar{
			LicensePlate: license,
			Name:         "name-" + strconv.Itoa(i),
			Model:        "Model" + strconv.Itoa(i),
			Performace:   float32(i) * 0.1,
			Year:         2000 + (i % 24),
		})

		orders = append(orders, &TestCarFixOrder{
			Code:         "FIX" + strconv.Itoa(i),
			LicensePlate: license,
			Amount:       float32(i) * 10,
		})

		notifs = append(notifs, &TestNotification{
			Content: "Message " + strconv.Itoa(i),
			TTL:     1,
		})
	}
	now := time.Now()

	_, err = carRepo.BulkCreate(cars, now)
	require.NoError(t, err)
	_, err = fixRepo.BulkCreate(orders, now)
	require.NoError(t, err)
	_, err = notifRepo.BulkCreate(notifs, now)
	require.NoError(t, err)
	require.NoError(t, uow.Commit())

	time.Sleep(2 * time.Second)
	checkTime := time.Now()

	uow2 := db.NewUnitOfWork(store, nil)
	nRepo, _ := db.GetRepository[TestNotification](uow2, UOWTemporalFC, testColumnFamilySelector, "test_schema", &db.DefaultIDGeneratorFactory{})

	for _, notif := range notifs {
		found, err := nRepo.FindByField("ID", notif.ID, checkTime)
		require.NoError(t, err)
		require.Nil(t, found, "TTL notification with ID %s should have expired", notif.ID)
	}
}
