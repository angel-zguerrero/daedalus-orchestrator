package utils

import (
	"fmt"
	"os"

	"github.com/rs/zerolog/log"
)

func ValidateEnvVar() error {
	env := os.Getenv("ENV")
	if env == "" {
		env = "development"
		os.Setenv("ENV", env)
	}

	switch env {
	case "development", "staging", "production":

		log.Info().
			Str("ENV", env).
			Msg("Valid ENV value")
		return nil
	default:
		log.Error().
			Str("ENV", env).
			Msg("Invalid ENV value")
		return fmt.Errorf("invalid ENV value: %s. Must be one of: development, staging, production", env)
	}
}
