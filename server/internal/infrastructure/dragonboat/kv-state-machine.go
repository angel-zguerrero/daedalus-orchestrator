package dragonboat

import (
	"bytes"
	"deadalus-orch/server/internal/infrastructure/db"
	"encoding/gob"
	"fmt"
	"io"

	"github.com/lni/dragonboat/v4/statemachine"
)

type OnDiskKVStateMachine struct {
	store db.KVStore
}

func NewOnDiskKVStateMachine(store db.KVStore) statemachine.IOnDiskStateMachine {
	return &OnDiskKVStateMachine{store: store}
}

func (s *OnDiskKVStateMachine) Open(stopc <-chan struct{}) (uint64, error) {
	// Devuelve el log index del último snapshot aplicado, 0 si no hay.
	return 0, nil
}

func (s *OnDiskKVStateMachine) Update(data []statemachine.Entry) ([]statemachine.Entry, error) {
	for i := range data {
		var cmd struct {
			Key   []byte
			Value []byte
		}

		if err := gob.NewDecoder(bytes.NewReader(data[i].Cmd)).Decode(&cmd); err != nil {
			return nil, err
		}

		if err := s.store.Put(cmd.Key, cmd.Value); err != nil {
			return nil, err
		}

		data[i].Result.Value = uint64(len(cmd.Value))
	}
	return data, nil
}

func (s *OnDiskKVStateMachine) Lookup(query interface{}) (interface{}, error) {
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

func (s *OnDiskKVStateMachine) Sync() error {
	return s.store.Flush()
}

func (s *OnDiskKVStateMachine) PrepareSnapshot() (interface{}, error) {
	return nil, nil
}

func (s *OnDiskKVStateMachine) SaveSnapshot(
	ctx interface{},
	w io.Writer,
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

		return enc.Encode(&entry)
	})

	if err != nil {
		return fmt.Errorf("snapshot save failed: %w", err)
	}

	return nil
}

func (s *OnDiskKVStateMachine) RecoverFromSnapshot(
	r io.Reader,
	done <-chan struct{},
) error {
	if err := s.store.ClearAll(); err != nil {
		return fmt.Errorf("failed to clear state: %w", err)
	}

	dec := gob.NewDecoder(r)

	for {
		select {
		case <-done:
			return fmt.Errorf("snapshot recovery cancelled")
		default:
		}

		var entry struct {
			Key   []byte
			Value []byte
		}

		if err := dec.Decode(&entry); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("decode failed: %w", err)
		}

		if err := s.store.Put(entry.Key, entry.Value); err != nil {
			return fmt.Errorf("put failed during snapshot recovery: %w", err)
		}
	}

	return nil
}

func (s *OnDiskKVStateMachine) Close() error {
	return nil
}

func (s *OnDiskKVStateMachine) GetHash() (uint64, error) {
	// Opcional: puedes devolver un hash del estado actual si quieres integridad
	return 0, nil
}
