package rest_api_admin

import (
	"context"
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// RestAdminAPI handles the administrative REST API endpoints.
type RestAdminAPI struct {
	MasterNode            *dragonboat.RaftNode
	TenantNodes           []*dragonboat.RaftNode
	TenantNodesDictionary map[string]*dragonboat.RaftNode
	TenantNodesLock       sync.Mutex
	ginEngine             *gin.Engine
	jwtKey                []byte
	jwtDuration           time.Duration
	server                *http.Server
	logger                zerolog.Logger
}
type zerologAdapter struct {
	logger zerolog.Logger
}

func (z zerologAdapter) Write(p []byte) (n int, err error) {
	z.logger.Info().Msg(string(p))
	return len(p), nil
}

// NewRestAdminAPI creates a new instance of RestAdminAPI.
func NewRestAdminAPI(MasterNode *dragonboat.RaftNode, TenantNodes []*dragonboat.RaftNode, TenantNodesDictionary map[string]*dragonboat.RaftNode, jwtSecretKey string, jwtAuthDuration time.Duration, logger zerolog.Logger) *RestAdminAPI {
	if MasterNode == nil {
		logger.Fatal().Msg("Admin API: Raft node cannot be nil")
	}
	if jwtSecretKey == "" {
		logger.Warn().Msg("Admin API: JWT secret key is empty. This is insecure.")
	}
	gin.DefaultWriter = zerologAdapter{logger}
	gin.DefaultErrorWriter = zerologAdapter{logger}
	engine := gin.Default()

	api := &RestAdminAPI{
		MasterNode:            MasterNode,
		TenantNodes:           TenantNodes,
		TenantNodesDictionary: TenantNodesDictionary,
		ginEngine:             engine,
		jwtKey:                []byte(jwtSecretKey),
		jwtDuration:           jwtAuthDuration,
		logger:                logger,
	}

	setupRoutes(engine, api)

	return api
}

// Start starts the Gin HTTP server for the admin API.
func (api *RestAdminAPI) Start(listenAddr string) error {
	if api.ginEngine == nil {
		return fmt.Errorf("Admin API Gin engine not initialized")
	}
	api.logger.Info().Str("address", listenAddr).Msg("🚀 Starting Admin REST API server...")

	api.server = &http.Server{
		Addr:    listenAddr,
		Handler: api.ginEngine,
	}

	if err := api.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		api.logger.Error().Err(err).Msg("❌ Failed to start Admin REST API server")
		return err
	}
	api.logger.Info().Msg("✅ Admin REST API server shut down gracefully.")
	return nil
}

// Shutdown gracefully shuts down the Gin HTTP server.
func (api *RestAdminAPI) Shutdown(ctx context.Context) error {
	api.logger.Info().Msg("🔌 Shutting down Admin REST API server...")
	if api.server != nil {
		return api.server.Shutdown(ctx)
	}
	return nil
}

func (api *RestAdminAPI) SetTenantNode(shardID int, tenantId string) *dragonboat.RaftNode {
	var tenant *dragonboat.RaftNode

	api.TenantNodesLock.Lock()
	for i := range api.TenantNodes {
		if api.TenantNodes[i].ShardID == uint64(shardID) {
			tenant = api.TenantNodes[i]
			break
		}
	}
	api.TenantNodesLock.Unlock()

	api.TenantNodesLock.Lock()
	api.TenantNodesDictionary[tenantId] = tenant
	api.TenantNodesLock.Unlock()
	return tenant
}
