package dragonboat

import (
	"bytes"
	"context"
	"deadalus-orch/server/internal/pkg/utils"
	commands "deadalus-orch/server/internal/usecase/command"
	general_command "deadalus-orch/server/internal/usecase/command/general"
	"encoding/gob"
	"fmt"
	"time"

	"github.com/rs/zerolog"
)

// ExecuteRepositoryCommand executes a repository command and parses the result to the specified type T.
// This function encapsulates the common pattern of:
// 1. Creating an FSM_Command with REPOSITORY_COMMAND type
// 2. Writing to the Raft node with timeout
// 3. Decoding the result
// 4. Parsing to the expected type
//
// Parameters:
//   - raftNode: The RaftNode instance to execute the command on
//   - ctx: The context for the operation
//   - cmd: The repository command to execute
//   - timeout: The timeout duration for the operation
//   - logger: The logger instance for error logging
//   - operationName: A descriptive name for the operation (used in error messages)
//
// Returns:
//   - The parsed result of type T
//   - An error if any step fails
func ExecuteRepositoryCommand[T any](
	raftNode *RaftNode,
	ctx context.Context,
	cmd commands.Command,
	timeout time.Duration,
	logger zerolog.Logger,
	operationName string,
) (T, error) {
	var zero T

	// Create timeout context
	writeCtx, writeCancel := context.WithTimeout(ctx, timeout)
	defer writeCancel()

	// Create FSM command
	fsmCmd := general_command.FSM_Command{
		Now:  utils.GetNowInInt(),
		Type: general_command.REPOSITORY_COMMAND,
		CMD:  cmd,
	}

	// Execute write operation
	result, err := raftNode.Write(writeCtx, fsmCmd)
	if err != nil {
		logger.Error().Err(err).Msgf("Failed to execute %s", operationName)
		return zero, fmt.Errorf("failed to execute %s: %w", operationName, err)
	}

	// Decode result
	buf := bytes.NewBuffer(result.Data)
	dec := gob.NewDecoder(buf)
	parsedResult := &commands.CommandResult{}
	if err := dec.Decode(parsedResult); err != nil {
		logger.Error().Err(err).Msgf("%s command returned unexpected result type", operationName)
		return zero, fmt.Errorf("%s command returned decode error: %w", operationName, err)
	}

	// Check for command-level errors
	if parsedResult.Error != "" {
		return zero, fmt.Errorf("%s failed: %s", operationName, parsedResult.Error)
	}

	// Parse to expected type
	typedResult, ok := parsedResult.Result.(T)
	if !ok {
		return zero, fmt.Errorf("%s returned unexpected result type, expected %T", operationName, zero)
	}

	return typedResult, nil
}
