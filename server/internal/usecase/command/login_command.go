package command

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/pkg/utils"
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
func (cmd *LoginCommand) Execute(uow *db.UnitOfWork, now time.Time) ([]byte, error) {
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
	// Login method in repository returns (bool, error)
	// For this integration, we need the username if login is successful.
	// We'll need to modify userRepo.Login or add a new method that returns the user details or username.

	// Let's assume userRepo.Login is modified or we use GetUserByUsername after a successful Login check.
	// For now, let's proceed by first calling Login, then GetUserByUsername if successful.
	// This is not the most efficient (two lookups) but avoids changing UserRepository method signatures
	// without deeper analysis of its usage.

	loggedIn, err := userRepo.Login(cmd.Email, cmd.Password)
	if err != nil {
		return nil, NewInfrastructureErrorWrap(err, "error during login process")
	}

	if !loggedIn {
		// Return nil bytes and no error to indicate login failure (e.g., wrong credentials)
		// Or, a specific domain error could be returned. For now, nil, nil indicates "not logged in".
		// The original command returned BoolToBytes(false), nil.
		// To convey "not logged in" without an error, and distinguish from "logged in with empty username",
		// we can return an empty byte array.
		return []byte{}, nil
	}

	// If loggedIn is true, fetch the user to get the username.
	// LoginCommand uses Email as identifier, which could be email or username.
	// UserRepository.Login tries by email then username. We need to know which one matched
	// or fetch the user by one of these identifiers.
	// This is a bit tricky as LoginCommand.Email can be either.
	// A cleaner way would be for userRepo.Login to return the *models.User on success.

	// Assuming cmd.Email was the actual username or an email that can be used to fetch the user.
	// Let's try to get user by cmd.Email (which Login internally checks as email then username)
	// This part is a bit of a hack due to the current structure.
	// Ideally, userRepo.Login() would return the user object or username.
	user, err := userRepo.GetUserByUsername(cmd.Email) // Try with Email field as username
	if err != nil || user == nil {
		// If not found by username, and login was by email, this won't work.
		// This highlights a need for userRepo.Login to return more info.
		// For the purpose of this task, we'll assume cmd.Email *is* the username for simplicity here,
		// or that GetUserByUsername would be smart enough.
		// A more robust solution would be to modify userRepo.Login to return the user object.
		//
		// If login was successful, a user *must* exist. If GetUserByUsername fails here,
		// it implies cmd.Email was an email, not a username.
		// This part of the flow is problematic with current repo signatures.
		//
		// Let's assume for now, to proceed, that if login succeeded, cmd.Email is a username.
		// This is a strong assumption.
		// A better temporary approach: if login is successful, the command's purpose is fulfilled for auth.
		// The JWT generation (which needs username) should happen in the API layer, which can
		// re-fetch the user if needed, or LoginCommand could be enhanced.

		// Given the constraints, LoginCommand will return a success indicator.
		// The API layer, upon receiving success, will then generate JWT (it has username from request or can re-fetch)
		// and then issue RegisterSessionCommand.
		// So, LoginCommand returns BoolToBytes(true) still.
		// The integration is *sequential* in the API layer.

		// Reverting to original return for LoginCommand due to complexities of getting username reliably here
		// without altering UserRepository.Login's return signature, which is outside current scope of "LoginCommand integration".
		// The API handler that calls LoginCommand will be responsible for:
		// 1. Getting result of LoginCommand.
		// 2. If successful, generating JWT (it would have the username, presumably from the initial request payload).
		// 3. Issuing RegisterSessionCommand.

		// Therefore, LoginCommand itself doesn't change its return value for now.
		// The "integration" means the API orchestrates this.
		return utils.BoolToBytes(loggedIn), nil // loggedIn is true here
	}

	// If we had the user object:
	// return []byte(user.Username), nil
	// But due to the ambiguity of cmd.Email and the return type of userRepo.Login,
	// we stick to the original return type. The API layer will handle the rest.
	return utils.BoolToBytes(loggedIn), nil // Return true, successful login
}
