package command

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"encoding/gob"
	"time"
)

func init() {
	gob.Register(LoginCommand{})
}

// LoginCommand represents a command to authenticate a user.
type LoginCommand struct {
	Email    string
	Password string
}

// Execute attempts to log in a user with the provided email and password.
// It uses the UserRepository to validate credentials.
// The IDGeneratorFactory is passed as nil to NewUserRepository as login itself doesn't generate new IDs.
func (cmd *LoginCommand) Execute(uow *db.UnitOfWork, now time.Time) (any, error) {
	// It's mentioned that ID generation should use a deterministic generator if needed.
	// For login, we are primarily validating credentials. NewUserRepository takes an IDGeneratorFactory.
	// If the underlying operations within UserRepository for login were to create audit logs
	// or session tokens requiring deterministic IDs, that factory would need to be appropriately configured
	// and passed down. For now, assuming login itself doesn't create entities requiring new IDs directly
	// within this command's scope, we pass nil. If deeper operations require it, this might need adjustment.
	userRepo, err := db.NewUserRepository(uow, nil) // Passing nil for IDGeneratorFactory
	if err != nil {
		return nil, err
	}

	// The existing UserRepository.Login method uses time.Now() internally.
	// This ideally should be refactored to accept `now time.Time` if strict determinism via the command's `now` is required.
	// However, sticking to the current task, we call it as is.
	// The impact is that freshness of data read by Login will depend on its internal time.Now(), not the command's `now`.
	loggedIn, err := userRepo.Login(cmd.Email, cmd.Password)
	if err != nil {
		return nil, err
	}

	return loggedIn, nil
}
