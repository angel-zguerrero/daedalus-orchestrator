package grpc_server

import (
	"context"
	"time"

	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	"deadalus-orch/server/internal/infrastructure/server/common"
	ratelimit_store "deadalus-orch/server/internal/infrastructure/server/limiter"
	"deadalus-orch/server/internal/pkg/config"
	bo "deadalus-orch/server/internal/usecase/business-logic"
	commands "deadalus-orch/server/internal/usecase/command"
	auth_command "deadalus-orch/server/internal/usecase/command/auth"
	general_command "deadalus-orch/server/internal/usecase/command/general"

	"fmt"
	"reflect"
	"strconv"
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

// UnaryTenantInterceptor returns a new unary server interceptor that extracts tenant information and injects it into the context
func UnaryTenantInterceptor(tenantBO *bo.TenantBO, serverConfig *common.ServerConfing, logger zerolog.Logger) grpc.UnaryServerInterceptor {
	// Methods that create tenants must bypass tenant lookup (tenant doesn't exist yet)
	skipTenantLookup := map[string]bool{
		"/tenant.TenantService/AssertTenant":     true,
		"/tenant.TenantService/AssertBulkTenant": true,
	}

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if skipTenantLookup[info.FullMethod] {
			return handler(ctx, req)
		}

		// Extract tenant code from the request using reflection
		tenantCode := extractTenantCodeFromRequest(req)
		if tenantCode == "" {
			// Si no hay tenant code en la request, continúa sin inyectar contexto
			return handler(ctx, req)
		}

		// Obtener información del tenant
		tenant, _, _, err := tenantBO.GetTenant(ctx, tenantCode)
		if err != nil {
			logger.Error().Err(err).Str("tenantCode", tenantCode).Msg("Failed to get tenant in gRPC interceptor")
			return nil, status.Errorf(codes.InvalidArgument, "Invalid tenant: %v", err)
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
			logger.Error().Str("tenantCode", tenantCode).Str("cfs", cfs).Int("shardId", tenant.ShardId).Msg("No node found for tenant shard in gRPC interceptor")
			return nil, status.Errorf(codes.Internal, "Tenant node not available")
		}

		// Crear el contexto del tenant
		tenantCtx := &common.TenantContext{
			Tenant: &tenant,
			Node:   node,
			CF:     cf,
			CFS:    cfs,
		}

		// Inyectar en el contexto
		newCtx := common.SetTenantContext(ctx, tenantCtx)

		return handler(newCtx, req)
	}
}

// extractTenantCodeFromRequest extrae el código de tenant de diferentes tipos de request usando reflexión
func extractTenantCodeFromRequest(req interface{}) string {
	if req == nil {
		return ""
	}

	v := reflect.ValueOf(req)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return ""
	}

	// Buscar campos comunes que contengan el tenant code
	tenantFields := []string{"TenantCode", "Code"}

	for _, fieldName := range tenantFields {
		field := v.FieldByName(fieldName)
		if field.IsValid() && field.Kind() == reflect.String {
			return field.String()
		}
	}

	return ""
}

type wrappedStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedStream) Context() context.Context {
	return w.ctx
}

