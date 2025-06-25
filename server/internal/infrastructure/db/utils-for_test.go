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
	testTTLSeconds = 3
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

func (m *MockKVStore) Get(AdminFC, key string, now time.Time) ([]byte, error) {
	args := m.Called(AdminFC, key, now)
	var s []byte
	if tmp := args.Get(0); tmp != nil {
		s = tmp.([]byte)
	}
	return s, args.Error(1)
}

func (m *MockKVStore) Delete(AdminFC, key string, now time.Time) error {
	args := m.Called(AdminFC, key, now)
	return args.Error(0)
}

func (r *MockKVStore) Exists(columnFamily, key string, now time.Time) (bool, error) {
	args := r.Called(columnFamily, key, now)
	return args.Bool(0), args.Error(1)
}

func (m *MockKVStore) Put(AdminFC, key string, value []byte, ttl int, now time.Time) error {
	args := m.Called(AdminFC, key, value, ttl, now)
	return args.Error(0)
}

func (m *MockKVStore) PutRaw(AdminFC, key string, value []byte) error {
	args := m.Called(AdminFC, key, value)
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

func (r *MockKVStore) Iterate(fn func(cfName string, key, value []byte) error) error {
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

func (m *MockKVStore) SearchByPatternPaginatedKV(cfName, pattern, cursor string, limit int, now time.Time) ([]db.KeyValuePair, string, error) {
	args := m.Called(cfName, pattern, cursor, limit, now)
	var s []db.KeyValuePair
	if tmp := args.Get(0); tmp != nil {
		s = tmp.([]db.KeyValuePair)
	}
	return s, "", args.Error(2)
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
