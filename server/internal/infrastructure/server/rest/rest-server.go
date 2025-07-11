package rest_server

import (
	"context"
	"deadalus-orch/server/internal/infrastructure/server/common"
	"deadalus-orch/server/internal/pkg/config"
	"time"

	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

type RestServer struct {
	Config    *common.RestServerConfing
	GinEngine *gin.Engine
}
type zerologAdapter struct {
	logger zerolog.Logger
}

func (z zerologAdapter) Write(p []byte) (n int, err error) {
	z.logger.Info().Msg(string(p))
	return len(p), nil
}

// NewRestServer creates a new instance of RestServer.
func NewRestServer(config *common.RestServerConfing) *RestServer {
	if config.MasterNode == nil {
		config.Logger.Fatal().Msg("Admin API: Raft node cannot be nil")
	}

	gin.DefaultWriter = zerologAdapter{config.Logger}
	gin.DefaultErrorWriter = zerologAdapter{config.Logger}
	engine := gin.Default()

	server := &RestServer{
		Config:    config,
		GinEngine: engine,
	}

	server.setupRoutes(engine)
	return server
}

// Start starts the Gin HTTP server for the admin API.
func (s *RestServer) Start() error {
	if s.GinEngine == nil {
		return fmt.Errorf("admin API Gin engine not initialized")
	}

	listenAddr := fmt.Sprintf("%s:%d", config.GlobalConfiguration.AdminListenAddrHost, config.GlobalConfiguration.AdminListenAddrPort)
	s.Config.Logger.Info().Str("address", listenAddr).Msg("🚀 Starting Admin REST API server...")

	s.Config.Server = &http.Server{
		Addr:    listenAddr,
		Handler: s.GinEngine,
	}

	if err := s.Config.Server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		s.Config.Logger.Error().Err(err).Msg("❌ Failed to start Admin REST API server")
		return err
	}
	s.Config.Logger.Info().Msg("✅ Admin REST API server shut down gracefully.")
	return nil
}

// Shutdown gracefully shuts down the Gin HTTP server.
func (s *RestServer) Shutdown() error {
	ctx, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()
	s.Config.Logger.Info().Msg("🔌 Shutting down Admin REST API server...")
	if s.Config.Server != nil {
		return s.Config.Server.Shutdown(ctx)
	}
	return nil
}
