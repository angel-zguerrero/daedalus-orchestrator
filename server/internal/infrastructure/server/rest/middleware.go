package rest_server

import (
	"context"
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	"deadalus-orch/server/internal/infrastructure/server/common"
	ratelimit "deadalus-orch/server/internal/infrastructure/server/limiter"
	"deadalus-orch/server/internal/pkg/config"
	bo "deadalus-orch/server/internal/usecase/business-logic"
	commands "deadalus-orch/server/internal/usecase/command"
	auth_command "deadalus-orch/server/internal/usecase/command/auth"
	general_command "deadalus-orch/server/internal/usecase/command/general"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/ulule/limiter/v3"
	mgin "github.com/ulule/limiter/v3/drivers/middleware/gin"
)

// tenantContextMiddleware creates a middleware that extracts tenant information and injects it into the context
func tenantContextMiddleware(tenantBO *bo.TenantBO, serverConfig *common.ServerConfing, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Solo aplica a rutas que tienen un parámetro :code de tenant
		tenantCode := c.Param("code")
		if tenantCode == "" {
			// Si no hay código de tenant, continúa sin inyectar contexto
			c.Next()
			return
		}

		// Obtener información del tenant
		tenant, _, _, err := tenantBO.GetTenant(c.Request.Context(), tenantCode)
		if err != nil {
			logger.Error().Err(err).Str("tenantCode", tenantCode).Msg("Failed to get tenant in middleware")
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant: " + err.Error()})
			c.Abort()
			return
		}

		// Construir CF y CFS
		cf := db.ColumnFamilyPrefix + strconv.Itoa(tenant.ColumnFamilyIndex)
		cfs := tenant.ID

		// Obtener el nodo correspondiente al tenant usando ShardId
		var node *dragonboat.RaftNode
		serverConfig.TenantNodesLock.Lock()
		for i := range serverConfig.TenantNodes {
			if serverConfig.TenantNodes[i].ShardID == uint64(tenant.ShardId) {
				node = serverConfig.TenantNodes[i]
				break
			}
		}
		serverConfig.TenantNodesLock.Unlock()

		if node == nil {
			logger.Error().Str("tenantCode", tenantCode).Str("cfs", cfs).Int("shardId", tenant.ShardId).Msg("No node found for tenant shard in REST middleware")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Tenant node not available"})
			c.Abort()
			return
		}

		// Crear el contexto del tenant
		tenantCtx := &common.TenantContext{
			Tenant: &tenant,
			Node:   node,
			CF:     cf,
			CFS:    cfs,
		}

		// Inyectar en el contexto de la request
		newCtx := common.SetTenantContext(c.Request.Context(), tenantCtx)
		c.Request = c.Request.WithContext(newCtx)

		c.Next()
	}
}

// unifiedAuthMiddleware handles both human admin session tokens and OAuth 2.0 machine tokens
// on the same set of routes. It branches on the token_type JWT claim:
//
//	"session" (or missing — backward compat): validates signature + checks KV session via Raft
//	"oauth": validates signature only (stateless), injects oauth_scopes, oauth_tenant_id,
//	         oauth_client_id into gin.Context for downstream requireScope middleware
//
// Backward compatibility: JWTs without a token_type claim (issued before this change)
// are treated as "session" so zero-downtime deploys work correctly.
func unifiedAuthMiddleware(MasterNode *dragonboat.RaftNode, logger zerolog.Logger, jwtKey []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		// ── Step 1: Extract Bearer token ─────────────────────────────────────
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			logger.Warn().Msg("Authorization header missing")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			logger.Warn().Msg("Invalid Authorization header format")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid Authorization header format"})
			return
		}
		tokenString := parts[1]

		// ── Step 2: Verify signature & expiry (common path) ───────────────────
		// Parse into MapClaims first to read token_type without committing to a struct
		rawToken, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return jwtKey, nil
		})
		if err != nil {
			if errors.Is(err, jwt.ErrTokenExpired) {
				logger.Warn().Msg("JWT token expired")
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token expired"})
			} else {
				logger.Warn().Err(err).Msg("Invalid JWT token")
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			}
			return
		}
		if !rawToken.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			return
		}

		mapClaims, ok := rawToken.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Malformed token claims"})
			return
		}

		// ── Step 3: Branch on token_type ──────────────────────────────────────
		tokenType, _ := mapClaims["token_type"].(string)
		// Default to "session" when missing — backward compat with pre-existing JWTs
		if tokenType == "" {
			tokenType = "session"
		}

		switch tokenType {

		case "oauth":
			// ── OAuth path: stateless, inject scopes into gin.Context ──────────
			oauthToken, err := jwt.ParseWithClaims(tokenString, &bo.OAuthClaims{}, func(t *jwt.Token) (interface{}, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method")
				}
				return jwtKey, nil
			})
			if err != nil || !oauthToken.Valid {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid OAuth token"})
				return
			}
			claims, ok := oauthToken.Claims.(*bo.OAuthClaims)
			if !ok {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Malformed OAuth token claims"})
				return
			}
			// Inject into gin.Context for downstream requireScope middleware
			c.Set("oauth_scopes", claims.Scopes)
			c.Set("oauth_tenant_id", claims.TenantID)
			c.Set("oauth_client_id", claims.Subject)
			logger.Debug().
				Str("client_id", claims.Subject).
				Strs("scopes", claims.Scopes).
				Msg("OAuth machine token authenticated")
			c.Next()

		default: // "session" or unknown (treated as session for backward compat)
			// ── Session path: verify KV session existence via Raft ────────────
			checkSessionCmd := &auth_command.CheckSessionExistsCommand{
				JWTToken: tokenString,
				JWTKey:   jwtKey,
			}
			queryCmd := &general_command.Query_Command{
				Command: &general_command.Repository_Command{CMD: checkSessionCmd},
				Now:     time.Now().UnixNano(),
			}
			ctx, cancel := context.WithTimeout(context.Background(), config.GlobalConfiguration.ApiRaftTimeout)
			defer cancel()

			result, err := MasterNode.Read(ctx, *queryCmd)
			if err != nil {
				logger.Error().Err(err).Msg("Failed to execute CheckSessionExistsCommand via Raft")
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify session"})
				return
			}
			sessionExists, err := commands.DecodeCommandResult[bool](result.([]byte))
			if err != nil {
				logger.Error().Err(err).Msg("Failed to decode session check result")
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify session: " + err.Error()})
				return
			}
			if !sessionExists {
				logger.Warn().Msg("Session does not exist or has been invalidated")
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Session is invalid or has expired"})
				return
			}
			// No oauth_scopes injected → requireScope will pass through for session tokens
			logger.Debug().Msg("Admin session token authenticated")
			c.Next()
		}
	}
}

