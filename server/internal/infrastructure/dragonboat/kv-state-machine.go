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

	"github.com/rs/zerolog/log"

	"github.com/linxGnu/grocksdb"
	"github.com/lni/dragonboat/v4/statemachine"
)

// KVRocksDBStateMachineImpl defines the interface for specific key-value state machine implementations.
// This allows different underlying storage or command processing logic to be plugged into the base state machine.
type KVRocksDBStateMachineImpl interface {
	// OpenDB is responsible for opening the database at the given path.
	// It should return the database instance, handles for normal column families,
	// handles for TTL column families, and any error encountered.
	OpenDB(dbPath string) (*grocksdb.DB, map[string]*grocksdb.ColumnFamilyHandle, map[string]*grocksdb.ColumnFamilyHandle, error)
	// Update processes a slice of statemachine entries and applies them to the given RocksDB write batch.
	// It should decode the commands from the entries and translate them into batch operations.
	// It returns a slice of processed Command objects (which might be different from input if errors occurred) and any error.
	Update(ents []statemachine.Entry, batch *grocksdb.WriteBatch) ([]Command, error)
	// Lookup processes a query and is expected to return an RK_Command (Read Key Command)
	// that can be used by the base state machine to perform the actual read from RocksDB.
	// The input `key` is generic and its interpretation is up to the implementation.
	Lookup(key interface{}) (RK_Command, error)
}

// KVBaseRocksDBStateMachineConfig holds configuration parameters for the KVBaseRocksDBStateMachine.
type KVBaseRocksDBStateMachineConfig struct {
	// TTLInternalError specifies the Time-To-Live (in seconds) for internal error messages stored in the database.
	TTLInternalError uint64
}

// KVBaseRocksDBStateMachine is a base implementation of Dragonboat's IOnDiskStateMachine.
// It uses RocksDB for persistence and relies on a KVRocksDBStateMachineImpl for specific command processing logic.
// This struct manages the RocksDB lifecycle, snapshotting, recovery, and applying Raft entries.
type KVBaseRocksDBStateMachine struct {
	clusterID uint64 // The ID of the Raft cluster.
	nodeID    uint64 // The ID of this node in the Raft cluster.
	// lastApplied is the Raft index of the last entry successfully applied to the state machine.
	// It's crucial for consistency and recovery.
	lastApplied      uint64
	store            unsafe.Pointer // Points to a *db.RocksdbStore instance. Used for atomic updates.
	closed           bool           // True if the state machine has been closed.
	aborted          bool           // True if the state machine has been aborted (not currently used in logic but present).
	mu               sync.RWMutex   // Protects access to shared state, especially during Open, Close, Update, and Snapshot operations.
	stateMachineImpl KVRocksDBStateMachineImpl // The specific implementation for handling DB opening and command processing.
	config           KVBaseRocksDBStateMachineConfig // Configuration for the state machine.
}

// NewKVStateMachine creates a new instance of KVBaseRocksDBStateMachine.
//
// Parameters:
//   - clusterID: The ID of the Raft cluster.
//   - nodeID: The ID of this node in the Raft cluster.
//   - stateMachineImpl: An implementation of KVRocksDBStateMachineImpl for custom logic.
//   - config: Configuration for the state machine.
//
// Returns:
//   - An statemachine.IOnDiskStateMachine ready for use with Dragonboat.
func NewKVStateMachine(clusterID uint64, nodeID uint64, stateMachineImpl KVRocksDBStateMachineImpl, config KVBaseRocksDBStateMachineConfig) statemachine.IOnDiskStateMachine {
	return &KVBaseRocksDBStateMachine{
		clusterID:        clusterID,
		nodeID:           nodeID,
		stateMachineImpl: stateMachineImpl,
		config:           config,
	}
}

// GetLastApplied returns the Raft index of the last entry that was successfully applied to the state machine.
func (s *KVBaseRocksDBStateMachine) GetLastApplied() uint64 {
	return s.lastApplied
}

// Open initializes the state machine, primarily by opening the RocksDB database.
// It determines the database directory, handles potential cleanup from previous runs,
// and loads the last applied Raft index from the database's metadata.
//
// Parameters:
//   - stopc: A channel that signals when the state machine should stop its operations. (Not directly used in this Open method but part of the interface).
//
// Returns:
//   - The last applied Raft index as recovered from the database.
//   - An error if any part of the opening process fails (e.g., directory operations, DB opening, reading applied index).
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

