package rest_server

import (
	"context"
	"deadalus-orch/server/internal/infrastructure/server/rest/common"
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
func (s *RestServer) Start(listenAddr string) error {
	if s.GinEngine == nil {
		return fmt.Errorf("Admin API Gin engine not initialized")
	}
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
func (s *RestServer) Shutdown(ctx context.Context) error {
	s.Config.Logger.Info().Msg("🔌 Shutting down Admin REST API server...")
	if s.Config.Server != nil {
		return s.Config.Server.Shutdown(ctx)
	}
	return nil
}