// requireScope enforces OAuth scope restrictions on routes that both admin users and
// OAuth Service Accounts can access.
//
// For OAUTH tokens: at least one of the required scopes must be present in the token.
//
//	Returns 403 Forbidden if none match.
//
// For SESSION tokens (admin users): ALWAYS passes through.
//
//	Admin users have implicit full access — no scope restriction is applied.
//
// Usage example:
//
//	tenantsGroup.POST("/:code/queue", requireScope("queues:create", "queues:admin"), queueCtrl.CreateQueueHandler)
func requireScope(required ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		rawScopes, isOAuthToken := c.Get("oauth_scopes")
		if !isOAuthToken {
			// No oauth_scopes in context → this is a session token → always allow
			c.Next()
			return
		}

		scopes, ok := rawScopes.([]string)
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Invalid scope format in token"})
			return
		}

		scopeSet := make(map[string]bool, len(scopes))
		for _, s := range scopes {
			scopeSet[s] = true
		}
		for _, req := range required {
			if scopeSet[req] {
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"error":           "Insufficient scope",
			"required_one_of": required,
		})
	}
}

// sessionOnlyMiddleware blocks OAuth machine tokens from accessing admin-only routes.
// Routes protected by this middleware are exclusively for human admin users.
// Any request authenticated via an OAuth token will receive 403 Forbidden.
//
// Apply to:
//   - OAuth App CRUD management routes (Service Accounts cannot manage other Service Accounts)
//   - User management routes
//   - Any other admin-only operational route
func sessionOnlyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, isOAuth := c.Get("oauth_scopes"); isOAuth {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "This endpoint requires an admin session. OAuth tokens are not permitted here.",
			})
			return
		}
		c.Next()
	}
}
func rateLimitMiddleware(MasterNode *dragonboat.RaftNode, keyStrategy string, Period time.Duration, Limit int64) gin.HandlerFunc {
	rate := limiter.Rate{
		Period: Period,
		Limit:  Limit,
	}

	store := ratelimit.NewRaftStore(MasterNode, "ratelimit", Period)

	var options mgin.Option
	if keyStrategy == "token" {
		options = mgin.WithKeyGetter(func(c *gin.Context) string {
			authHeader := c.GetHeader("Authorization")
			if authHeader == "" {
				log.Warn().Msg("Rate limiting by token: Authorization header missing, falling back to IP.")
				return c.ClientIP() // Fallback to IP if no token
			}
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				log.Warn().Msg("Rate limiting by token: Invalid Authorization header format, falling back to IP.")
				return c.ClientIP() // Fallback to IP if format is wrong
			}
			return parts[1] // Use token string as key
		})
	} else {
		// Default IP-based strategy, mgin handles this by default if no KeyGetter or specific context key is set.
		// Explicitly setting it for clarity.
		options = mgin.WithKeyGetter(func(c *gin.Context) string {
			return c.ClientIP()
		})
	}

	// It's important to pass the options to NewMiddleware.
	// If multiple options are needed in the future, they can be passed as additional arguments.
	return mgin.NewMiddleware(limiter.New(store, rate), options)
}

// authMiddleware creates a middleware handler for JWT authentication and session validation.