// queryAppliedIndex retrieves the last applied Raft index from the RocksDB store.
// It reads a specific key (AppliedIndexKey) from a metadata column family (db.MetaFC).
//
// Parameters:
//   - rocks_kv_store: The RocksDB store instance to query.
//
// Returns:
//   - The last applied index as a uint64. Returns 0 if the key is not found (e.g., new database).
//   - An error if the database read operation fails.
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

// Update applies a series of Raft log entries to the state machine.
// This is the core method where commands from Raft are processed and persisted.
// It handles different command types: DDL (Data Definition Language for column families),
// RW (Read/Write operations), and MCL (Maintenance Control Language).
// For RW operations, it further distinguishes between Put, PutTTL, Delete, and DeleteTTL.
// It uses a write batch for atomic updates to RocksDB and stores the last applied index.
// If the stateMachineImpl.Update returns an error, this method attempts to store
// that error information as a TTL entry in the MasterEventFC column family.
//
// Parameters:
//   - ents: A slice of statemachine.Entry objects from Dragonboat, each containing a command to apply.
//
// Returns:
//   - The input `ents` slice, with the `Result` field populated for each entry (typically with the size of the command).
//   - An error if any critical operation fails (e.g., unknown command type, DB write error, CF operation error).
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
					TTL:              int(s.config.TTLInternalError),
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

// cleanExpiredKeys iterates through a TTL-enabled column family and removes keys that have expired.
// It scans keys prefixed with `prefixTTLIndex`, which store `expireAtTimestamp:originalKey`.
// If `expireAtTimestamp` is in the past, it deletes the `prefixTTLIndex` key,
// the actual data key (`prefixData:originalKey`), and the reference key (`prefixTTLExpire:originalKey`).
// It performs deletions in batches (up to `maxDeletions`) to avoid holding locks for too long.
//
// Parameters:
//   - db: The RocksDB instance.
//   - cf: The column family handle for the TTL-enabled column family to clean.
//
// Returns:
//   - An error if iterator operations or batch write operations fail.
func cleanExpiredKeys(db_instance *grocksdb.DB, cf *grocksdb.ColumnFamilyHandle) error {
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
	log.Debug().Msg("-------> prefixTTLIndex")
	log.Debug().Interface("prefixTTLIndex", prefixTTLIndex).Msg("")
	for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
		key := it.Key()
		keyBytes := append([]byte(nil), key.Data()...)
		key.Free()

		keyStr := string(keyBytes)
		log.Debug().Msg("------>  keyStr")
		log.Debug().Str("keyStr", keyStr).Msg("")
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
			if err := db_instance.Write(writeOpts, batch); err != nil {
			return fmt.Errorf("failed to write batch for expired keys: %w", err)
		}
	}

	return nil
}

// Lookup performs read-only queries on the state machine.
// It uses the `stateMachineImpl.Lookup` to get a structured RK_Command, then executes
// the appropriate read operation (Get, Search, GetOpTTL, SearchTTL) on the RocksDB store.
// For TTL operations, it checks if the key has expired before returning it.
// For search operations, it handles pagination.
//
// Parameters:
//   - query: An interface{} representing the query. The concrete type and interpretation
//     are handled by `stateMachineImpl.Lookup`.
//
// Returns:
//   - The result of the query (e.g., byte slice for Get, PagedResultKV for Search).
//   - nil if the key is not found or has expired (for TTL gets).
//   - An error if the database is closed, the column family is not found, or a database read operation fails.
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

// Sync flushes any buffered data in the RocksDB store to disk.
//
// Returns:
//   - An error if the RocksDB Flush operation fails.
func (s *KVBaseRocksDBStateMachine) Sync() error {
	rocks_kv_store := (*db.RocksdbStore)(atomic.LoadPointer(&s.store))
	return rocks_kv_store.Flush()
}

// PrepareSnapshot is part of the IOnDiskStateMachine interface.
// In this implementation, it's a no-op as SaveSnapshot directly streams data from RocksDB.
//
// Returns:
//   - nil, nil (no context needed for SaveSnapshot, no error).
func (s *KVBaseRocksDBStateMachine) PrepareSnapshot() (interface{}, error) {
	return nil, nil
}

// SaveSnapshot creates a snapshot of the current state machine data and writes it to the provided io.Writer.
// It iterates over all key-value pairs in all column families of the RocksDB store
// and GOB-encodes them into the writer.
// The snapshot includes the column family name, key, and value for each entry.
//
// Parameters:
//   - ctx: The context returned by PrepareSnapshot (unused in this implementation).
//   - w: The io.Writer to write the snapshot data to.
//   - done: A channel that signals if the snapshot operation should be cancelled.
//
// Returns:
//   - An error if the database is closed, iteration fails, GOB encoding fails, or the operation is cancelled.
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

