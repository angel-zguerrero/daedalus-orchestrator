package rest_server

import (
	"context"
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/infrastructure/server/common"
	"deadalus-orch/server/internal/pkg/config"
	"deadalus-orch/shared/constants"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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
		config.Logger.Fatal().Msg("Rest API: Raft node cannot be nil")
	}

	gin.DefaultWriter = zerologAdapter{config.Logger}
	gin.DefaultErrorWriter = zerologAdapter{config.Logger}
	engine := gin.Default()

	// Ruta para servir archivos estáticos
	staticPath := resolveAngularDistPath(config)
	engine.Static("/admin/", staticPath)

	engine.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/rest-api/") {
			c.JSON(http.StatusNotFound, gin.H{"error": "REST API route not found"})
			return
		}

		if strings.HasPrefix(c.Request.URL.Path, "/admin") {
			c.File(filepath.Join(staticPath, "index.html"))
			return
		}

		c.JSON(http.StatusNotFound, gin.H{"error": "Route not found"})
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
	}

	// Production / release binary:
	// 1st priority — look for a 'web-admin' folder next to the executable.
	//   This covers the release tarball layout:
	//     daedalus-orchestrator_vX.Y.Z_<os>_<arch>/
	//       daedalus-orchestrator   (binary)
	//       web-admin/              (Angular assets)
	if execPath, err := os.Executable(); err == nil {
		execDir := filepath.Dir(execPath)
		candidate := filepath.Join(execDir, "web-admin")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			restServerConfing.Logger.Info().
				Str("path", candidate).
				Msg("📂 Serving web-admin from directory next to binary")
			return candidate
		}
	}

	// 2nd priority — system-wide install path (/var/lib/daedalus/data).
	base_path, err := db.DefaultPathProvider{}.GetDatabasePath()
	if err != nil {
		restServerConfing.Logger.Fatal().
			Err(err).
			Str("package", "rest_server").
			Str("func", "resolveAngularDistPath").
			Msgf("❌ Getting Rest web application path")
	}
	restServerConfing.Logger.Info().
		Str("path", base_path).
		Msg("📂 Serving web-admin from system path")
	return base_path

}

// Start starts the Gin HTTP server for the Rest API and the Web Admin.
func (s *RestServer) Start() error {
	if s.GinEngine == nil {
		return fmt.Errorf("Rest API Gin engine not initialized")
	}

	listenAddr := fmt.Sprintf("%s:%d", config.GlobalConfiguration.RestListenAddrHost, config.GlobalConfiguration.RestListenAddrPort)
	s.Config.Logger.Info().Str("address", listenAddr).Msg("🚀 Starting Rest REST API server.")
	s.Config.Logger.Info().Str("address", listenAddr+"/admin/").Msg("🚀 Starting Web Admin.")

	s.Config.Server = &http.Server{
		Addr:    listenAddr,
		Handler: s.GinEngine,
	}

	if err := s.Config.Server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		s.Config.Logger.Error().Err(err).Msg("❌ Failed to start Rest REST API server")
		return err
	}
	s.Config.Logger.Info().Msg("✅ Rest REST API server shut down gracefully.")
	return nil
}

// Shutdown gracefully shuts down the Gin HTTP server.
func (s *RestServer) Shutdown() error {
	ctx, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()
	s.Config.Logger.Info().Msg("🔌 Shutting down Rest REST API server.")
	if s.Config.Server != nil {
		return s.Config.Server.Shutdown(ctx)
	}
	return nil
}
