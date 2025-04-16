package config

import (
	"bufio"
	"deadalus-orch/shared/constants"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
)

func LoadOrDefault(path string) (*Config, error) {
	config := &Config{
		Port: 50052,
	}
	env := os.Getenv("ENV")

	if path == "" {

		if env == "development" {
			path = "../daedalus.conf"
		} else {
			path = "/etc/daedalus/daedalus.conf"
		}
	} else {
		if _, err := os.Stat(path); err == nil {
			log.Info().
				Str("path", path).
				Msg("✅ Using config file")
			cfg, err := LoadConfig(path)
			config = cfg
			if err != nil {
				log.Error().
					Str("path", path).
					Err(err).
					Msg("⚠️ Failed to load config file")
				return nil, err
			}
		} else {
			log.Error().
				Str("path", path).
				Err(err).
				Msg("⚠️ Failed to load config file")
			return nil, err
		}
	}

	if os.Getenv("PORT") != "" {
		p, err := strconv.Atoi(os.Getenv("PORT"))
		if err != nil {
			return nil, err
		}
		config.Port = p
	}

	if config.DBname == "" {
		config.DBname = os.Getenv("DB_NAME")
	}
	if config.DBname == "" {
		config.DBname = "daedalus.db"
	}

	if config.DefaultRootUser == "" {
		config.DefaultRootUser = os.Getenv("DEFAULT_ROOT_USER")
	}

	if config.DefaultRootPassword == "" {
		config.DefaultRootPassword = os.Getenv("DEFAULT_ROOT_PASSWORD")
	}

	if config.DefaultRootUser == "" {
		config.DefaultRootUser = "admin"
	}

	if config.DefaultRootPassword == "" {
		config.DefaultRootPassword = "admin"
	}

	if config.Port == 0 {
		config.Port = 50052
	}

	return config, nil
}

func LoadConfig(path string) (*Config, error) {
	configMap := make(map[string]string)

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
			configMap[key] = value
		}
	}
	config, err := mapToConfig(configMap)
	if err != nil {
		return nil, err
	}
	return ConfigFromMapToConfig(*config), scanner.Err()
}

func mapToConfig(data map[string]string) (*ConfigFromMap, error) {
	cfg := &ConfigFromMap{}

	for k, v := range data {
		switch k {
		case constants.ConfigPortKey:
			p, err := strconv.Atoi(v)
			if err != nil {
				return nil, fmt.Errorf("error parsing port: %w", err)
			}
			cfg.port = p

		case constants.ConfigDBName:
			cfg.db_name = v

		case constants.ConfigDefaultRootUser:
			cfg.default_root_user = v

		case constants.ConfigDefaultRootPassword:
			cfg.default_root_password = v
		}
	}

	return cfg, nil
}
