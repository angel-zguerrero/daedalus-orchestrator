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
	gob.Register(GetOAuthAppByClientIDCommand{})
	gob.Register(GetOAuthAppByIDCommand{})
	gob.Register(GetOAuthAppsByTenantCommand{})
	gob.Register(models.OAuthApp{})
}

// GetOAuthAppByClientIDCommand retrieves an OAuthApp by its public ClientID.
type GetOAuthAppByClientIDCommand struct {
	ClientID string
}

func (cmd *GetOAuthAppByClientIDCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	cr := &command.CommandResult{}
	repo, err := db.NewOAuthAppRepository(uow, &db.DeterministicIDGeneratorFactory{})
	if err != nil {
		cr.Error = err.Error()
		return *cr
	}
	app, err := repo.GetAppByClientID(cmd.ClientID, now)
	if err != nil {
		cr.Error = fmt.Sprintf("failed to get OAuth app by clientID: %v", err)
		return *cr
	}
	if app != nil {
		cr.Result = *app
	}
	return *cr
}

// GetOAuthAppByIDCommand retrieves an OAuthApp by its internal primary-key ID.
type GetOAuthAppByIDCommand struct {
	ID string
}

func (cmd *GetOAuthAppByIDCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	cr := &command.CommandResult{}
	repo, err := db.NewOAuthAppRepository(uow, &db.DeterministicIDGeneratorFactory{})
	if err != nil {
		cr.Error = err.Error()
		return *cr
	}
	app, err := repo.GetAppByID(cmd.ID, now)
	if err != nil {
		cr.Error = fmt.Sprintf("failed to get OAuth app by ID: %v", err)
		return *cr
	}
	if app != nil {
		cr.Result = *app
	}
	return *cr
}

// GetOAuthAppsByTenantCommand retrieves all apps belonging to a tenant.
type GetOAuthAppsByTenantCommand struct {
	TenantID string
}

func (cmd *GetOAuthAppsByTenantCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	cr := &command.CommandResult{}
	repo, err := db.NewOAuthAppRepository(uow, &db.DeterministicIDGeneratorFactory{})
	if err != nil {
		cr.Error = err.Error()
		return *cr
	}
	apps, err := repo.GetAppsByTenant(cmd.TenantID, now)
	if err != nil {
		cr.Error = fmt.Sprintf("failed to list OAuth apps for tenant: %v", err)
		return *cr
	}
	if apps == nil {
		apps = []models.OAuthApp{}
	}
	cr.Result = apps
	return *cr
}
