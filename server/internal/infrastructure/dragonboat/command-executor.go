package dragonboat

import (
	"bytes"
	"context"
	"deadalus-orch/server/internal/pkg/utils"
	commands "deadalus-orch/server/internal/usecase/command"
	general_command "deadalus-orch/server/internal/usecase/command/general"
	"encoding/gob"
	"fmt"
	"strings"
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

	// Only create a new timeout if the parent context doesn't already have a deadline.
	// This prevents double-wrapping (caller sets 30s, then this adds another timeout),
	// which causes "invalid deadline" errors when the parent expires first.
	writeCtx := ctx
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var writeCancel context.CancelFunc
		writeCtx, writeCancel = context.WithTimeout(ctx, timeout)
		defer writeCancel()
	}

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

	// Check for nil results
	if parsedResult.Result == nil {
		return zero, nil
	}

	// Parse to expected type
	typedResult, ok := parsedResult.Result.(T)
	if !ok {
		return zero, fmt.Errorf("%s returned unexpected result type, expected %T", operationName, zero)
	}

	return typedResult, nil
}

// ExecuteRepositoryQuery executes a repository query (read operation) and parses the result to the specified type T.
// This function encapsulates the common pattern of:
// 1. Creating a Query_Command with Repository_Command
// 2. Reading from the Raft node with timeout
// 3. Decoding the result
// 4. Parsing to the expected type
//
// Parameters:
//   - raftNode: The RaftNode instance to execute the query on
//   - ctx: The context for the operation
//   - cmd: The repository query command to execute
//   - timeout: The timeout duration for the operation
//   - logger: The logger instance for error logging
//   - operationName: A descriptive name for the operation (used in error messages)
//
// Returns:
//   - The parsed result of type T
//   - An error if any step fails
func ExecuteRepositoryQuery[T any](
	raftNode *RaftNode,
	ctx context.Context,
	cmd commands.Command,
	timeout time.Duration,
	logger zerolog.Logger,
	operationName string,
) (T, error) {
	var zero T

	// Create query command
	queryCommand := &general_command.Query_Command{
		Command: &general_command.Repository_Command{
			CMD: cmd,
		},
		Now: time.Now().UnixNano(),
	}

	// Only create a new timeout if the parent context doesn't already have a deadline.
	readCtx := ctx
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var readCancel context.CancelFunc
		readCtx, readCancel = context.WithTimeout(ctx, timeout)
		defer readCancel()
	}

	// Execute read operation
	result, err := raftNode.Read(readCtx, *queryCommand)
	if err != nil {
		// Handle special case for "nil pointer" errors which usually indicate "not found"
		if strings.Contains(err.Error(), "cannot encode nil pointer of type") {
			logger.Debug().Msgf("%s: entity not found", operationName)
			return zero, fmt.Errorf("%s: entity not found", operationName)
		}
		logger.Error().Err(err).Msgf("Failed to execute %s", operationName)
		return zero, fmt.Errorf("failed to execute %s: %w", operationName, err)
	}

	// Decode result
	buf := bytes.NewBuffer(result.([]byte))
	dec := gob.NewDecoder(buf)
	parsedResult := &commands.CommandResult{}
	if err := dec.Decode(parsedResult); err != nil {
		logger.Error().Err(err).Msgf("%s query returned unexpected result type", operationName)
		return zero, fmt.Errorf("%s query returned decode error: %w", operationName, err)
	}

	// Check for command-level errors
	if parsedResult.Error != "" {
		return zero, fmt.Errorf("%s failed: %s", operationName, parsedResult.Error)
	}

	// Check for nil results (entity not found)
	if parsedResult.Result == nil {
		return zero, nil
	}

	// Parse to expected type
	typedResult, ok := parsedResult.Result.(T)
	if !ok {
		return zero, fmt.Errorf("%s returned unexpected result type, expected %T", operationName, zero)
	}

	return typedResult, nil
}
