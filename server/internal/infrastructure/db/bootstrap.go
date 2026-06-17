package db

import (
	"deadalus-orch/shared/models"
	"fmt"

	"time"


	"github.com/rs/zerolog/log"
)

// BootstrapRootUser ensures the default root user exists in the KVStore.
// If the root user (as defined by `config.DefaultRootUser` and `config.DefaultRootPassword`)
// does not exist, it will be created.
//
// Parameters:
//   - kvStore: The KVStore implementation where user data is stored.
//   - config: The application configuration containing the default root user credentials.
//
// Returns:
//   - An error if any operation fails (e.g., accessing the KVStore, missing credentials),
//     or nil if the root user exists or is successfully created.
func BootstrapRootUser(userRepository UserRepository, id string, username string, email string, passwordHash string, now time.Time) error {
	root, err := userRepository.GetUserRoot(now)
	if err != nil {
		return fmt.Errorf("failed to get default root: %v", err)
	}

	if root == nil {
		if username == "" || passwordHash == "" {
			log.Info().Msg("ℹ️ Default root user credentials not provided in env. Initial setup required via UI.")
			return nil
		}

		log.Info().
			Str("username", username).
			Msg("🧑‍💻 Creating default root user")

		_, err = userRepository.CreateUser(models.CreateUser{
			ID:         id,
			Username:   username,
			Email:      email,
			Password:   passwordHash,
			IsRootUser: true,
		}, now)
		return err
	}

	log.Info().
		Str("Username", root.Username).
		Msg("✅ Root user exists")

	return nil
}
