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

func setupKVMasterPebble(t *testing.T, engine string) *dragonboat.KVBaseStateMachine {
	t.Helper()
	t.Setenv(constants.EnvVarMasterDBEngine, engine)
	config.LoadDefaultConfiguration()
	kv := dragonboat.NewMasterKVStateMachine(1, 1).(*dragonboat.KVBaseStateMachine)
	stopc := make(chan struct{})
	_, err := kv.Open(stopc)
	require.NoError(t, err)
	return kv
}
func setupKV(t *testing.T, engine string) *dragonboat.KVBaseStateMachine { // Changed return type
	t.Helper()
	t.Setenv(constants.EnvVarMasterDBEngine, engine)
	config.LoadDefaultConfiguration()
	kv := dragonboat.NewMasterKVStateMachine(1, 1).(*dragonboat.KVBaseStateMachine) // Changed dragonboat.NewMasterKVStateMachine to NewMasterKVStateMachine and dragonboat.KVBaseStateMachine to KVBaseStateMachine
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

	query := commands.Query_Command{
		Now: utils.GetNowInInt(),
		Command: commands.RK_Command{
			Key:              "lookup_key",
			ColumnFamilyName: db.DefaultFC,
		},
	}

	val, err := kv.Lookup(query)
	require.NoError(t, err)
	require.Equal(t, []byte("lookup_value"), val)
}

func TestPebble_Lookup_NonExistingKey(t *testing.T) {
	kv := setupKV(t, "pebble")
	defer kv.Close()

	query := commands.Query_Command{
		Now: utils.GetNowInInt(),
		Command: commands.RK_Command{
			Key:              "missing_key",
			ColumnFamilyName: db.DefaultFC,
		},
	}
	val, err := kv.Lookup(query)
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

	kv2 := dragonboat.NewMasterKVStateMachine(1, 1).(*dragonboat.KVBaseStateMachine) // Changed dragonboat.NewMasterKVStateMachine to NewMasterKVStateMachine and dragonboat.KVBaseStateMachine to KVBaseStateMachine
	stopc := make(chan struct{})
	_, err = kv2.Open(stopc)
	require.NoError(t, err)
	defer kv2.Close()

	err = kv2.RecoverFromSnapshot(&snap, ctx.Done())
	require.NoError(t, err)

	query1 := commands.Query_Command{
		Now: utils.GetNowInInt(),
		Command: commands.RK_Command{
			Key:              "snap_key",
			ColumnFamilyName: db.DefaultFC,
		},
	}

	val, err := kv2.Lookup(query1)
	require.NoError(t, err)
	require.Equal(t, []byte("snap_value"), val)

	query2 := commands.Query_Command{
		Now: utils.GetNowInInt(),
		Command: commands.RK_Command{
			Key:              dragonboat.AppliedIndexKey, // This refers to a const in the non-moved dragonboat package
			ColumnFamilyName: db.MetaFC,
		},
	}

	val, err = kv2.Lookup(query2)
	require.NoError(t, err)
	require.Equal(t, kv2.GetLastApplied(), binary.LittleEndian.Uint64(val.([]byte)))
}

