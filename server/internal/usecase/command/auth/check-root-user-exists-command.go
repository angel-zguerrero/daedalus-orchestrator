package auth_command

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"encoding/gob"
	"time"
)

func init() {
	gob.Register(CheckRootUserExistsCommand{})
}

// CheckRootUserExistsCommand represents a command to verify if a root user exists.
type CheckRootUserExistsCommand struct {
}

func (cmd *CheckRootUserExistsCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}
	userRepo, err := db.NewUserRepository(uow, &db.DeterministicIDGeneratorFactory{})
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	root, err := userRepo.GetUserRoot()
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	commandResult.Result = root != nil
	return *commandResult
}
