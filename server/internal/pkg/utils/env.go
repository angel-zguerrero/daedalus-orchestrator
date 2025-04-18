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
		env = string(constants.DEVELOPMENT)
		os.Setenv(constants.EnvVarEnvKey, env)
	}

	switch env {
	case string(constants.DEVELOPMENT), string(constants.STAGING), string(constants.PRODUCTION):
		log.Info().
			Str(constants.EnvVarEnvKey, env).
			Send()
	default:
		log.Error().
			Str(constants.EnvVarEnvKey, env).
			Msg("Invalid ENV value")
		return fmt.Errorf("invalid ENV value: %s. Must be one of: development, staging, production", env)
	}

	otlActived := os.Getenv(constants.EnvVarOtelActived)
	if otlActived == "" {
		otlActived = string(constants.OTEL_ACTIVE_TRUE)
		os.Setenv(constants.EnvVarOtelActived, otlActived)
	}

	switch otlActived {
	case string(constants.OTEL_ACTIVE_TRUE), string(constants.OTEL_ACTIVE_FALSE):

		log.Info().
			Str(constants.EnvVarOtelActived, otlActived).
			Send()
	default:
		log.Error().
			Str(constants.EnvVarOtelActived, otlActived).
			Msg("Invalid OtelActived value")
		return fmt.Errorf("invalid OtelActived value: %s. Must be one of: true, false", otlActived)
	}

	return nil
}
