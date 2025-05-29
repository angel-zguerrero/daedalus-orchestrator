package dragonboat

import (
	"bytes"
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/pkg/config"
	"encoding/gob"
	"fmt"

	"github.com/linxGnu/grocksdb"
	"github.com/lni/dragonboat/v4/statemachine"
)

// MasterKVBaseRocksDBStateMachine is an implementation of the KVRocksDBStateMachineImpl interface,
// specifically tailored for the master shard/node in a Dragonboat cluster.
// It defines how the master shard opens its database and processes Raft entries.
type MasterKVBaseRocksDBStateMachine struct {
}

// OpenDB opens the RocksDB database for the master shard.
// It calls db.OpenMasterDB, which is expected to set up the predefined column families
// (like AdminFC and MasterEventFC) for the master database.
//
// Parameters:
//   - dbPath: The file system path where the master RocksDB database is located or will be created.
//
// Returns:
//   - A pointer to the opened grocksdb.DB instance.
//   - A map of normal column family names to their grocksdb.ColumnFamilyHandle.
//   - A map of TTL column family names to their grocksdb.ColumnFamilyHandle.
//   - An error if the database cannot be opened.
func (r *MasterKVBaseRocksDBStateMachine) OpenDB(dbPath string) (*grocksdb.DB, map[string]*grocksdb.ColumnFamilyHandle, map[string]*grocksdb.ColumnFamilyHandle, error) {
	return db.OpenMasterDB(dbPath)
}

// Lookup prepares an RK_Command (Read Key Command) from a generic query object.
// For the master state machine, it expects the query to be already of type RK_Command.
// This method primarily validates the type of the query. The actual database lookup
// is performed by the generic KVBaseRocksDBStateMachine.
//
// Parameters:
//   - query: The query object, expected to be of type RK_Command.
//
// Returns:
//   - The RK_Command if the type assertion is successful.
//   - An empty RK_Command and an error if the query is not of the expected type.
func (r *MasterKVBaseRocksDBStateMachine) Lookup(query interface{}) (RK_Command, error) {
	lookupQuery, ok := query.(RK_Command)
	if !ok {
		return RK_Command{}, fmt.Errorf("expected query to be RK_Command, got %T", query)
	}

	return lookupQuery, nil
}

// Update decodes Raft log entries into Command structs.
// Each entry's `Cmd` field is expected to be a GOB-encoded byte slice representing a generic Command.
// This method deserializes these byte slices back into Command objects.
// The actual application of these commands to the RocksDB write batch is handled by
// the generic KVBaseRocksDBStateMachine's Update method, which calls this implementation.
//
// Parameters:
//   - ents: A slice of statemachine.Entry objects from Dragonboat.
//   - batch: The grocksdb.WriteBatch to which operations would be added (not directly used in this specific Update,
//     as it only decodes; the base SM handles batch operations).
//
// Returns:
//   - A slice of decoded Command objects.
//   - An error if GOB decoding fails for any entry.
func (r *MasterKVBaseRocksDBStateMachine) Update(ents []statemachine.Entry, batch *grocksdb.WriteBatch) ([]Command, error) {
	commands := make([]Command, len(ents))

	for i, ent := range ents {
		var cmd Command
		if err := gob.NewDecoder(bytes.NewReader(ent.Cmd)).Decode(&cmd); err != nil {
			return nil, fmt.Errorf("failed to decode command for entry at index %d (Raft index %d): %w", i, ent.Index, err)
		}
		commands[i] = cmd

	}
	return commands, nil
}

// NewMasterKVRocksDBStateMachine creates a new IOnDiskStateMachine suitable for the master shard.
// It wraps a MasterKVBaseRocksDBStateMachine instance within the generic NewKVStateMachine,
// providing the specific implementation details for the master.
//
// Parameters:
//   - clusterID: The ID of the Raft cluster.
//   - nodeID: The ID of this node in the Raft cluster.
//
// Returns:
//   - An statemachine.IOnDiskStateMachine configured for master shard operations.
func NewMasterKVRocksDBStateMachine(clusterID uint64, nodeID uint64) statemachine.IOnDiskStateMachine {
	return NewKVStateMachine(clusterID, nodeID, &MasterKVBaseRocksDBStateMachine{}, KVBaseRocksDBStateMachineConfig{
		TTLInternalError: config.GlobalConfiguration.TTLInternalError,
	})
}
