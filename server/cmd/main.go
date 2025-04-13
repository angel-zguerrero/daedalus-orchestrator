package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"

	"path/filepath"

	"os/user"

	db_access "deadalus-orch/server/internal/infrastructure/db"
	configUtils "deadalus-orch/server/internal/pkg/config-file-utils"
	constants "deadalus-orch/shared/constants"
	models "deadalus-orch/shared/models"

	"google.golang.org/grpc"
)

func isValidPort(p int) bool {
	return p >= 1024 && p <= 65535
}

func ensureDirExists(path string) error {
	dir := filepath.Dir(path)
	return os.MkdirAll(dir, 0755)
}

func getDatabasePath() (string, error) {
	env := os.Getenv("ENV")
	if env == "" {
		env = "development"
	}

	if env == "development" {
		usr, err := user.Current()
		if err != nil {
			return "", fmt.Errorf("could not get current user: %v", err)
		}
		return filepath.Join(usr.HomeDir, ".daedalus", "data"), nil
	}

	path := "/var/lib/daedalus/data"
	if err := os.MkdirAll(path, 0755); err != nil {
		return "", fmt.Errorf("could not create data directory: %v", err)
	}

	return filepath.Join(path), nil
}

func main() {
	flagConfig := flag.String("config", "", "Path to the daedalus.conf configuration file (optional)")
	flag.Parse()

	const defaultPort = 50052
	port := defaultPort
	config := map[string]string{}

	configPath, err := configUtils.FindConfigFile(*flagConfig)
	if err == nil {
		fmt.Println("✅ Configuration file found at:", configPath)
		config, err = configUtils.LoadConfig(configPath)
		if err != nil {
			fmt.Printf("❌ Failed to read configuration file: %v\n", err)
			log.Fatal(err)
		}
	} else {
		fmt.Println("⚠️ No daedalus.conf file found. Continuing without file-based configuration.")
	}

	if val, ok := config["port"]; ok {
		p, err := strconv.Atoi(val)
		if err != nil || !isValidPort(p) {
			fmt.Printf("❌ Invalid 'port' in config: '%s'. Using default: %d\n", val, defaultPort)
		} else {
			port = p
		}
	} else if val, ok := os.LookupEnv("PORT"); ok {
		p, err := strconv.Atoi(val)
		if err != nil || !isValidPort(p) {
			fmt.Printf("❌ Invalid PORT env var: '%s'. Using default: %d\n", val, defaultPort)
		} else {
			port = p
		}
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("❌ Failed to listen: %v", err)
	}
	defer lis.Close()

	dbName := config["db_name"]
	if dbName == "" {
		dbName = os.Getenv("DB_NAME")
	}
	if dbName == "" {
		dbName = "daedalus.db"
	}

	dbBasePath, err := getDatabasePath()
	if err != nil {
		fmt.Printf("❌ Failed generating db base path: %v\n", err)
		log.Fatal(err)
	}

	dbPath := filepath.Join(dbBasePath, dbName)

	if err := ensureDirExists(dbPath); err != nil {
		log.Fatalf("❌ Could not create directory for database: %v", err)
	}

	db, err := db_access.OpenDB(dbPath)
	if err != nil {
		fmt.Printf("❌ Failed to open database: %v\n", err)
		log.Fatal(err)
	}
	defer db.Close()

	defaultRootUserRoot, err := db_access.GetDefaultRootUserRoot(db)
	if err != nil {
		fmt.Printf("❌ Failed to read %v: %v\n", constants.DefaultRootUserRootKey, err)
		log.Fatal(err)
	}

	if defaultRootUserRoot == nil {
		userName := config["default_root_user"]
		if userName == "" {
			userName = os.Getenv("DEFAULT_ROOT_USER")
		}

		password := config["default_root_password"]
		if password == "" {
			password = os.Getenv("DEFAULT_ROOT_PASSWORD")
		}

		if userName == "" || password == "" {
			fmt.Println("❌ You must provide a default_root_user and default_root_password. Use daedalus.conf or environment variables.")
			return
		}

		fmt.Println("🧑‍💻 Default root user:", userName)
		fmt.Println("🔐 Root password: [HIDDEN]")

		err = db_access.PutDefaultRootUserRoot(db, models.CreateUser{userName, "noemail@daedalus.com", password})
		if err != nil {
			fmt.Printf("❌ Failed to create root user: %v\n", err)
			log.Fatal(err)
		}
	} else {
		existingUser, err := db_access.GetUser(db, defaultRootUserRoot.Username)
		if err != nil {
			fmt.Printf("❌ Failed to fetch user '%s': %v\n", defaultRootUserRoot.Username, err)
			log.Fatal(err)
		}

		if existingUser == nil {
			fmt.Printf("⚠️ User '%s' not found in system, creating it...\n", defaultRootUserRoot.Username)
			err = db_access.PutUser(db, *defaultRootUserRoot)
			if err != nil {
				fmt.Printf("❌ Failed to recreate default root user: %v\n", err)
				log.Fatal(err)
			}
			fmt.Println("✅ Default root user created:", defaultRootUserRoot.Username)
		} else {
			fmt.Println("✅ Default root user found:", defaultRootUserRoot.Username)
		}
	}

	s := grpc.NewServer()
	defer s.GracefulStop()

	log.Printf("🚀 Server listening at :%d\n", port)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("❌ Failed to serve: %v", err)
	}
}
