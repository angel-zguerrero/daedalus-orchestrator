package oauth_command

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"encoding/gob"
	"time"
)

func init() {
	gob.Register(RotateOAuthSecretCommand{})
}

type RotateOAuthSecretCommand struct {
	AppID         string
	NewSecretHash string
}

func (cmd *RotateOAuthSecretCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	repo, err := db.NewOAuthAppRepository(uow, idFactory)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	app, err := repo.FindByID(cmd.AppID, now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	if app == nil {
		commandResult.Error = "OAuthApp not found"
		return *commandResult
	}

	app.ClientSecretHash = cmd.NewSecretHash
	app.UpdatedAt = now.UnixMilli()

	if err := repo.Save(app, now); err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	return *commandResult
}
