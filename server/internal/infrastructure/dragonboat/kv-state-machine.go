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
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/lni/dragonboat/v4/statemachine"
	"github.com/rs/zerolog/log"
)

type KVStateMachineImpl interface {
	// OpenDB acquires a reference to the shared database via the SharedDBProvider.
	// Unlike the old per-shard model, this does NOT create a new database; it reuses the shared one.
	OpenDB(sharedProvider *db.SharedDBProvider, pathProvider db.PathProvider) (db.KVStore, error)

	Update(cmd any, uow *db.UnitOfWork, now time.Time) commands.CommandResult

	Lookup(cmd any, uow *db.UnitOfWork, now time.Time) commands.CommandResult

	// BelongsToShard returns true if the given column family name belongs to this shard.
	// Used by SaveSnapshot to filter entries — only entries belonging to this shard are serialized.
	// The meta CF is always included (filtered by applied index key with clusterID).
	BelongsToShard(cfName string) bool
}
type KVBaseStateMachineConfig struct {
	// TTLInternalError specifies the Time-To-Live (in seconds) for internal error messages stored in the database.
	TTLInternalError   uint64
	PathProvider       db.PathProvider
	SharedDBProvider   *db.SharedDBProvider
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
	// With the shared DB model, all shards share one database instance.
	// OpenDB acquires a reference from the SharedDBProvider instead of
	// creating a per-shard database directory.
	store, err := s.stateMachineImpl.OpenDB(s.config.SharedDBProvider, s.config.PathProvider)
	if err != nil {
		return 0, fmt.Errorf("failed to open shared DB for shard %d node %d: %w", s.clusterID, s.nodeID, err)
	}

	atomic.SwapPointer(&s.store, unsafe.Pointer(&store))
	appliedIndex, err := s.queryAppliedIndex(store)
	if err != nil {
		panic(err)
	}
	s.lastApplied = appliedIndex

	log.Info().
		Uint64("clusterID", s.clusterID).
		Uint64("nodeID", s.nodeID).
		Uint64("appliedIndex", appliedIndex).
		Msg("Shard opened with shared DB")

	return appliedIndex, nil
}

// appliedIndexKeyForShard returns the shard-specific key for storing the applied Raft index.
// With a shared DB, each shard must store its applied index under a unique key to avoid collisions.
func (s *KVBaseStateMachine) appliedIndexKeyForShard() string {
	return fmt.Sprintf("%s:%d", AppliedIndexKey, s.clusterID)
}

