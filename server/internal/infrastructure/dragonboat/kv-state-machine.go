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
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/linxGnu/grocksdb"
	"github.com/lni/dragonboat/v4/statemachine"
)

type KVRocksDBStateMachineImpl interface {
	OpenDB(dbPath string) (*grocksdb.DB, map[string]*grocksdb.ColumnFamilyHandle, map[string]*grocksdb.ColumnFamilyHandle, error)
	Update(ents []statemachine.Entry, batch *grocksdb.WriteBatch) ([]Command, error)
	Lookup(key interface{}) (RK_Command, error)
}
type KVBaseRocksDBStateMachineConfig struct {
	InternalErrorTTL uint64
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
	config           KVBaseRocksDBStateMachineConfig
}

func NewKVStateMachine(clusterID uint64, nodeID uint64, stateMachineImpl KVRocksDBStateMachineImpl, config KVBaseRocksDBStateMachineConfig) statemachine.IOnDiskStateMachine {
	return &KVBaseRocksDBStateMachine{
		clusterID:        clusterID,
		nodeID:           nodeID,
		stateMachineImpl: stateMachineImpl,
		config:           config,
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
	rocks_db, columnFamilyHandles, ttlColumnFamilyHandles, err := db.OpenMasterDB(dbdir)
	if err != nil {
		return 0, err
	}
	store := &db.RocksdbStore{
		DB:                     rocks_db,
		ColumnFamilyHandles:    columnFamilyHandles,
		TTLColumnFamilyHandles: ttlColumnFamilyHandles,
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
	commands, err := s.stateMachineImpl.Update(ents, batch)

	if err != nil {
		error_key := time.Now().UnixMilli()
		cmd := Command{
			Type: RW,
			CMD: RWK_Command{
				Op: Write,
				CMD: WK_Command{
					Key:              "internal-errors:" + fmt.Sprintf("%020d", error_key),
					Value:            []byte(err.Error()),
					ColumnFamilyName: db.MasterEventFC,
					TTL:              int(s.config.InternalErrorTTL),
					Op:               PutOpTTL,
				},
			},
		}
		commands = []Command{
			cmd,
		}
	}

	var dllFCEntries []int
	var rwEntries []int
	var mclEntries []int

	for i, cmd := range commands {
		switch cmd.Type {
		case DLL_FC:
			dllFCEntries = append(dllFCEntries, i)
		case RW:
			rwEntries = append(rwEntries, i)
		case MCL:
			mclEntries = append(mclEntries, i)
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
		case Add_TTL_CF_Op:
			cfName := ddlCmd.ColumnFamilyName
			if cfName == "" {
				return nil, fmt.Errorf("the family column name cannot be empty")
			}

			if _, exists := rocks_kv_store.TTLColumnFamilyHandles[cfName]; !exists {
				opts := grocksdb.NewDefaultOptions()
				defer opts.Destroy()

				cfh, err := rocks_kv_store.DB.CreateColumnFamily(opts, cfName)
				if err != nil {
					return nil, fmt.Errorf("error creando CF %s: %w", cfName, err)
				}
				rocks_kv_store.TTLColumnFamilyHandles[cfName] = cfh
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
		case Remove_TTL_CF_Op:
			cfh := rocks_kv_store.TTLColumnFamilyHandles[ddlCmd.ColumnFamilyName]
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

	for _, idx := range rwEntries {
		cmd := commands[idx]
		rwCmd, ok := cmd.CMD.(RWK_Command)
		if !ok {
			return nil, fmt.Errorf("expected RWK_Command for RW type, got %T", cmd.CMD)
		}
		switch rwCmd.Op {
		case Read:
			return nil, fmt.Errorf("Invalid read operation: %T", cmd.CMD)
		case Write:
			wCmd, ok := rwCmd.CMD.(WK_Command)
			if !ok {
				return nil, fmt.Errorf("expected WK_Command for RW type, got %T", cmd.CMD)
			}
			switch wCmd.Op {

			case PutOp:
				cfh := rocks_kv_store.ColumnFamilyHandles[wCmd.ColumnFamilyName]
				if cfh == nil {
					return nil, fmt.Errorf("Column Family not found: %s", wCmd.ColumnFamilyName)
				}
				batch.PutCF(cfh, []byte(wCmd.Key), wCmd.Value)
			case PutOpTTL:
				cfh := rocks_kv_store.TTLColumnFamilyHandles[wCmd.ColumnFamilyName]
				if cfh == nil {
					return nil, fmt.Errorf("Column Family not found: %s", wCmd.ColumnFamilyName)
				}

				ttlMillis := time.Now().Add(time.Duration(wCmd.TTL) * time.Second).UnixMilli()

				ttlRealKey := fmt.Sprintf("%s%s", prefixData, wCmd.Key)
				ttlExpireIndexKey := fmt.Sprintf("%s%s", prefixTTLExpire, wCmd.Key)

				oldTTLBytes, err := rocks_kv_store.Get(wCmd.ColumnFamilyName, ttlExpireIndexKey)
				if err != nil {
					return nil, fmt.Errorf("error reading previous TTL for key %s: %w", wCmd.Key, err)
				}
				if oldTTLBytes != nil {
					oldTTLMillis, err := strconv.ParseInt(string(oldTTLBytes), 10, 64)
					if err == nil {
						oldTTLIndexKey := fmt.Sprintf("%s%020d:%s", prefixTTLIndex, oldTTLMillis, wCmd.Key)
						batch.DeleteCF(cfh, []byte(oldTTLIndexKey))
					}
				}

				batch.PutCF(cfh, []byte(ttlRealKey), wCmd.Value)

				newTTLIndexKey := fmt.Sprintf("%s%020d:%s", prefixTTLIndex, ttlMillis, wCmd.Key)
				batch.PutCF(cfh, []byte(newTTLIndexKey), nil)

				batch.PutCF(cfh, []byte(ttlExpireIndexKey), []byte(strconv.FormatInt(ttlMillis, 10)))

			case DeleteOp:
				cfh := rocks_kv_store.ColumnFamilyHandles[wCmd.ColumnFamilyName]
				if cfh == nil {
					return nil, fmt.Errorf("Column Family not found %s", wCmd.ColumnFamilyName)
				}
				batch.DeleteCF(cfh, []byte(wCmd.Key))
			case DeleteOpTTL:
				cfh := rocks_kv_store.TTLColumnFamilyHandles[wCmd.ColumnFamilyName]
				if cfh == nil {
					return nil, fmt.Errorf("Column Family not found: %s", wCmd.ColumnFamilyName)
				}

				ttlExpireIndexKey := fmt.Sprintf("%s%s", prefixTTLExpire, wCmd.Key)
				oldTTLBytes, err := rocks_kv_store.Get(wCmd.ColumnFamilyName, ttlExpireIndexKey)
				if err != nil {
					return nil, fmt.Errorf("error reading previous TTL for key %s: %w", wCmd.Key, err)
				}

				if oldTTLBytes != nil {
					oldTTLMillis, err := strconv.ParseInt(string(oldTTLBytes), 10, 64)
					if err == nil {
						oldTTLIndexKey := fmt.Sprintf("%s%020d:%s", prefixTTLIndex, oldTTLMillis, wCmd.Key)
						batch.DeleteCF(cfh, []byte(oldTTLIndexKey))
					}
				}

				ttlRealKey := fmt.Sprintf("%s%s", prefixData, wCmd.Key)
				batch.DeleteCF(cfh, []byte(ttlRealKey))
				batch.DeleteCF(cfh, []byte(ttlExpireIndexKey))
			default:
				return nil, fmt.Errorf("unknown W Operation: %v", wCmd.Op)

			}
		default:
			return nil, fmt.Errorf("unknown RW Operation: %v", rwCmd.Op)
		}
		ents[idx].Result = statemachine.Result{Value: uint64(len(ents[idx].Cmd))}
	}

	for _, idx := range mclEntries {
		cmd := commands[idx]
		mlcCmd, ok := cmd.CMD.(MCLK_Command)
		if !ok {
			return nil, fmt.Errorf("expected MCLK_Command for RW type, got %T", cmd.CMD)
		}
		switch mlcCmd.Op {
		case ClearExpiredTTL:
			for name, handle := range rocks_kv_store.TTLColumnFamilyHandles {
				err = cleanExpiredKeys(rocks_kv_store.DB, handle)
				if err != nil {
					return nil, fmt.Errorf("error cleaning expired keys for CF %s: %w", name, err)
				}
			}
		default:
			return nil, fmt.Errorf("unknown MCL Operation: %v", mlcCmd.Op)
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

func cleanExpiredKeys(db *grocksdb.DB, cf *grocksdb.ColumnFamilyHandle) error {
	const maxDeletions = 1000
	var deleted int64

	readOpts := grocksdb.NewDefaultReadOptions()
	defer readOpts.Destroy()

	writeOpts := grocksdb.NewDefaultWriteOptions()
	defer writeOpts.Destroy()

	batch := grocksdb.NewWriteBatch()
	defer batch.Destroy()

	it := db.NewIteratorCF(readOpts, cf)
	defer it.Close()

	nowMillis := time.Now().UnixMilli()
	prefix := []byte(prefixTTLIndex)
	fmt.Println("-------> prefixTTLIndex")
	fmt.Println(prefixTTLIndex)
	for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
		key := it.Key()
		keyBytes := append([]byte(nil), key.Data()...)
		key.Free()

		keyStr := string(keyBytes)
		fmt.Println("------>  keyStr")
		fmt.Println(keyStr)
		trimmed := strings.TrimPrefix(keyStr, prefixTTLIndex)
		sepIdx := strings.IndexByte(trimmed, ':')
		if sepIdx <= 0 || sepIdx >= len(trimmed)-1 {
			continue
		}

		expireAtStr := trimmed[:sepIdx]
		originalKey := trimmed[sepIdx+1:]

		expireAt, err := strconv.ParseInt(expireAtStr, 10, 64)
		if err != nil {
			continue
		}

		if expireAt > nowMillis {
			break
		}

		dataKey := []byte(prefixData + originalKey)
		expireRefKey := []byte(prefixTTLExpire + originalKey)

		ro := grocksdb.NewDefaultReadOptions()
		defer ro.Destroy()

		batch.DeleteCF(cf, dataKey)
		batch.DeleteCF(cf, expireRefKey)
		batch.DeleteCF(cf, keyBytes)

		deleted++
		if deleted >= maxDeletions {
			break
		}
	}

	if err := it.Err(); err != nil {
		return fmt.Errorf("iterator error: %w", err)
	}

	if deleted > 0 {
		if err := db.Write(writeOpts, batch); err != nil {
			return fmt.Errorf("failed to write batch for expired keys: %w", err)
		}
	}

	return nil
}

func (s *KVBaseRocksDBStateMachine) Lookup(query interface{}) (interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rocks_kv_store := (*db.RocksdbStore)(atomic.LoadPointer(&s.store))
	if rocks_kv_store != nil {

		query, err := s.stateMachineImpl.Lookup(query)

		if err == nil && s.closed {
			return nil, errors.New("lookup returned valid result when DiskKV is already closed")
		}

		switch query.Op {

		case GetOp:
			var data []byte
			cfh := rocks_kv_store.ColumnFamilyHandles[query.ColumnFamilyName]
			if cfh == nil {
				return nil, fmt.Errorf("Column Family not found %s", query.ColumnFamilyName)
			}
			data, err = rocks_kv_store.Get(query.ColumnFamilyName, query.Key)
			if err != nil {
				return nil, err
			}
			if data != nil {
				return data, err
			}
		case Search:
			cfh := rocks_kv_store.ColumnFamilyHandles[query.ColumnFamilyName]
			if cfh == nil {
				return nil, fmt.Errorf("Column Family not found %s", query.ColumnFamilyName)
			}

			pairs, nextCursor, err := rocks_kv_store.SearchByPatternPaginatedKV(
				query.ColumnFamilyName,
				query.KeyPatter,
				query.Cursor,
				int(query.Limit),
			)
			if err != nil {
				return nil, err
			}

			if len(pairs) > 0 {
				result := &PagedResultKV{
					Data:       pairs, // Data ahora es []KeyValuePair
					NextCursor: []byte(nextCursor),
				}
				return result, nil
			}

		case GetOpTTL:
			var data []byte
			cfh := rocks_kv_store.TTLColumnFamilyHandles[query.ColumnFamilyName]
			if cfh == nil {
				return nil, fmt.Errorf("Column Family not found %s", query.ColumnFamilyName)
			}
			expireKey := fmt.Sprintf("%s%s", prefixTTLExpire, query.Key)
			expireBytes, err := rocks_kv_store.Get(query.ColumnFamilyName, expireKey)
			if err != nil {
				return nil, fmt.Errorf("failed to read expire key: %w", err)
			}
			if len(expireBytes) == 0 {
				return nil, nil
			}

			expireAt, err := strconv.ParseInt(string(expireBytes), 10, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid expire timestamp: %w", err)
			}

			if time.Now().UnixMilli() > expireAt {
				return nil, nil
			}

			dataKey := fmt.Sprintf("%s%s", prefixData, query.Key)
			data, err = rocks_kv_store.Get(query.ColumnFamilyName, dataKey)

			if err != nil {
				return nil, err
			}
			if data != nil {
				return data, err
			}
		case SearchTTL:
			cfh := rocks_kv_store.TTLColumnFamilyHandles[query.ColumnFamilyName]
			if cfh == nil {
				return nil, fmt.Errorf("Column Family not found %s", query.ColumnFamilyName)
			}

			var resultData []db.KeyValuePair
			cursor := query.Cursor
			remaining := int(query.Limit)

			for remaining > 0 {
				keyPatter := fmt.Sprintf("%s%s", prefixData, query.KeyPatter)
				pairs, nextCursor, err := rocks_kv_store.SearchByPatternPaginatedKV(
					query.ColumnFamilyName,
					keyPatter,
					cursor,
					remaining*2,
				)
				if err != nil {
					return nil, err
				}

				for _, pair := range pairs {
					key := strings.TrimPrefix(pair.Key, prefixData)
					expireKey := fmt.Sprintf("%s%s", prefixTTLExpire, key)

					expireBytes, err := rocks_kv_store.Get(query.ColumnFamilyName, expireKey)
					if err != nil || len(expireBytes) == 0 {
						continue
					}

					expireAt, err := strconv.ParseInt(string(expireBytes), 10, 64)
					if err != nil || time.Now().UnixMilli() > expireAt {
						continue
					}

					resultData = append(resultData, pair)
					remaining--

					if remaining == 0 {
						cursor = nextCursor
						break
					}
				}

				if nextCursor == "" {
					cursor = ""
					break
				}
				cursor = nextCursor
			}

			return &PagedResultKV{
				Data:       resultData,
				NextCursor: []byte(cursor),
			}, nil

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

	rocks_db, columnFamilyHandles, ttlColumnFamilyHandles, err := db.OpenMasterDB(dbdir)
	if err != nil {
		return err
	}
	rocks_db_store := &db.RocksdbStore{
		DB:                     rocks_db,
		ColumnFamilyHandles:    columnFamilyHandles,
		TTLColumnFamilyHandles: ttlColumnFamilyHandles,
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
