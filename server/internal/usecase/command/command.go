package command

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"encoding/json"
	"fmt"
	"time"
)

type CommandResult struct {
	Error  string `json:"error,omitempty"`
	Result any    `json:"result,omitempty"`
}

// Command defines the interface for all executable commands.
type Command interface {
	// Execute processes the command using the given UnitOfWork and timestamp.
	// The UnitOfWork must not be created internally by the command.
	// The now timestamp must be provided to the command.
	Execute(uow *db.UnitOfWork, now time.Time) CommandResult
}

// EncodeCommandResult encodes CommandResult into JSON bytes without using gob.
func EncodeCommandResult(res CommandResult) ([]byte, error) {
	var rawResult json.RawMessage
	if res.Result != nil {
		b, err := json.Marshal(res.Result)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal CommandResult payload: %w", err)
		}
		rawResult = b
	}

	dto := struct {
		Error string          `json:"error,omitempty"`
		Data  json.RawMessage `json:"data,omitempty"`
	}{
		Error: res.Error,
		Data:  rawResult,
	}

	return json.Marshal(dto)
}

// DecodeCommandResult decodes JSON bytes into target type T.
func DecodeCommandResult[T any](data []byte) (T, error) {
	var zero T
	if len(data) == 0 {
		return zero, nil
	}

	var dto struct {
		Error string          `json:"error,omitempty"`
		Data  json.RawMessage `json:"data,omitempty"`
	}

	if err := json.Unmarshal(data, &dto); err != nil {
		return zero, fmt.Errorf("command result JSON decode error: %w", err)
	}

	if dto.Error != "" {
		return zero, fmt.Errorf("%s", dto.Error)
	}

	if len(dto.Data) == 0 || string(dto.Data) == "null" {
		return zero, nil
	}

	var val T
	if err := json.Unmarshal(dto.Data, &val); err != nil {
		return zero, fmt.Errorf("command result data JSON decode error: %w", err)
	}

	return val, nil
}
