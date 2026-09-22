package oauth_command

import (
	"encoding/gob"
	"fmt"
	"time"

	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"deadalus-orch/shared/models"
)

func init() {
	gob.Register(CreateOAuthAppCommand{})
}

// CreateOAuthAppCommand creates a new OAuth app via Raft.
type CreateOAuthAppCommand struct {
	ID            string
	ClientID      string
	ClientSecret  string // plain-text secret; hashed inside Execute() by OAuthAppRepository
	Name          string
	TenantID      string
	Description   string
	AllowedScopes []string
}

func (cmd *CreateOAuthAppCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	res := &command.CommandResult{}
	repo, err := db.NewOAuthAppRepository(uow, &db.DefaultIDGeneratorFactory{})
	if err != nil {
		res.Error = fmt.Sprintf("failed to create OAuthApp repository: %v", err)
		return *res
	}

	input := models.CreateOAuthApp{
		ID:            cmd.ID,
		ClientID:      cmd.ClientID,
		ClientSecret:  cmd.ClientSecret,
		Name:          cmd.Name,
		TenantID:      cmd.TenantID,
		Description:   cmd.Description,
		AllowedScopes: cmd.AllowedScopes,
		Now:           now,
	}

	id, err := repo.CreateApp(input)
	if err != nil {
		res.Error = fmt.Sprintf("failed to create OAuth app: %v", err)
		return *res
	}

	res.Result = id
	return *res
}

var _ command.Command = &CreateOAuthAppCommand{}
