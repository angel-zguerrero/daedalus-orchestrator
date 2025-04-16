package utils

import (
	"fmt"
	"os"

	"deadalus-orch/shared/constants"

	"github.com/rs/zerolog/log"
)

func ValidateEnvVar() error {
	env := os.Getenv(constants.EnvVarEnvKey)
	if env == "" {
		env = "development"
		os.Setenv(constants.EnvVarEnvKey, env)
	}

	switch env {
	case "development", "staging", "production":

		log.Info().
			Str(constants.EnvVarEnvKey, env).
			Msg("Valid ENV value")
		return nil
	default:
		log.Error().
			Str(constants.EnvVarEnvKey, env).
			Msg("Invalid ENV value")
		return fmt.Errorf("invalid ENV value: %s. Must be one of: development, staging, production", env)
	}
}
