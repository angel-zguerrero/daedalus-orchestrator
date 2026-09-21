package oauth_command

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"deadalus-orch/shared/models"
	"encoding/gob"
	"time"
)

func init() {
	gob.Register(FindOAuthAppCommand{})
}

type FindOAuthAppCommand struct {
	ClientID string
}

func (cmd *FindOAuthAppCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	repo, err := db.NewOAuthAppRepository(uow, idFactory)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	app, err := repo.FindByClientID(cmd.ClientID, now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	var result *models.OAuthApp
	if app != nil {
		result = app
	}

	commandResult.Result = result
	return *commandResult
}
