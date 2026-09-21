package oauth_command

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"encoding/gob"
	"fmt"
	"time"
)

func init() {
	gob.Register(RotateOAuthSecretCommand{})
	gob.Register(RotateOAuthSecretResult{})
}

// RotateOAuthSecretResult carries both the updated app and the one-time plain-text secret.
// SECURITY: plainSecret must be returned to the caller ONCE in the HTTP response and then discarded.
type RotateOAuthSecretResult struct {
	PlainSecret string
}

// RotateOAuthSecretCommand generates a new client secret, hashes it, and persists the update.
// The Result field of CommandResult will hold a RotateOAuthSecretResult with the new plain-text secret.
type RotateOAuthSecretCommand struct {
	AppID string
}

func (cmd *RotateOAuthSecretCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	cr := &command.CommandResult{}
	repo, err := db.NewOAuthAppRepository(uow, &db.DeterministicIDGeneratorFactory{})
	if err != nil {
		cr.Error = err.Error()
		return *cr
	}

	newSecret, err := repo.RotateSecret(cmd.AppID, now)
	if err != nil {
		cr.Error = fmt.Sprintf("failed to rotate secret: %v", err)
		return *cr
	}

	cr.Result = RotateOAuthSecretResult{PlainSecret: newSecret}
	return *cr
}
