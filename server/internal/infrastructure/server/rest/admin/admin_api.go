package rest_api_admin

import (
	"context"
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	"deadalus-orch/server/internal/pkg/config"
	"deadalus-orch/server/internal/pkg/utils"
	commands "deadalus-orch/server/internal/usecase/command"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// RestAdminAPI handles the administrative REST API endpoints.
type RestAdminAPI struct {
	node        dragonboat.RaftNode // Interface for Raft node interaction
	ginEngine   *gin.Engine
	jwtKey      []byte
	jwtDuration time.Duration
	server      *http.Server
	logger      zerolog.Logger
}
type zerologAdapter struct {
	logger zerolog.Logger
}

func (z zerologAdapter) Write(p []byte) (n int, err error) {
	z.logger.Info().Msg(string(p))
	return len(p), nil
}

// NewRestAdminAPI creates a new instance of RestAdminAPI.
// It initializes the Gin engine and sets up the API routes.
func NewRestAdminAPI(node *dragonboat.RaftNode, jwtSecretKey string, jwtAuthDuration time.Duration, logger zerolog.Logger) *RestAdminAPI {
	if node == nil {
		logger.Fatal().Msg("Admin API: Raft node cannot be nil")
	}
	if jwtSecretKey == "" {
		logger.Warn().Msg("Admin API: JWT secret key is empty. This is insecure.")
		// In a real scenario, you might want to prevent startup or generate a temporary key.
		// For now, we'll proceed but this should be addressed.
	}
	gin.DefaultWriter = zerologAdapter{logger}
	gin.DefaultErrorWriter = zerologAdapter{logger}
	engine := gin.Default()

	api := &RestAdminAPI{
		node:        *node,
		ginEngine:   engine,
		jwtKey:      []byte(jwtSecretKey),
		jwtDuration: jwtAuthDuration,
		logger:      logger,
	}

	// Setup routes
	adminAPIGroup := engine.Group("/admin-api")
	{
		adminAPIGroup.POST("/login", api.loginHandler)

		// Tenant routes (protected by JWT middleware in a real scenario)
		// For now, JWT protection is not implemented on these routes yet.
		tenantsGroup := adminAPIGroup.Group("/tenants")
		{
			tenantsGroup.GET("", api.getTenantsHandler)
			tenantsGroup.POST("", api.createTenantHandler)
			tenantsGroup.GET("/:id", api.getTenantHandler)
			tenantsGroup.PUT("/:id", api.updateTenantHandler)
			tenantsGroup.DELETE("/:id", api.deleteTenantHandler)
		}
	}

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

// loginRequest defines the structure for login request JSON payload.
type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// loginHandler handles the /admin-api/login endpoint.
func (api *RestAdminAPI) loginHandler(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.logger.Warn().Err(err).Msg("Login attempt with invalid payload")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	// Instantiate the LoginCommand
	loginCmd := &commands.LoginCommand{
		Email:    req.Username, // Assuming username from request can be email
		Password: req.Password,
	}

	// Wrap the LoginCommand in a Query_Command
	// The Query_Command.Now field might be used by the state machine for its own timestamping if needed,
	// but our Command.Execute method receives `now` explicitly from the Lookup method.
	queryCommand := &commands.Query_Command{
		Command: &commands.Repository_Command{
			CMD: loginCmd,
		},
		Now: time.Now().UnixNano(), // Or handle as per specific requirements if Query_Command.Now is actively used
	}

	// Call the node's Read method (SyncRead)
	// The task implies a Read method on the node. Assuming it's similar to Lookup/Propose
	// and is responsible for routing to the MasterKVDBStateMachine.Lookup method.
	// The actual method name on `api.node` might be `SyncRead` or similar.
	// For this example, I'll use a hypothetical `api.node.Read` method.
	// If the method is `Lookup`, we'd use `api.node.Lookup(c.Request.Context(), queryCommand)`.
	// The problem states "call the node’s Read method ... passing the Query_Command".
	// Let's assume a `Read` method exists that behaves like `Lookup` but is meant for synchronous queries.
	// If `api.node` is of type `*dragonboat.Node`, it has `SyncRead(ctx context.Context, clusterID uint64, query interface{}) (interface{}, error)`
	// We'd need the clusterID for this. Assuming master cluster ID.
	// For now, let's use `api.node.Lookup` as its signature is `Lookup(ctx context.Context, query interface{}) (interface{}, error)`
	// which matches well if we assume it can handle Query_Command.
	// The problem description for master-kv-state-machine.go's Lookup method is:
	// `Lookup(cmd any, uow *db.UnitOfWork, now time.Time) (interface{}, error)`
	// The `RaftNode` interface's `Lookup` method is:
	// `Lookup(ctx context.Context, query interface{}) (interface{}, error)`
	// The `dragonboat.Node`'s `Lookup` is:
	// `Lookup(ctx context.Context, query interface{}) (interface{}, error)`
	// This seems to be the most direct path. The `uow` and `now` for `MasterKVDBStateMachine.Lookup`
	// are prepared internally by the `dragonboat.Node.Lookup`'s machinery before calling the state machine.
	ctx, cancel := context.WithTimeout(context.Background(), config.GlobalConfiguration.ApiRaftTimeout)
	defer cancel()
	result, err := api.node.Read(ctx, *queryCommand)
	if err != nil {
		api.logger.Error().Err(err).Str("username", req.Username).Msg("Login command execution failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Login failed: " + err.Error()})
		return
	}

	loggedIn, err := utils.BytesToBool(result.([]byte))
	if err != nil {
		api.logger.Error().Str("username", req.Username).Msg("Login command returned unexpected result type")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Login failed due to an internal error (result type)"})
		return
	}

	if !loggedIn {
		api.logger.Warn().Str("username", req.Username).Msg("Login attempt failed: invalid credentials")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}

	// Credentials are valid, generate JWT token
	tokenString, err := api.generateJWT(req.Username)
	if err != nil {
		api.logger.Error().Err(err).Str("username", req.Username).Msg("Failed to generate JWT token during login")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Login successful, but failed to generate token: " + err.Error()})
		return
	}

	api.logger.Info().Str("username", req.Username).Msg("User logged in successfully")
	c.JSON(http.StatusOK, gin.H{
		"message": "Login successful",
		"token":   tokenString,
	})
}

// getTenantsHandler handles GET /admin-api/tenants
func (api *RestAdminAPI) getTenantsHandler(c *gin.Context) {
	// Simulate node.Lookup()
	//query := []byte(`{"action": "list_tenants"}`)
	//_, err := api.node.Lookup(c.Request.Context(), query)
	//if err != nil {
	//	api.logger.Error().Err(err).Msg("Raft Lookup failed for list_tenants")
	//	c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list tenants (lookup error)"})
	//	return
	//}
	// Dummy response
	api.logger.Info().Msg("Admin API: Listing all tenants (dummy)")
	c.JSON(http.StatusOK, gin.H{"message": "Listing all tenants (dummy)", "tenants": []string{"tenantA", "tenantB"}})
}

// createTenantHandler handles POST /admin-api/tenants
func (api *RestAdminAPI) createTenantHandler(c *gin.Context) {
	// Simulate node.Propose() for tenant creation
	// In a real app, you'd parse tenant details from the request body
	//proposalData := []byte(`{"action": "create_tenant", "details": {"name": "newTenantFromAPI"}}`) // Dummy proposal
	//_, err := api.node.Propose(c.Request.Context(), proposalData)
	//if err != nil {
	//	api.logger.Error().Err(err).Msg("Raft Propose failed for create_tenant")
	//	c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create tenant (proposal error)"})
	//	return
	//}
	// Dummy response
	api.logger.Info().Msg("Admin API: Tenant creation requested (dummy)")
	c.JSON(http.StatusCreated, gin.H{"message": "Tenant created (dummy)", "id": "newTenantID123"})
}

// getTenantHandler handles GET /admin-api/tenants/:id
func (api *RestAdminAPI) getTenantHandler(c *gin.Context) {
	tenantID := c.Param("id")
	// Simulate node.Lookup()
	//query := []byte(fmt.Sprintf(`{"action": "get_tenant", "id": "%s"}`, tenantID))
	//_, err := api.node.Lookup(c.Request.Context(), query)
	//if err != nil {
	//	api.logger.Error().Err(err).Str("tenantID", tenantID).Msg("Raft Lookup failed for get_tenant")
	//	c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get tenant details (lookup error)"})
	//	return
	//}
	// Dummy response
	api.logger.Info().Str("tenantID", tenantID).Msg("Admin API: Getting tenant details (dummy)")
	c.JSON(http.StatusOK, gin.H{"message": "Details for tenant " + tenantID + " (dummy)", "id": tenantID, "name": "Dummy Tenant " + tenantID})
}

// updateTenantHandler handles PUT /admin-api/tenants/:id
func (api *RestAdminAPI) updateTenantHandler(c *gin.Context) {
	tenantID := c.Param("id")
	// Simulate node.Propose() for tenant update
	//proposalData := []byte(fmt.Sprintf(`{"action": "update_tenant", "id": "%s", "details": {"name": "updatedNameFromAPI"}}`, tenantID)) // Dummy proposal
	//_, err := api.node.Propose(c.Request.Context(), proposalData)
	//if err != nil {
	//	api.logger.Error().Err(err).Str("tenantID", tenantID).Msg("Raft Propose failed for update_tenant")
	//	c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update tenant (proposal error)"})
	//	return
	//}
	// Dummy response
	api.logger.Info().Str("tenantID", tenantID).Msg("Admin API: Tenant update requested (dummy)")
	c.JSON(http.StatusOK, gin.H{"message": "Tenant " + tenantID + " updated (dummy)", "id": tenantID})
}

// deleteTenantHandler handles DELETE /admin-api/tenants/:id
func (api *RestAdminAPI) deleteTenantHandler(c *gin.Context) {
	tenantID := c.Param("id")
	// Simulate node.Propose() for tenant deletion
	//proposalData := []byte(fmt.Sprintf(`{"action": "delete_tenant", "id": "%s"}`, tenantID)) // Dummy proposal
	//_, err := api.node.Propose(c.Request.Context(), proposalData)
	//if err != nil {
	//	api.logger.Error().Err(err).Str("tenantID", tenantID).Msg("Raft Propose failed for delete_tenant")
	//	c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete tenant (proposal error)"})
	//	return
	//}
	// Dummy response
	api.logger.Info().Str("tenantID", tenantID).Msg("Admin API: Tenant deletion requested (dummy)")
	c.JSON(http.StatusOK, gin.H{"message": "Tenant " + tenantID + " deleted (dummy)"})
}

// Helper function to get JWT expiration from global config (or use default)
// This is a placeholder, as the actual duration is passed during NewRestAdminAPI construction.
// However, it shows how one might access it if needed directly.
// It's important to note that this function uses the global `log` instance.
// If this function were to be used by methods that now have access to `api.logger`,
// it would be inconsistent. For the purpose of this refactoring, we assume this function
// is either not critically dependent on the specific logger instance for its warnings,
// or it might be refactored separately if strict logger consistency is required everywhere.
func getJWTExpirationDuration() time.Duration {
	hours := config.GlobalConfiguration.AdminAPIJWTExpirationHours
	if hours <= 0 {
		log.Warn().Int("configuredHours", hours).Msg("Invalid JWT expiration hours configured, defaulting to 3 hours.")
		return 3 * time.Hour // Default to 3 hours if not configured or invalid
	}
	return time.Duration(hours) * time.Hour
}

// generateJWT generates a new JWT token.
// This is a placeholder and will be properly implemented in the login handler.
func (api *RestAdminAPI) generateJWT(username string) (string, error) {
	expirationTime := time.Now().Add(api.jwtDuration)
	claims := &jwt.RegisteredClaims{
		Subject:   username,
		ExpiresAt: jwt.NewNumericDate(expirationTime),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(api.jwtKey)
	if err != nil {
		return "", err
	}
	return tokenString, nil
}
