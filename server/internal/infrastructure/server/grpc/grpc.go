package server

import (
	"deadalus-orch/server/internal/pkg/config"
	"fmt"
	"net"

	"github.com/rs/zerolog/log"

	pb "deadalus-orch/server/internal/infrastructure/common/proto/health/metrics"
	healthmetrics "deadalus-orch/server/internal/infrastructure/server/grpc/health"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"

	"google.golang.org/grpc"
)

type ListenerFunc func(network, address string) (net.Listener, error)

type GRPCServer interface {
	Serve(lis net.Listener) error
	GracefulStop()
	RegisterService(*grpc.ServiceDesc, interface{})
}

type GRPCServerFactory func() GRPCServer

func DefaultListener(network, address string) (net.Listener, error) {
	return net.Listen(network, address)
}

func DefaultGRPCServerFactory() GRPCServer {
	handler := otelgrpc.NewServerHandler()
	return grpc.NewServer(grpc.StatsHandler(handler))
}

func StartGRPC(
	config config.Config,
	listen ListenerFunc,
	gprcServerFactory GRPCServerFactory,
) error {

	port := 2000

	lis, err := listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}
	defer lis.Close()

	s := gprcServerFactory()
	defer s.GracefulStop()

	// registration gRPC implementations

	metricsSrv := healthmetrics.NewMetricsServer("main") // main or follower
	pb.RegisterMetricsServiceServer(s, metricsSrv)

	log.Info().
		Int("port", port).
		Msg("🚀 gRPC server listening")

	return s.Serve(lis)
}
