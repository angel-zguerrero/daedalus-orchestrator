package auth_command

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"encoding/gob"
	"time"
)

func init() {
	gob.Register(CheckDemoResolvedCommand{})
}

// CheckDemoResolvedCommand represents a command to check if demo mode has been permanently resolved.
type CheckDemoResolvedCommand struct {
}

func (cmd CheckDemoResolvedCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	val, err := uow.KVStore.Get(db.AdminFC, db.AdminFCSector, "demo_resolved", now)
	if err != nil {
		return command.CommandResult{
			Error: err.Error(),
		}
	}

	resolved := false
	if val != nil && string(val) == "true" {
		resolved = true
	}

	return command.CommandResult{
		Result: resolved,
	}
}
