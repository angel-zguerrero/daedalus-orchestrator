package rest_api_admin

import (
	"context"
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	ratelimit "deadalus-orch/server/internal/infrastructure/server/limiter"
	"deadalus-orch/server/internal/pkg/config"
	"deadalus-orch/server/internal/pkg/utils"
	commands "deadalus-orch/server/internal/usecase/command"
	"fmt"
	"net/http"
	"time"

	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/ulule/limiter/v3"
	"github.com/ulule/limiter/v3/drivers/middleware/stdlib"
)

// RestAdminAPI handles the administrative REST API endpoints.
type RestAdminAPI struct {
	node        *dragonboat.RaftNode // Interface for Raft node interaction
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
func NewRestAdminAPI(node *dragonboat.RaftNode, jwtSecretKey string, jwtAuthDuration time.Duration, logger zerolog.Logger) *RestAdminAPI {
	if node == nil {
		logger.Fatal().Msg("Admin API: Raft node cannot be nil")
	}
	if jwtSecretKey == "" {
		logger.Warn().Msg("Admin API: JWT secret key is empty. This is insecure.")
	}
	gin.DefaultWriter = zerologAdapter{logger}
	gin.DefaultErrorWriter = zerologAdapter{logger}
	engine := gin.Default()

	api := &RestAdminAPI{
		node:        node,
		ginEngine:   engine,
		jwtKey:      []byte(jwtSecretKey),
		jwtDuration: jwtAuthDuration,
		logger:      logger,
	}

	// Setup routes
	adminAPIGroup := engine.Group("/admin-api")
	{
		adminAPIGroup.POST("/login", api.loginHandler)

		tenantsGroup := adminAPIGroup.Group("/tenants")
		tenantsGroup.Use(api.authMiddleware())
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
	UsernameOrEmail string `json:"UsernameOrEmail" binding:"required"`
	Password        string `json:"password" binding:"required"`
}

// loginHandler handles the /admin-api/login endpoint.
func (api *RestAdminAPI) loginHandler(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.logger.Warn().Err(err).Msg("Login attempt with invalid payload")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	loginCmd := &commands.LoginCommand{
		UsernameOrEmail: req.UsernameOrEmail,
		Password:        req.Password,
	}

	queryCommand := &commands.Query_Command{
		Command: &commands.Repository_Command{
			CMD: loginCmd,
		},
		Now: time.Now().UnixNano(), // Or handle as per specific requirements if Query_Command.Now is actively used
	}

	ctx, cancel := context.WithTimeout(context.Background(), config.GlobalConfiguration.ApiRaftTimeout)
	defer cancel()
	result, err := api.node.Read(ctx, *queryCommand)
	if err != nil {
		api.logger.Error().Err(err).Str("username", req.UsernameOrEmail).Msg("Login command execution failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Login failed: " + err.Error()})
		return
	}

	loggedIn, err := utils.BytesToBool(result.([]byte))
	if err != nil {
		api.logger.Error().Str("username", req.UsernameOrEmail).Msg("Login command returned unexpected result type")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Login failed due to an internal error (result type)"})
		return
	}

	if !loggedIn {
		api.logger.Warn().Str("username", req.UsernameOrEmail).Msg("Login attempt failed: invalid credentials")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}

	tokenString, err := api.generateJWT(req.UsernameOrEmail)
	if err != nil {
		api.logger.Error().Err(err).Str("username", req.UsernameOrEmail).Msg("Failed to generate JWT token during login")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Login successful, but failed to generate token: " + err.Error()})
		return
	}

	registerSessionCmd := &commands.RegisterSessionCommand{
		JWTToken: tokenString,
		JWTKey:   api.jwtKey,
	}

	fsmCmd := commands.FSM_Command{
		Now:  utils.GetNowInInt(),
		Type: commands.REPOSITORY_COMMAND,
		CMD:  registerSessionCmd,
	}

	writeCtx, writeCancel := context.WithTimeout(context.Background(), config.GlobalConfiguration.ApiRaftTimeout) // Or a specific timeout for writes
	defer writeCancel()

	_, err = api.node.Write(writeCtx, fsmCmd)
	if err != nil {

		api.logger.Error().Err(err).Str("username", req.UsernameOrEmail).Msg("Failed to register session after login")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register session after login: " + err.Error()})
		return
	}

	api.logger.Info().Str("username", req.UsernameOrEmail).Msg("User logged in successfully and session registered")
	c.JSON(http.StatusOK, gin.H{
		"message": "Login successful",
		"token":   tokenString,
	})
}

// getTenantsHandler handles GET /admin-api/tenants
func (api *RestAdminAPI) getTenantsHandler(c *gin.Context) {

	c.JSON(http.StatusOK, gin.H{"message": "Listing all tenants (dummy)", "tenants": []string{"tenantA", "tenantB"}})
}

// createTenantHandler handles POST /admin-api/tenants
func (api *RestAdminAPI) createTenantHandler(c *gin.Context) {
	c.JSON(http.StatusCreated, gin.H{"message": "Tenant created (dummy)", "id": "newTenantID123"})
}

// getTenantHandler handles GET /admin-api/tenants/:id
func (api *RestAdminAPI) getTenantHandler(c *gin.Context) {
	tenantID := c.Param("id")

	c.JSON(http.StatusOK, gin.H{"message": "Details for tenant " + tenantID + " (dummy)", "id": tenantID, "name": "Dummy Tenant " + tenantID})
}

// updateTenantHandler handles PUT /admin-api/tenants/:id
func (api *RestAdminAPI) updateTenantHandler(c *gin.Context) {
	tenantID := c.Param("id")

	c.JSON(http.StatusOK, gin.H{"message": "Tenant " + tenantID + " updated (dummy)", "id": tenantID})
}

// deleteTenantHandler handles DELETE /admin-api/tenants/:id
func (api *RestAdminAPI) deleteTenantHandler(c *gin.Context) {
	tenantID := c.Param("id")

	c.JSON(http.StatusOK, gin.H{"message": "Tenant " + tenantID + " deleted (dummy)"})
}

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

func NewRateLimitMiddleware(raftNode *dragonboat.RaftNode) gin.HandlerFunc {
	// Define rate: 20 requests por minuto
	rate := limiter.Rate{
		Period: 1 * time.Minute,
		Limit:  20,
	}

	// Crear store personalizado
	store := ratelimit.NewRaftStore(raftNode, "ratelimit", 1*time.Minute)

	// Crear instancia del limiter
	limiterInstance := limiter.New(store, rate)

	// Crear middleware estilo stdlib
	limiterMiddleware := stdlib.NewMiddleware(limiterInstance)

	// Adaptarlo a gin.HandlerFunc
	return func(c *gin.Context) {
		// Adaptar gin context a http.Request y http.ResponseWriter
		limiterMiddleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Pasar el control a Gin
			c.Request = r
			c.Next()
		})).ServeHTTP(c.Writer, c.Request)
	}
}

