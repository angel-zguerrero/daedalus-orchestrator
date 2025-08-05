package general_command

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"encoding/gob"
	"fmt"
	"time"
)

func init() {
	gob.Register(DeleteColumnFamilySectorCommand{})
}

// DeleteColumnFamilySectorCommand represents a command to delete all keys from a specific ColumnFamilySector.
type DeleteColumnFamilySectorCommand struct {
	ColumnFamilySector string
	ColumnFamily       string
}

func (cmd *DeleteColumnFamilySectorCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}
	kvStore := uow.KVStore

	// Counter to track how many keys were deleted
	deletedCount := 0
	pageSize := 1000 // Process keys in batches of 1000

	// If no specific column family is specified, we need to handle all column families
	if cmd.ColumnFamily == "" {
		commandResult.Error = "ColumnFamily must be specified when using SearchByPatternPaginatedKV"
		return *commandResult
	}

	// Use SearchByPatternPaginatedKV to iterate through all keys in the specified column family and sector
	cursor := ""
	pattern := "*" // Get all keys in the sector

	for {
		// Search for keys matching the pattern in the specified column family and sector
		keyValuePairs, nextCursor, err := kvStore.SearchByPatternPaginatedKV(
			cmd.ColumnFamily,
			cmd.ColumnFamilySector,
			pattern,
			cursor,
			pageSize,
			now,
		)

		if err != nil {
			commandResult.Error = fmt.Sprintf("Error searching keys: %s", err.Error())
			return *commandResult
		}

		// Delete all keys found in this batch
		for _, kvp := range keyValuePairs {
			deleteErr := kvStore.Delete(cmd.ColumnFamily, cmd.ColumnFamilySector, kvp.Key, now)
			if deleteErr != nil {
				commandResult.Error = fmt.Sprintf("Error deleting key %s: %s", kvp.Key, deleteErr.Error())
				return *commandResult
			}
			deletedCount++
		}

		// If there's no next cursor, we've processed all keys
		if nextCursor == "" {
			break
		}
		cursor = nextCursor
	}

	commandResult.Result = map[string]interface{}{
		"success":      true,
		"deletedCount": deletedCount,
		"message":      fmt.Sprintf("Successfully deleted %d keys from ColumnFamilySector: %s in ColumnFamily: %s", deletedCount, cmd.ColumnFamilySector, cmd.ColumnFamily),
	}

	return *commandResult
}
