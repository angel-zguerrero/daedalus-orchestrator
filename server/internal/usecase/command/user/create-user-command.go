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
	gob.Register(CreateUserCommand{})
}

type CreateUserCommand struct {
	ID         string
	Username   string
	Email      string
	Password   string
	IsRootUser bool
}

func (cmd *CreateUserCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}
	idFactory := &db.DeterministicIDGeneratorFactory{}
	userRepo, err := db.NewUserRepository(uow, idFactory)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	if cmd.ID == "" {
		cmd.ID = idFactory.GenerateID()
	}

	if cmd.Username == "" || cmd.Password == "" {
		commandResult.Error = "username and password are required"
		return *commandResult
	}

	createdID, err := userRepo.CreateUser(models.CreateUser{
		ID:         cmd.ID,
		Username:   cmd.Username,
		Email:      cmd.Email,
		Password:   cmd.Password,
		IsRootUser: cmd.IsRootUser,
	}, now)

	if err != nil {
		commandResult.Error = fmt.Sprintf("failed to create user: %v", err)
		return *commandResult
	}

	commandResult.Result = createdID
	return *commandResult
}
