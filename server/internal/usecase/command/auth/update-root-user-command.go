package auth_command

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"encoding/gob"
	"fmt"
	"time"
)

func init() {
	gob.Register(UpdateRootUserCommand{})
}

// UpdateRootUserCommand represents a command to update root user credentials.
type UpdateRootUserCommand struct {
	Username string
	Password string
}

func (cmd *UpdateRootUserCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
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

	if root == nil {
		commandResult.Error = "root user does not exist"
		return *commandResult
	}

	if cmd.Username != "" {
		root.Username = cmd.Username
	}

	var newPassword *string
	if cmd.Password != "" {
		newPassword = &cmd.Password
	}

	success, err := userRepo.UpdateUser(*root, newPassword)
	if err != nil {
		commandResult.Error = fmt.Sprintf("failed to update root user: %v", err)
		return *commandResult
	}

	if !success {
		commandResult.Error = "failed to update root user"
		return *commandResult
	}

	commandResult.Result = "Root user updated successfully"
	return *commandResult
}
