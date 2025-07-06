//go:build rocksdb
// +build rocksdb

package dragonboat_test

import (
	"bytes"
	"context"
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	"deadalus-orch/server/internal/pkg/config"
	"deadalus-orch/server/internal/pkg/utils"
	commands "deadalus-orch/server/internal/usecase/command"
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

func TestOne(t *testing.T) {

	//dragonboat.Init(101, 1, "3435")
	//dragonboat.Init(101, 2, "3436")

	//time.Sleep(10 * time.Second)
}
func setupKVMaster(t *testing.T, engine string) *dragonboat.KVBaseStateMachine {
	t.Helper()
	t.Setenv(constants.EnvVarMasterDBEngine, engine)
	config.LoadDefaultConfiguration()
	kv := dragonboat.NewMasterKVStateMachine(dragonboat.TestPathProvider{Path: t.TempDir()}, 3000)(1, 1).(*dragonboat.KVBaseStateMachine)
	stopc := make(chan struct{})
	_, err := kv.Open(stopc)
	require.NoError(t, err)
	return kv
}
func TestOpen_Close(t *testing.T) {
	kv := setupKVMaster(t, "rocksdb")

	err := kv.Close()
	require.NoError(t, err)

	require.Panics(t, func() {
		_ = kv.Close()
	})
}

func TestUpdate_SingleEntry(t *testing.T) {
	kv := setupKVMaster(t, "rocksdb")
	defer kv.Close()

	var buf bytes.Buffer
	cmd := commands.FSM_Command{
		Now:  utils.GetNowInInt(),
		Type: commands.RW,
		CMD: commands.RWK_Command{
			Op: commands.Write,
			CMD: commands.WK_Command{
				Key:              "foo",
				Value:            []byte("bar"),
				ColumnFamilyName: db.DefaultFC,
				Op:               commands.PutOp,
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

func TestUpdate_AfterClose_Panics(t *testing.T) {
	kv := setupKVMaster(t, "rocksdb")
	_ = kv.Close()

	require.Panics(t, func() {
		_, _ = kv.Update(nil)
	})
}

func TestLookup_ExistingKey(t *testing.T) {
	kv := setupKVMaster(t, "rocksdb")
	defer kv.Close()

	var buf bytes.Buffer

	cmd := commands.FSM_Command{
		Now:  utils.GetNowInInt(),
		Type: commands.RW,
		CMD: commands.RWK_Command{
			Op: commands.Write,
			CMD: commands.WK_Command{
				Key:              "lookup_key",
				Value:            []byte("lookup_value"),
				ColumnFamilyName: db.DefaultFC,
				Op:               commands.PutOp,
			},
		},
	}

	require.NoError(t, gob.NewEncoder(&buf).Encode(cmd))

	_, err := kv.Update([]statemachine.Entry{
		{Cmd: buf.Bytes(), Index: kv.GetLastApplied() + 1},
	})
	require.NoError(t, err)

	require.NoError(t, err)
	query := commands.Query_Command{
		Now: utils.GetNowInInt(),
		Command: commands.RK_Command{
			Key:              "lookup_key",
			ColumnFamilyName: db.DefaultFC,
		},
	}
	var bufQ bytes.Buffer
	gob.NewEncoder(&bufQ).Encode(query)
	val, err := kv.Lookup(bufQ.Bytes())
	require.NoError(t, err)
	require.Equal(t, []byte("lookup_value"), val)
}

func TestLookup_NonExistingKey(t *testing.T) {
	kv := setupKVMaster(t, "rocksdb")
	defer kv.Close()

	query := commands.Query_Command{
		Now: utils.GetNowInInt(),
		Command: commands.RK_Command{
			Key:              "missing_key",
			ColumnFamilyName: db.DefaultFC,
		},
	}
	var buf bytes.Buffer
	gob.NewEncoder(&buf).Encode(query)

	val, err := kv.Lookup(buf.Bytes())
	require.NoError(t, err)
	require.Nil(t, val)
}

func TestSync(t *testing.T) {
	kv := setupKVMaster(t, "rocksdb")
	defer kv.Close()

	err := kv.Sync()
	require.NoError(t, err)
}

func TestSaveSnapshotAndRecover(t *testing.T) {
	kv := setupKVMaster(t, "rocksdb")

	var buf bytes.Buffer
	cmd := commands.FSM_Command{
		Now:  utils.GetNowInInt(),
		Type: commands.RW,
		CMD: commands.RWK_Command{
			Op: commands.Write,
			CMD: commands.WK_Command{
				Key:              "snap_key",
				Value:            []byte("snap_value"),
				ColumnFamilyName: db.DefaultFC,
				Op:               commands.PutOp,
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

	kv2 := dragonboat.NewMasterKVStateMachine(dragonboat.TestPathProvider{Path: t.TempDir()}, 3000)(1, 1).(*dragonboat.KVBaseStateMachine)
	stopc := make(chan struct{})
	_, err = kv2.Open(stopc)
	require.NoError(t, err)
	defer kv2.Close()

	err = kv2.RecoverFromSnapshot(&snap, ctx.Done())
	require.NoError(t, err)

	require.NoError(t, err)
	query1 := commands.Query_Command{
		Now: utils.GetNowInInt(),
		Command: commands.RK_Command{
			Key:              "snap_key",
			ColumnFamilyName: db.DefaultFC,
		},
	}

	var bufQ bytes.Buffer
	gob.NewEncoder(&bufQ).Encode(query1)
	val, err := kv2.Lookup(bufQ.Bytes())
	require.NoError(t, err)
	require.Equal(t, []byte("snap_value"), val)

	require.NoError(t, err)
	query2 := commands.Query_Command{
		Now: utils.GetNowInInt(),
		Command: commands.RK_Command{
			Key:              dragonboat.AppliedIndexKey,
			ColumnFamilyName: db.MetaFC,
		},
	}

	var buf2 bytes.Buffer
	gob.NewEncoder(&buf2).Encode(query2)
	val, err = kv2.Lookup(buf2.Bytes())
	require.NoError(t, err)
	require.Equal(t, kv2.GetLastApplied(), binary.LittleEndian.Uint64(val.([]byte)))
}

func TestSaveSnapshot_Cancelled(t *testing.T) {
	kv := setupKVMaster(t, "rocksdb")
	defer kv.Close()

	done := make(chan struct{})
	close(done)

	var buf bytes.Buffer
	cmd := commands.FSM_Command{
		Now:  utils.GetNowInInt(),
		Type: commands.RW,
		CMD: commands.RWK_Command{
			Op: commands.Write,
			CMD: commands.WK_Command{
				Key:              "snap_key",
				Value:            []byte("snap_value"),
				ColumnFamilyName: db.DefaultFC,
				Op:               commands.PutOp,
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

func TestRecoverSnapshot_Cancelled(t *testing.T) {
	kv := setupKVMaster(t, "rocksdb")
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

func TestUpdate_AddColumnFamily(t *testing.T) {
	kv := setupKVMaster(t, "rocksdb")
	defer kv.Close()

	var buf bytes.Buffer
	cmd := commands.FSM_Command{
		Type: commands.DDL_FC,
		CMD: commands.DDL_Command{
			ColumnFamilyName: "new_cf",
			Op:               commands.Add_CF_Op,
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
func TestUpdate_DropColumnFamily(t *testing.T) {
	kv := setupKVMaster(t, "rocksdb")
	defer kv.Close()
	{
		var buf bytes.Buffer
		cmd := commands.FSM_Command{
			Type: commands.DDL_FC,
			CMD: commands.DDL_Command{

				ColumnFamilyName: "to_delete_cf",
				Op:               commands.Add_CF_Op,
			},
		}
		err := gob.NewEncoder(&buf).Encode(cmd)
		require.NoError(t, err)

		require.NoError(t, err)

		_, err = kv.Update([]statemachine.Entry{
			{Cmd: buf.Bytes(), Index: kv.GetLastApplied() + 1},
		})
		require.NoError(t, err)
	}

	var buf bytes.Buffer
	cmd := commands.FSM_Command{
		Type: commands.DDL_FC,
		CMD: commands.DDL_Command{

			ColumnFamilyName: "to_delete_cf",
			Op:               commands.Remove_CF_Op,
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

func TestRead_SingleEntryIntoUpdate(t *testing.T) {
	kv := setupKVMaster(t, "rocksdb")
	defer kv.Close()

	var buf bytes.Buffer
	cmd := commands.FSM_Command{
		Now:  utils.GetNowInInt(),
		Type: commands.RW,
		CMD: commands.RWK_Command{
			Op: commands.Read,
			CMD: commands.RK_Command{
				Key:              "foo",
				ColumnFamilyName: db.DefaultFC,
				Op:               commands.GetOp,
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
	require.Contains(t, string(result[0].Result.Data), "Invalid read operation: command.RWK_Command")
}
func TestUpdate_PutWithTTL(t *testing.T) {
	kv := setupKVMaster(t, "rocksdb")
	defer kv.Close()

	var buf bytes.Buffer
	cmd := commands.FSM_Command{
		Now:  utils.GetNowInInt(),
		Type: commands.RW,
		CMD: commands.RWK_Command{
			Op: commands.Write,
			CMD: commands.WK_Command{
				Key:              "ttl_key",
				Value:            []byte("ttl_value"),
				ColumnFamilyName: db.MasterEventFC,
				TTL:              5,
				Op:               commands.PutOpTTL,
			},
		},
	}
	require.NoError(t, gob.NewEncoder(&buf).Encode(cmd))

	_, err := kv.Update([]statemachine.Entry{
		{Cmd: buf.Bytes(), Index: kv.GetLastApplied() + 1},
	})
	require.NoError(t, err)
}
func TestUpdate_DropTTLColumnFamily(t *testing.T) {
	kv := setupKVMaster(t, "rocksdb")
	defer kv.Close()
	{
		var buf bytes.Buffer
		cmd := commands.FSM_Command{
			Type: commands.DDL_FC,
			CMD: commands.DDL_Command{

				ColumnFamilyName: "to_delete_cf",
				Op:               commands.Add_TTL_CF_Op,
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
	cmd := commands.FSM_Command{
		Type: commands.DDL_FC,
		CMD: commands.DDL_Command{
			ColumnFamilyName: "to_delete_cf",
			Op:               commands.Remove_TTL_CF_Op,
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

func TestUpdate_DeleteWithTTL(t *testing.T) {
	kv := setupKVMaster(t, "rocksdb")
	defer kv.Close()

	var buf bytes.Buffer
	cmd := commands.FSM_Command{
		Now:  utils.GetNowInInt(),
		Type: commands.RW,
		CMD: commands.RWK_Command{
			Op: commands.Write,
			CMD: commands.WK_Command{
				Key:              "ttl_key",
				Value:            []byte("ttl_value"),
				ColumnFamilyName: db.MasterEventFC,
				TTL:              5,
				Op:               commands.PutOpTTL,
			},
		},
	}
	require.NoError(t, gob.NewEncoder(&buf).Encode(cmd))

	_, err := kv.Update([]statemachine.Entry{
		{Cmd: buf.Bytes(), Index: kv.GetLastApplied() + 1},
	})
	require.NoError(t, err)

	var bufDel bytes.Buffer
	cmd = commands.FSM_Command{
		Now:  utils.GetNowInInt(),
		Type: commands.RW,
		CMD: commands.RWK_Command{
			Op: commands.Write,
			CMD: commands.WK_Command{
				Key:              "ttl_key",
				ColumnFamilyName: db.MasterEventFC,
				TTL:              5,
				Op:               commands.DeleteOpTTL,
			},
		},
	}
	require.NoError(t, gob.NewEncoder(&bufDel).Encode(cmd))

	_, err = kv.Update([]statemachine.Entry{
		{Cmd: bufDel.Bytes(), Index: kv.GetLastApplied() + 1},
	})
	require.NoError(t, err)
}
func TestPutTTLStoresWithExpiration(t *testing.T) {
	kv := setupKVMaster(t, "rocksdb")
	defer kv.Close()

	var buf bytes.Buffer
	cmd := commands.FSM_Command{
		Now:  utils.GetNowInInt(),
		Type: commands.RW,
		CMD: commands.RWK_Command{
			Op: commands.Write,
			CMD: commands.WK_Command{
				Key:              "ttl_test_key",
				Value:            []byte("ttl_test_value"),
				ColumnFamilyName: db.MasterEventFC,
				TTL:              10,
				Op:               commands.PutOpTTL,
			},
		},
	}
	require.NoError(t, gob.NewEncoder(&buf).Encode(cmd))

	_, err := kv.Update([]statemachine.Entry{
		{Cmd: buf.Bytes(), Index: kv.GetLastApplied() + 1},
	})
	require.NoError(t, err)

	require.NoError(t, err)
	query := commands.Query_Command{
		Now: utils.GetNowInInt(),
		Command: commands.RK_Command{
			Key:              "ttl_test_key",
			ColumnFamilyName: db.MasterEventFC,
			Op:               commands.GetOpTTL,
		},
	}

	var bufQ bytes.Buffer
	gob.NewEncoder(&bufQ).Encode(query)
	val, err := kv.Lookup(bufQ.Bytes())
	require.NoError(t, err)
	require.Equal(t, []byte("ttl_test_value"), val)
}

func TestTTLExpirationRemovesKey(t *testing.T) {
	kv := setupKVMaster(t, "rocksdb")
	defer kv.Close()

	key := "expiring_key"

	// Put TTL entry
	var buf bytes.Buffer
	cmd := commands.FSM_Command{
		Now:  utils.GetNowInInt(),
		Type: commands.RW,
		CMD: commands.RWK_Command{
			Op: commands.Write,
			CMD: commands.WK_Command{
				Key:              key,
				Value:            []byte("soon_gone"),
				ColumnFamilyName: db.MasterEventFC,
				TTL:              1,
				Op:               commands.PutOpTTL,
			},
		},
	}
	require.NoError(t, gob.NewEncoder(&buf).Encode(cmd))

	_, err := kv.Update([]statemachine.Entry{
		{Cmd: buf.Bytes(), Index: kv.GetLastApplied() + 1},
	})
	require.NoError(t, err)

	time.Sleep(2 * time.Second)

	require.NoError(t, err)
	query := commands.Query_Command{
		Now: utils.GetNowInInt(),
		Command: commands.RK_Command{
			Key:              key,
			ColumnFamilyName: db.MasterEventFC,
			Op:               commands.GetOpTTL,
		},
	}
	var bufQ bytes.Buffer
	gob.NewEncoder(&bufQ).Encode(query)
	val, err := kv.Lookup(bufQ.Bytes())
	require.NoError(t, err)
	require.Nil(t, val)
}

func TestDeleteTTLRemovesFromCFAndExpirations(t *testing.T) {
	kv := setupKVMaster(t, "rocksdb")
	defer kv.Close()

	key := "delete_ttl_key"

	// Insert with TTL
	var bufPut bytes.Buffer
	cmdPut := commands.FSM_Command{
		Now:  utils.GetNowInInt(),
		Type: commands.RW,
		CMD: commands.RWK_Command{
			Op: commands.Write,
			CMD: commands.WK_Command{
				Key:              key,
				Value:            []byte("value"),
				ColumnFamilyName: db.MasterEventFC,
				TTL:              60,
				Op:               commands.PutOpTTL,
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
	cmdDel := commands.FSM_Command{
		Now:  utils.GetNowInInt(),
		Type: commands.RW,
		CMD: commands.RWK_Command{
			Op: commands.Write,
			CMD: commands.WK_Command{
				Key:              key,
				ColumnFamilyName: db.MasterEventFC,
				Op:               commands.DeleteOpTTL,
			},
		},
	}
	require.NoError(t, gob.NewEncoder(&bufDel).Encode(cmdDel))

	_, err = kv.Update([]statemachine.Entry{
		{Cmd: bufDel.Bytes(), Index: kv.GetLastApplied() + 1},
	})
	require.NoError(t, err)

	require.NoError(t, err)
	query := commands.Query_Command{
		Now: utils.GetNowInInt(),
		Command: commands.RK_Command{
			Key:              key,
			ColumnFamilyName: db.MasterEventFC,
			Op:               commands.GetOpTTL,
		},
	}
	var bufQ bytes.Buffer
	gob.NewEncoder(&bufQ).Encode(query)
	val, err := kv.Lookup(bufQ.Bytes())
	require.NoError(t, err)
	require.Nil(t, val)
}

func TestKVStateMachine_ClearExpiredTTL(t *testing.T) {
	kv := setupKVMaster(t, "rocksdb")
	defer kv.Close()

	key := "expiredKey"
	value := []byte("some-value")

	cmd := commands.FSM_Command{
		Now:  utils.GetNowInInt(),
		Type: commands.RW,
		CMD: commands.RWK_Command{
			Op: commands.Write,
			CMD: commands.WK_Command{
				Key:              key,
				Value:            value,
				ColumnFamilyName: db.MasterEventFC,
				TTL:              1,
				Op:               commands.PutOpTTL,
			},
		},
	}
	data := encodeCommandR(t, cmd)

	_, err := kv.Update([]statemachine.Entry{
		{Cmd: data, Index: kv.GetLastApplied() + 1},
	})
	if err != nil {
		t.Fatalf("failed to insert key with TTL: %v", err)
	}

	time.Sleep(2 * time.Second)

	clearCmd := commands.FSM_Command{
		Type: commands.MCL,
		CMD: commands.MCLK_Command{
			Op: commands.ClearExpiredTTL,
		},
	}
	data = encodeCommandR(t, clearCmd)

	_, err = kv.Update([]statemachine.Entry{
		{Cmd: data, Index: kv.GetLastApplied() + 1},
	})
	if err != nil {
		t.Fatalf("failed to clear expired TTL entries: %v", err)
	}

	require.NoError(t, err)
	query := commands.Query_Command{
		Now: utils.GetNowInInt(),
		Command: commands.RK_Command{
			Op:               commands.GetOpTTL,
			ColumnFamilyName: db.MasterEventFC,
			Key:              key,
		},
	}
	var buf bytes.Buffer
	gob.NewEncoder(&buf).Encode(query)
	val, err := kv.Lookup(buf.Bytes())
	require.NoError(t, err)
	require.Nil(t, val)
}
func encodeCommandR(t *testing.T, cmd commands.FSM_Command) []byte {
	var buf bytes.Buffer
	err := gob.NewEncoder(&buf).Encode(cmd)
	if err != nil {
		t.Fatalf("failed to encode command: %v", err)
	}
	return buf.Bytes()
}
func TestUpdate_UnknownCommandType(t *testing.T) {
	kv := setupKVMaster(t, "rocksdb")
	defer kv.Close()

	cmd := commands.FSM_Command{
		Type: 999,
		CMD:  nil,
	}
	data := encodeCommandR(t, cmd)

	result, err := kv.Update([]statemachine.Entry{
		{Cmd: data, Index: kv.GetLastApplied() + 1},
	})
	require.NoError(t, err) // Update call itself should not error out
	require.NotNil(t, result)
	require.Len(t, result, 1)
	// Expect an error message in Result.Data
	require.Contains(t, string(result[0].Result.Data), "unknown command type: 999")
}

func TestUpdate_UnknownWriteOp(t *testing.T) {
	kv := setupKVMaster(t, "rocksdb")
	defer kv.Close()

	cmd := commands.FSM_Command{
		Now:  utils.GetNowInInt(),
		Type: commands.RW,
		CMD: commands.RWK_Command{
			Op: commands.Write,
			CMD: commands.WK_Command{
				Key:              "badop",
				Value:            []byte("x"),
				ColumnFamilyName: db.DefaultFC,
				Op:               999,
			},
		},
	}
	data := encodeCommandR(t, cmd)

	result, err := kv.Update([]statemachine.Entry{
		{Cmd: data, Index: kv.GetLastApplied() + 1},
	})
	require.NoError(t, err) // Update call itself should not error out
	require.NotNil(t, result)
	require.Len(t, result, 1)
	// Expect an error message in Result.Data
	require.Contains(t, string(result[0].Result.Data), "unknown W Operation: 999")
}

func TestUpdate_InvalidNowField(t *testing.T) {
	kv := setupKVMaster(t, "rocksdb")
	defer kv.Close()

	var buf bytes.Buffer
	cmd := commands.FSM_Command{
		Now:  0, // Invalid 'Now' field
		Type: commands.RW,
		CMD: commands.RWK_Command{
			Op: commands.Write,
			CMD: commands.WK_Command{
				Key:              "foo",
				Value:            []byte("bar"),
				ColumnFamilyName: db.DefaultFC,
				Op:               commands.PutOp,
			},
		},
	}

	err := gob.NewEncoder(&buf).Encode(cmd)
	require.NoError(t, err)

	entries := []statemachine.Entry{
		{Cmd: buf.Bytes(), Index: kv.GetLastApplied() + 1},
	}
	result, err := kv.Update(entries)
	require.NoError(t, err) // The Update method itself should not error for this validation
	require.Len(t, result, 1)
	require.Equal(t, uint64(len(buf.Bytes())), result[0].Result.Value)
	require.Equal(t, commands.ErrMissingOrInvalidNowField.Error(), string(result[0].Result.Data))

	// Verify that the data was not actually written
	queryCmd := commands.Query_Command{
		Now: utils.GetNowInInt(), // Use a valid 'Now' for lookup
		Command: commands.RK_Command{
			Key:              "foo",
			ColumnFamilyName: db.DefaultFC,
			Op:               commands.GetOp,
		},
	}
	var queryBuf bytes.Buffer
	err = gob.NewEncoder(&queryBuf).Encode(queryCmd)
	require.NoError(t, err)

	lookedUpValue, err := kv.Lookup(queryBuf.Bytes())
	require.NoError(t, err)
	require.Nil(t, lookedUpValue, "Data should not have been written due to invalid 'Now' field")
}

func TestLookup_InvalidNowField(t *testing.T) {
	kv := setupKVMaster(t, "rocksdb")
	defer kv.Close()

	// Attempt to lookup a key with an invalid 'Now' field
	queryCmd := commands.Query_Command{
		Now: 0, // Invalid 'Now' field
		Command: commands.RK_Command{
			Key:              "any_key",
			ColumnFamilyName: db.DefaultFC,
			Op:               commands.GetOp,
		},
	}
	var queryBuf bytes.Buffer
	err := gob.NewEncoder(&queryBuf).Encode(queryCmd)
	require.NoError(t, err)

	_, err = kv.Lookup(queryBuf.Bytes())
	require.Error(t, err)
	require.Equal(t, commands.ErrMissingOrInvalidNowField, err)
}

func TestUpdate_ValidNowField(t *testing.T) {
	kv := setupKVMaster(t, "rocksdb")
	defer kv.Close()

	validNow := utils.GetNowInInt()
	require.Greater(t, validNow, int64(0), "Generated 'Now' should be valid")

	var buf bytes.Buffer
	cmd := commands.FSM_Command{
		Now:  validNow, // Valid 'Now' field
		Type: commands.RW,
		CMD: commands.RWK_Command{
			Op: commands.Write,
			CMD: commands.WK_Command{
				Key:              "valid_now_key",
				Value:            []byte("valid_now_value"),
				ColumnFamilyName: db.DefaultFC,
				Op:               commands.PutOp,
			},
		},
	}

	err := gob.NewEncoder(&buf).Encode(cmd)
	require.NoError(t, err)

	entries := []statemachine.Entry{
		{Cmd: buf.Bytes(), Index: kv.GetLastApplied() + 1},
	}
	result, err := kv.Update(entries)
	require.NoError(t, err)
	require.Len(t, result, 1)
	require.Equal(t, uint64(len(buf.Bytes())), result[0].Result.Value)
	// For valid command, Data might be nil or empty, not an error message
	if len(result[0].Result.Data) > 0 {
		require.NotEqual(t, commands.ErrMissingOrInvalidNowField.Error(), string(result[0].Result.Data))
	}

	// Verify that the data was actually written
	queryCmd := commands.Query_Command{
		Now: utils.GetNowInInt(), // Use a valid 'Now' for lookup
		Command: commands.RK_Command{
			Key:              "valid_now_key",
			ColumnFamilyName: db.DefaultFC,
			Op:               commands.GetOp,
		},
	}
	var queryBuf bytes.Buffer
	err = gob.NewEncoder(&queryBuf).Encode(queryCmd)
	require.NoError(t, err)

	lookedUpValue, err := kv.Lookup(queryBuf.Bytes())
	require.NoError(t, err)
	require.Equal(t, []byte("valid_now_value"), lookedUpValue)
}

func TestLookup_ValidNowField(t *testing.T) {
	kv := setupKVMaster(t, "rocksdb")
	defer kv.Close()

	// First, write a key to lookup
	keyToLookup := "lookup_valid_now"
	valueToLookup := "some_value_for_valid_now_lookup"
	var writeBuf bytes.Buffer
	writeCmd := commands.FSM_Command{
		Now:  utils.GetNowInInt(), // Valid 'Now' for write
		Type: commands.RW,
		CMD: commands.RWK_Command{
			Op: commands.Write,
			CMD: commands.WK_Command{
				Key:              keyToLookup,
				Value:            []byte(valueToLookup),
				ColumnFamilyName: db.DefaultFC,
				Op:               commands.PutOp,
			},
		},
	}
	err := gob.NewEncoder(&writeBuf).Encode(writeCmd)
	require.NoError(t, err)
	_, err = kv.Update([]statemachine.Entry{{Cmd: writeBuf.Bytes(), Index: kv.GetLastApplied() + 1}})
	require.NoError(t, err)

	// Attempt to lookup the key with a valid 'Now' field
	validNow := utils.GetNowInInt()
	require.Greater(t, validNow, int64(0), "Generated 'Now' for lookup should be valid")

	queryCmd := commands.Query_Command{
		Now: validNow, // Valid 'Now' field
		Command: commands.RK_Command{
			Key:              keyToLookup,
			ColumnFamilyName: db.DefaultFC,
			Op:               commands.GetOp,
		},
	}
	var queryBuf bytes.Buffer
	err = gob.NewEncoder(&queryBuf).Encode(queryCmd)
	require.NoError(t, err)

	lookedUpValue, err := kv.Lookup(queryBuf.Bytes())
	require.NoError(t, err)
	require.Equal(t, []byte(valueToLookup), lookedUpValue)
}

func TestRecoverFromSnapshot_InvalidData(t *testing.T) {
	kv := setupKVMaster(t, "rocksdb")
	defer kv.Close()

	data := []byte("not-a-valid-gob")
	r := bytes.NewReader(data)

	err := kv.RecoverFromSnapshot(r, make(chan struct{}))
	require.Error(t, err)
	require.Contains(t, err.Error(), "decode failed")
}

type failingRocksdbImpl struct {
	dragonboat.KVStateMachineImpl
}

func (f *failingRocksdbImpl) Update(ents []statemachine.Entry, batch *db.WriteBatch) ([]commands.FSM_Command, error) {
	return nil, errors.New("simulated update error")
}
func (f *failingRocksdbImpl) Lookup(key interface{}) (commands.RK_Command, error) {
	return commands.RK_Command{}, nil
}
func (f *failingRocksdbImpl) OpenDB(string) (db.KVStore, error) {
	return nil, nil
}

func TestLookup_Search_MultipleResults(t *testing.T) {
	kv := setupKVMaster(t, "rocksdb")
	defer kv.Close()

	// Insert some entries
	entries := []commands.WK_Command{
		{Key: "user:1", Value: []byte("a"), ColumnFamilyName: db.DefaultFC, Op: commands.PutOp},
		{Key: "user:2", Value: []byte("b"), ColumnFamilyName: db.DefaultFC, Op: commands.PutOp},
		{Key: "user:3", Value: []byte("c"), ColumnFamilyName: db.DefaultFC, Op: commands.PutOp},
		{Key: "user_x:3", Value: []byte("c"), ColumnFamilyName: db.DefaultFC, Op: commands.PutOp},
	}

	for _, entry := range entries {
		cmd := commands.FSM_Command{
			Now:  utils.GetNowInInt(),
			Type: commands.RW,
			CMD:  commands.RWK_Command{Op: commands.Write, CMD: entry},
		}
		data := encodeCommandR(t, cmd)

		_, err := kv.Update([]statemachine.Entry{
			{Cmd: data, Index: kv.GetLastApplied() + 1},
		})
		require.NoError(t, err)
	}

	// Search with pattern "user:"

	query := commands.Query_Command{
		Now: utils.GetNowInInt(),
		Command: commands.RK_Command{
			KeyPattern:       "user:*",
			ColumnFamilyName: db.DefaultFC,
			Cursor:           "",
			Limit:            10,
			Op:               commands.Search,
		},
	}

	var buf bytes.Buffer
	gob.NewEncoder(&buf).Encode(query)
	res, err := kv.Lookup(buf.Bytes())
	require.NoError(t, err)
	paged := res.(*dragonboat.PagedResultKV)
	require.Len(t, paged.Data, 3)
}

func TestLookup_SearchTTL_OnlyValidResults(t *testing.T) {
	kv := setupKVMaster(t, "rocksdb")
	defer kv.Close()

	// Insert 3 TTL entries: one expired, two valid
	entries := []struct {
		key   string
		value string
		ttl   int
	}{
		{"k:1", "v1", 1}, // expired
		{"k:2", "v2", 30},
		{"k:3", "v3", 30},
	}

	for _, e := range entries {
		cmd := commands.FSM_Command{
			Type: commands.RW,
			Now:  utils.GetNowInInt(),
			CMD: commands.RWK_Command{
				Op: commands.Write,
				CMD: commands.WK_Command{
					Key:              e.key,
					Value:            []byte(e.value),
					ColumnFamilyName: db.MasterEventFC,
					TTL:              e.ttl,
					Op:               commands.PutOpTTL,
				},
			},
		}

		data := encodeCommandR(t, cmd)

		_, err := kv.Update([]statemachine.Entry{
			{Cmd: data, Index: kv.GetLastApplied() + 1},
		})
		require.NoError(t, err)
	}
	time.Sleep(2 * time.Second)

	query := commands.Query_Command{
		Now: utils.GetNowInInt(),
		Command: commands.RK_Command{
			KeyPattern:       "k:*",
			ColumnFamilyName: db.MasterEventFC,
			Cursor:           "",
			Limit:            10,
			Op:               commands.SearchTTL,
		},
	}

	var buf bytes.Buffer
	gob.NewEncoder(&buf).Encode(query)

	res, err := kv.Lookup(buf.Bytes())
	require.NoError(t, err)
	paged := res.(*dragonboat.PagedResultKV)
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
func TestSaveSnapshotAndRecoverRocksToPebble(t *testing.T) {
	kvRocksDB := setupKVMaster(t, "rocksdb")
	var buf bytes.Buffer
	cmd := commands.FSM_Command{
		Now:  utils.GetNowInInt(),
		Type: commands.RW,
		CMD: commands.RWK_Command{
			Op: commands.Write,
			CMD: commands.WK_Command{
				Key:              "snap_key",
				Value:            []byte("snap_value"),
				ColumnFamilyName: db.DefaultFC,
				Op:               commands.PutOp,
			},
		},
	}

	require.NoError(t, gob.NewEncoder(&buf).Encode(cmd))

	_, err := kvRocksDB.Update([]statemachine.Entry{
		{Cmd: buf.Bytes(), Index: kvRocksDB.GetLastApplied() + 2},
	})
	require.NoError(t, err)

	var snap bytes.Buffer
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()

	err = kvRocksDB.SaveSnapshot(nil, &snap, ctx.Done())
	require.NoError(t, err)

	_ = kvRocksDB.Close()
	kvPebble := setupKVMaster(t, "pebble")
	require.NoError(t, err)
	defer kvPebble.Close()

	err = kvPebble.RecoverFromSnapshot(&snap, ctx.Done())
	require.NoError(t, err)

	require.NoError(t, err)
	query1 := commands.Query_Command{
		Now: utils.GetNowInInt(),
		Command: commands.RK_Command{
			Key:              "snap_key",
			ColumnFamilyName: db.DefaultFC,
		},
	}
	var bufQ bytes.Buffer
	gob.NewEncoder(&bufQ).Encode(query1)
	val, err := kvPebble.Lookup(bufQ.Bytes())
	require.NoError(t, err)
	require.Equal(t, []byte("snap_value"), val)

	require.NoError(t, err)
	query2 := commands.Query_Command{
		Now: utils.GetNowInInt(),
		Command: commands.RK_Command{
			Key:              dragonboat.AppliedIndexKey,
			ColumnFamilyName: db.MetaFC,
		},
	}
	var bufQ2 bytes.Buffer
	gob.NewEncoder(&bufQ2).Encode(query2)
	val, err = kvPebble.Lookup(bufQ2.Bytes())
	require.NoError(t, err)
	require.Equal(t, kvPebble.GetLastApplied(), binary.LittleEndian.Uint64(val.([]byte)))
}
