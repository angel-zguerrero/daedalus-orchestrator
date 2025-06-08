package dragonboat

import (
	"bytes"
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/pkg/config"
	"encoding/gob"
	"fmt"

	"github.com/lni/dragonboat/v4/statemachine"
)

type MasterKVDBStateMachine struct {
}

func (r *MasterKVDBStateMachine) OpenDB(dbPath string) (db.KVStore, error) {
	return db.OpenMasterDB(dbPath)
}

func (r *MasterKVDBStateMachine) Lookup(query interface{}) (RK_Command, error) {
	lookupQuery, ok := query.(RK_Command)
	if !ok {
		return RK_Command{}, fmt.Errorf("expected query to be RK_Command, got %T", query)
	}

	return lookupQuery, nil
}

func (r *MasterKVDBStateMachine) Update(ents []statemachine.Entry, batch *db.WriteBatch) ([]Command, error) {
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

func NewMasterKVStateMachine(clusterID uint64, nodeID uint64) statemachine.IOnDiskStateMachine {
	return NewKVStateMachine(clusterID, nodeID, &MasterKVDBStateMachine{}, KVBaseStateMachineConfig{
		TTLInternalError: config.GlobalConfiguration.TTLInternalError,
	})
}