func TestPebble_SaveSnapshot_Cancelled(t *testing.T) {
	kv := setupKV(t, "pebble")
	defer kv.Close()

	done := make(chan struct{})
	close(done)

	err := kv.SaveSnapshot(nil, io.Discard, done)
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
func TestPebble_Update_DropColumnFamily(t *testing.T) {
	kv := setupKV(t, "pebble")
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

func TestPebble_Read_SingleEntryIntoUpdate(t *testing.T) {
	kv := setupKV(t, "pebble")
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
	require.Contains(t, string(result[0].Result.Data), "Invalid read operation: command.RWK_Command") // Changed dragonboat.RWK_Command to commands.RWK_Command
}
func TestPebble_Update_PutWithTTL(t *testing.T) {
	kv := setupKV(t, "pebble")
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
func TestPebble_Update_DropTTLColumnFamily(t *testing.T) {
	kv := setupKV(t, "pebble")
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

func TestPebble_Update_DeleteWithTTL(t *testing.T) {
	kv := setupKV(t, "pebble")
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
func TestPebble_PutTTLStoresWithExpiration(t *testing.T) {
	kv := setupKV(t, "pebble")
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

	query := commands.Query_Command{
		Now: utils.GetNowInInt(),
		Command: commands.RK_Command{
			Key:              "ttl_test_key",
			ColumnFamilyName: db.MasterEventFC,
			Op:               commands.GetOpTTL,
		},
	}

	val, err := kv.Lookup(query)
	require.NoError(t, err)
	require.Equal(t, []byte("ttl_test_value"), val)
}

func TestPebble_TTLExpirationRemovesKey(t *testing.T) {
	kv := setupKV(t, "pebble")
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

	query := commands.Query_Command{
		Now: utils.GetNowInInt(),
		Command: commands.RK_Command{
			Key:              key,
			ColumnFamilyName: db.MasterEventFC,
			Op:               commands.GetOpTTL,
		},
	}
	val, err := kv.Lookup(query)
	require.NoError(t, err)
	require.Nil(t, val)
}

func TestPebble_DeleteTTLRemovesFromCFAndExpirations(t *testing.T) {
	kv := setupKV(t, "pebble")
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

	query := commands.Query_Command{
		Now: utils.GetNowInInt(),
		Command: commands.RK_Command{
			Key:              key,
			ColumnFamilyName: db.MasterEventFC,
			Op:               commands.GetOpTTL,
		},
	}
	val, err := kv.Lookup(query)
	require.NoError(t, err)
	require.Nil(t, val)
}

func TestPebble_KVStateMachine_ClearExpiredTTL(t *testing.T) {
	kv := setupKV(t, "pebble")
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
	data := encodeCommand(t, cmd)

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
	data = encodeCommand(t, clearCmd)

	_, err = kv.Update([]statemachine.Entry{
		{Cmd: data, Index: kv.GetLastApplied() + 1},
	})
	if err != nil {
		t.Fatalf("failed to clear expired TTL entries: %v", err)
	}

	query := commands.Query_Command{
		Now: utils.GetNowInInt(),
		Command: commands.RK_Command{
			Op:               commands.GetOpTTL,
			ColumnFamilyName: db.MasterEventFC,
			Key:              key,
		},
	}
	val, err := kv.Lookup(query)
	require.NoError(t, err)
	require.Nil(t, val)
}
func encodeCommand(t *testing.T, cmd commands.FSM_Command) []byte { // Changed dragonboat.FSM_Command to commands.FSM_Command
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

	cmd := commands.FSM_Command{
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

	cmd := commands.FSM_Command{
		Now:  utils.GetNowInInt(),
		Type: commands.RW,
		CMD: commands.RWK_Command{
			Op: commands.Write,
			CMD: commands.WK_Command{
				Key:              "badop",
				Value:            []byte("x"),
				ColumnFamilyName: db.DefaultFC,
				Op:               999, // This is an unknown op
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

func (f *failingImpl) Update(ents []statemachine.Entry, batch *db.WriteBatch) ([]commands.FSM_Command, error) { // Changed dragonboat.FSM_Command to commands.FSM_Command
	return nil, errors.New("simulated update error")
}
func (f *failingImpl) Lookup(key interface{}) (commands.RK_Command, error) { // Changed dragonboat.RK_Command to commands.RK_Command
	return commands.RK_Command{}, nil
}
func (f *failingImpl) OpenDB(string) (db.KVStore, error) {
	return nil, nil
}

func TestPebble_Lookup_Search_MultipleResults(t *testing.T) {
	kv := setupKV(t, "pebble")
	defer kv.Close()

	// Insert some entries
	entries := []commands.WK_Command{ // Changed dragonboat.WK_Command to commands.WK_Command
		{Key: "user:1", Value: []byte("a"), ColumnFamilyName: db.DefaultFC, Op: commands.PutOp},   // Changed dragonboat.PutOp to commands.PutOp
		{Key: "user:2", Value: []byte("b"), ColumnFamilyName: db.DefaultFC, Op: commands.PutOp},   // Changed dragonboat.PutOp to commands.PutOp
		{Key: "user:3", Value: []byte("c"), ColumnFamilyName: db.DefaultFC, Op: commands.PutOp},   // Changed dragonboat.PutOp to commands.PutOp
		{Key: "user_x:3", Value: []byte("c"), ColumnFamilyName: db.DefaultFC, Op: commands.PutOp}, // Changed dragonboat.PutOp to commands.PutOp
	}

	for _, entry := range entries {
		cmd := commands.FSM_Command{
			Now:  utils.GetNowInInt(),
			Type: commands.RW,
			CMD:  commands.RWK_Command{Op: commands.Write, CMD: entry},
		}
		data := encodeCommand(t, cmd)

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
			Op:               commands.Search, // Changed dragonboat.Search to commands.Search
		},
	}

	res, err := kv.Lookup(query)
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
		cmd := commands.FSM_Command{
			Now:  utils.GetNowInInt(),
			Type: commands.RW,
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

		data := encodeCommand(t, cmd)

		_, err := kv.Update([]statemachine.Entry{
			{Cmd: data, Index: kv.GetLastApplied() + 1},
		})
		require.NoError(t, err)
	}
	time.Sleep(2 * time.Second)

	query := commands.Query_Command{
		Now: utils.GetNowInInt(),
		Command: commands.RK_Command{
			KeyPattern:       "k*",
			ColumnFamilyName: db.MasterEventFC,
			Cursor:           "",
			Limit:            10,
			Op:               commands.SearchTTL, // Changed dragonboat.SearchTTL to commands.SearchTTL
		},
	}

	res, err := kv.Lookup(query)
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
func TestSaveSnapshotAndRecoverPebbleToRocksDB(t *testing.T) {
	kvPebble := setupKVMasterPebble(t, "pebble")
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
	kvRocksDB := setupKVMasterPebble(t, "rocksdb")
	require.NoError(t, err)
	defer kvRocksDB.Close()

	err = kvRocksDB.RecoverFromSnapshot(&snap, ctx.Done())
	require.NoError(t, err)

	query1 := commands.Query_Command{
		Now: utils.GetNowInInt(),
		Command: commands.RK_Command{
			Key:              "snap_key",
			ColumnFamilyName: db.DefaultFC,
		},
	}

	val, err := kvRocksDB.Lookup(query1)
	require.NoError(t, err)
	require.Equal(t, []byte("snap_value"), val)

	query2 := commands.Query_Command{
		Now: utils.GetNowInInt(),
		Command: commands.RK_Command{
			Key:              dragonboat.AppliedIndexKey, // This refers to a const in the non-moved dragonboat package
			ColumnFamilyName: db.MetaFC,
		},
	}

	val, err = kvRocksDB.Lookup(query2)
	require.NoError(t, err)
	require.Equal(t, kvRocksDB.GetLastApplied(), binary.LittleEndian.Uint64(val.([]byte)))
}
