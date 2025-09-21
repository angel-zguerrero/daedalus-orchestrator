package rest_server

import (
	"bytes"
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
	"encoding/gob"
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
func tenantContextMiddleware(tenantBO *bo.TenantBO, tenantNodesDictionary map[string]*dragonboat.RaftNode, logger zerolog.Logger) gin.HandlerFunc {
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

		// Obtener el nodo correspondiente al tenant
		node, exists := tenantNodesDictionary[cfs]
		if !exists {
			logger.Error().Str("tenantCode", tenantCode).Str("cfs", cfs).Msg("No node found for tenant")
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

func authMiddleware(MasterNode *dragonboat.RaftNode, logger zerolog.Logger, jwtKey []byte) gin.HandlerFunc {
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

		claims := &jwt.RegisteredClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return jwtKey, nil
		})

		if err != nil {
			if err == jwt.ErrTokenExpired {
				logger.Warn().Msg("JWT token expired")
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token expired"})
			} else {
				logger.Warn().Err(err).Msg("Invalid JWT token")
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			}
			return
		}

		if !token.Valid {
			logger.Warn().Msg("JWT token is invalid")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			return
		}

		// Locally validated, now check session existence via Raft
		checkSessionCmd := &auth_command.CheckSessionExistsCommand{
			JWTToken: tokenString,
			JWTKey:   jwtKey, // Assuming the command needs the key for its own validation if any
		}

		queryCmd := &general_command.Query_Command{
			Command: &general_command.Repository_Command{
				CMD: checkSessionCmd,
			},
			Now: time.Now().UnixNano(),
		}

		ctx, cancel := context.WithTimeout(context.Background(), config.GlobalConfiguration.ApiRaftTimeout)
		defer cancel()

		result, err := MasterNode.Read(ctx, *queryCmd)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to execute CheckSessionExistsCommand via Raft")
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify session"})
			return
		}

		buf := bytes.NewBuffer(result.([]byte))
		dec := gob.NewDecoder(buf)
		parsedResult := &commands.CommandResult{}
		if err := dec.Decode(parsedResult); err != nil {
			logger.Error().Err(err).Msg("Session does not exist or has been invalidated")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Session does not exist or has been invalidated:" + err.Error()})
			return
		}

		sessionExists := parsedResult.Result.(bool)

		if !sessionExists {
			logger.Warn().Str("token_subject", claims.Subject).Msg("Session does not exist or has been invalidated")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Session is invalid or has expired"})
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
