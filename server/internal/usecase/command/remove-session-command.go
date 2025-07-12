package command

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"encoding/gob"
	"time"
	// "fmt" // Will be needed if we add logging or more complex error handling
)

func init() {
	gob.Register(RemoveSessionCommand{})
}

// RemoveSessionCommand represents a command to register a user session
// using a JWT token.
type RemoveSessionCommand struct {
	JWTToken string

	JWTKey []byte
}

func (cmd *RemoveSessionCommand) Execute(uow *db.UnitOfWork, now time.Time) CommandResult {
	commandResult := &CommandResult{}
	if cmd.JWTToken == "" {
		commandResult.Error = "JWTToken cannot be empty for RemoveSessionCommand"
		return *commandResult
	}
	if len(cmd.JWTKey) == 0 {
		commandResult.Error = "JWTKey cannot be empty for RemoveSessionCommand"
		return *commandResult
	}

	idFactory := &db.DeterministicIDGeneratorFactory{}

	sessionRepo, err := db.NewSessionRepository(uow, idFactory, cmd.JWTKey)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	err = sessionRepo.RemoveSession(cmd.JWTToken, now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	commandResult.Result = true
	return *commandResult
}
