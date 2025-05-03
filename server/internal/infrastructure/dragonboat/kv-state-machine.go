package dragonboat

import (
	"bytes"
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

type KVStateMachine struct {
	clusterID   uint64
	nodeID      uint64
	lastApplied uint64
	store       unsafe.Pointer // we will use db.KVStore
	closed      bool
	aborted     bool
	mu          sync.RWMutex
}

func NewKVStateMachine(clusterID uint64, nodeID uint64) statemachine.IOnDiskStateMachine {
	return &KVStateMachine{
		clusterID: clusterID,
		nodeID:    nodeID,
	}
}
func (s *KVStateMachine) GetLastApplied() uint64 {
	return s.lastApplied
}
func (s *KVStateMachine) Open(stopc <-chan struct{}) (uint64, error) {
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
	rocks_db, columnFamilyHandles, err := db.OpenDB(dbdir)
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

func (s *KVStateMachine) queryAppliedIndex(rocks_kv_store *db.RocksdbStore) (uint64, error) {
	result, err := rocks_kv_store.Get(db.MetaFC, AppliedIndexKey)
	if err != nil {
		return 0, err
	}

	if len(result) == 0 {
		return 0, nil
	}
	return binary.LittleEndian.Uint64(result), nil
}

func (s *KVStateMachine) Update(ents []statemachine.Entry) ([]statemachine.Entry, error) {
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

	var dllFCEntries []int
	var rwEntries []int

	commands := make([]Command, len(ents))
	for i, ent := range ents {
		var cmd Command
		if err := gob.NewDecoder(bytes.NewReader(ent.Cmd)).Decode(&cmd); err != nil {
			return nil, err
		}
		commands[i] = cmd

		switch cmd.Type {
		case DLL_FC:
			dllFCEntries = append(dllFCEntries, i)
		case RW:
			rwEntries = append(rwEntries, i)
		default:
			return nil, fmt.Errorf("unknown command type: %v", cmd.Type)
		}
	}

	for _, idx := range dllFCEntries {
		cmd := commands[idx]
		ddlCmd, ok := cmd.CMD.(DDL_Command)
		if !ok {
			return nil, fmt.Errorf("expected DDL_Command for DLL type, got %T", cmd.CMD)
		}
		switch ddlCmd.Op {
		case Add_CF_Op:
			cfName := ddlCmd.ColumnFamilyName
			if cfName == "" {
				return nil, fmt.Errorf("the family column name cannot be empty")
			}

			if _, exists := rocks_kv_store.ColumnFamilyHandles[cfName]; !exists {
				opts := grocksdb.NewDefaultOptions()
				defer opts.Destroy()

				cfh, err := rocks_kv_store.DB.CreateColumnFamily(opts, cfName)
				if err != nil {
					return nil, fmt.Errorf("error creando CF %s: %w", cfName, err)
				}
				rocks_kv_store.ColumnFamilyHandles[cfName] = cfh
			}
		case Remove_CF_Op:
			cfh := rocks_kv_store.ColumnFamilyHandles[ddlCmd.ColumnFamilyName]
			if cfh == nil {
				return nil, fmt.Errorf("Column Family not found %T to Drop process", ddlCmd.ColumnFamilyName)
			}
			err := rocks_kv_store.DB.DropColumnFamily(cfh)
			if err != nil {
				return nil, err
			}
			cfh.Destroy()
		}
		ents[idx].Result = statemachine.Result{Value: uint64(len(ents[idx].Cmd))}
	}

	batch := grocksdb.NewWriteBatch()
	defer batch.Destroy()
	for _, idx := range rwEntries {
		cmd := commands[idx]
		rwCmd, ok := cmd.CMD.(RWK_Command)
		if !ok {
			return nil, fmt.Errorf("expected RWK_Command for RW type, got %T", cmd.CMD)
		}
		switch rwCmd.Op {
		case PutOp:
			cfh := rocks_kv_store.ColumnFamilyHandles[rwCmd.ColumnFamilyName]
			if cfh == nil {
				return nil, fmt.Errorf("Column Family not found: %s", rwCmd.ColumnFamilyName)
			}
			batch.PutCF(cfh, []byte(rwCmd.Key), rwCmd.Value)
		case DeleteOp:
			cfh := rocks_kv_store.ColumnFamilyHandles[rwCmd.ColumnFamilyName]
			if cfh == nil {
				return nil, fmt.Errorf("Column Family not found %s", rwCmd.ColumnFamilyName)
			}
			batch.DeleteCF(cfh, []byte(rwCmd.Key))
		default:
			return nil, fmt.Errorf("unknown RW Operation: %v", rwCmd.Op)
		}
		ents[idx].Result = statemachine.Result{Value: uint64(len(ents[idx].Cmd))}
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

func (s *KVStateMachine) Lookup(key interface{}) (interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rocks_kv_store := (*db.RocksdbStore)(atomic.LoadPointer(&s.store))
	if rocks_kv_store != nil {

		lookupQuery, ok := key.(LookupQuery)
		if !ok {
			return nil, fmt.Errorf("expected key to be string, got %T", key)
		}

		data, err := rocks_kv_store.Get(lookupQuery.ColumnFamilyName, lookupQuery.Key)

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

func (s *KVStateMachine) Sync() error {
	rocks_kv_store := (*db.RocksdbStore)(atomic.LoadPointer(&s.store))
	return rocks_kv_store.Flush()
}

func (s *KVStateMachine) PrepareSnapshot() (interface{}, error) {
	return nil, nil
}

func (s *KVStateMachine) SaveSnapshot(
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

func (s *KVStateMachine) RecoverFromSnapshot(
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

	rocks_db, columnFamilyHandles, err := db.OpenDB(dbdir)
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

func (s *KVStateMachine) Close() error {
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
