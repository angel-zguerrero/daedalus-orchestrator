package command

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"encoding/gob"
	"time"
)

func init() {
	gob.Register(CommandResult{})
}

type CommandResult struct {
	Error  string
	Result any
}

// Command defines the interface for all executable commands.
type Command interface {
	// Execute processes the command using the given UnitOfWork and timestamp.
	// The UnitOfWork must not be created internally by the command.
	// The now timestamp must be provided to the command.
	Execute(uow *db.UnitOfWork, now time.Time) CommandResult
}
