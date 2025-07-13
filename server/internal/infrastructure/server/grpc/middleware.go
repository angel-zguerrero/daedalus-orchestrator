package grpc_server

import (
	"context"
	"time"

	"deadalus-orch/server/internal/infrastructure/dragonboat"
	ratelimit_store "deadalus-orch/server/internal/infrastructure/server/limiter"
	"deadalus-orch/server/internal/pkg/config"
	commands "deadalus-orch/server/internal/usecase/command"
	auth_command "deadalus-orch/server/internal/usecase/command/auth"
	general_command "deadalus-orch/server/internal/usecase/command/general"

	"bytes"
	"encoding/gob"
	"fmt"
	"strings"

	"net"

	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog"
	"github.com/ulule/limiter/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"

	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// UnaryAuthInterceptor returns a new unary server interceptor that authenticates requests.
func UnaryAuthInterceptor(MasterNode *dragonboat.RaftNode, logger zerolog.Logger, jwtKey []byte) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			logger.Warn().Msg("UnaryAuthInterceptor: Missing metadata")
			return nil, status.Errorf(codes.Unauthenticated, "missing metadata")
		}

		if !strings.HasSuffix(info.FullMethod, "AuthService/Login") {
			authHeaders := md.Get("authorization")
			if len(authHeaders) == 0 {
				logger.Warn().Msg("UnaryAuthInterceptor: Authorization header missing")
				return nil, status.Errorf(codes.Unauthenticated, "authorization token is required")
			}

			authHeader := authHeaders[0]
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				logger.Warn().Msg("UnaryAuthInterceptor: Invalid Authorization header format")
				return nil, status.Errorf(codes.Unauthenticated, "authorization token format is 'Bearer <token>'")
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
					logger.Warn().Msg("UnaryAuthInterceptor: JWT token expired")
					return nil, status.Errorf(codes.Unauthenticated, "token expired")
				}
				logger.Warn().Err(err).Msg("UnaryAuthInterceptor: Invalid JWT token")
				return nil, status.Errorf(codes.Unauthenticated, "invalid token")
			}

			if !token.Valid {
				logger.Warn().Msg("UnaryAuthInterceptor: JWT token is invalid")
				return nil, status.Errorf(codes.Unauthenticated, "invalid token")
			}

			// Locally validated, now check session existence via Raft
			checkSessionCmd := &auth_command.CheckSessionExistsCommand{
				JWTToken: tokenString,
				JWTKey:   jwtKey,
			}

			queryCmd := &general_command.Query_Command{
				Command: &general_command.Repository_Command{
					CMD: checkSessionCmd,
				},
				Now: time.Now().UnixNano(),
			}

			raftCtx, cancel := context.WithTimeout(context.Background(), config.GlobalConfiguration.ApiRaftTimeout)
			defer cancel()

			result, err := MasterNode.Read(raftCtx, *queryCmd)
			if err != nil {
				logger.Error().Err(err).Msg("UnaryAuthInterceptor: Failed to execute CheckSessionExistsCommand via Raft")
				return nil, status.Errorf(codes.Internal, "failed to verify session")
			}

			buf := bytes.NewBuffer(result.([]byte))
			dec := gob.NewDecoder(buf)
			parsedResult := &commands.CommandResult{}
			if err := dec.Decode(parsedResult); err != nil {
				logger.Error().Err(err).Msg("UnaryAuthInterceptor: Session does not exist or has been invalidated - decode error")
				return nil, status.Errorf(codes.Internal, "failed to decode session verification result")
			}

			sessionExists, ok := parsedResult.Result.(bool)
			if !ok {
				logger.Error().Msg("UnaryAuthInterceptor: Unexpected type for session existence result")
				return nil, status.Errorf(codes.Internal, "failed to interpret session verification result")
			}

			if !sessionExists {
				logger.Warn().Str("token_subject", claims.Subject).Msg("UnaryAuthInterceptor: Session does not exist or has been invalidated")
				return nil, status.Errorf(codes.Unauthenticated, "session is invalid or has expired")
			}

			// Add claims to context for downstream use if needed
			// newCtx := context.WithValue(ctx, "user_claims", claims)
			// return handler(newCtx, req)
		}

		return handler(ctx, req)
	}
}

