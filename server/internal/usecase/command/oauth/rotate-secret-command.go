package oauth_command

import (
	"encoding/gob"
	"fmt"
	"time"

	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
)

func init() {
	gob.Register(RotateOAuthSecretCommand{})
	gob.Register(RotateOAuthSecretResult{})
}

// RotateOAuthSecretCommand generates and persists a new secret for an OAuthApp.
type RotateOAuthSecretCommand struct {
	AppID string
}

type RotateOAuthSecretResult struct {
	PlainSecret string
}

func (cmd *RotateOAuthSecretCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	res := &command.CommandResult{}
	repo, err := db.NewOAuthAppRepository(uow, &db.DefaultIDGeneratorFactory{})
	if err != nil {
		res.Error = fmt.Sprintf("failed to create OAuthApp repository: %v", err)
		return *res
	}

	plainSecret, err := repo.RotateSecret(cmd.AppID, now)
	if err != nil {
		res.Error = fmt.Sprintf("failed to rotate OAuth secret: %v", err)
		return *res
	}

	res.Result = RotateOAuthSecretResult{PlainSecret: plainSecret}
	return *res
}

var _ command.Command = &RotateOAuthSecretCommand{}
