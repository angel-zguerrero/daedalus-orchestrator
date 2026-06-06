package db

import (
	"deadalus-orch/shared/models"
	"fmt"

	"deadalus-orch/server/internal/pkg/config"

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
func BootstrapRootUser(userRepository UserRepository, config config.Config) error {
	root, err := userRepository.GetUserRoot()
	if err != nil {
		return fmt.Errorf("failed to get default root: %v", err)
	}

	if root == nil {
		username := config.DefaultRootUser
		password := config.DefaultRootPassword

		if username == "" || password == "" {
			log.Info().Msg("ℹ️ Default root user credentials not provided in env. Initial setup required via UI.")
			return nil
		}

		log.Info().
			Str("username", username).
			Msg("🧑‍💻 Creating default root user")

		_, err = userRepository.CreateUser(models.CreateUser{
			ID:         "94adc9e9e1194d39aaf7f9cfc392ee48",
			Username:   username,
			Email:      "noemail@daedalus.com",
			Password:   password,
			IsRootUser: true,
		})
		return err
	}

	log.Info().
		Str("Username", root.Username).
		Msg("✅ Root user exists")

	return nil
}
