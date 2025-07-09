package rest_server

import (
	"bytes"
	"context"
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	ratelimit "deadalus-orch/server/internal/infrastructure/server/limiter"
	"deadalus-orch/server/internal/pkg/config"
	commands "deadalus-orch/server/internal/usecase/command"
	"encoding/gob"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/ulule/limiter/v3"
	mgin "github.com/ulule/limiter/v3/drivers/middleware/gin"
)

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
		checkSessionCmd := &commands.CheckSessionExistsCommand{
			JWTToken: tokenString,
			JWTKey:   jwtKey, // Assuming the command needs the key for its own validation if any
		}

		queryCmd := &commands.Query_Command{
			Command: &commands.Repository_Command{
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
			logger.Error().Err(err).Msg("Paginate tenants command failed")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Paginate tenants command failed"})
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
