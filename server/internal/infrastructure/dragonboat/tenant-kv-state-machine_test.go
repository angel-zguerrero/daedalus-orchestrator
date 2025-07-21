package dragonboat_test

import (
	"bytes"
	"context"
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	"deadalus-orch/server/internal/pkg/config" // Added
	"deadalus-orch/server/internal/pkg/utils"
	general_command "deadalus-orch/server/internal/usecase/command/general"
	"encoding/binary"
	"encoding/gob"
	"io"
	"testing"
	"time"

	"github.com/lni/dragonboat/v4/statemachine"
	"github.com/stretchr/testify/require"
)

func setupTenantKV(t *testing.T) *dragonboat.KVBaseStateMachine {
	t.Helper()
	// Ensure config is loaded as NewTenantKVRocksDBStateMachine might depend on it for paths
	err := config.LoadDefaultConfiguration()
	require.NoError(t, err, "Failed to load default configuration for test setup")

	kv := dragonboat.NewTenantKVStateMachine(dragonboat.TestPathProvider{Path: t.TempDir()})(1, 2).(*dragonboat.KVBaseStateMachine) // Changed dragonboat.NewTenantKVStateMachine to NewTenantKVStateMachine and dragonboat.KVBaseStateMachine to KVBaseStateMachine
	stopc := make(chan struct{})
	_, err = kv.Open(stopc)
	require.NoError(t, err)
	return kv
}

func TestTenantOne(t *testing.T) {

	//dragonboat.Init(101, 1, "3435")
	//dragonboat.Init(101, 2, "3436")

	//time.Sleep(10 * time.Second)
}
func TestTenantOpen_Close(t *testing.T) {
	kv := setupTenantKV(t)

	err := kv.Close()
	require.NoError(t, err)

	require.Panics(t, func() {
		_ = kv.Close()
	})
}

func TestTenantUpdate_SingleEntry(t *testing.T) {
	kv := setupTenantKV(t)
	defer kv.Close()

	var buf bytes.Buffer
	cmd := general_command.FSM_Command{
		Now:  utils.GetNowInInt(),
		Type: general_command.RW,
		CMD: general_command.RWK_Command{
			Op: general_command.Write,
			CMD: general_command.WK_Command{
				Key:                "foo",
				Value:              []byte("bar"),
				ColumnFamilyName:   db.DefaultFC,
				ColumnFamilySector: db.DefaultFCSector,
				Op:                 general_command.PutOp,
			},
		},
	}

	err := gob.NewEncoder(&buf).Encode(cmd)
	require.NoError(t, err)

	result, err := kv.Update([]statemachine.Entry{
		{Cmd: buf.Bytes(), Index: kv.GetLastApplied() + 1},
	})
	require.NoError(t, err)
	require.Equal(t, uint64(len(buf.Bytes())), result[0].Result.Value) // Adjusted index for result
}

func TestTenantUpdate_AfterClose_Panics(t *testing.T) {
	kv := setupTenantKV(t)
	_ = kv.Close()

	require.Panics(t, func() {
		_, _ = kv.Update(nil)
	})
}

func TestTenantLookup_ExistingKey(t *testing.T) {
	kv := setupTenantKV(t)
	defer kv.Close()

	var buf bytes.Buffer

	cmd := general_command.FSM_Command{
		Now:  utils.GetNowInInt(),
		Type: general_command.RW,
		CMD: general_command.RWK_Command{
			Op: general_command.Write,
			CMD: general_command.WK_Command{
				Key:                "lookup_key",
				Value:              []byte("lookup_value"),
				ColumnFamilyName:   db.DefaultFC,
				ColumnFamilySector: db.DefaultFCSector,
				Op:                 general_command.PutOp,
			},
		},
	}

	require.NoError(t, gob.NewEncoder(&buf).Encode(cmd))

	_, err := kv.Update([]statemachine.Entry{
		{Cmd: buf.Bytes(), Index: kv.GetLastApplied() + 1},
	})
	require.NoError(t, err)

	query := general_command.Query_Command{
		Now: utils.GetNowInInt(),
		Command: general_command.RK_Command{
			Key:                "lookup_key",
			ColumnFamilyName:   db.DefaultFC,
			ColumnFamilySector: db.DefaultFCSector,
		},
	}
	var bufQ bytes.Buffer
	gob.NewEncoder(&bufQ).Encode(query)
	val, err := kv.Lookup(bufQ.Bytes())
	require.NoError(t, err)
	require.Equal(t, []byte("lookup_value"), val)
}

