package server

import (
	"deadalus-orch/server/internal/pkg/config"
	"fmt"
	"net"

	"github.com/rs/zerolog/log"

	healthmetrics "deadalus-orch/server/internal/infrastructure/server/grpc/metrics"
	pb "deadalus-orch/server/internal/infrastructure/server/grpc/proto/health/metrics"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"

	"google.golang.org/grpc"
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
//
// Returns:
//   - An instance of GRPCServer.
type GRPCServerFactory func() GRPCServer

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
// It creates a standard grpc.Server with OpenTelemetry gRPC server stats handler enabled.
//
// Returns:
//   - A GRPCServer instance (which is a *grpc.Server).
func DefaultGRPCServerFactory() GRPCServer {
	// Enable OpenTelemetry instrumentation for the gRPC server.
	handler := otelgrpc.NewServerHandler()
	return grpc.NewServer(grpc.StatsHandler(handler))
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
	config_app config.Config, // Renamed to avoid conflict with package name 'config'
	listen ListenerFunc,
	gprcServerFactory GRPCServerFactory,
) error {

	// TODO: Port should be configurable via the `config_app` parameter.
	port := 2000 // Default port for the gRPC server.

	lis, err := listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}
	defer lis.Close()

	s := gprcServerFactory()
	defer s.GracefulStop()

	// registration gRPC implementations

	metricsSrv := healthmetrics.NewMetricsServer() // main or follower
	pb.RegisterMetricsServiceServer(s, metricsSrv)

	log.Info().
		Int("port", port).
		Msg("🚀 gRPC server listening")

	return s.Serve(lis)
}
