package dragonboat

import (
	"bytes"
	"deadalus-orch/server/internal/infrastructure/db"
	"encoding/gob"

	"github.com/linxGnu/grocksdb"
	"github.com/lni/dragonboat/v4/statemachine"
)

type TenantKVBaseRocksDBStateMachine struct {
}

func (r *TenantKVBaseRocksDBStateMachine) OpenDB(dbPath string) (*grocksdb.DB, map[string]*grocksdb.ColumnFamilyHandle, error) {
	return db.OpenTenantDB(dbPath)
}

func (r *TenantKVBaseRocksDBStateMachine) Lookup(rocks_kv_store *db.RocksdbStore, lookupQuery LookupQuery) (interface{}, error) {
	return rocks_kv_store.Get(lookupQuery.ColumnFamilyName, lookupQuery.Key)
}

func (r *TenantKVBaseRocksDBStateMachine) Update(rocks_kv_store *db.RocksdbStore, ents []statemachine.Entry, batch *grocksdb.WriteBatch) ([]Command, error) {
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
	return NewKVStateMachine(clusterID, nodeID, &TenantKVBaseRocksDBStateMachine{})
}
