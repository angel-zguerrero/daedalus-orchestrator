package utils

import (
	"fmt"
	"os"

	"deadalus-orch/shared/constants"

	"github.com/rs/zerolog/log"
)

// ValidateEnvVar checks critical environment variables for the application.
// It ensures that `ENV` (or its equivalent `constants.EnvVarEnvKey`) is set to one of
// "development", "staging", or "production". If not set, it defaults to "development".
// It also ensures that `OTEL_ACTIVED` (or `constants.EnvVarOtelActived`) is set to "true" or "false".
// If not set, it defaults to "true" (enabling OpenTelemetry).
// For any invalid value, it logs an error and returns an error.
// Valid or defaulted values are logged at Info level.
//
// Returns:
//   - An error if any of the validated environment variables have an invalid value.
//   - nil if all validated environment variables are valid or successfully defaulted.
func ValidateEnvVar() error {
	// Validate ENV (constants.EnvVarEnvKey)
	env := os.Getenv(constants.EnvVarEnvKey)
	if env == "" {
		env = string(constants.DEVELOPMENT) // Default to "development"
		os.Setenv(constants.EnvVarEnvKey, env)
		log.Info().Str(constants.EnvVarEnvKey, env).Msg("Defaulted to development environment")
	}

	switch constants.Env(env) { // Cast to constants.Env for direct comparison with typed constants
	case constants.DEVELOPMENT, constants.STAGING, constants.PRODUCTION:
		log.Info().
			Str(constants.EnvVarEnvKey, env).
			Msg("Environment validated")
	default:
		log.Error().
			Str(constants.EnvVarEnvKey, env).
			Msg("Invalid ENV value")
		return fmt.Errorf("invalid %s value: %s. Must be one of: %s, %s, %s",
			constants.EnvVarEnvKey, env, constants.DEVELOPMENT, constants.STAGING, constants.PRODUCTION)
	}

	// Validate OTEL_ACTIVED (constants.EnvVarOtelActived)
	otelActived := os.Getenv(constants.EnvVarOtelActived)
	if otelActived == "" {
		otelActived = string(constants.OTEL_ACTIVE_TRUE) // Default to "true" (OTEL enabled)
		os.Setenv(constants.EnvVarOtelActived, otelActived)
		log.Info().Str(constants.EnvVarOtelActived, otelActived).Msg("Defaulted to OTEL_ACTIVED=true")
	}

	switch otelActived { // Cast to constants.OtelActive
	case constants.OTEL_ACTIVE_TRUE, constants.OTEL_ACTIVE_FALSE:
		log.Info().
			Str(constants.EnvVarOtelActived, otelActived).
			Msg("OTEL_ACTIVED validated")
	default:
		log.Error().
			Str(constants.EnvVarOtelActived, otelActived).
			Msg("Invalid OTEL_ACTIVED value")
		return fmt.Errorf("invalid %s value: %s. Must be one of: %s, %s",
			constants.EnvVarOtelActived, otelActived, constants.OTEL_ACTIVE_TRUE, constants.OTEL_ACTIVE_FALSE)
	}

	return nil
}
