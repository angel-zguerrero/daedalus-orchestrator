package user_command

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"deadalus-orch/shared/models"
	"encoding/gob"
	"time"
)

func init() {
	gob.Register(GetUsersCommand{})
	gob.Register(db.FindResult[models.User]{})
}

type GetUsersCommand struct {
	Filter string
	Cursor string
	Limit  int
}

func (cmd *GetUsersCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}
	userRepo, err := db.NewUserRepository(uow, &db.DeterministicIDGeneratorFactory{})
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	limit := cmd.Limit
	if limit <= 0 {
		limit = 50
	}

	result, err := userRepo.GetUsers(cmd.Filter, cmd.Cursor, limit, now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	if result != nil {
		commandResult.Result = *result
	} else {
		commandResult.Result = db.FindResult[models.User]{Entities: []models.User{}}
	}
	return *commandResult
}
