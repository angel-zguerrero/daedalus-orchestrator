package auth_command

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"encoding/gob"
	"time"
)

func init() {
	gob.Register(BootstrapRootUserCommand{})
}

// BootstrapRootUserCommand represents a command to authenticate a user.
type BootstrapRootUserCommand struct {
	ID           string
	Username     string
	Email        string
	PasswordHash string
}

func (cmd *BootstrapRootUserCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}
	userRepo, err := db.NewUserRepository(uow, &db.DeterministicIDGeneratorFactory{}) // Passing nil for IDGeneratorFactory
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	err = db.BootstrapRootUser(*userRepo, cmd.ID, cmd.Username, cmd.Email, cmd.PasswordHash, now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}
	return *commandResult
}