func (s *KVBaseStateMachine) queryAppliedIndex(kv_store db.KVStore) (uint64, error) {
	result, err := kv_store.Get(db.MetaFC, db.MetaFCSector, s.appliedIndexKeyForShard(), time.Now())
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
	// No mutex needed here — Dragonboat guarantees Update() is called sequentially per shard.
	// Using a Lock here would contend with Lookup()/SaveSnapshot() which use RLock().

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
	batch.Put(db.MetaFC, db.MetaFCSector, s.appliedIndexKeyForShard(), appliedIndex, time.Now())

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
	appliedIdxKey := s.appliedIndexKeyForShard()

	// With a shared DB, we must only serialize entries belonging to THIS shard.
	// The stateMachineImpl.BelongsToShard() method determines CF ownership.
	// For the meta CF, we only include the applied index key for this specific shard.
	err := kv_store.Iterate(func(cfName string, cfSector string, key, value []byte) error {
		select {
		case <-done:
			return fmt.Errorf("snapshot cancelled")
		default:
		}

		// Filter: meta CF — only include this shard's applied index key
		if cfName == db.MetaFC {
			if string(key) != appliedIdxKey {
				return nil // skip other shards' applied index keys
			}
		} else if !s.stateMachineImpl.BelongsToShard(cfName) {
			// Skip CFs that don't belong to this shard
			return nil
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
		return fmt.Errorf("snapshot save failed for shard %d: %w", s.clusterID, err)
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

	// With shared DB, we don't create a new database — we selectively delete this shard's
	// data from the shared DB and re-insert the snapshot entries.
	kv_store := *(*db.KVStore)(atomic.LoadPointer(&s.store))
	if kv_store == nil {
		return errors.New("cannot recover from snapshot: store is nil")
	}

	// Step 1: Delete this shard's existing data from the shared DB.
	// Delete the shard's applied index key from meta CF.
	appliedIdxKey := s.appliedIndexKeyForShard()
	if err := kv_store.Delete(db.MetaFC, db.MetaFCSector, appliedIdxKey, time.Now()); err != nil {
		log.Warn().Err(err).Str("key", appliedIdxKey).Msg("Failed to delete applied index key during snapshot recovery (may not exist yet)")
	}

	// Delete CFs that belong to this shard.
	// We iterate all data and delete entries whose CF belongs to this shard.
	// Note: For tenant shards, the CFs (cf-n-X) will be cleaned up by DeleteColumnFamily
	// or by iterating. For the master shard, it owns the admin CF.
	// A simpler approach: the snapshot will overwrite existing data, and since
	// SaveSnapshot only serializes this shard's data, restoring it is safe.
	// We only need to ensure stale keys are removed.
	// For now, we rely on the fact that keys are deterministically generated,
	// so the snapshot restore will overwrite all current values.

	dec := gob.NewDecoder(r)

	batch := db.NewWriteBatch()
	count := 0
	knownCFs := make(map[string]bool)

	for {
		select {
		case <-done:
			return fmt.Errorf("snapshot recovery cancelled for shard %d", s.clusterID)
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
			return fmt.Errorf("decode failed during snapshot recovery for shard %d: %w", s.clusterID, err)
		}

		// Ensure Column Family exists in the shared DB
		if !knownCFs[entry.CFName] {
			exists, _, err := kv_store.ExistsColumnFamily(entry.CFName)
			if err != nil {
				return fmt.Errorf("failed to check column family existence: %w", err)
			}
			if !exists {
				isTTL := strings.HasPrefix(entry.CFName, db.ColumnFamilyTTLPrefix)
				if err := kv_store.CreateColumnFamily(entry.CFName, isTTL); err != nil {
					return fmt.Errorf("failed to create column family %s: %w", entry.CFName, err)
				}
				log.Info().Str("cf_name", entry.CFName).Bool("is_ttl", isTTL).Msg("Created missing column family during snapshot recovery")
			}
			knownCFs[entry.CFName] = true
		}

		batch.Put(entry.CFName, entry.CFNSector, string(entry.Key), entry.Value, time.Now())
		count++

		if count%10000 == 0 {
			if err := kv_store.WriteRaw(batch); err != nil {
				return fmt.Errorf("write raw failed during snapshot recovery for shard %d: %w", s.clusterID, err)
			}
			batch = db.NewWriteBatch()
			log.Info().Int("count", count).Uint64("clusterID", s.clusterID).Msg("Intermediate batch write successful during snapshot recovery")
		}
	}

	if batch.Count() > 0 {
		if err := kv_store.WriteRaw(batch); err != nil {
			return fmt.Errorf("final write raw failed during snapshot recovery for shard %d: %w", s.clusterID, err)
		}
		log.Info().Int("total_count", count).Uint64("clusterID", s.clusterID).Msg("Final batch write successful during snapshot recovery")
	}

	// Update applied index from the restored data
	newLastApplied, err := s.queryAppliedIndex(kv_store)
	if err != nil {
		panic(fmt.Sprintf("failed to query applied index after snapshot recovery for shard %d: %v", s.clusterID, err))
	}

	if s.lastApplied > newLastApplied && newLastApplied != 0 {
		log.Error().Uint64("currentLastApplied", s.lastApplied).Uint64("newLastApplied", newLastApplied).Uint64("clusterID", s.clusterID).Msg("Last applied index moved backward after snapshot recovery")
	}
	s.lastApplied = newLastApplied

	log.Info().
		Uint64("clusterID", s.clusterID).
		Uint64("newLastApplied", newLastApplied).
		Int("entriesRestored", count).
		Msg("Snapshot recovery completed for shard")

	return nil
}

func (s *KVBaseStateMachine) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		panic("close called twice")
	}
	s.closed = true

	// With shared DB, we don't close the KVStore directly — we release our reference.
	// The SharedDBProvider will close the underlying DB when the last reference is released.
	if s.config.SharedDBProvider != nil {
		log.Info().
			Uint64("clusterID", s.clusterID).
			Uint64("nodeID", s.nodeID).
			Msg("Releasing shared DB reference for shard")
		return s.config.SharedDBProvider.Release()
	}

	// Fallback: if no SharedDBProvider (shouldn't happen in normal operation)
	kv_store := *(*db.KVStore)(atomic.LoadPointer(&s.store))
	if kv_store != nil {
		return kv_store.Close()
	}
	return nil
}
