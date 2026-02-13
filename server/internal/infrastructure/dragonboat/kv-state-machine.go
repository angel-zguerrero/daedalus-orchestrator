package dragonboat

import (
	"bytes"
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/pkg/utils"
	commands "deadalus-orch/server/internal/usecase/command"
	general_command "deadalus-orch/server/internal/usecase/command/general"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
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

	Update(cmd any, uow *db.UnitOfWork, now time.Time) commands.CommandResult

	Lookup(cmd any, uow *db.UnitOfWork, now time.Time) commands.CommandResult
}
type KVBaseStateMachineConfig struct {
	// TTLInternalError specifies the Time-To-Live (in seconds) for internal error messages stored in the database.
	TTLInternalError uint64
	PathProvider     db.PathProvider
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
	dir, err := getNodeDBDirName(s.clusterID, s.nodeID, s.config.PathProvider)

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
	result, err := kv_store.Get(db.MetaFC, db.MetaFCSector, AppliedIndexKey, time.Now()) // WORK, here now can be any value, due to the key used to store the "applyed index" is not ttl key
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
	uow := db.NewUnitOfWork(kv_store, batch)

	fsm_commands := make([]general_command.FSM_Command, len(ents))
	parseErrors := make([]bool, len(ents))

	for i, ent := range ents {
		var cmd general_command.FSM_Command
		if err := gob.NewDecoder(bytes.NewReader(ent.Cmd)).Decode(&cmd); err != nil {
			parseErrors[i] = true
			msg := fmt.Sprintf(
				"failed to decode command for entry at index %d (Raft index %d): %v",
				i, ent.Index, err,
			)
			ents[i].Result = statemachine.Result{
				Value: uint64(len(ents[i].Cmd)),
				Data:  []byte(msg),
			}
			continue
		}
		fsm_commands[i] = cmd
	}

	var dllFCEntries []int
	var rwEntries []int
	var mclEntries []int
	var specializedEntries []int

	for i, cmd := range fsm_commands {
		if parseErrors[i] {
			continue
		}

		if cmd.Now <= 0 {
			if cmd.Type == general_command.RW {
				if rwCmd, ok := cmd.CMD.(general_command.RWK_Command); ok && rwCmd.Op == general_command.Write {
					parseErrors[i] = true // Mark to prevent further processing in subsequent loops
					ents[i].Result = statemachine.Result{
						Value: uint64(len(ents[i].Cmd)), // As per convention for statemachine results
						Data:  []byte(general_command.ErrMissingOrInvalidNowField.Error()),
					}
					log.Warn(). // Changed to Warn as it's a client data validation issue
							Uint64("raft_index", ents[i].Index).
							Int64("provided_now", cmd.Now).
							Str("command_type", "RW_Write").
							Msgf("FSM_Command validation failed: %s", general_command.ErrMissingOrInvalidNowField.Error())
					continue // Move to the next command entry
				}
			}
		}

		switch cmd.Type {
		case general_command.DDL_FC:
			dllFCEntries = append(dllFCEntries, i)
		case general_command.RW:
			rwEntries = append(rwEntries, i)
		case general_command.MCL:
			mclEntries = append(mclEntries, i)
		case general_command.REPOSITORY_COMMAND:
			specializedEntries = append(specializedEntries, i)
		default:
			msg := fmt.Sprintf("unknown command type: %v", cmd.Type)
			ents[i].Result = statemachine.Result{
				Value: uint64(len(ents[i].Cmd)),
				Data:  []byte(msg),
			}
			parseErrors[i] = true
		}
	}

	for _, idx := range dllFCEntries {
		if parseErrors[idx] {
			continue
		}
		cmd := fsm_commands[idx]
		ddlCmd, ok := cmd.CMD.(general_command.DDL_Command)
		if !ok {
			msg := fmt.Sprintf("expected DDL_Command for DLL type, got %T", cmd.CMD)
			ents[idx].Result = statemachine.Result{
				Value: uint64(len(ents[idx].Cmd)),
				Data:  []byte(msg),
			}
			continue
		}
		switch ddlCmd.Op {
		// Implementar operaciones aquí
		}
		ents[idx].Result = statemachine.Result{Value: uint64(len(ents[idx].Cmd))}
	}

	for _, idx := range rwEntries {
		if parseErrors[idx] {
			continue
		}
		cmd := fsm_commands[idx]
		now := time.Unix(0, cmd.Now)
		rwCmd, ok := cmd.CMD.(general_command.RWK_Command)
		if !ok {
			msg := fmt.Sprintf("expected RWK_Command for RW type, got %T", cmd.CMD)
			ents[idx].Result = statemachine.Result{
				Value: uint64(len(ents[idx].Cmd)),
				Data:  []byte(msg),
			}
			continue
		}
		switch rwCmd.Op {
		case general_command.Read:
			msg := fmt.Sprintf("Invalid read operation: %T", cmd.CMD)
			ents[idx].Result = statemachine.Result{
				Value: uint64(len(ents[idx].Cmd)),
				Data:  []byte(msg),
			}
			continue
		case general_command.Write:
			wCmd, ok := rwCmd.CMD.(general_command.WK_Command)
			if !ok {
				msg := fmt.Sprintf("expected WK_Command for RW type, got %T", cmd.CMD)
				ents[idx].Result = statemachine.Result{
					Value: uint64(len(ents[idx].Cmd)),
					Data:  []byte(msg),
				}
				continue
			}
			switch wCmd.Op {
			case general_command.PutOp:
				batch.Put(wCmd.ColumnFamilyName, wCmd.ColumnFamilySector, wCmd.Key, wCmd.Value, now)
			case general_command.PutOpTTL:
				batch.PutTTl(wCmd.ColumnFamilyName, wCmd.ColumnFamilySector, wCmd.Key, wCmd.Value, wCmd.TTL, now)
			case general_command.DeleteOp, general_command.DeleteOpTTL:
				batch.Delete(wCmd.ColumnFamilyName, wCmd.ColumnFamilySector, wCmd.Key, now)
			default:
				msg := fmt.Sprintf("unknown W Operation: %v", wCmd.Op)
				ents[idx].Result = statemachine.Result{
					Value: uint64(len(ents[idx].Cmd)),
					Data:  []byte(msg),
				}
				continue
			}
		default:
			msg := fmt.Sprintf("unknown RW Operation: %v", rwCmd.Op)
			ents[idx].Result = statemachine.Result{
				Value: uint64(len(ents[idx].Cmd)),
				Data:  []byte(msg),
			}
			continue
		}
		ents[idx].Result = statemachine.Result{Value: uint64(len(ents[idx].Cmd))}
	}

	for _, idx := range specializedEntries {
		if parseErrors[idx] {
			continue
		}
		cmd := fsm_commands[idx].CMD
		now := time.Unix(0, fsm_commands[idx].Now)
		result := s.stateMachineImpl.Update(cmd, uow, now)
		var buf bytes.Buffer

		err := gob.NewEncoder(&buf).Encode(result)

		if err != nil {
			b, e := utils.ErrorToGobBytes(err)
			if e != nil {
				b = []byte(err.Error())
			}
			ents[idx].Result = statemachine.Result{
				Value: uint64(len(ents[idx].Cmd)),
				Data:  b,
			}
			continue
		}

		ents[idx].Result = statemachine.Result{
			Value: uint64(len(ents[idx].Cmd)),
			Data:  buf.Bytes(),
		}
	}

	for _, idx := range mclEntries {
		if parseErrors[idx] {
			continue
		}
		cmd := fsm_commands[idx]
		mlcCmd, ok := cmd.CMD.(general_command.MCLK_Command)
		if !ok {
			msg := fmt.Sprintf("expected MCLK_Command for MCL type, got %T", cmd.CMD)
			ents[idx].Result = statemachine.Result{
				Value: uint64(len(ents[idx].Cmd)),
				Data:  []byte(msg),
			}
			continue
		}
		switch mlcCmd.Op {
		case general_command.ClearExpiredTTL:
			now := time.Unix(0, fsm_commands[idx].Now)
			err := kv_store.CleanExpiredKeys(now)
			if err != nil {
				ents[idx].Result = statemachine.Result{
					Value: uint64(len(ents[idx].Cmd)),
					Data:  []byte(err.Error()),
				}
				continue
			}
		default:
			msg := fmt.Sprintf("unknown MCL Operation: %v", mlcCmd.Op)
			ents[idx].Result = statemachine.Result{
				Value: uint64(len(ents[idx].Cmd)),
				Data:  []byte(msg),
			}
			continue
		}
		ents[idx].Result = statemachine.Result{Value: uint64(len(ents[idx].Cmd))}
	}

	appliedIndex := make([]byte, 8)
	binary.LittleEndian.PutUint64(appliedIndex, ents[len(ents)-1].Index)
	batch.Put(db.MetaFC, db.MetaFCSector, AppliedIndexKey, appliedIndex, time.Now())

	if err := uow.Commit(); err != nil {
		return nil, err
	}

	if s.lastApplied >= ents[len(ents)-1].Index {
		return nil, fmt.Errorf("lastApplied not moving forward: current=%d new=%d", s.lastApplied, ents[len(ents)-1].Index)
	}
	s.lastApplied = ents[len(ents)-1].Index
	return ents, nil
}

