package auth_command

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"deadalus-orch/shared/models"
	"encoding/gob"
	"fmt"
	"time"
)

func init() {
	gob.Register(SetupRootUserCommand{})
}

// SetupRootUserCommand represents a command to initialize root user credentials.
type SetupRootUserCommand struct {
	Username string
	Password string
}

func (cmd *SetupRootUserCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
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

	if root != nil {
		commandResult.Error = "root user already exists"
		return *commandResult
	}

	if cmd.Username == "" || cmd.Password == "" {
		commandResult.Error = "username and password are required"
		return *commandResult
	}

	_, err = userRepo.CreateUser(models.CreateUser{
		ID:         "94adc9e9e1194d39aaf7f9cfc392ee48",
		Username:   cmd.Username,
		Email:      "noemail@daedalus.com",
		Password:   cmd.Password,
		IsRootUser: true,
	})
	if err != nil {
		commandResult.Error = fmt.Sprintf("failed to create root user: %v", err)
		return *commandResult
	}

	commandResult.Result = "Root user created successfully"
	return *commandResult
}
