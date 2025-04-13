package config_file_utils

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func FindConfigFile(forcePath string) (string, error) {
	var paths []string
	if forcePath != "" {
		paths = []string{
			forcePath,
		}
	} else {
		env := os.Getenv("ENV")
		if env == "" {
			env = "development"
		}

		if env == "development" {
			paths = []string{
				"/etc/daedalus/daedalus.conf",
				"../daedalus.conf",
			}
		} else {
			paths = []string{
				"/etc/daedalus/daedalus.conf",
			}
		}
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("daedalus.conf file not found in any known path")
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
		config[key] = value
	}

	return config, scanner.Err()
}
