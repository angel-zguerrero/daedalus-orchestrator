package command

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/pkg/utils"
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

func (cmd *LoginCommand) Execute(uow *db.UnitOfWork, now time.Time) ([]byte, error) {

	userRepo, err := db.NewUserRepository(uow, nil) // Passing nil for IDGeneratorFactory
	if err != nil {
		return nil, err
	}

	loggedIn, err := userRepo.Login(cmd.UsernameOrEmail, cmd.Password)
	if err != nil {
		return nil, err
	}

	if !loggedIn {

		return []byte{}, nil
	}

	user, err := userRepo.GetUserByUsername(cmd.UsernameOrEmail) // Try with Email field as username
	if err != nil || user == nil {

		return utils.BoolToBytes(loggedIn), nil
	}

	return utils.BoolToBytes(loggedIn), nil
}
