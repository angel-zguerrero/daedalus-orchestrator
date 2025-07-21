package dragonboat_test

import (
	"bytes"
	"context"
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	"deadalus-orch/server/internal/pkg/config"
	"deadalus-orch/server/internal/pkg/utils"
	general_command "deadalus-orch/server/internal/usecase/command/general"
	"deadalus-orch/shared/constants"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/lni/dragonboat/v4/statemachine"
	"github.com/stretchr/testify/require"
)

func setupKVMasterPebble(t *testing.T, engine string) *dragonboat.KVBaseStateMachine {
	t.Helper()
	t.Setenv(constants.EnvVarMasterDBEngine, engine)
	config.LoadDefaultConfiguration()
	kv := dragonboat.NewMasterKVStateMachine(dragonboat.TestPathProvider{Path: t.TempDir()})(1, 1).(*dragonboat.KVBaseStateMachine)
	stopc := make(chan struct{})
	_, err := kv.Open(stopc)
	require.NoError(t, err)
	return kv
}
func setupKV(t *testing.T, engine string) *dragonboat.KVBaseStateMachine { // Changed return type
	t.Helper()
	t.Setenv(constants.EnvVarMasterDBEngine, engine)
	config.LoadDefaultConfiguration()
	kv := dragonboat.NewMasterKVStateMachine(dragonboat.TestPathProvider{Path: t.TempDir()})(1, 1).(*dragonboat.KVBaseStateMachine) // Changed dragonboat.NewMasterKVStateMachine to NewMasterKVStateMachine and dragonboat.KVBaseStateMachine to KVBaseStateMachine
	stopc := make(chan struct{})
	_, err := kv.Open(stopc)
	require.NoError(t, err)
	return kv
}
func TestPebble_Open_Close(t *testing.T) {
	kv := setupKV(t, "pebble")

	err := kv.Close()
	require.NoError(t, err)

	require.Panics(t, func() {
		_ = kv.Close()
	})
}

func TestPebble_Update_SingleEntry(t *testing.T) {
	kv := setupKV(t, "pebble")
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
	require.Equal(t, uint64(len(buf.Bytes())), result[0].Result.Value)
}

func TestPebble_Update_AfterClose_Panics(t *testing.T) {
	kv := setupKV(t, "pebble")
	_ = kv.Close()

	require.Panics(t, func() {
		_, _ = kv.Update(nil)
	})
}

func TestPebble_Lookup_ExistingKey(t *testing.T) {
	kv := setupKV(t, "pebble")
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

	var buf3 bytes.Buffer
	gob.NewEncoder(&buf3).Encode(query)
	val, err := kv.Lookup(buf3.Bytes())
	require.NoError(t, err)
	require.Equal(t, []byte("lookup_value"), val)
}

func TestPebble_Lookup_NonExistingKey(t *testing.T) {
	kv := setupKV(t, "pebble")
	defer kv.Close()

	query := general_command.Query_Command{
		Now: utils.GetNowInInt(),
		Command: general_command.RK_Command{
			Key:                "missing_key",
			ColumnFamilyName:   db.DefaultFC,
			ColumnFamilySector: db.DefaultFCSector,
		},
	}
	var buf bytes.Buffer
	gob.NewEncoder(&buf).Encode(query)

	val, err := kv.Lookup(buf.Bytes())
	require.NoError(t, err)
	require.Nil(t, val)
}

func TestPebble_Sync(t *testing.T) {
	kv := setupKV(t, "pebble")
	defer kv.Close()

	err := kv.Sync()
	require.NoError(t, err)
}

func TestPebble_SaveSnapshotAndRecover(t *testing.T) {
	kv := setupKV(t, "pebble")

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
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()

	err = kv.SaveSnapshot(nil, &snap, ctx.Done())
	require.NoError(t, err)

	_ = kv.Close()

	kv2 := dragonboat.NewMasterKVStateMachine(dragonboat.TestPathProvider{Path: t.TempDir()})(1, 1).(*dragonboat.KVBaseStateMachine) // Changed dragonboat.NewMasterKVStateMachine to NewMasterKVStateMachine and dragonboat.KVBaseStateMachine to KVBaseStateMachine
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

	var buf1 bytes.Buffer
	gob.NewEncoder(&buf1).Encode(query1)

	val, err := kv2.Lookup(buf1.Bytes())
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

	var buf2 bytes.Buffer
	gob.NewEncoder(&buf2).Encode(query2)
	val, err = kv2.Lookup(buf2.Bytes())
	require.NoError(t, err)
	require.Equal(t, kv2.GetLastApplied(), binary.LittleEndian.Uint64(val.([]byte)))
}

