package server

import (
	"fmt"
	"net"

	"github.com/rs/zerolog/log"

	"deadalus-orch/server/internal/infrastructure/server/common"
	healthmetrics "deadalus-orch/server/internal/infrastructure/server/grpc/metrics"
	pb "deadalus-orch/server/internal/infrastructure/server/grpc/proto/health/metrics"
	pbT "deadalus-orch/server/internal/infrastructure/server/grpc/proto/pb/tenant"
	"deadalus-orch/server/internal/infrastructure/server/grpc/tenant"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"

	"time"

	"google.golang.org/grpc"

	"deadalus-orch/server/internal/infrastructure/dragonboat"

	"github.com/rs/zerolog"
)

// ListenerFunc is a function type that abstracts the creation of a net.Listener.
// This allows for different listener implementations, such as TCP or in-memory listeners for testing.
//
// Parameters:
//   - network: The network type (e.g., "tcp", "unix").
//   - address: The address to listen on (e.g., ":2000", "/tmp/grpc.sock").
//
// Returns:
//   - A net.Listener instance.
//   - An error if the listener cannot be created.
type ListenerFunc func(network, address string) (net.Listener, error)

// GRPCServer defines an interface for a gRPC server, allowing for implementations
// other than the standard grpc.Server (e.g., for testing or custom server behavior).
type GRPCServer interface {
	// Serve accepts incoming connections on the listener lis, creating a new
	// ServerTransport and service goroutine for each. The service goroutines
	// read gRPC requests and then call the registered handlers to reply to them.
	// Serve returns when lis.Accept fails with a non-temporary error.
	// lis will be closed when this method returns.
	// Serve will return a non-nil error unless GracefulStop is called.
	Serve(lis net.Listener) error
	// GracefulStop stops the gRPC server gracefully. It stops the server from
	// accepting new connections and waits for all active RPCs to finish.
	// It does not interrupt any active RPCs.
	GracefulStop()
	// RegisterService registers a service and its implementation to the gRPC
	// server. It is called from the IDL generated code. This must be called before
	// invoking Serve.
	RegisterService(sd *grpc.ServiceDesc, ss interface{})
}

// GRPCServerFactory is a function type that creates and returns an instance of GRPCServer.
// This allows for customizing the gRPC server creation, for example, to include interceptors or options.
type GRPCServerFactory func(
	masterNode *dragonboat.RaftNode,
	logger zerolog.Logger,
	jwtKey []byte,
	rateLimitStrategy string,
	rateLimitPeriod time.Duration,
	rateLimitCount int64,
) GRPCServer

// DefaultListener is the default implementation of ListenerFunc.
// It uses net.Listen to create a standard network listener.
//
// Parameters:
//   - network: The network type (e.g., "tcp", "unix").
//   - address: The address to listen on.
//
// Returns:
//   - A net.Listener instance.
//   - An error if net.Listen fails.
func DefaultListener(network, address string) (net.Listener, error) {
	return net.Listen(network, address)
}

// DefaultGRPCServerFactory is the default implementation of GRPCServerFactory.
// It creates a standard grpc.Server with OpenTelemetry gRPC server stats handler enabled
// and configured unary interceptors for auth and rate limiting.
//
// Returns:
//   - A GRPCServer instance (which is a *grpc.Server).
func DefaultGRPCServerFactory(
	masterNode *dragonboat.RaftNode,
	logger zerolog.Logger,
	jwtKey []byte,
	rateLimitStrategy string,
	rateLimitPeriod time.Duration,
	rateLimitCount int64,
) GRPCServer {
	// Enable OpenTelemetry instrumentation for the gRPC server.
	otelHandler := otelgrpc.NewServerHandler()

	// Setup our interceptors
	authInterceptor := UnaryAuthInterceptor(masterNode, logger, jwtKey)
	rateLimitInterceptor := UnaryRateLimitInterceptor(masterNode, logger, rateLimitStrategy, rateLimitPeriod, rateLimitCount)

	return grpc.NewServer(
		grpc.StatsHandler(otelHandler),
		grpc.ChainUnaryInterceptor(
			authInterceptor,
			rateLimitInterceptor,
			// Add other interceptors here if needed in the future
		),
	)
}

// StartGRPC initializes and starts the gRPC server.
// It creates a listener, gets a gRPC server instance using the provided factory,
// registers gRPC services (currently MetricsService), and starts serving requests.
// The server will gracefully stop when its operations are complete or if an error occurs.
//
// Parameters:
//   - config: The application configuration (currently unused in this function but passed for future use).
//   - listen: A ListenerFunc used to create the network listener.
//   - gprcServerFactory: A GRPCServerFactory used to create the gRPC server instance.
//
// Returns:
//   - An error if starting the listener or serving fails. Returns nil if the server
//     starts and shuts down gracefully without serving errors.
func StartGRPC(
	serverConfig *common.RestServerConfing, // Use existing config struct
	listen ListenerFunc,
	gprcServerFactory GRPCServerFactory,
) error {

	port := 2000 // Use passed gRPC port

	lis, err := listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}
	defer lis.Close()

	// Pass dependencies to the factory
	s := gprcServerFactory(
		serverConfig.MasterNode,
		serverConfig.Logger,
		serverConfig.JwtKey,
		"token",
		1*time.Minute,
		20,
	)
	defer s.GracefulStop()

	// registration gRPC implementations

	metricsSrv := healthmetrics.NewMetricsServer() // main or follower
	pb.RegisterMetricsServiceServer(s, metricsSrv)

	tenantSrv := tenant.NewTenantService(serverConfig) // main or follower
	pbT.RegisterTenantServiceServer(s, tenantSrv)

	log.Info().
		Int("port", port).
		Msg("🚀 gRPC server listening")

	return s.Serve(lis)
}
