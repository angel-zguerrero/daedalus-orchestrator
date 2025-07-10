package command

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"encoding/gob"
	"time"
)

func init() {
	gob.Register(LoginCommand{})
}

// LoginCommand represents a command to authenticate a user.
type LoginCommand struct {
	UsernameOrEmail string
	Password        string
}

func (cmd *LoginCommand) Execute(uow *db.UnitOfWork, now time.Time) CommandResult {
	commandResult := &CommandResult{}
	idFactory := &db.DeterministicIDGeneratorFactory{}
	userRepo, err := db.NewUserRepository(uow, idFactory) // Passing nil for IDGeneratorFactory
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	loggedIn, err := userRepo.Login(cmd.UsernameOrEmail, cmd.Password)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}
	commandResult.Result = loggedIn

	return *commandResult
}
