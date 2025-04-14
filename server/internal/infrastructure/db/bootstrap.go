package db

import (
	"deadalus-orch/shared/models"
	"fmt"

	"github.com/linxGnu/grocksdb"
)

func BootstrapRootUser(db *grocksdb.DB, config map[string]string) error {
	root, err := GetDefaultRootUserRoot(db)
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
		return PutDefaultRootUserRoot(db, models.CreateUser{
			Username: username,
			Email:    "noemail@daedalus.com",
			Password: password,
		})
	}

	existing, err := GetUser(db, root.Username)
	if err != nil {
		return err
	}

	if existing == nil {
		return PutUser(db, *root)
	}

	fmt.Println("✅ Root user exists:", root.Username)
	return nil
}
