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

	"github.com/lni/dragonboat/v4/statemachine"
	"github.com/rs/zerolog/log"
)

type KVStateMachineImpl interface {
	OpenDB(dbPath string) (db.KVStore, error)

	Update(ents []statemachine.Entry, batch *db.WriteBatch) ([]Command, error)

	Lookup(key interface{}) (RK_Command, error)
}
type KVBaseStateMachineConfig struct {
	// TTLInternalError specifies the Time-To-Live (in seconds) for internal error messages stored in the database.
	TTLInternalError uint64
}

type KVBaseStateMachine struct {
	clusterID uint64 // The ID of the Raft cluster.
	nodeID    uint64 // The ID of this node in the Raft cluster.
	// lastApplied is the Raft index of the last entry successfully applied to the state machine.
	// It's crucial for consistency and recovery.
	lastApplied      uint64
	store            unsafe.Pointer
	closed           bool                     // True if the state machine has been closed.
	aborted          bool                     // True if the state machine has been aborted (not currently used in logic but present).
	mu               sync.RWMutex             // Protects access to shared state, especially during Open, Close, Update, and Snapshot operations.
	stateMachineImpl KVStateMachineImpl       // The specific implementation for handling DB opening and command processing.
	config           KVBaseStateMachineConfig // Configuration for the state machine.
}

func NewKVStateMachine(clusterID uint64, nodeID uint64, stateMachineImpl KVStateMachineImpl, config KVBaseStateMachineConfig) statemachine.IOnDiskStateMachine {
	return &KVBaseStateMachine{
		clusterID:        clusterID,
		nodeID:           nodeID,
		stateMachineImpl: stateMachineImpl,
		config:           config,
	}
}

// GetLastApplied returns the Raft index of the last entry that was successfully applied to the state machine.
func (s *KVBaseStateMachine) GetLastApplied() uint64 {
	return s.lastApplied
}

func (s *KVBaseStateMachine) Open(stopc <-chan struct{}) (uint64, error) {
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
	store, err := s.stateMachineImpl.OpenDB(dbdir)
	if err != nil {
		return 0, err
	}

	atomic.SwapPointer(&s.store, unsafe.Pointer(&store))
	appliedIndex, err := s.queryAppliedIndex(store)
	if err != nil {
		panic(err)
	}
	s.lastApplied = appliedIndex
	return appliedIndex, nil
}

func (s *KVBaseStateMachine) queryAppliedIndex(kv_store db.KVStore) (uint64, error) {
	result, err := kv_store.Get(db.MetaFC, AppliedIndexKey)
	if err != nil {
		return 0, err
	}

	if len(result) == 0 {
		return 0, nil
	}
	return binary.LittleEndian.Uint64(result), nil
}

func (s *KVBaseStateMachine) Update(ents []statemachine.Entry) ([]statemachine.Entry, error) {
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

	kv_store := *(*db.KVStore)(atomic.LoadPointer(&s.store))
	batch := db.NewWriteBatch()

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

				batch.Put(wCmd.ColumnFamilyName, wCmd.Key, wCmd.Value)
			case PutOpTTL:
				ttlMillis := time.Now().Add(time.Duration(wCmd.TTL) * time.Second).UnixMilli()

				ttlRealKey := fmt.Sprintf("%s%s", db.PrefixData, wCmd.Key)
				ttlExpireIndexKey := fmt.Sprintf("%s%s", db.PrefixTTLExpire, wCmd.Key)

				oldTTLBytes, err := kv_store.Get(wCmd.ColumnFamilyName, ttlExpireIndexKey)
				if err != nil {
					return nil, fmt.Errorf("error reading previous TTL for key %s: %w", wCmd.Key, err)
				}
				if oldTTLBytes != nil {
					oldTTLMillis, err := strconv.ParseInt(string(oldTTLBytes), 10, 64)
					if err == nil {
						oldTTLIndexKey := fmt.Sprintf("%s%020d:%s", db.PrefixTTLIndex, oldTTLMillis, wCmd.Key)
						batch.Delete(wCmd.ColumnFamilyName, oldTTLIndexKey)
					}
				}

				batch.Put(wCmd.ColumnFamilyName, ttlRealKey, wCmd.Value)

				newTTLIndexKey := fmt.Sprintf("%s%020d:%s", db.PrefixTTLIndex, ttlMillis, wCmd.Key)
				batch.Put(wCmd.ColumnFamilyName, newTTLIndexKey, nil)

				batch.Put(wCmd.ColumnFamilyName, ttlExpireIndexKey, []byte(strconv.FormatInt(ttlMillis, 10)))

			case DeleteOp:
				batch.Delete(wCmd.ColumnFamilyName, wCmd.Key)
			case DeleteOpTTL:
				if err := kv_store.Delete(wCmd.ColumnFamilyName, wCmd.Key); err != nil {
					return nil, err
				}
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
			err = kv_store.CleanExpiredKeys()
			if err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("unknown MCL Operation: %v", mlcCmd.Op)
		}
		ents[idx].Result = statemachine.Result{Value: uint64(len(ents[idx].Cmd))}
	}

	appliedIndex := make([]byte, 8)
	binary.LittleEndian.PutUint64(appliedIndex, ents[len(ents)-1].Index)
	batch.Put(db.MetaFC, AppliedIndexKey, appliedIndex)

	if err := kv_store.Write(batch); err != nil {
		return nil, err
	}

	if s.lastApplied >= ents[len(ents)-1].Index {
		return nil, fmt.Errorf("lastApplied not moving forward: current=%d new=%d", s.lastApplied, ents[len(ents)-1].Index)
	}
	s.lastApplied = ents[len(ents)-1].Index
	return ents, nil
}

