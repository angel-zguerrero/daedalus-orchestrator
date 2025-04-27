package dragonboat_test

import (
	"bytes"
	"context"
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	"encoding/binary"
	"encoding/gob"
	"io"
	"testing"
	"time"

	"github.com/lni/dragonboat/v4/statemachine"
	"github.com/stretchr/testify/require"
)

func setupKV(t *testing.T) *dragonboat.KVStateMachine {
	t.Helper()
	kv := dragonboat.NewKVStateMachine(1, 1).(*dragonboat.KVStateMachine)
	stopc := make(chan struct{})
	_, err := kv.Open(stopc)
	require.NoError(t, err)
	return kv
}

func TestOne(t *testing.T) {

	//dragonboat.Init(1, 1, "3435")
	//dragonboat.Init(1, 2, "3436")

	//time.Sleep(240 * time.Second)
}
func TestOpen_Close(t *testing.T) {
	kv := setupKV(t)

	err := kv.Close()
	require.NoError(t, err)

	require.Panics(t, func() {
		_ = kv.Close()
	})
}

func TestUpdate_SingleEntry(t *testing.T) {
	kv := setupKV(t)
	defer kv.Close()

	var buf bytes.Buffer
	cmd := struct {
		Key   []byte
		Value []byte
	}{
		Key:   []byte("foo"),
		Value: []byte("bar"),
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
	kv := setupKV(t)
	_ = kv.Close()

	require.Panics(t, func() {
		_, _ = kv.Update(nil)
	})
}

func TestLookup_ExistingKey(t *testing.T) {
	kv := setupKV(t)
	defer kv.Close()

	var buf bytes.Buffer
	cmd := struct {
		Key   []byte
		Value []byte
	}{
		Key:   []byte("lookup_key"),
		Value: []byte("lookup_value"),
	}
	require.NoError(t, gob.NewEncoder(&buf).Encode(cmd))

	_, err := kv.Update([]statemachine.Entry{
		{Cmd: buf.Bytes(), Index: kv.GetLastApplied() + 1},
	})
	require.NoError(t, err)

	// Buscar
	val, err := kv.Lookup([]byte("lookup_key"))
	require.NoError(t, err)
	require.Equal(t, []byte("lookup_value"), val)
}

func TestLookup_NonExistingKey(t *testing.T) {
	kv := setupKV(t)
	defer kv.Close()

	val, err := kv.Lookup([]byte("missing_key"))
	require.NoError(t, err)
	require.Nil(t, val)
}

func TestLookup_InvalidType(t *testing.T) {
	kv := setupKV(t)
	defer kv.Close()

	_, err := kv.Lookup("not_bytes")
	require.Error(t, err)
}

func TestSync(t *testing.T) {
	kv := setupKV(t)
	defer kv.Close()

	err := kv.Sync()
	require.NoError(t, err)
}

func TestSaveSnapshotAndRecover(t *testing.T) {
	kv := setupKV(t)

	var buf bytes.Buffer
	cmd := struct {
		Key   []byte
		Value []byte
	}{
		Key:   []byte("snap_key"),
		Value: []byte("snap_value"),
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

	kv2 := dragonboat.NewKVStateMachine(1, 1).(*dragonboat.KVStateMachine)
	stopc := make(chan struct{})
	_, err = kv2.Open(stopc)
	require.NoError(t, err)
	defer kv2.Close()

	err = kv2.RecoverFromSnapshot(&snap, ctx.Done())
	require.NoError(t, err)

	val, err := kv2.Lookup([]byte("snap_key"))
	require.NoError(t, err)
	require.Equal(t, []byte("snap_value"), val)

	val, err = kv2.Lookup([]byte(dragonboat.AppliedIndexKey))
	require.NoError(t, err)
	require.Equal(t, kv2.GetLastApplied(), binary.LittleEndian.Uint64(val.([]byte)))
}

func TestSaveSnapshot_Cancelled(t *testing.T) {
	kv := setupKV(t)
	defer kv.Close()

	done := make(chan struct{})
	close(done) // ya cancelado

	err := kv.SaveSnapshot(nil, io.Discard, done)
	require.Error(t, err)
	require.Contains(t, err.Error(), "snapshot cancelled")
}

func TestRecoverSnapshot_Cancelled(t *testing.T) {
	kv := setupKV(t)
	defer kv.Close()

	// simulamos un reader infinito
	r, w := io.Pipe()
	done := make(chan struct{})
	close(done)

	go func() {
		// no escribimos nada
		_ = w.Close()
	}()

	err := kv.RecoverFromSnapshot(r, done)
	require.Error(t, err)
	require.Contains(t, err.Error(), "snapshot recovery cancelled")
}
