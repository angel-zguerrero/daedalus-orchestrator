package db

import (
	"deadalus-orch/shared/models"
	"fmt"

	"deadalus-orch/server/internal/pkg/config"

	"github.com/rs/zerolog/log"
)

func BootstrapRootUser(kvStore KVStore, config config.Config) error {
	root, err := GetDefaultRootUserRoot(kvStore)
	if err != nil {
		return fmt.Errorf("failed to get default root: %v", err)
	}

	if root == nil {
		username := config.DefaultRootUser
		password := config.DefaultRootPassword

		if username == "" || password == "" {
			return fmt.Errorf("missing default root user/password")
		}

		log.Info().
			Str("username", username).
			Msg("🧑‍💻 Creating default root user")

		return PutDefaultRootUserRoot(kvStore, models.CreateUser{
			Username: username,
			Email:    "noemail@daedalus.com",
			Password: password,
		})
	}

	existing, err := GetUser(kvStore, root.Username)
	if err != nil {
		return err
	}

	if existing == nil {
		return PutUser(kvStore, *root)
	}

	log.Info().
		Str("Username", root.Username).
		Msg("✅ Root user exists")

	return nil
}
