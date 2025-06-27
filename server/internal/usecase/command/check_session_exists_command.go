package command

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/pkg/utils"
	"encoding/gob"
	"errors"
	"time"
	// "fmt" // Will be needed if we add logging or more complex error handling
)

func init() {
	gob.Register(CheckSessionExistsCommand{})
}

// CheckSessionExistsCommand
// using a JWT token.
type CheckSessionExistsCommand struct {
	JWTToken string

	JWTKey []byte
}

func (cmd *CheckSessionExistsCommand) Execute(uow *db.UnitOfWork, now time.Time) ([]byte, error) {
	if cmd.JWTToken == "" {
		return nil, errors.New("JWTToken cannot be empty for CheckSessionExistsCommand")
	}
	if len(cmd.JWTKey) == 0 {
		return nil, errors.New("JWTKey cannot be empty for CheckSessionExistsCommand")
	}

	idFactory := &db.DefaultIDGeneratorFactory{}

	sessionRepo, err := db.NewSessionRepository(uow, idFactory, cmd.JWTKey)
	if err != nil {
		return nil, err
	}

	exists, err := sessionRepo.SessionExists(cmd.JWTToken, now)
	if err != nil {
		return nil, err
	}

	return utils.BoolToBytes(exists), nil
}
