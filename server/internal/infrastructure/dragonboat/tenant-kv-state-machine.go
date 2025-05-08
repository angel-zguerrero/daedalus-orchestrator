package dragonboat

import (
	"bytes"
	"deadalus-orch/server/internal/infrastructure/db"
	"encoding/gob"
	"fmt"

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

func (r *TenantKVBaseRocksDBStateMachine) Update(rocks_kv_store *db.RocksdbStore, ents []statemachine.Entry, batch *grocksdb.WriteBatch) ([]statemachine.Entry, error) {

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
	return ents, nil
}

func NewTenantKVRocksDBStateMachine(clusterID uint64, nodeID uint64) statemachine.IOnDiskStateMachine {
	return NewKVStateMachine(clusterID, nodeID, &TenantKVBaseRocksDBStateMachine{})
}
