package tenant_summary_command

import (
	"bytes"
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"encoding/gob"
	"time"
)

func init() {
	gob.Register(GetLastUpdateAtFromCommand{})
}

type GetLastUpdateAtFromCommand struct {
	// No fields needed for this query command
}

func (cmd *GetLastUpdateAtFromCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	// Get the KV store
	kvStore := uow.KVStore

	// Prepare the key
	key := "last-update-at-from"

	// Get the value from the specified Column Family and Sector
	valueBytes, err := kvStore.Get(db.AdminFC, db.AdminFCSector, key, now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	if valueBytes == nil {
		// Key doesn't exist, return zero time
		commandResult.Result = time.Time{}
		return *commandResult
	}

	// Deserialize the time using gob
	buf := bytes.NewBuffer(valueBytes)
	decoder := gob.NewDecoder(buf)
	var lastUpdateAt time.Time
	err = decoder.Decode(&lastUpdateAt)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	commandResult.Result = lastUpdateAt
	return *commandResult
}
