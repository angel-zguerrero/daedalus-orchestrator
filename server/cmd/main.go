package main

import (
	"deadalus-orch/server/internal/app"
	"deadalus-orch/server/internal/pkg/config"
	"deadalus-orch/server/internal/pkg/utils"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog/log"
)

// main is the entry point of the application.
// It initializes the application by:
// - Setting up a channel to listen for system signals (SIGINT, SIGTERM) for graceful shutdown.
// - Validating essential environment variables.
// - Loading the default application configuration.
// - Starting the application using app.Run().
// The function will block until a system signal is received.
func main() {
	// Help flag handling and flag parsing is now done in config.LoadDefaultConfiguration()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	err := utils.ValidateEnvVar()
	if err != nil {
		log.Fatal().
			Err(err).
			Str("package", "app").
			Str("func", "Run").
			Msgf("❌ Failed validation of ENV var")
	}

	err = config.LoadDefaultConfiguration()
	if err != nil {
		log.Fatal().
			Err(err).
			Str("package", "app").
			Str("func", "Run").
			Msgf("❌ Failed loading configuration")
	}

	app.Run()
	<-stop
}