func TestTenantLookup_NonExistingKey(t *testing.T) {
	kv := setupTenantKV(t)
	defer kv.Close()

	query := general_command.Query_Command{
		Now: utils.GetNowInInt(),
		Command: general_command.RK_Command{
			Key:                "missing_key",
			ColumnFamilyName:   db.DefaultFC,
			ColumnFamilySector: db.DefaultFCSector,
		},
	}
	var bufQ bytes.Buffer
	gob.NewEncoder(&bufQ).Encode(query)
	val, err := kv.Lookup(bufQ.Bytes())

	require.NoError(t, err)
	require.Nil(t, val)
}

func TestTenantSync(t *testing.T) {
	kv := setupTenantKV(t)
	defer kv.Close()

	err := kv.Sync()
	require.NoError(t, err)
}

func TestTenantSaveSnapshotAndRecover(t *testing.T) {
	kv := setupTenantKV(t)

	var buf bytes.Buffer
	cmd := general_command.FSM_Command{
		Now:  utils.GetNowInInt(),
		Type: general_command.RW,
		CMD: general_command.RWK_Command{
			Op: general_command.Write,
			CMD: general_command.WK_Command{
				Key:                "snap_key",
				Value:              []byte("snap_value"),
				ColumnFamilyName:   db.DefaultFC,
				ColumnFamilySector: db.DefaultFCSector,
				Op:                 general_command.PutOp,
			},
		},
	}

	require.NoError(t, gob.NewEncoder(&buf).Encode(cmd))

	_, err := kv.Update([]statemachine.Entry{
		{Cmd: buf.Bytes(), Index: kv.GetLastApplied() + 1},
	})
	require.NoError(t, err)

	var snap bytes.Buffer
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err = kv.SaveSnapshot(nil, &snap, ctx.Done())
	require.NoError(t, err)

	_ = kv.Close()

	// Also ensure config is loaded for the second instance if it's path dependent
	err = config.LoadDefaultConfiguration()
	require.NoError(t, err, "Failed to load default configuration for test setup (kv2)")
	kv2 := dragonboat.NewTenantKVStateMachine(dragonboat.TestPathProvider{Path: t.TempDir()})(1, 2).(*dragonboat.KVBaseStateMachine) // Changed dragonboat.NewTenantKVStateMachine to NewTenantKVStateMachine and dragonboat.KVBaseStateMachine to KVBaseStateMachine
	stopc := make(chan struct{})
	_, err = kv2.Open(stopc)
	require.NoError(t, err)
	defer kv2.Close()

	err = kv2.RecoverFromSnapshot(&snap, ctx.Done())
	require.NoError(t, err)

	query1 := general_command.Query_Command{
		Now: utils.GetNowInInt(),
		Command: general_command.RK_Command{
			Key:                "snap_key",
			ColumnFamilyName:   db.DefaultFC,
			ColumnFamilySector: db.DefaultFCSector,
		},
	}
	var bufQ bytes.Buffer
	gob.NewEncoder(&bufQ).Encode(query1)
	val, err := kv2.Lookup(bufQ.Bytes())

	require.NoError(t, err)
	require.Equal(t, []byte("snap_value"), val)

	query2 := general_command.Query_Command{
		Now: utils.GetNowInInt(),
		Command: general_command.RK_Command{
			Key:                dragonboat.AppliedIndexKey, // This refers to a const in the non-moved dragonboat package
			ColumnFamilyName:   db.MetaFC,
			ColumnFamilySector: db.MetaFCSector,
		},
	}
	var bufQ2 bytes.Buffer
	gob.NewEncoder(&bufQ2).Encode(query2)
	val, err = kv2.Lookup(bufQ2.Bytes())

	require.NoError(t, err)
	require.Equal(t, kv2.GetLastApplied(), binary.LittleEndian.Uint64(val.([]byte)))
}

