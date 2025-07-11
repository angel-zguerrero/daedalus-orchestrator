package rest_server

import (
	"context"
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/infrastructure/server/common"
	"deadalus-orch/server/internal/pkg/config"
	"deadalus-orch/shared/constants"
	"fmt"
	"net/http"
	"path/filepath"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

type RestServer struct {
	Config    *common.ServerConfing
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
func NewRestServer(config *common.ServerConfing) *RestServer {
	if config.MasterNode == nil {
		config.Logger.Fatal().Msg("Admin API: Raft node cannot be nil")
	}

	gin.DefaultWriter = zerologAdapter{config.Logger}
	gin.DefaultErrorWriter = zerologAdapter{config.Logger}
	engine := gin.Default()

	// Ruta para servir archivos estáticos
	staticPath := resolveAngularDistPath(config)
	engine.Static("/admin/", staticPath)

	engine.NoRoute(func(c *gin.Context) {
		c.File(filepath.Join(staticPath, "index.html"))
	})

	server := &RestServer{
		Config:    config,
		GinEngine: engine,
	}

	server.setupRoutes(engine)
	return server
}

func resolveAngularDistPath(restServerConfing *common.ServerConfing) string {

	if config.GlobalConfiguration.Env == string(constants.DEVELOPMENT) {
		_, filename, _, ok := runtime.Caller(0)
		if !ok {
			restServerConfing.Logger.Fatal().
				Str("package", "rest_server").
				Str("func", "resolveAngularDistPath").
				Msgf("❌ Cannot resolve current file path")
		}

		baseDir := filepath.Dir(filename)
		var staticPath string
		staticPath = filepath.Join(baseDir, "../../../../../web-admin/dist/daedalus-web-admin/browser")
		absPath, err := filepath.Abs(staticPath)
		if err != nil {
			restServerConfing.Logger.Fatal().
				Err(err).
				Str("package", "rest_server").
				Str("func", "resolveAngularDistPath").
				Msgf("❌ Cannot resolve absolute path to Angular dist")
		}

		return absPath
	} else {
		base_path, err := db.DefaultPathProvider{}.GetDatabasePath()
		if err != nil {
			restServerConfing.Logger.Fatal().
				Err(err).
				Str("package", "rest_server").
				Str("func", "resolveAngularDistPath").
				Msgf("❌ Getting admin web application path")
		}
		return base_path
	}

}

// Start starts the Gin HTTP server for the admin API.
func (s *RestServer) Start() error {
	if s.GinEngine == nil {
		return fmt.Errorf("admin API Gin engine not initialized")
	}

	listenAddr := fmt.Sprintf("%s:%d", config.GlobalConfiguration.AdminListenAddrHost, config.GlobalConfiguration.AdminListenAddrPort)
	s.Config.Logger.Info().Str("address", listenAddr).Msg("🚀 Starting Admin REST API server.")

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
	s.Config.Logger.Info().Msg("🔌 Shutting down Admin REST API server.")
	if s.Config.Server != nil {
		return s.Config.Server.Shutdown(ctx)
	}
	return nil
}
