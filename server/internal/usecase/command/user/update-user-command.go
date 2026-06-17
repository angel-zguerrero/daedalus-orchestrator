package user_command

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"deadalus-orch/shared/models"
	"encoding/gob"
	"fmt"
	"time"
)

func init() {
	gob.Register(UpdateUserCommand{})
}

type UpdateUserCommand struct {
	ID       string
	Username string
	Email    string
	Password string
}

func (cmd *UpdateUserCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}
	userRepo, err := db.NewUserRepository(uow, &db.DeterministicIDGeneratorFactory{})
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	if cmd.ID == "" {
		commandResult.Error = "id is required"
		return *commandResult
	}

	success, err := userRepo.UpdateUser(models.UpdateUser{
		ID:       cmd.ID,
		Username: cmd.Username,
		Email:    cmd.Email,
		Password: cmd.Password,
	}, now)

	if err != nil {
		commandResult.Error = fmt.Sprintf("failed to update user: %v", err)
		return *commandResult
	}

	commandResult.Result = success
	return *commandResult
}
