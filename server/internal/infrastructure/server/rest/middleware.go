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
		tenantCode := c.Param("code")
		if tenantCode == "" {
			c.Next()
			return
		}

		tenant, _, _, err := tenantBO.GetTenant(c.Request.Context(), tenantCode)
		if err != nil {
			logger.Error().Err(err).Str("tenantCode", tenantCode).Msg("Failed to get tenant in middleware")
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant: " + err.Error()})
			c.Abort()
			return
		}

		// Tenant-boundary enforcement for OAuth tokens:
		// If oauth_tenant_id is present and non-empty (Tenant OAuth App), it MUST match target tenant ID.
		// If oauth_tenant_id is empty (Global OAuth App), bypass check for cross-tenant access.
		if oauthTenantIDRaw, exists := c.Get("oauth_tenant_id"); exists {
			if oauthTenantID, ok := oauthTenantIDRaw.(string); ok && oauthTenantID != "" {
				if oauthTenantID != tenant.ID {
					logger.Warn().
						Str("token_tenant_id", oauthTenantID).
						Str("request_tenant_id", tenant.ID).
						Msg("Tenant boundary violation for OAuth token")
					c.JSON(http.StatusForbidden, gin.H{
						"error": fmt.Sprintf("OAuth token is restricted to tenant %s", oauthTenantID),
					})
					c.Abort()
					return
				}
			}
		}

		cf := db.ColumnFamilyPrefix + strconv.Itoa(tenant.ColumnFamilyIndex)
		cfs := tenant.ID

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

		tenantCtx := &common.TenantContext{
			Tenant: &tenant,
			Node:   node,
			CF:     cf,
			CFS:    cfs,
		}

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
func unifiedAuthMiddleware(MasterNode *dragonboat.RaftNode, logger zerolog.Logger, jwtKey []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
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

		tokenType, _ := mapClaims["token_type"].(string)
		if tokenType == "" {
			tokenType = "session"
		}

		switch tokenType {

		case "oauth":
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
			c.Set("oauth_scopes", claims.Scopes)
			c.Set("oauth_tenant_id", claims.TenantID)
			c.Set("oauth_client_id", claims.Subject)
			logger.Debug().
				Str("client_id", claims.Subject).
				Strs("scopes", claims.Scopes).
				Msg("OAuth machine token authenticated")
			c.Next()

		default:
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
			logger.Debug().Msg("Admin session token authenticated")
			c.Next()
		}
	}
}

// requireScope enforces OAuth scope restrictions on routes that both admin users and
// OAuth Service Accounts can access.
func requireScope(required ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		rawScopes, isOAuthToken := c.Get("oauth_scopes")
		if !isOAuthToken {
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
				return c.ClientIP()
			}
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				log.Warn().Msg("Rate limiting by token: Invalid Authorization header format, falling back to IP.")
				return c.ClientIP()
			}
			return parts[1]
		})
	} else {
		options = mgin.WithKeyGetter(func(c *gin.Context) string {
			return c.ClientIP()
		})
	}

	return mgin.NewMiddleware(limiter.New(store, rate), options)
}
