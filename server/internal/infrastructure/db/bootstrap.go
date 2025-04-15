package db

import (
	"deadalus-orch/shared/models"
	"fmt"
)

func BootstrapRootUser(kvStore KVStore, config map[string]string) error {
	root, err := GetDefaultRootUserRoot(kvStore)
	if err != nil {
		return fmt.Errorf("failed to get default root: %v", err)
	}

	if root == nil {
		username := config["default_root_user"]
		password := config["default_root_password"]

		if username == "" || password == "" {
			return fmt.Errorf("missing default root user/password")
		}

		fmt.Println("🧑‍💻 Creating default root user:", username)
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

	fmt.Println("✅ Root user exists:", root.Username)
	return nil
}
