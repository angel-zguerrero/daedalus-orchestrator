package app

import (
	"deadalus-orch/server/internal/infrastructure/server/common"
	grpc_server "deadalus-orch/server/internal/infrastructure/server/grpc"
	"deadalus-orch/server/internal/pkg/config"
	"time"

	"github.com/rs/zerolog/log"
)

func (app *Application) StartGrpcAPI() {
	app.GrpcLock.Lock()
	defer app.GrpcLock.Unlock()
	if app.GrpcAPI == nil {
		jwtSecret := config.GlobalConfiguration.RestAPIJWTSecret
		jwtDuration := time.Hour * time.Duration(config.GlobalConfiguration.RestAPIJWTExpirationHours)

		log.Info().Msg("grpc API JWT Expiration: " + jwtDuration.String())

		// Pass the global log.Logger instance, which is configured in app.Run()
		serverConfig := &common.ServerConfing{
			MasterNode:            app.MasterNode,
			TenantNodes:           app.TenantNodes,
			TenantNodesDictionary: app.TenantNodesDictionary,
			JwtKey:                []byte(jwtSecret),
			JwtDuration:           jwtDuration,
			Logger:                log.Logger,
		}
		grpcAPI, _ := grpc_server.NewGrpcServer(serverConfig)
		app.GrpcAPI = grpcAPI

		go func() {
			if err := app.GrpcAPI.Start(); err != nil {
				log.Error().Err(err).Msg("❌ grpc API server failed to start or shut down with error")
			}
		}()

	} else if app.GrpcAPI != nil {
		log.Info().Msg("grpc API already running or was previously started.")
	}
}

func (app *Application) CloseGrpcAPI() {
	app.GrpcLock.Lock()
	defer app.GrpcLock.Unlock()
	if app.GrpcAPI != nil {
		log.Info().Msg("Closing grpc api .")

		app.GrpcAPI.Shutdown()
		log.Info().Msg("✅ grpc API closed successfully.")
		app.GrpcAPI = nil
	}
}
