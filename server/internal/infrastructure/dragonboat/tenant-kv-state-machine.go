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

type TenantKVBaseRocksDBStateMachine struct {
}

func (r *TenantKVBaseRocksDBStateMachine) OpenDB(dbPath string) (*grocksdb.DB, map[string]*grocksdb.ColumnFamilyHandle, map[string]*grocksdb.ColumnFamilyHandle, error) {
	return db.OpenTenantDB(dbPath)
}

func (r *TenantKVBaseRocksDBStateMachine) Lookup(query interface{}) (RK_Command, error) {
	lookupQuery, ok := query.(RK_Command)
	if !ok {
		return RK_Command{}, fmt.Errorf("expected key to be string, got %T", query)
	}

	return lookupQuery, nil
}

func (r *TenantKVBaseRocksDBStateMachine) Update(ents []statemachine.Entry, batch *grocksdb.WriteBatch) ([]Command, error) {
	commands := make([]Command, len(ents))

	for i, ent := range ents {
		var cmd Command
		if err := gob.NewDecoder(bytes.NewReader(ent.Cmd)).Decode(&cmd); err != nil {
			return nil, err
		}
		commands[i] = cmd

	}
	return commands, nil
}

func NewTenantKVRocksDBStateMachine(clusterID uint64, nodeID uint64) statemachine.IOnDiskStateMachine {
	return NewKVStateMachine(clusterID, nodeID, &TenantKVBaseRocksDBStateMachine{}, KVBaseRocksDBStateMachineConfig{
		TTLInternalError: config.GlobalConfiguration.TTLInternalError,
	})
}