func TestTenantSaveSnapshot_Cancelled(t *testing.T) {
	kv := setupTenantKV(t)
	defer kv.Close()

	done := make(chan struct{})
	close(done)

	var buf bytes.Buffer
	cmd := general_command.FSM_Command{
		Now:  utils.GetNowInInt(),
		Type: general_command.RW,
		CMD: general_command.RWK_Command{
			Op: general_command.Write,
			CMD: general_command.WK_Command{
				Key:                "snap_key",
				Value:              []byte("snap_value"),
				ColumnFamilyName:   db.DefaultFC,
				ColumnFamilySector: db.DefaultFCSector,
				Op:                 general_command.PutOp,
			},
		},
	}

	require.NoError(t, gob.NewEncoder(&buf).Encode(cmd))

	_, err := kv.Update([]statemachine.Entry{
		{Cmd: buf.Bytes(), Index: kv.GetLastApplied() + 1},
	})
	require.NoError(t, err)

	err = kv.SaveSnapshot(nil, io.Discard, done)
	require.Error(t, err)
	require.Contains(t, err.Error(), "snapshot cancelled")
}

func TestTenantRecoverSnapshot_Cancelled(t *testing.T) {
	kv := setupTenantKV(t)
	defer kv.Close()

	r, w := io.Pipe()
	done := make(chan struct{})
	close(done)

	go func() {
		_ = w.Close()
	}()

	err := kv.RecoverFromSnapshot(r, done)
	require.Error(t, err)
	require.Contains(t, err.Error(), "snapshot recovery cancelled")
}

func TestTenantUpdate_AddColumnFamily(t *testing.T) {
	kv := setupTenantKV(t)
	defer kv.Close()

	var buf bytes.Buffer
	cmd := general_command.FSM_Command{
		Type: general_command.DDL_FC,
		CMD: general_command.DDL_Command{
			ColumnFamilyName: "new_cf",
			Op:               general_command.Add_CF_Op,
		},
	}
	err := gob.NewEncoder(&buf).Encode(cmd)
	require.NoError(t, err)

	result, err := kv.Update([]statemachine.Entry{
		{Cmd: buf.Bytes(), Index: kv.GetLastApplied() + 1},
	})
	require.NoError(t, err)
	require.Equal(t, uint64(len(buf.Bytes())), result[0].Result.Value) // Adjusted index
}
func TestTenantUpdate_DropColumnFamily(t *testing.T) {
	kv := setupTenantKV(t)
	defer kv.Close()
	{
		var buf bytes.Buffer
		cmd := general_command.FSM_Command{
			Type: general_command.DDL_FC,
			CMD: general_command.DDL_Command{

				ColumnFamilyName: "to_delete_cf",
				Op:               general_command.Add_CF_Op,
			},
		}
		err := gob.NewEncoder(&buf).Encode(cmd)
		require.NoError(t, err)

		_, err = kv.Update([]statemachine.Entry{
			{Cmd: buf.Bytes(), Index: kv.GetLastApplied() + 1},
		})
		require.NoError(t, err)
	}

	var buf bytes.Buffer
	cmd := general_command.FSM_Command{
		Type: general_command.DDL_FC,
		CMD: general_command.DDL_Command{

			ColumnFamilyName: "to_delete_cf",
			Op:               general_command.Remove_CF_Op,
		},
	}
	err := gob.NewEncoder(&buf).Encode(cmd)
	require.NoError(t, err)

	result, err := kv.Update([]statemachine.Entry{
		{Cmd: buf.Bytes(), Index: kv.GetLastApplied() + 1},
	})
	require.NoError(t, err)
	require.Equal(t, uint64(len(buf.Bytes())), result[0].Result.Value) // Adjusted index
}