func TestPebble_SaveSnapshot_Cancelled(t *testing.T) {
	kv := setupKV(t, "pebble")
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

func TestPebble_RecoverSnapshot_Cancelled(t *testing.T) {
	kv := setupKV(t, "pebble")
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

func TestPebble_Update_AddColumnFamily(t *testing.T) {
	kv := setupKV(t, "pebble")
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
	require.Equal(t, uint64(len(buf.Bytes())), result[0].Result.Value)
}
func TestPebble_Update_DropColumnFamily(t *testing.T) {
	kv := setupKV(t, "pebble")
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
	require.Equal(t, uint64(len(buf.Bytes())), result[0].Result.Value)
}

func TestPebble_Read_SingleEntryIntoUpdate(t *testing.T) {
	kv := setupKV(t, "pebble")
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
func TestPebble_Update_PutWithTTL(t *testing.T) {
	kv := setupKV(t, "pebble")
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
				ColumnFamilyName:   db.MasterEventFC,
				ColumnFamilySector: db.MasterEventFCSector,
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
func TestPebble_Update_DropTTLColumnFamily(t *testing.T) {
	kv := setupKV(t, "pebble")
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
	require.Equal(t, uint64(len(buf.Bytes())), result[0].Result.Value)
}

func TestPebble_Update_DeleteWithTTL(t *testing.T) {
	kv := setupKV(t, "pebble")
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
				ColumnFamilyName:   db.MasterEventFC,
				ColumnFamilySector: db.MasterEventFCSector,
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
				ColumnFamilyName:   db.MasterEventFC,
				ColumnFamilySector: db.MasterEventFCSector,
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
func TestPebble_PutTTLStoresWithExpiration(t *testing.T) {
	kv := setupKV(t, "pebble")
	defer kv.Close()

	var buf bytes.Buffer
	cmd := general_command.FSM_Command{
		Now:  utils.GetNowInInt(),
		Type: general_command.RW,
		CMD: general_command.RWK_Command{
			Op: general_command.Write,
			CMD: general_command.WK_Command{
				Key:                "ttl_test_key",
				Value:              []byte("ttl_test_value"),
				ColumnFamilyName:   db.MasterEventFC,
				ColumnFamilySector: db.MasterEventFCSector,
				TTL:                10,
				Op:                 general_command.PutOpTTL,
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
			Key:                "ttl_test_key",
			ColumnFamilyName:   db.MasterEventFC,
			ColumnFamilySector: db.MasterEventFCSector,
			Op:                 general_command.GetOpTTL,
		},
	}

	// Use gob to encode the query comman
	var bufQuery bytes.Buffer
	gob.NewEncoder(&bufQuery).Encode(query)
	val, err := kv.Lookup(bufQuery.Bytes())
	require.NoError(t, err)
	require.Equal(t, []byte("ttl_test_value"), val)
}

func TestPebble_TTLExpirationRemovesKey(t *testing.T) {
	kv := setupKV(t, "pebble")
	defer kv.Close()

	key := "expiring_key"

	// Put TTL entry
	var buf bytes.Buffer
	cmd := general_command.FSM_Command{
		Now:  utils.GetNowInInt(),
		Type: general_command.RW,
		CMD: general_command.RWK_Command{
			Op: general_command.Write,
			CMD: general_command.WK_Command{
				Key:                key,
				Value:              []byte("soon_gone"),
				ColumnFamilyName:   db.MasterEventFC,
				ColumnFamilySector: db.MasterEventFCSector,
				TTL:                1,
				Op:                 general_command.PutOpTTL,
			},
		},
	}
	require.NoError(t, gob.NewEncoder(&buf).Encode(cmd))

	_, err := kv.Update([]statemachine.Entry{
		{Cmd: buf.Bytes(), Index: kv.GetLastApplied() + 1},
	})
	require.NoError(t, err)

	time.Sleep(2 * time.Second)

	query := general_command.Query_Command{
		Now: utils.GetNowInInt(),
		Command: general_command.RK_Command{
			Key:                key,
			ColumnFamilyName:   db.MasterEventFC,
			ColumnFamilySector: db.MasterEventFCSector,
			Op:                 general_command.GetOpTTL,
		},
	}
	// Use gob to encode the query command
	var bufQuery bytes.Buffer
	gob.NewEncoder(&bufQuery).Encode(query)
	val, err := kv.Lookup(bufQuery.Bytes())
	require.NoError(t, err)
	require.Nil(t, val)
}

func TestPebble_DeleteTTLRemovesFromCFAndExpirations(t *testing.T) {
	kv := setupKV(t, "pebble")
	defer kv.Close()

	key := "delete_ttl_key"

	// Insert with TTL
	var bufPut bytes.Buffer
	cmdPut := general_command.FSM_Command{
		Now:  utils.GetNowInInt(),
		Type: general_command.RW,
		CMD: general_command.RWK_Command{
			Op: general_command.Write,
			CMD: general_command.WK_Command{
				Key:                key,
				Value:              []byte("value"),
				ColumnFamilyName:   db.MasterEventFC,
				ColumnFamilySector: db.MasterEventFCSector,
				TTL:                60,
				Op:                 general_command.PutOpTTL,
			},
		},
	}
	require.NoError(t, gob.NewEncoder(&bufPut).Encode(cmdPut))

	_, err := kv.Update([]statemachine.Entry{
		{Cmd: bufPut.Bytes(), Index: kv.GetLastApplied() + 1},
	})
	require.NoError(t, err)

	// Delete it
	var bufDel bytes.Buffer
	cmdDel := general_command.FSM_Command{
		Now:  utils.GetNowInInt(),
		Type: general_command.RW,
		CMD: general_command.RWK_Command{
			Op: general_command.Write,
			CMD: general_command.WK_Command{
				Key:                key,
				ColumnFamilyName:   db.MasterEventFC,
				ColumnFamilySector: db.MasterEventFCSector,
				Op:                 general_command.DeleteOpTTL,
			},
		},
	}
	require.NoError(t, gob.NewEncoder(&bufDel).Encode(cmdDel))

	_, err = kv.Update([]statemachine.Entry{
		{Cmd: bufDel.Bytes(), Index: kv.GetLastApplied() + 1},
	})
	require.NoError(t, err)

	query := general_command.Query_Command{
		Now: utils.GetNowInInt(),
		Command: general_command.RK_Command{
			Key:                key,
			ColumnFamilyName:   db.MasterEventFC,
			ColumnFamilySector: db.MasterEventFCSector,
			Op:                 general_command.GetOpTTL,
		},
	}
	// Use gob to encode the query command
	var bufQuery bytes.Buffer
	gob.NewEncoder(&bufQuery).Encode(query)
	val, err := kv.Lookup(bufQuery.Bytes())
	require.NoError(t, err)
	require.Nil(t, val)
}

func TestPebble_KVStateMachine_ClearExpiredTTL(t *testing.T) {
	kv := setupKV(t, "pebble")
	defer kv.Close()

	key := "expiredKey"
	value := []byte("some-value")

	cmd := general_command.FSM_Command{
		Now:  utils.GetNowInInt(),
		Type: general_command.RW,
		CMD: general_command.RWK_Command{
			Op: general_command.Write,
			CMD: general_command.WK_Command{
				Key:                key,
				Value:              value,
				ColumnFamilyName:   db.MasterEventFC,
				ColumnFamilySector: db.MasterEventFCSector,
				TTL:                1,
				Op:                 general_command.PutOpTTL,
			},
		},
	}
	data := encodeCommand(t, cmd)

	_, err := kv.Update([]statemachine.Entry{
		{Cmd: data, Index: kv.GetLastApplied() + 1},
	})
	if err != nil {
		t.Fatalf("failed to insert key with TTL: %v", err)
	}

	time.Sleep(2 * time.Second)

	clearCmd := general_command.FSM_Command{
		Type: general_command.MCL,
		CMD: general_command.MCLK_Command{
			Op: general_command.ClearExpiredTTL,
		},
	}
	data = encodeCommand(t, clearCmd)

	_, err = kv.Update([]statemachine.Entry{
		{Cmd: data, Index: kv.GetLastApplied() + 1},
	})
	if err != nil {
		t.Fatalf("failed to clear expired TTL entries: %v", err)
	}

	query := general_command.Query_Command{
		Now: utils.GetNowInInt(),
		Command: general_command.RK_Command{
			Op:                 general_command.GetOpTTL,
			ColumnFamilyName:   db.MasterEventFC,
			ColumnFamilySector: db.MasterEventFCSector,
			Key:                key,
		},
	}
	var buf bytes.Buffer
	gob.NewEncoder(&buf).Encode(query)
	val, err := kv.Lookup(buf.Bytes())
	require.NoError(t, err)
	require.Nil(t, val)
}
func encodeCommand(t *testing.T, cmd general_command.FSM_Command) []byte { // Changed dragonboat.FSM_Command to general_command.FSM_Command
	var buf bytes.Buffer
	err := gob.NewEncoder(&buf).Encode(cmd)
	if err != nil {
		t.Fatalf("failed to encode command: %v", err)
	}
	return buf.Bytes()
}
func TestPebble_Update_UnknownCommandType(t *testing.T) {
	kv := setupKV(t, "pebble")
	defer kv.Close()

	cmd := general_command.FSM_Command{
		Type: 999, // This is an unknown type
		CMD:  nil,
	}
	data := encodeCommand(t, cmd)

	result, err := kv.Update([]statemachine.Entry{
		{Cmd: data, Index: kv.GetLastApplied() + 1},
	})
	require.NoError(t, err) // Update call itself should not error out
	require.NotNil(t, result)
	require.Len(t, result, 1)
	// Expect an error message in Result.Data
	require.Contains(t, string(result[0].Result.Data), "unknown command type: 999")
}

func TestPebble_Update_UnknownWriteOp(t *testing.T) {
	kv := setupKV(t, "pebble")
	defer kv.Close()

	cmd := general_command.FSM_Command{
		Now:  utils.GetNowInInt(),
		Type: general_command.RW,
		CMD: general_command.RWK_Command{
			Op: general_command.Write,
			CMD: general_command.WK_Command{
				Key:                "badop",
				Value:              []byte("x"),
				ColumnFamilyName:   db.DefaultFC,
				ColumnFamilySector: db.DefaultFCSector,
				Op:                 999, // This is an unknown op
			},
		},
	}
	data := encodeCommand(t, cmd)

	result, err := kv.Update([]statemachine.Entry{
		{Cmd: data, Index: kv.GetLastApplied() + 1},
	})
	require.NoError(t, err) // Update call itself should not error out
	require.NotNil(t, result)
	require.Len(t, result, 1)
	// Expect an error message in Result.Data
	require.Contains(t, string(result[0].Result.Data), "unknown W Operation: 999")
}
func TestPebble_RecoverFromSnapshot_InvalidData(t *testing.T) {
	kv := setupKV(t, "pebble")
	defer kv.Close()

	data := []byte("not-a-valid-gob")
	r := bytes.NewReader(data)

	err := kv.RecoverFromSnapshot(r, make(chan struct{}))
	require.Error(t, err)
	require.Contains(t, err.Error(), "decode failed")
}

type failingImpl struct {
	dragonboat.KVStateMachineImpl // Changed dragonboat.KVStateMachineImpl to KVStateMachineImpl
}

func (f *failingImpl) Update(ents []statemachine.Entry, batch *db.WriteBatch) ([]general_command.FSM_Command, error) { // Changed dragonboat.FSM_Command to general_command.FSM_Command
	return nil, errors.New("simulated update error")
}
func (f *failingImpl) Lookup(key interface{}) (general_command.RK_Command, error) { // Changed dragonboat.RK_Command to general_command.RK_Command
	return general_command.RK_Command{}, nil
}
func (f *failingImpl) OpenDB(string) (db.KVStore, error) {
	return nil, nil
}

func TestPebble_Lookup_Search_MultipleResults(t *testing.T) {
	kv := setupKV(t, "pebble")
	defer kv.Close()

	// Insert some entries
	entries := []general_command.WK_Command{ // Changed dragonboat.WK_Command to general_command.WK_Command
		{Key: "user:1", Value: []byte("a"), ColumnFamilyName: db.DefaultFC, ColumnFamilySector: db.DefaultFCSector, Op: general_command.PutOp},   // Changed dragonboat.PutOp to general_command.PutOp
		{Key: "user:2", Value: []byte("b"), ColumnFamilyName: db.DefaultFC, ColumnFamilySector: db.DefaultFCSector, Op: general_command.PutOp},   // Changed dragonboat.PutOp to general_command.PutOp
		{Key: "user:3", Value: []byte("c"), ColumnFamilyName: db.DefaultFC, ColumnFamilySector: db.DefaultFCSector, Op: general_command.PutOp},   // Changed dragonboat.PutOp to general_command.PutOp
		{Key: "user_x:3", Value: []byte("c"), ColumnFamilyName: db.DefaultFC, ColumnFamilySector: db.DefaultFCSector, Op: general_command.PutOp}, // Changed dragonboat.PutOp to general_command.PutOp
	}

	for _, entry := range entries {
		cmd := general_command.FSM_Command{
			Now:  utils.GetNowInInt(),
			Type: general_command.RW,
			CMD:  general_command.RWK_Command{Op: general_command.Write, CMD: entry},
		}
		data := encodeCommand(t, cmd)

		_, err := kv.Update([]statemachine.Entry{
			{Cmd: data, Index: kv.GetLastApplied() + 1},
		})
		require.NoError(t, err)
	}

	// Search with pattern "user:"
	query := general_command.Query_Command{
		Now: utils.GetNowInInt(),
		Command: general_command.RK_Command{
			KeyPattern:         "user:*",
			ColumnFamilyName:   db.DefaultFC,
			ColumnFamilySector: db.DefaultFCSector,
			Cursor:             "",
			Limit:              10,
			Op:                 general_command.Search, // Changed dragonboat.Search to general_command.Search
		},
	}

	var buf bytes.Buffer
	gob.NewEncoder(&buf).Encode(query) // Encode the query command
	res, err := kv.Lookup(buf.Bytes())
	require.NoError(t, err)
	paged := res.(*dragonboat.PagedResultKV) // Changed dragonboat.PagedResultKV to PagedResultKV
	require.Len(t, paged.Data, 3)
}

func TestPebble_Lookup_SearchTTL_OnlyValidResults(t *testing.T) {
	kv := setupKV(t, "pebble")
	defer kv.Close()

	// Insert 3 TTL entries: one expired, two valid
	entries := []struct {
		key   string
		value string
		ttl   int
	}{
		{"k1", "v1", 1}, // expired
		{"k2", "v2", 30},
		{"k3", "v3", 30},
	}

	for _, e := range entries {
		cmd := general_command.FSM_Command{
			Now:  utils.GetNowInInt(),
			Type: general_command.RW,
			CMD: general_command.RWK_Command{
				Op: general_command.Write,
				CMD: general_command.WK_Command{
					Key:                e.key,
					Value:              []byte(e.value),
					ColumnFamilyName:   db.MasterEventFC,
					ColumnFamilySector: db.MasterEventFCSector,
					TTL:                e.ttl,
					Op:                 general_command.PutOpTTL,
				},
			},
		}

		data := encodeCommand(t, cmd)

		_, err := kv.Update([]statemachine.Entry{
			{Cmd: data, Index: kv.GetLastApplied() + 1},
		})
		require.NoError(t, err)
	}
	time.Sleep(2 * time.Second)

	query := general_command.Query_Command{
		Now: utils.GetNowInInt(),
		Command: general_command.RK_Command{
			KeyPattern:         "k*",
			ColumnFamilyName:   db.MasterEventFC,
			ColumnFamilySector: db.MasterEventFCSector,
			Cursor:             "",
			Limit:              10,
			Op:                 general_command.SearchTTL, // Changed dragonboat.SearchTTL to general_command.SearchTTL
		},
	}

	var buf bytes.Buffer
	gob.NewEncoder(&buf).Encode(query) // Encode the query command
	res, err := kv.Lookup(buf.Bytes())
	require.NoError(t, err)
	paged := res.(*dragonboat.PagedResultKV) // Changed dragonboat.PagedResultKV to PagedResultKV
	require.Len(t, paged.Data, 2)

	validValues := []string{"v2", "v3"}
	actual := paged.Data
	require.Len(t, actual, 2)
	for _, expected := range validValues {
		found := false
		for _, kv := range actual {
			if string(kv.Value) == expected {
				found = true
				break
			}
		}
		require.True(t, found, "missing value %s", expected)
	}
}
func TestSaveSnapshotAndRecoverPebbleToPebbleDB(t *testing.T) {
	kvPebble := setupKVMasterPebble(t, "pebble")
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

	_, err := kvPebble.Update([]statemachine.Entry{
		{Cmd: buf.Bytes(), Index: kvPebble.GetLastApplied() + 1},
	})
	require.NoError(t, err)

	var snap bytes.Buffer
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()

	err = kvPebble.SaveSnapshot(nil, &snap, ctx.Done())
	require.NoError(t, err)

	_ = kvPebble.Close()
	pebbleDB := setupKVMasterPebble(t, "pebble")
	require.NoError(t, err)
	defer pebbleDB.Close()

	err = pebbleDB.RecoverFromSnapshot(&snap, ctx.Done())
	require.NoError(t, err)

	query1 := general_command.Query_Command{
		Now: utils.GetNowInInt(),
		Command: general_command.RK_Command{
			Key:                "snap_key",
			ColumnFamilyName:   db.DefaultFC,
			ColumnFamilySector: db.DefaultFCSector,
		},
	}

	var buf2 bytes.Buffer
	gob.NewEncoder(&buf2).Encode(query1)
	val, err := pebbleDB.Lookup(buf2.Bytes())
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

	var buf3 bytes.Buffer
	gob.NewEncoder(&buf3).Encode(query2)
	val, err = pebbleDB.Lookup(buf3.Bytes())
	require.NoError(t, err)
	require.Equal(t, pebbleDB.GetLastApplied(), binary.LittleEndian.Uint64(val.([]byte)))
}
