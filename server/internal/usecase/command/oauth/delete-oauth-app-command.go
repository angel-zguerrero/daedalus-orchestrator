package oauth_command

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"encoding/gob"
	"fmt"
	"time"
)

func init() {
	gob.Register(DeleteOAuthAppCommand{})
}

// DeleteOAuthAppCommand permanently removes an OAuth application by its internal ID.
type DeleteOAuthAppCommand struct {
	ID string
}

func (cmd *DeleteOAuthAppCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	cr := &command.CommandResult{}
	repo, err := db.NewOAuthAppRepository(uow, &db.DeterministicIDGeneratorFactory{})
	if err != nil {
		cr.Error = err.Error()
		return *cr
	}
	deleted, err := repo.DeleteApp(cmd.ID, now)
	if err != nil {
		cr.Error = fmt.Sprintf("failed to delete OAuth app: %v", err)
		return *cr
	}
	cr.Result = deleted
	return *cr
}
