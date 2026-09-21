package oauth_command

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"deadalus-orch/shared/models"
	"encoding/gob"
	"time"
)

func init() {
	gob.Register(RegisterOAuthTokenCommand{})
}

type RegisterOAuthTokenCommand struct {
	Token models.OAuthToken
}

func (cmd *RegisterOAuthTokenCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	repo, err := db.NewOAuthTokenRepository(uow, idFactory)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	if err := repo.Save(&cmd.Token, now); err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	return *commandResult
}
