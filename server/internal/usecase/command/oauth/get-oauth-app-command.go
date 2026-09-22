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
	gob.Register(GetOAuthAppByClientIDCommand{})
	gob.Register(GetOAuthAppByIDCommand{})
	gob.Register(GetOAuthAppsByTenantCommand{})
}

// GetOAuthAppByClientIDCommand retrieves an OAuthApp by ClientID.
type GetOAuthAppByClientIDCommand struct {
	ClientID string
}

func (cmd *GetOAuthAppByClientIDCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	res := &command.CommandResult{}
	repo, err := db.NewOAuthAppRepository(uow, &db.DefaultIDGeneratorFactory{})
	if err != nil {
		res.Error = fmt.Sprintf("failed to create OAuthApp repository: %v", err)
		return *res
	}

	app, err := repo.GetAppByClientID(cmd.ClientID, now)
	if err != nil {
		res.Error = err.Error()
		return *res
	}
	if app == nil {
		res.Result = models.OAuthApp{}
		return *res
	}
	res.Result = *app
	return *res
}

// GetOAuthAppByIDCommand retrieves an OAuthApp by internal ID.
type GetOAuthAppByIDCommand struct {
	ID string
}

func (cmd *GetOAuthAppByIDCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	res := &command.CommandResult{}
	repo, err := db.NewOAuthAppRepository(uow, &db.DefaultIDGeneratorFactory{})
	if err != nil {
		res.Error = fmt.Sprintf("failed to create OAuthApp repository: %v", err)
		return *res
	}

	app, err := repo.GetAppByID(cmd.ID, now)
	if err != nil {
		res.Error = err.Error()
		return *res
	}
	if app == nil {
		res.Result = models.OAuthApp{}
		return *res
	}
	res.Result = *app
	return *res
}

// GetOAuthAppsByTenantCommand retrieves all OAuthApps for a tenant.
type GetOAuthAppsByTenantCommand struct {
	TenantID string
}

func (cmd *GetOAuthAppsByTenantCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	res := &command.CommandResult{}
	repo, err := db.NewOAuthAppRepository(uow, &db.DefaultIDGeneratorFactory{})
	if err != nil {
		res.Error = fmt.Sprintf("failed to create OAuthApp repository: %v", err)
		return *res
	}

	apps, err := repo.GetAppsByTenant(cmd.TenantID, now)
	if err != nil {
		res.Error = err.Error()
		return *res
	}
	res.Result = apps
	return *res
}

var _ command.Command = &GetOAuthAppByClientIDCommand{}
var _ command.Command = &GetOAuthAppByIDCommand{}
var _ command.Command = &GetOAuthAppsByTenantCommand{}