// authMiddleware creates a middleware handler for JWT authentication and session validation.
func (api *RestAdminAPI) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			api.logger.Warn().Msg("Authorization header missing")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			api.logger.Warn().Msg("Invalid Authorization header format")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid Authorization header format"})
			return
		}
		tokenString := parts[1]

		claims := &jwt.RegisteredClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return api.jwtKey, nil
		})

		if err != nil {
			if err == jwt.ErrTokenExpired {
				api.logger.Warn().Msg("JWT token expired")
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token expired"})
			} else {
				api.logger.Warn().Err(err).Msg("Invalid JWT token")
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			}
			return
		}

		if !token.Valid {
			api.logger.Warn().Msg("JWT token is invalid")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			return
		}

		// Locally validated, now check session existence via Raft
		checkSessionCmd := &commands.CheckSessionExistsCommand{
			JWTToken: tokenString,
			JWTKey:   api.jwtKey, // Assuming the command needs the key for its own validation if any
		}

		queryCmd := &commands.Query_Command{
			Command: &commands.Repository_Command{
				CMD: checkSessionCmd,
			},
			Now: time.Now().UnixNano(),
		}

		ctx, cancel := context.WithTimeout(context.Background(), config.GlobalConfiguration.ApiRaftTimeout)
		defer cancel()

		resultBytes, err := api.node.Read(ctx, *queryCmd)
		if err != nil {
			api.logger.Error().Err(err).Msg("Failed to execute CheckSessionExistsCommand via Raft")
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify session"})
			return
		}

		sessionExists, err := utils.BytesToBool(resultBytes.([]byte))
		if err != nil {
			api.logger.Error().Err(err).Msg("CheckSessionExistsCommand returned unexpected result type")
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to interpret session verification result"})
			return
		}

		if !sessionExists {
			api.logger.Warn().Str("token_subject", claims.Subject).Msg("Session does not exist or has been invalidated")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Session is invalid or has expired"})
			return
		}

		c.Next()
	}
}
