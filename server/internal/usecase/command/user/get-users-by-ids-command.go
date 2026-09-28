package user_command

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"deadalus-orch/shared/models"
	"encoding/gob"
	"time"
)

func init() {
	gob.Register(GetUsersByIdsCommand{})
	gob.Register([]models.User{})
}

type GetUsersByIdsCommand struct {
	IDs []string
}

func (cmd *GetUsersByIdsCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}
	userRepo, err := db.NewUserRepository(uow, &db.DeterministicIDGeneratorFactory{})
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	users, err := userRepo.GetUsersByIDs(cmd.IDs, now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	safeUsers := make([]models.User, 0, len(users))
	for _, u := range users {
		if u != nil {
			safeUser := *u
			safeUser.PasswordHash = ""
			safeUsers = append(safeUsers, safeUser)
		}
	}

	commandResult.Result = safeUsers
	return *commandResult
}
