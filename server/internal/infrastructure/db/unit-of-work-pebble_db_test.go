package db_test

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"strconv"
	"testing"
	"time"

	"github.com/linxGnu/grocksdb"
	"github.com/stretchr/testify/require"
)

const UOWTestFCPebble = "test_fc"
const UOWDefaultFCPebble = "default"
const UOWTemporalFCPebble = "temporal_fc"

type TestCarPebbleFixOrderPebble struct {
	ID           string `orm:"primary-key"`
	Code         string `orm:"unique"`
	LicensePlate string
	Amount       float32
}

func (TestCarPebbleFixOrderPebble) TableName() string {
	return "car_fix_orders"
}

type TestCarPebble struct {
	ID           string `orm:"primary-key"`
	LicensePlate string `orm:"unique"`
	Name         string
	Model        string
	Performace   float32
	Year         int
}

func (TestCarPebble) TableName() string {
	return "cars"
}

type TestNotificationPebble struct {
	ID      string `orm:"primary-key"`
	Content string
	TTL     int `orm:"ttl"`
}

func (TestNotificationPebble) TableName() string {
	return "cars"
}

func newRocksdbStoreForUOWPebble(t *testing.T) *db.RocksdbStore {
	tmpDir := t.TempDir()
	opts := grocksdb.NewDefaultOptions()
	opts.SetCreateIfMissing(true)
	opts.SetCreateIfMissingColumnFamilies(true)
	goOp := grocksdb.NewDefaultOptions()

	rocks, cfHs, err := grocksdb.OpenDbColumnFamilies(opts, tmpDir, []string{UOWDefaultFCPebble, UOWTestFCPebble, UOWTemporalFCPebble}, []*grocksdb.Options{goOp, goOp, goOp})
	require.NoError(t, err)
	t.Cleanup(func() { rocks.Close() })

	columnFamilyNames, err := grocksdb.ListColumnFamilies(opts, tmpDir)
	require.NoError(t, err)

	cfMap := make(map[string]*grocksdb.ColumnFamilyHandle, len(columnFamilyNames)-1)
	for i, name := range columnFamilyNames {
		if name != UOWTemporalFCPebble {
			cfMap[name] = cfHs[i]
		}
	}

	ttlCFMap := make(map[string]*grocksdb.ColumnFamilyHandle, len(columnFamilyNames)-2)
	for i, name := range columnFamilyNames {
		if name == UOWTemporalFCPebble {
			ttlCFMap[name] = cfHs[i]
		}
	}

	return &db.RocksdbStore{
		DB:                     rocks,
		ColumnFamilyHandles:    cfMap,
		TTLColumnFamilyHandles: ttlCFMap,
	}
}

func newTestUOWDefaultIdGeneratorPebble(t *testing.T) (*db.UnitOfWork, db.KVStore, *db.Repository[TestCarPebble], *db.Repository[TestCarPebbleFixOrderPebble], *db.Repository[TestNotificationPebble], error) {
	store := newRocksdbStoreForUOWPebble(t)
	uow := db.NewUnitOfWork(store)

	carRepo, err := db.GetRepository[TestCarPebble](uow, UOWTestFCPebble, "test_schema", &db.DefaultIDGeneratorFactory{})
	if err != nil {
		t.Fatalf("failed to create car repo: %v", err)
	}
	testCarFixOrderRepo, err := db.GetRepository[TestCarPebbleFixOrderPebble](uow, UOWTestFCPebble, "test_schema", &db.DefaultIDGeneratorFactory{})
	if err != nil {
		t.Fatalf("failed to create fix order repo: %v", err)
	}
	testNotificationRepo, err := db.GetRepository[TestNotificationPebble](uow, UOWTemporalFCPebble, "test_schema", &db.DefaultIDGeneratorFactory{})
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

	car := &TestCarPebble{
		LicensePlate: "ABC123",
		Name:         "Toyota",
		Model:        "Corolla",
		Performace:   1.6,
		Year:         2020,
	}

	order := &TestCarPebbleFixOrderPebble{
		Code:         "ORD-001",
		LicensePlate: "ABC123",
		Amount:       123.45,
	}

	notif := &TestNotificationPebble{
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

	err = uow.Commit(now)
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

	uow2 := db.NewUnitOfWork(store)
	nRepo, _ := db.GetRepository[TestNotificationPebble](uow2, UOWTemporalFCPebble, "test_schema", &db.DefaultIDGeneratorFactory{})
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

	var notifs []*TestNotificationPebble
	for i := 0; i < 1000; i++ {
		n := &TestNotificationPebble{
			Content: "Notify " + strconv.Itoa(i),
			TTL:     1,
		}
		notifs = append(notifs, n)
	}
	now := time.Now()

	_, err = notifRepo.BulkCreate(notifs, now)
	require.NoError(t, err)
	require.NoError(t, uow.Commit(now))

	t.Log("Waiting TTL expiration for bulk notifications...")
	time.Sleep(2 * time.Second)
	checkTime := time.Now()

	uow2 := db.NewUnitOfWork(store)
	nRepo, _ := db.GetRepository[TestNotificationPebble](uow2, UOWTemporalFCPebble, "test_schema", &db.DefaultIDGeneratorFactory{})

	for _, notif := range notifs {
		found, err := nRepo.FindByField("ID", notif.ID, checkTime)
		require.NoError(t, err)
		require.Nil(t, found, "expected notification with ID %s to be expired", notif.ID)
	}
}
func TestPebbleUnitOfWork_MassiveMixedEntities(t *testing.T) {
	uow, store, carRepo, fixRepo, notifRepo, err := newTestUOWDefaultIdGeneratorPebble(t)
	require.NoError(t, err)

	var cars []*TestCarPebble
	var orders []*TestCarPebbleFixOrderPebble
	var notifs []*TestNotificationPebble

	for i := 0; i < 1000; i++ {
		license := "XYZ" + strconv.Itoa(i)
		cars = append(cars, &TestCarPebble{
			LicensePlate: license,
			Name:         "name-" + strconv.Itoa(i),
			Model:        "Model" + strconv.Itoa(i),
			Performace:   float32(i) * 0.1,
			Year:         2000 + (i % 24),
		})

		orders = append(orders, &TestCarPebbleFixOrderPebble{
			Code:         "FIX" + strconv.Itoa(i),
			LicensePlate: license,
			Amount:       float32(i) * 10,
		})

		notifs = append(notifs, &TestNotificationPebble{
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
	require.NoError(t, uow.Commit(now))

	time.Sleep(2 * time.Second)
	checkTime := time.Now()

	uow2 := db.NewUnitOfWork(store)
	nRepo, _ := db.GetRepository[TestNotificationPebble](uow2, UOWTemporalFCPebble, "test_schema", &db.DefaultIDGeneratorFactory{})

	for _, notif := range notifs {
		found, err := nRepo.FindByField("ID", notif.ID, checkTime)
		require.NoError(t, err)
		require.Nil(t, found, "TTL notification with ID %s should have expired", notif.ID)
	}
}
