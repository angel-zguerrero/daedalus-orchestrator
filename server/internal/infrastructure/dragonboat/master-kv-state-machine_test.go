package dragonboat_test

import (
	"bytes"
	"context"
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	"deadalus-orch/server/internal/pkg/config"
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
	kv := dragonboat.NewMasterKVStateMachine(1, 1).(*dragonboat.KVBaseStateMachine)
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
	cmd := dragonboat.Command{
		Type: dragonboat.RW,
		CMD: dragonboat.RWK_Command{
			Op: dragonboat.Write,
			CMD: dragonboat.WK_Command{
				Key:              "foo",
				Value:            []byte("bar"),
				ColumnFamilyName: db.DefaultFC,
				Op:               dragonboat.PutOp,
			},
		},
	}

	err := gob.NewEncoder(&buf).Encode(cmd)
	require.NoError(t, err)

	entry := statemachine.Entry{
		Cmd:   buf.Bytes(),
		Index: kv.GetLastApplied() + 1,
	}

	result, err := kv.Update([]statemachine.Entry{entry})
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

	cmd := dragonboat.Command{
		Type: dragonboat.RW,
		CMD: dragonboat.RWK_Command{
			Op: dragonboat.Write,
			CMD: dragonboat.WK_Command{
				Key:              "lookup_key",
				Value:            []byte("lookup_value"),
				ColumnFamilyName: db.DefaultFC,
				Op:               dragonboat.PutOp,
			},
		},
	}

	require.NoError(t, gob.NewEncoder(&buf).Encode(cmd))

	_, err := kv.Update([]statemachine.Entry{
		{Cmd: buf.Bytes(), Index: kv.GetLastApplied() + 1},
	})
	require.NoError(t, err)

	query := dragonboat.RK_Command{
		Key:              "lookup_key",
		ColumnFamilyName: db.DefaultFC,
	}

	val, err := kv.Lookup(query)
	require.NoError(t, err)
	require.Equal(t, []byte("lookup_value"), val)
}

func TestLookup_NonExistingKey(t *testing.T) {
	kv := setupKVMaster(t, "rocksdb")
	defer kv.Close()

	query := dragonboat.RK_Command{
		Key:              "missing_key",
		ColumnFamilyName: db.DefaultFC,
	}
	val, err := kv.Lookup(query)
	require.NoError(t, err)
	require.Nil(t, val)
}

