package command

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/pkg/config"
	"encoding/gob"
	"time"
)

func init() {
	gob.Register(BootstrapRootUserCommand{})
}

// BootstrapRootUserCommand represents a command to authenticate a user.
type BootstrapRootUserCommand struct {
}

func (cmd *BootstrapRootUserCommand) Execute(uow *db.UnitOfWork, now time.Time) ([]byte, error) {
	userRepo, err := db.NewUserRepository(uow, &db.DeterministicIDGeneratorFactory{}) // Passing nil for IDGeneratorFactory
	if err != nil {
		return nil, err
	}

	err = db.BootstrapRootUser(*userRepo, *config.GlobalConfiguration)

	return nil, err
}
