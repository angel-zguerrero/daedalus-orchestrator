package command

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"encoding/gob"
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

func (cmd *RegisterSessionCommand) Execute(uow *db.UnitOfWork, now time.Time) CommandResult {
	commandResult := &CommandResult{}
	if cmd.JWTToken == "" {
		commandResult.Error = "JWTToken cannot be empty for RegisterSessionCommand"
		return *commandResult
	}
	if len(cmd.JWTKey) == 0 {
		commandResult.Error = "JWTKey cannot be empty for RegisterSessionCommand"
		return *commandResult
	}

	idFactory := &db.DeterministicIDGeneratorFactory{}

	sessionRepo, err := db.NewSessionRepository(uow, idFactory, cmd.JWTKey)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	err = sessionRepo.RegisterSession(cmd.JWTToken, now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	return *commandResult
}
