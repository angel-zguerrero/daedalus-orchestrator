package auth_command

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"encoding/gob"
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

func (cmd *CheckSessionExistsCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}
	if cmd.JWTToken == "" {
		commandResult.Error = "JWTToken cannot be empty for CheckSessionExistsCommand"
		return *commandResult
	}
	if len(cmd.JWTKey) == 0 {

		commandResult.Error = "JWTKey cannot be empty for CheckSessionExistsCommand"
		return *commandResult
	}

	idFactory := &db.DeterministicIDGeneratorFactory{}

	sessionRepo, err := db.NewSessionRepository(uow, idFactory, cmd.JWTKey)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	exists, err := sessionRepo.SessionExists(cmd.JWTToken, now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}
	commandResult.Result = exists
	return *commandResult
}
