package dragonboat

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/linxGnu/grocksdb"
	"github.com/lni/dragonboat/v4/statemachine"
)

type KVRocksDBStateMachineImpl interface {
	OpenDB(dbPath string) (*grocksdb.DB, map[string]*grocksdb.ColumnFamilyHandle, error)
	Update(rocks_kv_store *db.RocksdbStore, ents []statemachine.Entry, batch *grocksdb.WriteBatch) ([]statemachine.Entry, error)
	Lookup(rocks_kv_store *db.RocksdbStore, key LookupQuery) (interface{}, error)
}

type KVBaseRocksDBStateMachine struct {
	clusterID        uint64
	nodeID           uint64
	lastApplied      uint64
	store            unsafe.Pointer // we will use db.KVStore
	closed           bool
	aborted          bool
	mu               sync.RWMutex
	stateMachineImpl KVRocksDBStateMachineImpl
}

func NewKVStateMachine(clusterID uint64, nodeID uint64, stateMachineImpl KVRocksDBStateMachineImpl) statemachine.IOnDiskStateMachine {
	return &KVBaseRocksDBStateMachine{
		clusterID:        clusterID,
		nodeID:           nodeID,
		stateMachineImpl: stateMachineImpl,
	}
}
func (s *KVBaseRocksDBStateMachine) GetLastApplied() uint64 {
	return s.lastApplied
}
func (s *KVBaseRocksDBStateMachine) Open(stopc <-chan struct{}) (uint64, error) {
	dir, err := getNodeDBDirName(s.clusterID, s.nodeID)
	if err != nil {
		panic(err)
	}
	if err := createNodeDataDir(dir); err != nil {
		panic(err)
	}
	var dbdir string
	if !isNewRun(dir) {
		if err := cleanupNodeDataDir(dir); err != nil {
			return 0, err
		}
		var err error
		dbdir, err = getCurrentDBDirName(dir)
		if err != nil {
			return 0, err
		}
		if _, err := os.Stat(dbdir); err != nil {
			if os.IsNotExist(err) {
				panic("db dir unexpectedly deleted")
			}
		}
	} else {
		dbdir = getNewRandomDBDirName(dir)
		if err := saveCurrentDBDirName(dir, dbdir); err != nil {
			return 0, err
		}
		if err := replaceCurrentDBFile(dir); err != nil {
			return 0, err
		}
	}
	rocks_db, columnFamilyHandles, err := db.OpenMasterDB(dbdir)
	if err != nil {
		return 0, err
	}
	store := &db.RocksdbStore{
		DB:                  rocks_db,
		ColumnFamilyHandles: columnFamilyHandles,
	}
	atomic.SwapPointer(&s.store, unsafe.Pointer(store))
	appliedIndex, err := s.queryAppliedIndex(store)
	if err != nil {
		panic(err)
	}
	s.lastApplied = appliedIndex
	return appliedIndex, nil
}

func (s *KVBaseRocksDBStateMachine) queryAppliedIndex(rocks_kv_store *db.RocksdbStore) (uint64, error) {
	result, err := rocks_kv_store.Get(db.MetaFC, AppliedIndexKey)
	if err != nil {
		return 0, err
	}

	if len(result) == 0 {
		return 0, nil
	}
	return binary.LittleEndian.Uint64(result), nil
}

