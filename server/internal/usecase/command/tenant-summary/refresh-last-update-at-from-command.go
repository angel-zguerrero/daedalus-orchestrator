package tenant_summary_command

import (
	"bytes"
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"encoding/gob"
	"time"
)

func init() {
	gob.Register(RefreshLastUpdateAtFromCommand{})
}

type RefreshLastUpdateAtFromCommand struct {
	LastUpdateAtFrom time.Time
}

func (cmd *RefreshLastUpdateAtFromCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	// Get the KV store
	kvStore := uow.KVStore

	// Prepare the key
	key := "last-update-at-from"

	// Serialize the time using gob
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	err := encoder.Encode(cmd.LastUpdateAtFrom)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	// Store the serialized time
	err = kvStore.Put(db.AdminFC, db.AdminFCSector, key, buf.Bytes(), 0, now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	commandResult.Result = "last-update-at-from updated successfully"
	return *commandResult
}
