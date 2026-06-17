package user_command

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"encoding/gob"
	"fmt"
	"time"
)

func init() {
	gob.Register(DeleteUserCommand{})
}

type DeleteUserCommand struct {
	Username string
}

func (cmd *DeleteUserCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}
	userRepo, err := db.NewUserRepository(uow, &db.DeterministicIDGeneratorFactory{})
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	if cmd.Username == "" {
		commandResult.Error = "username is required"
		return *commandResult
	}

	success, err := userRepo.DeleteUser(cmd.Username, now)
	if err != nil {
		commandResult.Error = fmt.Sprintf("failed to delete user: %v", err)
		return *commandResult
	}

	commandResult.Result = success
	return *commandResult
}