// UnaryRateLimitInterceptor returns a new unary server interceptor that rate-limits requests.
func UnaryRateLimitInterceptor(MasterNode *dragonboat.RaftNode, logger zerolog.Logger, keyStrategy string, period time.Duration, limit int64) grpc.UnaryServerInterceptor {

	rate := limiter.Rate{
		Period: period,
		Limit:  limit,
	}

	store := ratelimit_store.NewRaftStore(MasterNode, "grpc_ratelimit", period)
	limiterInstance := limiter.New(store, rate)

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		var key string

		if keyStrategy == "token" {
			md, ok := metadata.FromIncomingContext(ctx)
			if ok {
				authHeaders := md.Get("authorization")
				if len(authHeaders) > 0 {
					authHeader := authHeaders[0]
					parts := strings.Split(authHeader, " ")
					if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
						key = parts[1] // Use token string as key
					}
				}
			}
			// If token is not found or format is wrong, fallback to IP
			if key == "" {
				logger.Warn().Msg("UnaryRateLimitInterceptor: Rate limiting by token: Authorization header missing or invalid, falling back to IP.")
				// Fallback to IP if no token or invalid format
				p, ok := peer.FromContext(ctx)
				if ok {
					// p.Addr could be of type *net.TCPAddr or *net.UnixAddr
					// We are interested in the IP part for TCP
					if tcpAddr, ok := p.Addr.(*net.TCPAddr); ok {
						key = tcpAddr.IP.String()
					} else {
						// For other address types or if IP cannot be determined, use a generic key or skip rate limiting
						// For simplicity, we'll use a generic key here, but this might need refinement
						key = p.Addr.String() // This might not be ideal for all Addr types
						logger.Warn().Str("address", key).Msg("UnaryRateLimitInterceptor: Could not determine client IP for rate limiting, using full address string.")
					}
				} else {
					logger.Error().Msg("UnaryRateLimitInterceptor: Failed to extract peer details for IP-based rate limiting fallback.")
					return nil, status.Errorf(codes.Internal, "failed to determine client identity for rate limiting")
				}
			}
		} else { // Default to IP-based strategy
			p, ok := peer.FromContext(ctx)
			if !ok {
				logger.Error().Msg("UnaryRateLimitInterceptor: Failed to extract peer details for IP-based rate limiting.")
				return nil, status.Errorf(codes.Internal, "failed to determine client identity for rate limiting")
			}
			if tcpAddr, ok := p.Addr.(*net.TCPAddr); ok {
				key = tcpAddr.IP.String()
			} else {
				key = p.Addr.String()
				logger.Warn().Str("address", key).Msg("UnaryRateLimitInterceptor: Could not determine client IP for rate limiting, using full address string for IP strategy.")
			}
		}

		if key == "" { // Should not happen if logic above is correct, but as a safeguard
			logger.Error().Msg("UnaryRateLimitInterceptor: Rate limiting key is empty after attempting to derive it.")
			return nil, status.Errorf(codes.Internal, "could not determine rate limiting key")
		}

		limiterCtx, err := limiterInstance.Get(ctx, key)
		if err != nil {
			logger.Error().Err(err).Str("key", key).Msg("UnaryRateLimitInterceptor: Error getting rate limit from store")
			return nil, status.Errorf(codes.Internal, "failed to check rate limit")
		}

		if limiterCtx.Reached {
			logger.Warn().Str("key", key).Int64("limit", limiterCtx.Limit).Msg("UnaryRateLimitInterceptor: Rate limit reached")
			// Also set headers when rate limit is reached
			headers := metadata.New(map[string]string{
				"x-ratelimit-limit":     fmt.Sprintf("%d", limiterCtx.Limit),
				"x-ratelimit-remaining": fmt.Sprintf("%d", limiterCtx.Remaining),
				"x-ratelimit-reset":     fmt.Sprintf("%d", limiterCtx.Reset),
			})
			if err := grpc.SetHeader(ctx, headers); err != nil {
				logger.Error().Err(err).Msg("UnaryRateLimitInterceptor: Failed to set rate limit headers on rate limit reached")
			}
			return nil, status.Errorf(codes.ResourceExhausted, "too many requests, please try again later")
		}

		// Set rate limit headers
		headers := metadata.New(map[string]string{
			"x-ratelimit-limit":     fmt.Sprintf("%d", limiterCtx.Limit),
			"x-ratelimit-remaining": fmt.Sprintf("%d", limiterCtx.Remaining),
			"x-ratelimit-reset":     fmt.Sprintf("%d", limiterCtx.Reset),
		})
		if err := grpc.SetHeader(ctx, headers); err != nil {
			// Log the error but don't fail the request, as header setting is secondary to request processing.
			logger.Error().Err(err).Msg("UnaryRateLimitInterceptor: Failed to set rate limit headers")
		}

		return handler(ctx, req)
	}
}
