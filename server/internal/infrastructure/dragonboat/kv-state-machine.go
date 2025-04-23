package dragonboat

import (
	"bytes"
	"deadalus-orch/server/internal/infrastructure/db"
	"encoding/gob"
	"fmt"
	"io"

	"github.com/lni/dragonboat/v4/statemachine"
)

type KVStateMachine struct {
	store db.KVStore
}

func NewKVStateMachine(store db.KVStore) statemachine.IStateMachine {
	return &KVStateMachine{store: store}
}

func (s *KVStateMachine) Update(data statemachine.Entry) (statemachine.Result, error) {
	var cmd struct {
		Key   []byte
		Value []byte
	}

	dec := gob.NewDecoder(bytes.NewReader(data.Cmd))
	if err := dec.Decode(&cmd); err != nil {
		return statemachine.Result{}, err
	}

	err := s.store.Put(cmd.Key, cmd.Value)
	if err != nil {
		return statemachine.Result{}, err
	}

	return statemachine.Result{Value: uint64(len(cmd.Value))}, nil
}

func (s *KVStateMachine) Lookup(query interface{}) (interface{}, error) {
	key, ok := query.([]byte)
	if !ok {
		return nil, fmt.Errorf("invalid query type")
	}

	slice, err := s.store.Get(key)
	if err != nil {
		return nil, err
	}

	defer slice.Free()
	if !slice.Exists() {
		return nil, nil
	}

	return slice.Data(), nil
}

func (s *KVStateMachine) Close() error { return nil }

func (s *KVStateMachine) SaveSnapshot(
	w io.Writer,
	fc statemachine.ISnapshotFileCollection,
	done <-chan struct{},
) error {
	enc := gob.NewEncoder(w)

	err := s.store.Iterate(func(key, value []byte) error {
		select {
		case <-done:
			return fmt.Errorf("snapshot cancelled")
		default:
		}

		entry := struct {
			Key   []byte
			Value []byte
		}{
			Key:   append([]byte(nil), key...),
			Value: append([]byte(nil), value...),
		}

		if err := enc.Encode(&entry); err != nil {
			return fmt.Errorf("encoding failed: %w", err)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("snapshot save failed: %w", err)
	}

	return nil
}

func (s *KVStateMachine) RecoverFromSnapshot(
	r io.Reader,
	files []statemachine.SnapshotFile,
	done <-chan struct{},
) error {
	// Limpiar el estado actual antes de aplicar snapshot
	if err := s.store.ClearAll(); err != nil {
		return fmt.Errorf("failed to clear existing state: %w", err)
	}

	dec := gob.NewDecoder(r)

	for {
		select {
		case <-done:
			return fmt.Errorf("snapshot recovery cancelled")
		default:
			// seguimos
		}

		var entry struct {
			Key   []byte
			Value []byte
		}

		err := dec.Decode(&entry)
		if err != nil {
			if err == io.EOF {
				break // Terminamos la lectura
			}
			return fmt.Errorf("failed to decode snapshot: %w", err)
		}

		if err := s.store.Put(entry.Key, entry.Value); err != nil {
			return fmt.Errorf("failed to write key in snapshot recovery: %w", err)
		}
	}

	return nil
}
