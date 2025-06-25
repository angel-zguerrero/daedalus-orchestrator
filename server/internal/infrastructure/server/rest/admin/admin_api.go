package rest_api_admin

import (
	"context"
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	"deadalus-orch/server/internal/pkg/config"
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

	// Simulate authentication: In a real app, you'd check req.Username and req.Password against a store.
	// For this task, we'll assume any provided username/password is valid for simulation purposes,
	// but we need to ensure they are not empty as per binding:"required".
	if req.Username == "" || req.Password == "" {
		// This case should ideally be caught by `binding:"required"`, but as a safeguard:
		api.logger.Warn().Msg("Login attempt with empty username or password")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username and password are required"})
		return
	}

	// Simulate node.Propose()
	// In a real scenario, the proposal would contain information about the login attempt,
	// e.g., to be recorded in a distributed log or to trigger some cluster-wide action.
	// For now, we just simulate a successful proposal.
	//proposalData := []byte(fmt.Sprintf(`{"action": "user_login", "username": "%s"}`, req.Username))
	//_, err := api.node.Propose(c.Request.Context(), proposalData) // Using request context

	//if err != nil {
	// If Propose returns an error, it means the Raft operation failed.
	//	api.logger.Error().Err(err).Str("username", req.Username).Msg("Raft Propose failed during login attempt")
	//	c.JSON(http.StatusInternalServerError, gin.H{"error": "Login failed due to an internal error (proposal)"})
	//	return
	//}

	// Simulate successful proposal (as per requirement, Propose is emulated to always return true, error is nil)
	// For now, we assume if err is nil, the proposal was "successful" for the purpose of JWT generation.

	// Generate JWT token
	tokenString, err := api.generateJWT(req.Username)
	if err != nil {
		api.logger.Error().Err(err).Str("username", req.Username).Msg("Failed to generate JWT token during login")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Login failed due to an internal error (token generation)"})
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