// StreamTenantInterceptor returns a new stream server interceptor that extracts tenant information and injects it into the context
func StreamTenantInterceptor(tenantBO *bo.TenantBO, serverConfig *common.ServerConfing, logger zerolog.Logger) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// Note: Tenant code for streams usually needs to be extracted from metadata or the first message.
		// For our case, we will rely on metadata if it's there. 
		// If not, the stream handler itself will need to extract it from the first message (which is what we did in the handlers).
		// However, to keep it consistent, if the handler already extracts it (like PublishStream), we don't strictly need it here unless we enforce it.
		// We'll pass it through as we rely on the handler to call common.MustGetTenantData after parsing the first message if it's in the message.
		// Wait, common.MustGetTenantData expects it in the context.
		// If it's not in the context, MustGetTenantData panics.
		// The stream request might not have the tenant code in metadata, but in the message itself.
		// Ah, looking at `ExchangeService.PublishStream`, it does `common.MustGetTenantData(ctx)` *before* reading any message.
		// This means the tenant code MUST be in the context, typically from metadata injected by the client, or we must extract it.
		// In UnaryTenantInterceptor, it extracts from the `req` interface. For streams, `req` isn't available upfront.
		// How does the client send it? Usually via metadata for streams.
		// Let's implement extracting from metadata for StreamTenantInterceptor.

		md, ok := metadata.FromIncomingContext(ss.Context())
		if !ok {
			return handler(srv, ss) // Let handler deal with missing tenant or panic
		}

		tenantCodes := md.Get("x-tenant-code") // Example header, adjust if client uses something else
		if len(tenantCodes) == 0 {
			// Try "tenant-code"
			tenantCodes = md.Get("tenant-code")
		}

		if len(tenantCodes) == 0 {
			return handler(srv, ss)
		}

		tenantCode := tenantCodes[0]

		tenant, _, _, err := tenantBO.GetTenant(ss.Context(), tenantCode)
		if err != nil {
			logger.Error().Err(err).Str("tenantCode", tenantCode).Msg("Failed to get tenant in gRPC stream interceptor")
			return status.Errorf(codes.InvalidArgument, "Invalid tenant: %v", err)
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
			logger.Error().Str("tenantCode", tenantCode).Str("cfs", cfs).Int("shardId", tenant.ShardId).Msg("No node found for tenant shard in gRPC stream interceptor")
			return status.Errorf(codes.Internal, "Tenant node not available")
		}

		tenantCtx := &common.TenantContext{
			Tenant: &tenant,
			Node:   node,
			CF:     cf,
			CFS:    cfs,
		}

		newCtx := common.SetTenantContext(ss.Context(), tenantCtx)
		wrapped := &wrappedStream{ServerStream: ss, ctx: newCtx}

		return handler(srv, wrapped)
	}
}

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

			sessionExists, err := commands.DecodeCommandResult[bool](result.([]byte))
			if err != nil {
				logger.Error().Err(err).Msg("UnaryAuthInterceptor: Session does not exist or has been invalidated - decode error")
				return nil, status.Errorf(codes.Internal, "failed to decode session verification result")
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

// StreamAuthInterceptor returns a new stream server interceptor that authenticates requests.
func StreamAuthInterceptor(MasterNode *dragonboat.RaftNode, logger zerolog.Logger, jwtKey []byte) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := ss.Context()
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			logger.Warn().Msg("StreamAuthInterceptor: Missing metadata")
			return status.Errorf(codes.Unauthenticated, "missing metadata")
		}

		// Similar logic as UnaryAuthInterceptor
		if !strings.HasSuffix(info.FullMethod, "AuthService/Login") {
			authHeaders := md.Get("authorization")
			if len(authHeaders) == 0 {
				logger.Warn().Msg("StreamAuthInterceptor: Authorization header missing")
				return status.Errorf(codes.Unauthenticated, "authorization token is required")
			}

			authHeader := authHeaders[0]
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				logger.Warn().Msg("StreamAuthInterceptor: Invalid Authorization header format")
				return status.Errorf(codes.Unauthenticated, "authorization token format is 'Bearer <token>'")
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
					logger.Warn().Msg("StreamAuthInterceptor: JWT token expired")
					return status.Errorf(codes.Unauthenticated, "token expired")
				}
				logger.Warn().Err(err).Msg("StreamAuthInterceptor: Invalid JWT token")
				return status.Errorf(codes.Unauthenticated, "invalid token")
			}

			if !token.Valid {
				logger.Warn().Msg("StreamAuthInterceptor: JWT token is invalid")
				return status.Errorf(codes.Unauthenticated, "invalid token")
			}

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
				logger.Error().Err(err).Msg("StreamAuthInterceptor: Failed to execute CheckSessionExistsCommand via Raft")
				return status.Errorf(codes.Internal, "failed to verify session")
			}

			sessionExists, err := commands.DecodeCommandResult[bool](result.([]byte))
			if err != nil {
				logger.Error().Err(err).Msg("StreamAuthInterceptor: Session does not exist or has been invalidated - decode error")
				return status.Errorf(codes.Internal, "failed to decode session verification result")
			}

			if !sessionExists {
				logger.Warn().Str("token_subject", claims.Subject).Msg("StreamAuthInterceptor: Session does not exist or has been invalidated")
				return status.Errorf(codes.Unauthenticated, "session is invalid or has expired")
			}
		}

		return handler(srv, ss)
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

// StreamRateLimitInterceptor returns a new stream server interceptor that rate-limits requests.
func StreamRateLimitInterceptor(MasterNode *dragonboat.RaftNode, logger zerolog.Logger, keyStrategy string, period time.Duration, limit int64) grpc.StreamServerInterceptor {
	rate := limiter.Rate{
		Period: period,
		Limit:  limit,
	}

	store := ratelimit_store.NewRaftStore(MasterNode, "grpc_ratelimit", period)
	limiterInstance := limiter.New(store, rate)

	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := ss.Context()
		var key string

		if keyStrategy == "token" {
			md, ok := metadata.FromIncomingContext(ctx)
			if ok {
				authHeaders := md.Get("authorization")
				if len(authHeaders) > 0 {
					authHeader := authHeaders[0]
					parts := strings.Split(authHeader, " ")
					if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
						key = parts[1]
					}
				}
			}
			if key == "" {
				p, ok := peer.FromContext(ctx)
				if ok {
					if tcpAddr, ok := p.Addr.(*net.TCPAddr); ok {
						key = tcpAddr.IP.String()
					} else {
						key = p.Addr.String()
					}
				} else {
					return status.Errorf(codes.Internal, "failed to determine client identity for rate limiting")
				}
			}
		} else {
			p, ok := peer.FromContext(ctx)
			if !ok {
				return status.Errorf(codes.Internal, "failed to determine client identity for rate limiting")
			}
			if tcpAddr, ok := p.Addr.(*net.TCPAddr); ok {
				key = tcpAddr.IP.String()
			} else {
				key = p.Addr.String()
			}
		}

		if key == "" {
			return status.Errorf(codes.Internal, "could not determine rate limiting key")
		}

		limiterCtx, err := limiterInstance.Get(ctx, key)
		if err != nil {
			return status.Errorf(codes.Internal, "failed to check rate limit")
		}

		if limiterCtx.Reached {
			headers := metadata.New(map[string]string{
				"x-ratelimit-limit":     fmt.Sprintf("%d", limiterCtx.Limit),
				"x-ratelimit-remaining": fmt.Sprintf("%d", limiterCtx.Remaining),
				"x-ratelimit-reset":     fmt.Sprintf("%d", limiterCtx.Reset),
			})
			_ = grpc.SetHeader(ctx, headers)
			return status.Errorf(codes.ResourceExhausted, "too many requests, please try again later")
		}

		headers := metadata.New(map[string]string{
			"x-ratelimit-limit":     fmt.Sprintf("%d", limiterCtx.Limit),
			"x-ratelimit-remaining": fmt.Sprintf("%d", limiterCtx.Remaining),
			"x-ratelimit-reset":     fmt.Sprintf("%d", limiterCtx.Reset),
		})
		_ = grpc.SetHeader(ctx, headers)

		return handler(srv, ss)
	}
}
