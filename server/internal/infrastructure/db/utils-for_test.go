package db_test

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"sync"
	"time"

	"github.com/stretchr/testify/mock"
)

const (
	numNormalKeys = 2000
	numTTLKeys    = 2000
	normalCF      = "test_normal_cf"
	ttlCF         = "test_ttl_cf"
	// Small TTL for testing expiry quickly
	testTTLSeconds           = 3
	testColumnFamilySelector = "test-selector"
)

type testKeyData struct {
	ColumnFamily string
	Key          string
	Value        string
	IsTTL        bool
	InsertTime   time.Time     // To estimate expiry for TTL keys
	ExpectedTTL  time.Duration // The TTL value set
}

type MockKVStore struct {
	mock.Mock
}

func (m *MockKVStore) Get(AdminFC, cfSelector, key string, now time.Time) ([]byte, error) {
	args := m.Called(AdminFC, cfSelector, key, now)
	var s []byte
	if tmp := args.Get(0); tmp != nil {
		s = tmp.([]byte)
	}
	return s, args.Error(1)
}

func (m *MockKVStore) Delete(AdminFC, cfSelector, key string, now time.Time) error {
	args := m.Called(AdminFC, cfSelector, key, now)
	return args.Error(0)
}

func (r *MockKVStore) Exists(columnFamily, cfSelector, key string, now time.Time) (bool, error) {
	args := r.Called(columnFamily, cfSelector, key, now)
	return args.Bool(0), args.Error(1)
}

func (m *MockKVStore) Put(AdminFC, cfSelector, key string, value []byte, ttl int, now time.Time) error {
	args := m.Called(AdminFC, cfSelector, key, value, ttl, now)
	return args.Error(0)
}

func (m *MockKVStore) PutRaw(AdminFC, cfSelector, key string, value []byte) error {
	args := m.Called(AdminFC, cfSelector, key, value)
	return args.Error(0)
}

func (m *MockKVStore) Write(batch *db.WriteBatch) error {
	args := m.Called(batch)
	return args.Error(0)
}

func (m *MockKVStore) WriteRaw(batch *db.WriteBatch) error {
	args := m.Called(batch)
	return args.Error(0)
}

func (m *MockKVStore) DumpAll() (interface{}, error) {
	args := m.Called()
	var s []byte
	if tmp := args.Get(0); tmp != nil {
		s = tmp.([]byte)
	}
	return s, args.Error(1)
}

func (r *MockKVStore) Iterate(fn func(cfName string, cfSelector string, key, value []byte) error) error {
	return nil
}

func (r *MockKVStore) ClearAll() error {
	return nil
}

func (r *MockKVStore) Flush() error {
	return nil
}

func (r *MockKVStore) Close() error {
	return nil
}

func (r *MockKVStore) CleanExpiredKeys(now time.Time) error {
	args := r.Called(now)
	return args.Error(0)
}

func (m *MockKVStore) SearchByPatternPaginatedKV(cfName, cfSelector, pattern, cursor string, limit int, now time.Time) ([]db.KeyValuePair, string, error) {
	args := m.Called(cfName, cfSelector, pattern, cursor, limit, now)
	var s []db.KeyValuePair
	if tmp := args.Get(0); tmp != nil {
		s = tmp.([]db.KeyValuePair)
	}
	return s, "", args.Error(2)
}

func (m *MockKVStore) CreateColumnFamily(columnFamilyName string, isTtl bool) error {
	args := m.Called(columnFamilyName, isTtl)
	return args.Error(0)
}

func (m *MockKVStore) DeleteColumnFamily(columnFamilyName string) error {
	args := m.Called(columnFamilyName)
	return args.Error(0)
}

func (m *MockKVStore) ExistsColumnFamily(columnFamilyName string) (bool, bool, error) {
	args := m.Called(columnFamilyName)
	return args.Bool(0), args.Bool(1), args.Error(2)
}

type TestIDGeneratorFactory struct {
	ids   []string
	index int
	mu    sync.Mutex
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

func NewTestIDGeneratorFactory(ids []string) *TestIDGeneratorFactory {
	return &TestIDGeneratorFactory{
		ids: ids,
	}
}

const TestFC = "test_fc"
const DefaultFC = "default"
const TemporalFC = "temporal_fc"

type testEntity struct {
	ID       string `orm:"primary-key"`
	Name     string `orm:"unique"`
	LastName string
	Age      int
	TTL      int `orm:"ttl"`
}

func (testEntity) TableName() string {
	return "users"
}

type ConditionalUniqueEntity struct {
	ID                     string `orm:"primary-key"`
	Name                   string
	UniqueValue            string `orm:"unique,ignore-is-true:ShouldIgnoreUniqueness"`
	ShouldIgnoreUniqueness bool
}

func (e ConditionalUniqueEntity) TableName() string {
	return "conditional_unique_entities"
}

type InvalidConditionalEntityBadRef struct {
	ID          string `orm:"primary-key"`
	UniqueValue string `orm:"ignore-is-true:NonExistentFlag"`
}

func (e InvalidConditionalEntityBadRef) TableName() string { return "invalid_cond_bad_ref" }

type InvalidConditionalEntityBadType struct {
	ID          string `orm:"primary-key"`
	UniqueValue string `orm:"ignore-is-true:NonBoolFlag"`
	NonBoolFlag int
}

func (e InvalidConditionalEntityBadType) TableName() string { return "invalid_cond_bad_type" }

type User struct {
	ID   string `orm:"primary-key"`
	Name string `orm:"unique"`
}

func (User) TableName() string {
	return "users"
}

type InvalidConditionalEmptyField struct {
	ID          string `orm:"primary-key"`
	UniqueValue string `orm:"ignore-is-true:"`
}

func (e InvalidConditionalEmptyField) TableName() string {
	return "conditional_unique_entities"
}

type NestedMetaTest struct {
	UniqueID    string `orm:"unique"`
	OTValue     string
	Description string
}

type NestedEntityTest struct {
	ID   string `orm:"primary-key"`
	Data string
	Meta NestedMetaTest
}

func (NestedEntityTest) TableName() string {
	return "nested_entities_test"
}

type NoPrimary struct {
	Name string
}

func (NoPrimary) TableName() string {
	return "nopk"
}

type TempEntity struct {
	ID    string `orm:"primary-key"`
	Token string `orm:"unique"`
	TTL   int    `orm:"ttl"`
}

func (TempEntity) TableName() string {
	return "temporal_entities"
}

type InvalidTempEntity struct {
	ID    string `orm:"primary-key"`
	Token string `orm:"unique"`
	ttl   int    `orm:"ttl"`
}

func (InvalidTempEntity) TableName() string {
	return "invalid_temporal_entities"
}

type Invalid struct {
	Name string `orm:"unique"`
}

func (Invalid) TableName() string {
	return "invalid"
}

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
const UOWTestFC = "test_fc"
const UOWDefaultFC = "default"
const UOWTemporalFC = "temporal_fc"

type TestCarFixOrder struct {
	ID           string `orm:"primary-key"`
	Code         string `orm:"unique"`
	LicensePlate string
	Amount       float32
}

func (TestCarFixOrder) TableName() string {
	return "car_fix_orders"
}

type TestCar struct {
	ID           string `orm:"primary-key"`
	LicensePlate string `orm:"unique"`
	Name         string
	Model        string
	Performace   float32
	Year         int
}

func (TestCar) TableName() string {
	return "cars"
}

type TestNotification struct {
	ID      string `orm:"primary-key"`
	Content string
	TTL     int `orm:"ttl"`
}

func (TestNotification) TableName() string {
	return "cars"
}
