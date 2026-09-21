package oauth_command

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"deadalus-orch/shared/models"
	"encoding/gob"
	"time"
)

func init() {
	gob.Register(CreateOAuthAppCommand{})
}

type CreateOAuthAppCommand struct {
	App models.OAuthApp
}

func (cmd *CreateOAuthAppCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	repo, err := db.NewOAuthAppRepository(uow, idFactory)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	cmd.App.CreatedAt = now.UnixMilli()
	cmd.App.UpdatedAt = now.UnixMilli()

	if err := repo.Save(&cmd.App, now); err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	return *commandResult
}