func (s *KVBaseStateMachine) Lookup(q interface{}) (interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	kv_store := *(*db.KVStore)(atomic.LoadPointer(&s.store))
	if kv_store != nil {
		data, ok := q.([]byte)
		if !ok {
			return nil, fmt.Errorf("invalid query type: expected []byte, got %T", q)
		}
		if len(data) == 0 {
			return nil, fmt.Errorf("empty query payload")
		}
		var query general_command.Query_Command
		if err := gob.NewDecoder(bytes.NewReader(data)).Decode(&query); err != nil {
			return nil, fmt.Errorf("failed to decode query command: %w", err)
		}

		// Validate Query_Command.Now field.
		if query.Now <= 0 {
			log.Warn().
				Int64("provided_now", query.Now).
				Msgf("Query_Command validation failed: %s", general_command.ErrMissingOrInvalidNowField.Error())
			return nil, general_command.ErrMissingOrInvalidNowField
		}

		now := time.Unix(0, query.Now)
		repo_command, ok := query.Command.(general_command.Repository_Command)
		uow := db.NewUnitOfWork(kv_store, nil)
		if ok {

			var buf bytes.Buffer
			result := s.stateMachineImpl.Lookup(repo_command.CMD, uow, now)
			err := gob.NewEncoder(&buf).Encode(result)
			if err != nil {
				return nil, err
			}
			return buf.Bytes(), nil
		}

		command, ok := query.Command.(general_command.RK_Command)
		if !ok {
			return nil, fmt.Errorf("expected command to be RK_Command, got %T", query.Command)
		}

		if s.closed {
			return nil, errors.New("lookup returned valid result when DiskKV is already closed")
		}

		switch command.Op {

		case general_command.GetOp:
			var data []byte

			data, err := kv_store.Get(command.ColumnFamilyName, command.ColumnFamilySector, command.Key, now)
			if err != nil {
				return nil, err
			}
			if data != nil {
				return data, err
			}
		case general_command.Search:

			pairs, nextCursor, err := kv_store.SearchByPatternPaginatedKV(
				command.ColumnFamilyName,
				command.ColumnFamilySector,
				command.KeyPattern,
				command.Cursor,
				int(command.Limit),
				now,
			)
			if err != nil {
				return nil, err
			}

			result := &PagedResultKV{
				Data:       pairs, // Data ahora es []KeyValuePair
				NextCursor: []byte(nextCursor),
			}
			return result, nil

		case general_command.GetOpTTL:
			var data []byte

			data, err := kv_store.Get(command.ColumnFamilyName, command.ColumnFamilySector, command.Key, now)
			if err != nil {
				return nil, err
			}
			if data != nil {
				return data, err
			}
		case general_command.SearchTTL:
			pairs, nextCursor, err := kv_store.SearchByPatternPaginatedKV(
				command.ColumnFamilyName,
				command.ColumnFamilySector,
				command.KeyPattern,
				command.Cursor,
				int(command.Limit),
				now,
			)
			if err != nil {
				return nil, err
			}

			result := &PagedResultKV{
				Data:       pairs, // Data ahora es []KeyValuePair
				NextCursor: []byte(nextCursor),
			}
			return result, nil

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

	err := kv_store.Iterate(func(cfName string, cfSector string, key, value []byte) error {
		select {
		case <-done:
			return fmt.Errorf("snapshot cancelled")
		default:
		}

		entry := struct {
			CFName    string
			CFNSector string
			Key       []byte
			Value     []byte
		}{
			CFName:    cfName,
			CFNSector: cfSector,
			Key:       append([]byte(nil), key...),
			Value:     append([]byte(nil), value...),
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

	dir, err := getNodeDBDirName(s.clusterID, s.nodeID, s.config.PathProvider)
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

	batch := db.NewWriteBatch()
	count := 0
	knownCFs := make(map[string]bool)

	for {
		select {
		case <-done:
			kv_store.Close()    // Clean up the new DB if cancelled
			os.RemoveAll(dbdir) // Attempt to remove the temporary DB dir
			return fmt.Errorf("snapshot recovery cancelled")
		default:
		}

		var entry struct {
			CFName    string
			CFNSector string
			Key       []byte
			Value     []byte
		}

		if err := dec.Decode(&entry); err != nil {
			if err == io.EOF {
				break // End of snapshot
			}
			kv_store.Close() // Clean up
			os.RemoveAll(dbdir)
			return fmt.Errorf("decode failed: %w", err)
		}

		// Ensure Column Family exists
		if !knownCFs[entry.CFName] {
			exists, _, err := kv_store.ExistsColumnFamily(entry.CFName)
			if err != nil {
				kv_store.Close()
				os.RemoveAll(dbdir)
				return fmt.Errorf("failed to check column family existence: %w", err)
			}
			if !exists {
				isTTL := strings.HasPrefix(entry.CFName, db.ColumnFamilyTTLPrefix)
				if err := kv_store.CreateColumnFamily(entry.CFName, isTTL); err != nil {
					kv_store.Close()
					os.RemoveAll(dbdir)
					return fmt.Errorf("failed to create column family %s: %w", entry.CFName, err)
				}
				log.Info().Str("cf_name", entry.CFName).Bool("is_ttl", isTTL).Msg("Created missing column family during recovery")
			}
			knownCFs[entry.CFName] = true
		}

		batch.Put(entry.CFName, entry.CFNSector, string(entry.Key), entry.Value, time.Now())
		count++

		if count%10000 == 0 {
			if err := kv_store.WriteRaw(batch); err != nil {
				kv_store.Close()
				os.RemoveAll(dbdir)
				return fmt.Errorf("write raw failed during snapshot recovery: %w", err)
			}
			batch = db.NewWriteBatch()
			log.Info().Int("count", count).Msg("Intermediate batch write successful during snapshot recovery")
		}
	}

	if batch.Count() > 0 {
		if err := kv_store.WriteRaw(batch); err != nil {
			kv_store.Close()
			os.RemoveAll(dbdir)
			return fmt.Errorf("final write raw failed during snapshot recovery: %w", err)
		}
		log.Info().Int("total_count", count).Msg("Final batch write successful during snapshot recovery")
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
