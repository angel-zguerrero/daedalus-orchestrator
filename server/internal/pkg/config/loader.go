package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func LoadOrDefault(path string) (map[string]string, error) {
	config := map[string]string{}

	env := os.Getenv("ENV")
	if env == "" {
		env = "development"
	}
	fmt.Println("*******env")
	fmt.Println(env)
	switch env {
	case "development", "staging", "production":
		// valid
	default:
		return nil, fmt.Errorf("invalid ENV value: %s. Must be one of: development, staging, production", env)
	}

	if path == "" {

		if env == "development" {
			path = "../daedalus.conf"
		} else {
			path = "/etc/daedalus/daedalus.conf"
		}
	} else {
		if _, err := os.Stat(path); err == nil {
			fmt.Println("✅ Using config file:", path)
			cfg, err := LoadConfig(path)
			config = cfg
			if err != nil {
				return nil, err
			}
			fmt.Println("⚠️ Failed to load config file:", err)
		} else {
			return nil, err
		}
	}

	if config["db_name"] == "" {
		config["db_name"] = os.Getenv("DB_NAME")
	}
	if config["db_name"] == "" {
		config["db_name"] = "daedalus.db"
	}

	if config["default_root_user"] == "" {
		config["default_root_user"] = os.Getenv("DEFAULT_ROOT_USER")
	}

	if config["default_root_password"] == "" {
		config["default_root_password"] = os.Getenv("DEFAULT_ROOT_PASSWORD")
	}

	if config["default_root_user"] == "" {
		config["default_root_user"] = "admin"
	}

	if config["default_root_password"] == "" {
		config["default_root_password"] = "admin"
	}

	return config, nil
}

func LoadConfig(path string) (map[string]string, error) {
	config := make(map[string]string)

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if key != "" && value != "" {
			config[key] = value
		}
	}

	return config, scanner.Err()
}
