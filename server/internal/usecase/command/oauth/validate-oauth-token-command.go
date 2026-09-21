package oauth_command

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"deadalus-orch/shared/models"
	"encoding/gob"
	"time"
)

func init() {
	gob.Register(ValidateOAuthTokenCommand{})
}

type ValidateOAuthTokenCommand struct {
	TokenHash string
}

func (cmd *ValidateOAuthTokenCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	repo, err := db.NewOAuthTokenRepository(uow, idFactory)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	token, err := repo.FindByTokenHash(cmd.TokenHash, now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	var result *models.OAuthToken
	if token != nil {
		result = token
	}

	commandResult.Result = result
	return *commandResult
}
