package oauth_command

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"deadalus-orch/shared/models"
	"encoding/gob"
	"fmt"
	"time"
)

func init() {
	gob.Register(CreateOAuthAppCommand{})
}

// CreateOAuthAppCommand persists a new OAuth application.
// SECURITY: ClientSecret must be the plain-text secret.
// The repository will hash it with bcrypt before storage.
type CreateOAuthAppCommand struct {
	ID            string
	ClientID      string
	ClientSecret  string // plain text — will be bcrypt-hashed by the repo
	Name          string
	TenantID      string
	Description   string
	AllowedScopes []string
}

func (cmd *CreateOAuthAppCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	cr := &command.CommandResult{}
	repo, err := db.NewOAuthAppRepository(uow, &db.DeterministicIDGeneratorFactory{})
	if err != nil {
		cr.Error = err.Error()
		return *cr
	}

	if cmd.Name == "" || cmd.ClientID == "" || cmd.ClientSecret == "" || cmd.TenantID == "" {
		cr.Error = "name, clientID, clientSecret, and tenantID are required"
		return *cr
	}

	createdID, err := repo.CreateApp(models.CreateOAuthApp{
		ID:            cmd.ID,
		ClientID:      cmd.ClientID,
		ClientSecret:  cmd.ClientSecret,
		Name:          cmd.Name,
		TenantID:      cmd.TenantID,
		Description:   cmd.Description,
		AllowedScopes: cmd.AllowedScopes,
		Now:           now,
	})
	if err != nil {
		cr.Error = fmt.Sprintf("failed to create OAuth app: %v", err)
		return *cr
	}

	cr.Result = createdID
	return *cr
}