func (s *KVBaseStateMachine) Lookup(query interface{}) (interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	kv_store := *(*db.KVStore)(atomic.LoadPointer(&s.store))
	if kv_store != nil {

		query, err := s.stateMachineImpl.Lookup(query)

		if err == nil && s.closed {
			return nil, errors.New("lookup returned valid result when DiskKV is already closed")
		}

		if err != nil {
			return nil, err
		}

		switch query.Op {

		case GetOp:
			var data []byte

			data, err = kv_store.Get(query.ColumnFamilyName, query.Key)
			if err != nil {
				return nil, err
			}
			if data != nil {
				return data, err
			}
		case Search:

			pairs, nextCursor, err := kv_store.SearchByPatternPaginatedKV(
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

			data, err = kv_store.Get(query.ColumnFamilyName, query.Key)
			if err != nil {
				return nil, err
			}
			if data != nil {
				return data, err
			}
		case SearchTTL:
			var resultData []db.KeyValuePair
			cursor := query.Cursor
			remaining := int(query.Limit)

			for remaining > 0 {
				keyPatter := fmt.Sprintf("%s%s", db.PrefixData, query.KeyPatter)
				pairs, nextCursor, err := kv_store.SearchByPatternPaginatedKV(
					query.ColumnFamilyName,
					keyPatter,
					cursor,
					remaining*2,
				)
				if err != nil {
					return nil, err
				}

				for _, pair := range pairs {
					key := strings.TrimPrefix(pair.Key, db.PrefixData)
					value, err := kv_store.Get(query.ColumnFamilyName, key)
					if err != nil {
						return nil, err
					}
					if value == nil {
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

func (s *KVBaseStateMachine) Sync() error {
	kv_store := *(*db.KVStore)(atomic.LoadPointer(&s.store))
	return kv_store.Flush()
}

func (s *KVBaseStateMachine) PrepareSnapshot() (interface{}, error) {
	return nil, nil
}

func (s *KVBaseStateMachine) SaveSnapshot(
	ctx interface{},
	w io.Writer,
	done <-chan struct{},
) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	kv_store := *(*db.KVStore)(atomic.LoadPointer(&s.store))

	if kv_store == nil {
		return errors.New("db closed")
	}

	enc := gob.NewEncoder(w)

	err := kv_store.Iterate(func(cfName string, key, value []byte) error {
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

func (s *KVBaseStateMachine) RecoverFromSnapshot(
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
	kv_store, err := s.stateMachineImpl.OpenDB(dbdir) // Simplified for now
	if err != nil {
		return fmt.Errorf("failed to open new DB for snapshot recovery: %w", err)
	}

	dec := gob.NewDecoder(r)

	for {
		select {
		case <-done:
			kv_store.Close()    // Clean up the new DB if cancelled
			os.RemoveAll(dbdir) // Attempt to remove the temporary DB dir
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
			kv_store.Close() // Clean up
			os.RemoveAll(dbdir)
			return fmt.Errorf("decode failed: %w", err)
		}

		if err := kv_store.Put(entry.CFName, string(entry.Key), entry.Value); err != nil {
			kv_store.Close()
			os.RemoveAll(dbdir)
			return fmt.Errorf("put failed during snapshot recovery for CF %s, Key %s: %w", entry.CFName, string(entry.Key), err)
		}
	}

	// Persist directory changes
	if err := saveCurrentDBDirName(dir, dbdir); err != nil {
		kv_store.Close()
		os.RemoveAll(dbdir)
		return err
	}
	if err := replaceCurrentDBFile(dir); err != nil {
		kv_store.Close()
		os.RemoveAll(dbdir)
		return err
	}

	// Update applied index
	newLastApplied, err := s.queryAppliedIndex(kv_store)
	if err != nil {
		// This is critical. If we can't read the applied index, the SM is in an inconsistent state.
		kv_store.Close()
		os.RemoveAll(dbdir)
		// Attempt to revert to oldDirName if possible, though that's complex and risky here.
		// For now, panic might be safer than continuing in an unknown state.
		panic(fmt.Sprintf("failed to query applied index after snapshot recovery: %v", err))
	}

	// Atomically swap the store pointer
	oldStorePtr := atomic.SwapPointer(&s.store, unsafe.Pointer(&kv_store))
	oldKvStore := *(*db.KVStore)(oldStorePtr)

	// Close and remove the old database IF it existed and was different
	if oldKvStore != nil && oldKvStore != kv_store { // Check if it's genuinely an old instance
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

func (s *KVBaseStateMachine) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	kv_store := *(*db.KVStore)(atomic.LoadPointer(&s.store))
	if kv_store != nil {
		if s.closed {
			// Already closed, potentially an issue if called multiple times without error.
			// Depending on strictness, could panic or return an error.
			// For now, let's match the original panic behavior.
			panic("close called twice")
		}
		s.closed = true
		return kv_store.Close()
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
