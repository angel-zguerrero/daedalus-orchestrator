package user_command

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"deadalus-orch/shared/models"
	"encoding/gob"
	"time"
)

func init() {
	gob.Register(GetUserByIdCommand{})
	gob.Register(models.User{})
}

type GetUserByIdCommand struct {
	ID string
}

func (cmd *GetUserByIdCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}
	userRepo, err := db.NewUserRepository(uow, &db.DeterministicIDGeneratorFactory{})
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	user, err := userRepo.GetUserByID(cmd.ID, now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	if user != nil {
		safeUser := *user
		safeUser.PasswordHash = ""
		commandResult.Result = safeUser
	} else {
		commandResult.Result = models.User{}
	}
	return *commandResult
}
