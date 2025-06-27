package command

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"encoding/gob"
	"errors"
	"time"
	// "fmt" // Will be needed if we add logging or more complex error handling
)

func init() {
	gob.Register(RegisterSessionCommand{})
}

// RegisterSessionCommand represents a command to register a user session
// using a JWT token.
type RegisterSessionCommand struct {
	JWTToken string

	JWTKey []byte
}

func (cmd *RegisterSessionCommand) Execute(uow *db.UnitOfWork, now time.Time) ([]byte, error) {
	if cmd.JWTToken == "" {
		return nil, errors.New("JWTToken cannot be empty for RegisterSessionCommand")
	}
	if len(cmd.JWTKey) == 0 {
		return nil, errors.New("JWTKey cannot be empty for RegisterSessionCommand")
	}

	idFactory := &db.DefaultIDGeneratorFactory{}

	sessionRepo, err := db.NewSessionRepository(uow, idFactory, cmd.JWTKey)
	if err != nil {
		return nil, err
	}

	err = sessionRepo.RegisterSession(cmd.JWTToken, now)
	if err != nil {
		return nil, err
	}

	return nil, nil
}
