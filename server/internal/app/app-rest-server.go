package app

import (
	"deadalus-orch/server/internal/infrastructure/server/common"
	rest_server "deadalus-orch/server/internal/infrastructure/server/rest"
	"deadalus-orch/server/internal/pkg/config"
	"time"

	"github.com/rs/zerolog/log"
)

func (app *Application) StartRestAPI() {
	app.ApiLock.Lock()
	defer app.ApiLock.Unlock()
	if app.RestAPI == nil {
		jwtSecret := config.GlobalConfiguration.RestAPIJWTSecret
		jwtDuration := time.Hour * time.Duration(config.GlobalConfiguration.RestAPIJWTExpirationHours)

		log.Info().Msg("Rest API JWT Expiration: " + jwtDuration.String())

		// Pass the global log.Logger instance, which is configured in app.Run()
		serverConfig := &common.ServerConfing{
			MasterNode:            app.MasterNode,
			TenantNodes:           app.TenantNodes,
			TenantNodesDictionary: app.TenantNodesDictionary,
			JwtKey:                []byte(jwtSecret),
			JwtDuration:           jwtDuration,
			Logger:                log.Logger,
		}
		app.RestAPI = rest_server.NewRestServer(serverConfig)

		go func() {
			if err := app.RestAPI.Start(); err != nil {
				log.Error().Err(err).Msg("❌ Rest API server failed to start or shut down with error")
			}
		}()

	} else if app.RestAPI != nil {
		log.Info().Msg("Rest API already running or was previously started.")
	}
}

func (app *Application) CloseRestAPI() {
	app.ApiLock.Lock()
	defer app.ApiLock.Unlock()
	if app.RestAPI != nil {
		log.Info().Msg("Closing Rest app.")

		if err := app.RestAPI.Shutdown(); err != nil {
			log.Error().Err(err).Msg("❌ Error during Rest API shutdown")
		} else {
			log.Info().Msg("✅ Rest API closed successfully.")
		}
		app.RestAPI = nil
	}
}
