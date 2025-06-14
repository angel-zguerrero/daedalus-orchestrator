package dragonboat_test

import (
	"bytes"
	"context"
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	"deadalus-orch/server/internal/pkg/config" // Added
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

	kv := dragonboat.NewTenantKVStateMachine(1, 2).(*dragonboat.KVBaseStateMachine)
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

func TestTenantLookup_NonExistingKey(t *testing.T) {
	kv := setupTenantKV(t)
	defer kv.Close()

	query := dragonboat.RK_Command{
		Key:              "missing_key",
		ColumnFamilyName: db.DefaultFC,
	}
	val, err := kv.Lookup(query)
	require.NoError(t, err)
	require.Nil(t, val)
}

func TestTenantLookup_InvalidType(t *testing.T) {
	kv := setupTenantKV(t)
	defer kv.Close()

	_, err := kv.Lookup(0)
	require.Error(t, err)
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err = kv.SaveSnapshot(nil, &snap, ctx.Done())
	require.NoError(t, err)

	_ = kv.Close()

	// Also ensure config is loaded for the second instance if it's path dependent
	err = config.LoadDefaultConfiguration()
	require.NoError(t, err, "Failed to load default configuration for test setup (kv2)")
	kv2 := dragonboat.NewTenantKVStateMachine(1, 2).(*dragonboat.KVBaseStateMachine)
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

func TestTenantSaveSnapshot_Cancelled(t *testing.T) {
	kv := setupTenantKV(t)
	defer kv.Close()

	done := make(chan struct{})
	close(done)

	err := kv.SaveSnapshot(nil, io.Discard, done)
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
func TestTenantUpdate_DropColumnFamily(t *testing.T) {
	kv := setupTenantKV(t)
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

func TestTenantRead_SingleEntryIntoUpdate(t *testing.T) {
	kv := setupTenantKV(t)
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
func TestTenantUpdate_PutWithTTL(t *testing.T) {
	kv := setupTenantKV(t)
	defer kv.Close()

	var buf bytes.Buffer
	cmd := dragonboat.Command{
		Type: dragonboat.RW,
		CMD: dragonboat.RWK_Command{
			Op: dragonboat.Write,
			CMD: dragonboat.WK_Command{
				Key:              "ttl_key",
				Value:            []byte("ttl_value"),
				ColumnFamilyName: db.TenantEventFC,
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
func TestTenantUpdate_DropTTLColumnFamily(t *testing.T) {
	kv := setupTenantKV(t)
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

func TestTenantUpdate_DeleteWithTTL(t *testing.T) {
	kv := setupTenantKV(t)
	defer kv.Close()

	var buf bytes.Buffer
	cmd := dragonboat.Command{
		Type: dragonboat.RW,
		CMD: dragonboat.RWK_Command{
			Op: dragonboat.Write,
			CMD: dragonboat.WK_Command{
				Key:              "ttl_key",
				Value:            []byte("ttl_value"),
				ColumnFamilyName: db.TenantEventFC,
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
				ColumnFamilyName: db.TenantEventFC,
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
