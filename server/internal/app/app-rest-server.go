package app

import (
	"deadalus-orch/server/internal/infrastructure/server/common"
	rest_server "deadalus-orch/server/internal/infrastructure/server/rest"
	"deadalus-orch/server/internal/pkg/config"
	"time"

	"github.com/rs/zerolog/log"
)

func (app *Application) StartAdminAPI() {
	app.ApiLock.Lock()
	defer app.ApiLock.Unlock()
	if app.RestAPI == nil {
		jwtSecret := config.GlobalConfiguration.AdminAPIJWTSecret
		jwtDuration := time.Hour * time.Duration(config.GlobalConfiguration.AdminAPIJWTExpirationHours)

		log.Info().Msg("Admin API JWT Expiration: " + jwtDuration.String())

		// Pass the global log.Logger instance, which is configured in app.Run()
		serverConfig := &common.RestServerConfing{
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
				log.Error().Err(err).Msg("❌ Admin API server failed to start or shut down with error")
			}
		}()

	} else if app.RestAPI != nil {
		log.Info().Msg("Admin API already running or was previously started.")
	}
}

func (app *Application) CloseAdminAPI() {
	app.ApiLock.Lock()
	defer app.ApiLock.Unlock()
	if app.RestAPI != nil {
		log.Info().Msg("Closing Admin app.")

		if err := app.RestAPI.Shutdown(); err != nil {
			log.Error().Err(err).Msg("❌ Error during Admin API shutdown")
		} else {
			log.Info().Msg("✅ Admin API closed successfully.")
		}
		app.RestAPI = nil
	}
}