func TestTenantRead_SingleEntryIntoUpdate(t *testing.T) {
	kv := setupTenantKV(t)
	defer kv.Close()

	var buf bytes.Buffer
	cmd := general_command.FSM_Command{
		Now:  utils.GetNowInInt(),
		Type: general_command.RW,
		CMD: general_command.RWK_Command{
			Op: general_command.Read,
			CMD: general_command.RK_Command{
				Key:                "foo",
				ColumnFamilyName:   db.DefaultFC,
				ColumnFamilySector: db.DefaultFCSector,
				Op:                 general_command.GetOp,
			},
		},
	}

	err := gob.NewEncoder(&buf).Encode(cmd)
	require.NoError(t, err)

	result, err := kv.Update([]statemachine.Entry{
		{Cmd: buf.Bytes(), Index: kv.GetLastApplied() + 1},
	})
	require.NoError(t, err) // Update call itself should not error out
	require.NotNil(t, result)
	require.Len(t, result, 1)
	// Expect an error message in Result.Data due to invalid operation type
	require.Contains(t, string(result[0].Result.Data), "Invalid read operation: general_command.RWK_Command") // Changed dragonboat.RWK_Command to general_command.RWK_Command
}
func TestTenantUpdate_PutWithTTL(t *testing.T) {
	kv := setupTenantKV(t)
	defer kv.Close()

	var buf bytes.Buffer
	cmd := general_command.FSM_Command{
		Now:  utils.GetNowInInt(),
		Type: general_command.RW,
		CMD: general_command.RWK_Command{
			Op: general_command.Write,
			CMD: general_command.WK_Command{
				Key:                "ttl_key",
				Value:              []byte("ttl_value"),
				ColumnFamilyName:   db.TenantEventFC,
				ColumnFamilySector: db.TenantEventFCSector,
				TTL:                5,
				Op:                 general_command.PutOpTTL,
			},
		},
	}
	require.NoError(t, gob.NewEncoder(&buf).Encode(cmd))

	_, err := kv.Update([]statemachine.Entry{
		{Cmd: buf.Bytes(), Index: kv.GetLastApplied() + 1},
	})
	require.NoError(t, err)
}
func TestTenantUpdate_DropTTLColumnFamily(t *testing.T) {
	kv := setupTenantKV(t)
	defer kv.Close()
	{
		var buf bytes.Buffer
		cmd := general_command.FSM_Command{
			Type: general_command.DDL_FC,
			CMD: general_command.DDL_Command{

				ColumnFamilyName: "to_delete_cf",
				Op:               general_command.Add_TTL_CF_Op,
			},
		}
		err := gob.NewEncoder(&buf).Encode(cmd)
		require.NoError(t, err)

		_, err = kv.Update([]statemachine.Entry{
			{Cmd: buf.Bytes(), Index: kv.GetLastApplied() + 1},
		})
		require.NoError(t, err)
	}

	var buf bytes.Buffer
	cmd := general_command.FSM_Command{
		Type: general_command.DDL_FC,
		CMD: general_command.DDL_Command{
			ColumnFamilyName: "to_delete_cf",
			Op:               general_command.Remove_TTL_CF_Op,
		},
	}
	err := gob.NewEncoder(&buf).Encode(cmd)
	require.NoError(t, err)

	result, err := kv.Update([]statemachine.Entry{
		{Cmd: buf.Bytes(), Index: kv.GetLastApplied() + 1},
	})
	require.NoError(t, err)
	require.Equal(t, uint64(len(buf.Bytes())), result[0].Result.Value) // Adjusted index
}

func TestTenantUpdate_DeleteWithTTL(t *testing.T) {
	kv := setupTenantKV(t)
	defer kv.Close()

	var buf bytes.Buffer
	cmd := general_command.FSM_Command{
		Now:  utils.GetNowInInt(),
		Type: general_command.RW,
		CMD: general_command.RWK_Command{
			Op: general_command.Write,
			CMD: general_command.WK_Command{
				Key:                "ttl_key",
				Value:              []byte("ttl_value"),
				ColumnFamilyName:   db.TenantEventFC,
				ColumnFamilySector: db.TenantEventFCSector,
				TTL:                5,
				Op:                 general_command.PutOpTTL,
			},
		},
	}
	require.NoError(t, gob.NewEncoder(&buf).Encode(cmd))

	_, err := kv.Update([]statemachine.Entry{
		{Cmd: buf.Bytes(), Index: kv.GetLastApplied() + 1},
	})
	require.NoError(t, err)

	var bufDel bytes.Buffer
	cmd = general_command.FSM_Command{
		Now:  utils.GetNowInInt(),
		Type: general_command.RW,
		CMD: general_command.RWK_Command{
			Op: general_command.Write,
			CMD: general_command.WK_Command{
				Key:                "ttl_key",
				ColumnFamilyName:   db.TenantEventFC,
				ColumnFamilySector: db.TenantEventFCSector,
				TTL:                5,
				Op:                 general_command.DeleteOpTTL,
			},
		},
	}
	require.NoError(t, gob.NewEncoder(&bufDel).Encode(cmd))

	_, err = kv.Update([]statemachine.Entry{
		{Cmd: bufDel.Bytes(), Index: kv.GetLastApplied() + 1},
	})
	require.NoError(t, err)
}
