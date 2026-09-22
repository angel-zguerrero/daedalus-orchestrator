package oauth_command

import (
	"encoding/gob"
	"fmt"
	"time"

	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
)

func init() {
	gob.Register(DeleteOAuthAppCommand{})
}

// DeleteOAuthAppCommand deletes an OAuthApp by ID.
type DeleteOAuthAppCommand struct {
	ID string
}

func (cmd *DeleteOAuthAppCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	res := &command.CommandResult{}
	repo, err := db.NewOAuthAppRepository(uow, &db.DefaultIDGeneratorFactory{})
	if err != nil {
		res.Error = fmt.Sprintf("failed to create OAuthApp repository: %v", err)
		return *res
	}

	deleted, err := repo.DeleteApp(cmd.ID, now)
	if err != nil {
		res.Error = fmt.Sprintf("failed to delete OAuth app: %v", err)
		return *res
	}

	res.Result = deleted
	return *res
}

var _ command.Command = &DeleteOAuthAppCommand{}