func TestLookup_InvalidType(t *testing.T) {
	kv := setupKVMaster(t, "rocksdb")
	defer kv.Close()

	_, err := kv.Lookup(0)
	require.Error(t, err)
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
	cmd := dragonboat.Command{
		Type: dragonboat.RW,
		CMD: dragonboat.RWK_Command{
			Op: dragonboat.Write,
			CMD: dragonboat.WK_Command{
				Key:              "snap_key",
				Value:            []byte("snap_value"),
				ColumnFamilyName: db.DefaultFC,
				Op:               dragonboat.PutOp,
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

	kv2 := dragonboat.NewMasterKVStateMachine(1, 1).(*dragonboat.KVBaseStateMachine)
	stopc := make(chan struct{})
	_, err = kv2.Open(stopc)
	require.NoError(t, err)
	defer kv2.Close()

	err = kv2.RecoverFromSnapshot(&snap, ctx.Done())
	require.NoError(t, err)

	query := dragonboat.RK_Command{
		Key:              "snap_key",
		ColumnFamilyName: db.DefaultFC,
	}

	val, err := kv2.Lookup(query)

	require.NoError(t, err)
	require.Equal(t, []byte("snap_value"), val)

	query = dragonboat.RK_Command{
		Key:              dragonboat.AppliedIndexKey,
		ColumnFamilyName: db.MetaFC,
	}

	val, err = kv2.Lookup(query)
	require.NoError(t, err)
	require.Equal(t, kv2.GetLastApplied(), binary.LittleEndian.Uint64(val.([]byte)))
}

func TestSaveSnapshot_Cancelled(t *testing.T) {
	kv := setupKVMaster(t, "rocksdb")
	defer kv.Close()

	done := make(chan struct{})
	close(done)

	err := kv.SaveSnapshot(nil, io.Discard, done)
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
	cmd := dragonboat.Command{
		Type: dragonboat.DDL_FC,
		CMD: dragonboat.DDL_Command{
			ColumnFamilyName: "new_cf",
			Op:               dragonboat.Add_CF_Op,
		},
	}
	err := gob.NewEncoder(&buf).Encode(cmd)
	require.NoError(t, err)

	entry := statemachine.Entry{
		Cmd:   buf.Bytes(),
		Index: kv.GetLastApplied() + 1,
	}

	result, err := kv.Update([]statemachine.Entry{entry})
	require.NoError(t, err)
	require.Equal(t, uint64(len(buf.Bytes())), result[0].Result.Value)
}
func TestUpdate_DropColumnFamily(t *testing.T) {
	kv := setupKVMaster(t, "rocksdb")
	defer kv.Close()
	{
		var buf bytes.Buffer
		cmd := dragonboat.Command{
			Type: dragonboat.DDL_FC,
			CMD: dragonboat.DDL_Command{

				ColumnFamilyName: "to_delete_cf",
				Op:               dragonboat.Add_CF_Op,
			},
		}
		err := gob.NewEncoder(&buf).Encode(cmd)
		require.NoError(t, err)

		entry := statemachine.Entry{
			Cmd:   buf.Bytes(),
			Index: kv.GetLastApplied() + 1,
		}

		_, err = kv.Update([]statemachine.Entry{entry})
		require.NoError(t, err)
	}

	var buf bytes.Buffer
	cmd := dragonboat.Command{
		Type: dragonboat.DDL_FC,
		CMD: dragonboat.DDL_Command{

			ColumnFamilyName: "to_delete_cf",
			Op:               dragonboat.Remove_CF_Op,
		},
	}
	err := gob.NewEncoder(&buf).Encode(cmd)
	require.NoError(t, err)

	entry := statemachine.Entry{
		Cmd:   buf.Bytes(),
		Index: kv.GetLastApplied() + 1,
	}

	result, err := kv.Update([]statemachine.Entry{entry})
	require.NoError(t, err)
	require.Equal(t, uint64(len(buf.Bytes())), result[0].Result.Value)
}

func TestRead_SingleEntryIntoUpdate(t *testing.T) {
	kv := setupKVMaster(t, "rocksdb")
	defer kv.Close()

	var buf bytes.Buffer
	cmd := dragonboat.Command{
		Type: dragonboat.RW,
		CMD: dragonboat.RWK_Command{
			Op: dragonboat.Read,
			CMD: dragonboat.RK_Command{
				Key:              "foo",
				ColumnFamilyName: db.DefaultFC,
				Op:               dragonboat.GetOp,
			},
		},
	}

	err := gob.NewEncoder(&buf).Encode(cmd)
	require.NoError(t, err)

	entry := statemachine.Entry{
		Cmd:   buf.Bytes(),
		Index: kv.GetLastApplied() + 1,
	}

	result, err := kv.Update([]statemachine.Entry{entry})
	require.Error(t, err)
	require.Nil(t, result)
}
func TestUpdate_PutWithTTL(t *testing.T) {
	kv := setupKVMaster(t, "rocksdb")
	defer kv.Close()

	var buf bytes.Buffer
	cmd := dragonboat.Command{
		Type: dragonboat.RW,
		CMD: dragonboat.RWK_Command{
			Op: dragonboat.Write,
			CMD: dragonboat.WK_Command{
				Key:              "ttl_key",
				Value:            []byte("ttl_value"),
				ColumnFamilyName: db.MasterEventFC,
				TTL:              5,
				Op:               dragonboat.PutOpTTL,
			},
		},
	}
	require.NoError(t, gob.NewEncoder(&buf).Encode(cmd))

	entry := statemachine.Entry{
		Cmd:   buf.Bytes(),
		Index: kv.GetLastApplied() + 1,
	}

	_, err := kv.Update([]statemachine.Entry{entry})
	require.NoError(t, err)
}
func TestUpdate_DropTTLColumnFamily(t *testing.T) {
	kv := setupKVMaster(t, "rocksdb")
	defer kv.Close()
	{
		var buf bytes.Buffer
		cmd := dragonboat.Command{
			Type: dragonboat.DDL_FC,
			CMD: dragonboat.DDL_Command{

				ColumnFamilyName: "to_delete_cf",
				Op:               dragonboat.Add_TTL_CF_Op,
			},
		}
		err := gob.NewEncoder(&buf).Encode(cmd)
		require.NoError(t, err)

		entry := statemachine.Entry{
			Cmd:   buf.Bytes(),
			Index: kv.GetLastApplied() + 1,
		}

		_, err = kv.Update([]statemachine.Entry{entry})
		require.NoError(t, err)
	}

	var buf bytes.Buffer
	cmd := dragonboat.Command{
		Type: dragonboat.DDL_FC,
		CMD: dragonboat.DDL_Command{
			ColumnFamilyName: "to_delete_cf",
			Op:               dragonboat.Remove_TTL_CF_Op,
		},
	}
	err := gob.NewEncoder(&buf).Encode(cmd)
	require.NoError(t, err)

	entry := statemachine.Entry{
		Cmd:   buf.Bytes(),
		Index: kv.GetLastApplied() + 1,
	}

	result, err := kv.Update([]statemachine.Entry{entry})
	require.NoError(t, err)
	require.Equal(t, uint64(len(buf.Bytes())), result[0].Result.Value)
}

func TestUpdate_DeleteWithTTL(t *testing.T) {
	kv := setupKVMaster(t, "rocksdb")
	defer kv.Close()

	var buf bytes.Buffer
	cmd := dragonboat.Command{
		Type: dragonboat.RW,
		CMD: dragonboat.RWK_Command{
			Op: dragonboat.Write,
			CMD: dragonboat.WK_Command{
				Key:              "ttl_key",
				Value:            []byte("ttl_value"),
				ColumnFamilyName: db.MasterEventFC,
				TTL:              5,
				Op:               dragonboat.PutOpTTL,
			},
		},
	}
	require.NoError(t, gob.NewEncoder(&buf).Encode(cmd))

	entry := statemachine.Entry{
		Cmd:   buf.Bytes(),
		Index: kv.GetLastApplied() + 1,
	}

	_, err := kv.Update([]statemachine.Entry{entry})
	require.NoError(t, err)

	var bufDel bytes.Buffer
	cmd = dragonboat.Command{
		Type: dragonboat.RW,
		CMD: dragonboat.RWK_Command{
			Op: dragonboat.Write,
			CMD: dragonboat.WK_Command{
				Key:              "ttl_key",
				ColumnFamilyName: db.MasterEventFC,
				TTL:              5,
				Op:               dragonboat.DeleteOpTTL,
			},
		},
	}
	require.NoError(t, gob.NewEncoder(&bufDel).Encode(cmd))

	entry = statemachine.Entry{
		Cmd:   bufDel.Bytes(),
		Index: kv.GetLastApplied() + 1,
	}

	_, err = kv.Update([]statemachine.Entry{entry})
	require.NoError(t, err)
}
func TestPutTTLStoresWithExpiration(t *testing.T) {
	kv := setupKVMaster(t, "rocksdb")
	defer kv.Close()

	var buf bytes.Buffer
	cmd := dragonboat.Command{
		Type: dragonboat.RW,
		CMD: dragonboat.RWK_Command{
			Op: dragonboat.Write,
			CMD: dragonboat.WK_Command{
				Key:              "ttl_test_key",
				Value:            []byte("ttl_test_value"),
				ColumnFamilyName: db.MasterEventFC,
				TTL:              10,
				Op:               dragonboat.PutOpTTL,
			},
		},
	}
	require.NoError(t, gob.NewEncoder(&buf).Encode(cmd))
	entry := statemachine.Entry{Cmd: buf.Bytes(), Index: kv.GetLastApplied() + 1}
	_, err := kv.Update([]statemachine.Entry{entry})
	require.NoError(t, err)

	query := dragonboat.RK_Command{
		Key:              "ttl_test_key",
		ColumnFamilyName: db.MasterEventFC,
		Op:               dragonboat.GetOpTTL,
	}

	val, err := kv.Lookup(query)
	require.NoError(t, err)
	require.Equal(t, []byte("ttl_test_value"), val)
}

func TestTTLExpirationRemovesKey(t *testing.T) {
	kv := setupKVMaster(t, "rocksdb")
	defer kv.Close()

	key := "expiring_key"

	// Put TTL entry
	var buf bytes.Buffer
	cmd := dragonboat.Command{
		Type: dragonboat.RW,
		CMD: dragonboat.RWK_Command{
			Op: dragonboat.Write,
			CMD: dragonboat.WK_Command{
				Key:              key,
				Value:            []byte("soon_gone"),
				ColumnFamilyName: db.MasterEventFC,
				TTL:              1,
				Op:               dragonboat.PutOpTTL,
			},
		},
	}
	require.NoError(t, gob.NewEncoder(&buf).Encode(cmd))
	entry := statemachine.Entry{Cmd: buf.Bytes(), Index: kv.GetLastApplied() + 1}
	_, err := kv.Update([]statemachine.Entry{entry})
	require.NoError(t, err)

	time.Sleep(2 * time.Second)

	query := dragonboat.RK_Command{
		Key:              key,
		ColumnFamilyName: db.MasterEventFC,
		Op:               dragonboat.GetOpTTL,
	}
	val, err := kv.Lookup(query)
	require.NoError(t, err)
	require.Nil(t, val)
}

func TestDeleteTTLRemovesFromCFAndExpirations(t *testing.T) {
	kv := setupKVMaster(t, "rocksdb")
	defer kv.Close()

	key := "delete_ttl_key"

	// Insert with TTL
	var bufPut bytes.Buffer
	cmdPut := dragonboat.Command{
		Type: dragonboat.RW,
		CMD: dragonboat.RWK_Command{
			Op: dragonboat.Write,
			CMD: dragonboat.WK_Command{
				Key:              key,
				Value:            []byte("value"),
				ColumnFamilyName: db.MasterEventFC,
				TTL:              60,
				Op:               dragonboat.PutOpTTL,
			},
		},
	}
	require.NoError(t, gob.NewEncoder(&bufPut).Encode(cmdPut))
	entry := statemachine.Entry{Cmd: bufPut.Bytes(), Index: kv.GetLastApplied() + 1}
	_, err := kv.Update([]statemachine.Entry{entry})
	require.NoError(t, err)

	// Delete it
	var bufDel bytes.Buffer
	cmdDel := dragonboat.Command{
		Type: dragonboat.RW,
		CMD: dragonboat.RWK_Command{
			Op: dragonboat.Write,
			CMD: dragonboat.WK_Command{
				Key:              key,
				ColumnFamilyName: db.MasterEventFC,
				Op:               dragonboat.DeleteOpTTL,
			},
		},
	}
	require.NoError(t, gob.NewEncoder(&bufDel).Encode(cmdDel))
	entry = statemachine.Entry{Cmd: bufDel.Bytes(), Index: kv.GetLastApplied() + 1}
	_, err = kv.Update([]statemachine.Entry{entry})
	require.NoError(t, err)

	query := dragonboat.RK_Command{
		Key:              key,
		ColumnFamilyName: db.MasterEventFC,
		Op:               dragonboat.GetOpTTL,
	}
	val, err := kv.Lookup(query)
	require.NoError(t, err)
	require.Nil(t, val)
}

func TestKVStateMachine_ClearExpiredTTL(t *testing.T) {
	kv := setupKVMaster(t, "rocksdb")
	defer kv.Close()

	key := "expiredKey"
	value := []byte("some-value")

	cmd := dragonboat.Command{
		Type: dragonboat.RW,
		CMD: dragonboat.RWK_Command{
			Op: dragonboat.Write,
			CMD: dragonboat.WK_Command{
				Key:              key,
				Value:            value,
				ColumnFamilyName: db.MasterEventFC,
				TTL:              1,
				Op:               dragonboat.PutOpTTL,
			},
		},
	}
	data := encodeCommandR(t, cmd)

	entry := statemachine.Entry{Cmd: data, Index: kv.GetLastApplied() + 1}

	_, err := kv.Update([]statemachine.Entry{entry})
	if err != nil {
		t.Fatalf("failed to insert key with TTL: %v", err)
	}

	time.Sleep(2 * time.Second)

	clearCmd := dragonboat.Command{
		Type: dragonboat.MCL,
		CMD: dragonboat.MCLK_Command{
			Op: dragonboat.ClearExpiredTTL,
		},
	}
	data = encodeCommandR(t, clearCmd)
	entry = statemachine.Entry{Cmd: data, Index: kv.GetLastApplied() + 1}
	_, err = kv.Update([]statemachine.Entry{entry})
	if err != nil {
		t.Fatalf("failed to clear expired TTL entries: %v", err)
	}

	query := dragonboat.RK_Command{
		Op:               dragonboat.GetOpTTL,
		ColumnFamilyName: db.MasterEventFC,
		Key:              key,
	}
	val, err := kv.Lookup(query)
	require.NoError(t, err)
	require.Nil(t, val)
}
func encodeCommandR(t *testing.T, cmd dragonboat.Command) []byte {
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

	cmd := dragonboat.Command{
		Type: 999,
		CMD:  nil,
	}
	data := encodeCommandR(t, cmd)

	entry := statemachine.Entry{Cmd: data, Index: kv.GetLastApplied() + 1}
	_, err := kv.Update([]statemachine.Entry{entry})
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown command type")
}

func TestUpdate_UnknownWriteOp(t *testing.T) {
	kv := setupKVMaster(t, "rocksdb")
	defer kv.Close()

	cmd := dragonboat.Command{
		Type: dragonboat.RW,
		CMD: dragonboat.RWK_Command{
			Op: dragonboat.Write,
			CMD: dragonboat.WK_Command{
				Key:              "badop",
				Value:            []byte("x"),
				ColumnFamilyName: db.DefaultFC,
				Op:               999,
			},
		},
	}
	data := encodeCommandR(t, cmd)
	entry := statemachine.Entry{Cmd: data, Index: kv.GetLastApplied() + 1}

	_, err := kv.Update([]statemachine.Entry{entry})
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown W Operation")
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

func (f *failingRocksdbImpl) Update(ents []statemachine.Entry, batch *db.WriteBatch) ([]dragonboat.Command, error) {
	return nil, errors.New("simulated update error")
}
func (f *failingRocksdbImpl) Lookup(key interface{}) (dragonboat.RK_Command, error) {
	return dragonboat.RK_Command{}, nil
}
func (f *failingRocksdbImpl) OpenDB(string) (db.KVStore, error) {
	return nil, nil
}

func TestLookup_Search_MultipleResults(t *testing.T) {
	kv := setupKVMaster(t, "rocksdb")
	defer kv.Close()

	// Insert some entries
	entries := []dragonboat.WK_Command{
		{Key: "user:1", Value: []byte("a"), ColumnFamilyName: db.DefaultFC, Op: dragonboat.PutOp},
		{Key: "user:2", Value: []byte("b"), ColumnFamilyName: db.DefaultFC, Op: dragonboat.PutOp},
		{Key: "user:3", Value: []byte("c"), ColumnFamilyName: db.DefaultFC, Op: dragonboat.PutOp},
		{Key: "user_x:3", Value: []byte("c"), ColumnFamilyName: db.DefaultFC, Op: dragonboat.PutOp},
	}

	for _, entry := range entries {
		cmd := dragonboat.Command{
			Type: dragonboat.RW,
			CMD:  dragonboat.RWK_Command{Op: dragonboat.Write, CMD: entry},
		}
		data := encodeCommandR(t, cmd)
		_, err := kv.Update([]statemachine.Entry{{Cmd: data, Index: kv.GetLastApplied() + 1}})
		require.NoError(t, err)
	}

	// Search with pattern "user:"
	query := dragonboat.RK_Command{
		KeyPattern:       "user:*",
		ColumnFamilyName: db.DefaultFC,
		Cursor:           "",
		Limit:            10,
		Op:               dragonboat.Search,
	}

	res, err := kv.Lookup(query)
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
		cmd := dragonboat.Command{
			Type: dragonboat.RW,
			CMD: dragonboat.RWK_Command{
				Op: dragonboat.Write,
				CMD: dragonboat.WK_Command{
					Key:              e.key,
					Value:            []byte(e.value),
					ColumnFamilyName: db.MasterEventFC,
					TTL:              e.ttl,
					Op:               dragonboat.PutOpTTL,
				},
			},
		}

		data := encodeCommandR(t, cmd)
		_, err := kv.Update([]statemachine.Entry{{Cmd: data, Index: kv.GetLastApplied() + 1}})
		require.NoError(t, err)
	}
	time.Sleep(2 * time.Second)

	query := dragonboat.RK_Command{
		KeyPattern:       "k:*",
		ColumnFamilyName: db.MasterEventFC,
		Cursor:           "",
		Limit:            10,
		Op:               dragonboat.SearchTTL,
	}

	res, err := kv.Lookup(query)
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