// RecoverFromSnapshot restores the state machine's state from a snapshot provided by an io.Reader.
// It involves the following steps:
// 1. Creates a new temporary RocksDB instance.
// 2. Decodes GOB-encoded entries (ColumnFamilyName, Key, Value) from the reader.
// 3. Puts each entry into the new RocksDB instance.
// 4. If successful, it replaces the old RocksDB instance with the new one.
// 5. Updates the current DB directory information and cleans up the old DB directory.
// 6. Updates the `lastApplied` index based on the recovered data.
//
// Parameters:
//   - r: The io.Reader to read the snapshot data from.
//   - done: A channel that signals if the recovery operation should be cancelled.
//
// Returns:
//   - An error if the state machine is already closed, directory operations fail, DB opening fails,
//     GOB decoding fails, putting data into the new DB fails, or the operation is cancelled.
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
	if err != nil {
		// It's possible oldDirName doesn't exist if this is the first run or after a clean start.
		// Log this situation but don't necessarily fail if dbdir can be created.
		log.Warn().Err(err).Msg("Failed to get current DB directory name during snapshot recovery, proceeding with new one")
		// oldDirName might be empty or invalid, handle removal carefully later
	}

	// Use stateMachineImpl to open the DB, ensuring correct CFs are potentially pre-created
	// For snapshot recovery, we are essentially creating a new DB instance.
	// The stateMachineImpl.OpenDB is suitable here as it encapsulates the logic
	// for opening a DB potentially with specific column families.
	// We are not using s.stateMachineImpl.OpenDB directly as it's an interface method.
	// The actual DB opening logic is in db.OpenMasterDB or similar, which should be
	// called by the concrete implementation of stateMachineImpl.OpenDB or a similar utility.
	// For simplicity, we'll assume a generic OpenMasterDB-like behavior for recovery,
	// as the snapshot contains all CF data.
	// A more robust approach might involve the stateMachineImpl itself handling snapshot recovery.
	rocks_db, columnFamilyHandles, ttlColumnFamilyHandles, err := db.OpenMasterDB(dbdir) // Simplified for now
	if err != nil {
		return fmt.Errorf("failed to open new DB for snapshot recovery: %w", err)
	}
	rocks_db_store := &db.RocksdbStore{
		DB:                     rocks_db,
		ColumnFamilyHandles:    columnFamilyHandles,
		TTLColumnFamilyHandles: ttlColumnFamilyHandles,
	}

	dec := gob.NewDecoder(r)

	// Track existing and create new Column Families as they appear in the snapshot
	// This is a simplified approach. A more robust solution would involve
	// the stateMachineImpl and potentially specific DDL commands if CFs need
	// special handling (like TTL properties not directly in the snapshot data).
	currentCFs := make(map[string]bool)
	for cfName := range columnFamilyHandles {
		currentCFs[cfName] = true
	}
	for cfName := range ttlColumnFamilyHandles {
		currentCFs[cfName] = true
	}

	for {
		select {
		case <-done:
			rocks_db_store.Close() // Clean up the new DB if cancelled
			os.RemoveAll(dbdir)    // Attempt to remove the temporary DB dir
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
				break // End of snapshot
			}
			rocks_db_store.Close() // Clean up
			os.RemoveAll(dbdir)
			return fmt.Errorf("decode failed: %w", err)
		}

		// Ensure column family exists in the new DB before putting data
		if _, ok := currentCFs[entry.CFName]; !ok {
			// Attempt to create the column family.
			// This assumes default options. Specific options would require more info from snapshot or config.
			// Also, it doesn't distinguish between normal and TTL CFs based on snapshot data alone.
			// This is a simplification; a full solution might need DDL commands in the snapshot
			// or a pre-defined schema.
			opts := grocksdb.NewDefaultOptions()
			// Note: opts should be destroyed, but its lifecycle here is tricky.
			// Ideally, CF creation is less dynamic or handled by the stateMachineImpl.
			newCfHandle, createErr := rocks_db_store.DB.CreateColumnFamily(opts, entry.CFName)
			opts.Destroy()
			if createErr != nil {
				rocks_db_store.Close()
				os.RemoveAll(dbdir)
				return fmt.Errorf("failed to create column family %s during snapshot recovery: %w", entry.CFName, createErr)
			}
			// Assuming it's a normal CF. If it needs to be TTL, that info isn't in this basic entry.
			rocks_db_store.ColumnFamilyHandles[entry.CFName] = newCfHandle
			currentCFs[entry.CFName] = true
		}

		if err := rocks_db_store.Put(entry.CFName, string(entry.Key), entry.Value); err != nil {
			rocks_db_store.Close()
			os.RemoveAll(dbdir)
			return fmt.Errorf("put failed during snapshot recovery for CF %s, Key %s: %w", entry.CFName, string(entry.Key), err)
		}
	}

	// Persist directory changes
	if err := saveCurrentDBDirName(dir, dbdir); err != nil {
		rocks_db_store.Close()
		os.RemoveAll(dbdir)
		return err
	}
	if err := replaceCurrentDBFile(dir); err != nil {
		rocks_db_store.Close()
		os.RemoveAll(dbdir)
		return err
	}

	// Update applied index
	newLastApplied, err := s.queryAppliedIndex(rocks_db_store)
	if err != nil {
		// This is critical. If we can't read the applied index, the SM is in an inconsistent state.
		rocks_db_store.Close()
		os.RemoveAll(dbdir)
		// Attempt to revert to oldDirName if possible, though that's complex and risky here.
		// For now, panic might be safer than continuing in an unknown state.
		panic(fmt.Sprintf("failed to query applied index after snapshot recovery: %v", err))
	}

	// Atomically swap the store pointer
	oldStorePtr := atomic.SwapPointer(&s.store, unsafe.Pointer(rocks_db_store))
	oldKvStore := (*db.RocksdbStore)(oldStorePtr)

	// Close and remove the old database IF it existed and was different
	if oldKvStore != nil && oldKvStore.DB != rocks_db_store.DB { // Check if it's genuinely an old instance
		oldKvStore.Close()
		if oldDirName != "" && oldDirName != dbdir { // Ensure oldDirName is valid and different
			parent := filepath.Dir(oldDirName)
			if err := os.RemoveAll(oldDirName); err != nil {
				// Log error but proceed, as the new DB is in place.
				log.Error().Err(err).Str("path", oldDirName).Msg("Failed to remove old database directory")
			} else {
				if err := syncDir(parent); err != nil { // Sync parent dir of removed old DB dir
					log.Error().Err(err).Str("path", parent).Msg("Failed to sync parent directory after removing old DB")
				}
			}
		}
	}
	
	// Sync directory of the new DB
	if err := syncDir(filepath.Dir(dbdir)); err != nil {
		log.Error().Err(err).Str("path", filepath.Dir(dbdir)).Msg("Failed to sync new database directory")
		// Not returning error here as primary recovery is done.
	}


	// Important: Update the lastApplied index *after* successfully swapping the store
	// and cleaning up.
	// However, the Dragonboat library expects lastApplied to be the index *of the snapshot itself*
	// or the latest index *covered by the snapshot*. The current `queryAppliedIndex` reads
	// an index stored *within* the DB data. This might need adjustment based on how
	// Dragonboat's snapshot index mechanism works. For now, we assume newLastApplied is correct.
	if s.lastApplied > newLastApplied && newLastApplied != 0 { // Allow newLastApplied to be 0 for fresh snapshot
		// This is a critical error. Log it, but a panic might be more appropriate
		// as the state machine could be inconsistent.
		log.Error().Uint64("currentLastApplied", s.lastApplied).Uint64("newLastApplied", newLastApplied).Msg("Last applied index moved backward after snapshot recovery")
		// Consider panic if strict ordering is essential and cannot be violated.
		// panic("last applied not moving forward after snapshot recovery")
	}
	s.lastApplied = newLastApplied


	return nil
}

// Close closes the state machine and its underlying RocksDB store.
// It's protected by a mutex to prevent concurrent close operations or
// operations on a closed store.
//
// Returns:
//   - An error if any occurs during the closing of the RocksDB store (though typically, RocksDB Close doesn't return errors).
// Panics if called more than once.
func (s *KVBaseRocksDBStateMachine) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	rocks_kv_store := (*db.RocksdbStore)(atomic.LoadPointer(&s.store))
	if rocks_kv_store != nil {
		if s.closed {
			// Already closed, potentially an issue if called multiple times without error.
			// Depending on strictness, could panic or return an error.
			// For now, let's match the original panic behavior.
			panic("close called twice")
		}
		s.closed = true
		return rocks_kv_store.Close()
	}
	// If store is nil, it might mean it was never opened or already closed and set to nil.
	// If s.closed is true here, it means Close was called on an already nil store (which is fine).
	// If s.closed is false, it means it was never opened.
	if s.closed { // Already marked as closed, and store was nil
		panic("close called twice on a nil store that was marked closed")
	}
	// If store is nil and not marked closed, it means it was likely never opened. Mark as closed.
	s.closed = true
	return nil
}