func (s *KVBaseRocksDBStateMachine) Update(ents []statemachine.Entry) ([]statemachine.Entry, error) {
	if s.aborted {
		panic("update() called after abort set to true")
	}
	if s.closed {
		panic("update called after Close()")
	}
	if len(ents) == 0 {
		return nil, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	rocks_kv_store := (*db.RocksdbStore)(atomic.LoadPointer(&s.store))
	batch := grocksdb.NewWriteBatch()
	defer batch.Destroy()
	ents, err := s.stateMachineImpl.Update(rocks_kv_store, ents, batch)

	if err != nil {
		return nil, err
	}

	appliedIndex := make([]byte, 8)
	binary.LittleEndian.PutUint64(appliedIndex, ents[len(ents)-1].Index)
	batch.PutCF(rocks_kv_store.ColumnFamilyHandles[db.MetaFC], []byte(AppliedIndexKey), appliedIndex)

	if err := rocks_kv_store.Write(batch); err != nil {
		return nil, err
	}

	if s.lastApplied >= ents[len(ents)-1].Index {
		return nil, fmt.Errorf("lastApplied not moving forward: current=%d new=%d", s.lastApplied, ents[len(ents)-1].Index)
	}
	s.lastApplied = ents[len(ents)-1].Index
	return ents, nil
}

type LookupQuery struct {
	Key              string
	ColumnFamilyName string
}

func (s *KVBaseRocksDBStateMachine) Lookup(key interface{}) (interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rocks_kv_store := (*db.RocksdbStore)(atomic.LoadPointer(&s.store))
	if rocks_kv_store != nil {

		lookupQuery, ok := key.(LookupQuery)
		if !ok {
			return nil, fmt.Errorf("expected key to be string, got %T", key)
		}

		data, err := s.stateMachineImpl.Lookup(rocks_kv_store, lookupQuery)

		if err == nil && s.closed {
			panic("lookup returned valid result when DiskKV is already closed")
		}

		if data != nil {
			return data, err
		}
		return nil, nil
	}
	return nil, errors.New("db closed")
}

func (s *KVBaseRocksDBStateMachine) Sync() error {
	rocks_kv_store := (*db.RocksdbStore)(atomic.LoadPointer(&s.store))
	return rocks_kv_store.Flush()
}

func (s *KVBaseRocksDBStateMachine) PrepareSnapshot() (interface{}, error) {
	return nil, nil
}

func (s *KVBaseRocksDBStateMachine) SaveSnapshot(
	ctx interface{},
	w io.Writer,
	done <-chan struct{},
) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rocks_kv_store := (*db.RocksdbStore)(atomic.LoadPointer(&s.store))

	if rocks_kv_store == nil {
		return errors.New("db closed")
	}

	enc := gob.NewEncoder(w)

	err := rocks_kv_store.Iterate(func(cfName string, key, value []byte) error {
		select {
		case <-done:
			return fmt.Errorf("snapshot cancelled")
		default:
		}

		entry := struct {
			CFName string
			Key    []byte
			Value  []byte
		}{
			CFName: cfName,
			Key:    append([]byte(nil), key...),
			Value:  append([]byte(nil), value...),
		}

		return enc.Encode(&entry)
	})

	if err != nil {
		return fmt.Errorf("snapshot save failed: %w", err)
	}

	return nil
}

func (s *KVBaseRocksDBStateMachine) RecoverFromSnapshot(
	r io.Reader,
	done <-chan struct{},
) error {
	if s.closed {
		panic("recover from snapshot called after Close()")
	}

	dir, err := getNodeDBDirName(s.clusterID, s.nodeID)
	if err != nil {
		return err
	}
	dbdir := getNewRandomDBDirName(dir)
	oldDirName, err := getCurrentDBDirName(dir)

	rocks_db, columnFamilyHandles, err := db.OpenMasterDB(dbdir)
	if err != nil {
		return err
	}
	rocks_db_store := &db.RocksdbStore{
		DB:                  rocks_db,
		ColumnFamilyHandles: columnFamilyHandles,
	}

	dec := gob.NewDecoder(r)

	for {
		select {
		case <-done:
			return fmt.Errorf("snapshot recovery cancelled")
		default:
		}

		var entry struct {
			CFName string
			Key    []byte
			Value  []byte
		}

		if err := dec.Decode(&entry); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("decode failed: %w", err)
		}

		if err := rocks_db_store.Put(entry.CFName, string(entry.Key), entry.Value); err != nil {
			return fmt.Errorf("put failed during snapshot recovery: %w", err)
		}
	}

	if err := saveCurrentDBDirName(dir, dbdir); err != nil {
		return err
	}
	if err := replaceCurrentDBFile(dir); err != nil {
		return err
	}
	newLastApplied, err := s.queryAppliedIndex(rocks_db_store)
	if err != nil {
		panic(err)
	}

	if s.lastApplied > newLastApplied {
		panic("last applied not moving forward")
	}
	s.lastApplied = newLastApplied
	old := (*db.RocksdbStore)(atomic.SwapPointer(&s.store, unsafe.Pointer(rocks_db_store)))

	if old != nil {
		old.Close()
	}
	parent := filepath.Dir(oldDirName)
	if err := os.RemoveAll(oldDirName); err != nil {
		return err
	}
	return syncDir(parent)
}

func (s *KVBaseRocksDBStateMachine) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	rocks_kv_store := (*db.RocksdbStore)(atomic.LoadPointer(&s.store))
	if rocks_kv_store != nil && !s.closed {
		s.closed = true
		rocks_kv_store.Close()
	} else {
		if s.closed {
			panic("close called twice")
		}
	}
	return nil
}
