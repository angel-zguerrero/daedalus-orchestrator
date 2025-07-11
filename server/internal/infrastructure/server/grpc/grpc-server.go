package grpc_server

import (
	"fmt"
	"net"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"

	"deadalus-orch/server/internal/infrastructure/server/common"
	healthmetrics "deadalus-orch/server/internal/infrastructure/server/grpc/metrics"
	pb "deadalus-orch/server/internal/infrastructure/server/grpc/proto/health/metrics"
	pbT "deadalus-orch/server/internal/infrastructure/server/grpc/proto/pb/tenant"
	"deadalus-orch/server/internal/infrastructure/server/grpc/tenant"
)

type GrpcServer struct {
	Config     *common.RestServerConfing
	grpcServer *grpc.Server
	listener   net.Listener
}

func NewGrpcServer(config *common.RestServerConfing) (*GrpcServer, error) {
	if config.MasterNode == nil {
		config.Logger.Fatal().Msg("gRPC: Raft node cannot be nil")
	}

	port := 2000
	listenAddr := fmt.Sprintf(":%d", port)
	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return nil, fmt.Errorf("gRPC: failed to listen: %w", err)
	}

	otelHandler := otelgrpc.NewServerHandler()
	authInterceptor := UnaryAuthInterceptor(config.MasterNode, config.Logger, config.JwtKey)
	rateLimitInterceptor := UnaryRateLimitInterceptor(config.MasterNode, config.Logger, "token", time.Minute, 20)

	server := grpc.NewServer(
		grpc.StatsHandler(otelHandler),
		grpc.ChainUnaryInterceptor(authInterceptor, rateLimitInterceptor),
	)

	// Registrar servicios
	pb.RegisterMetricsServiceServer(server, healthmetrics.NewMetricsServer())
	pbT.RegisterTenantServiceServer(server, tenant.NewTenantService(config))

	return &GrpcServer{
		Config:     config,
		grpcServer: server,
		listener:   lis,
	}, nil
}

func (s *GrpcServer) Start() error {
	addr := s.listener.Addr().String()
	s.Config.Logger.Info().Str("address", addr).Msg("🚀 Starting gRPC server...")
	if err := s.grpcServer.Serve(s.listener); err != nil {
		s.Config.Logger.Error().Err(err).Msg("❌ Failed to start gRPC server")
		return err
	}
	s.Config.Logger.Info().Msg("✅ gRPC server shut down gracefully.")
	return nil
}

func (s *GrpcServer) Shutdown() {
	s.Config.Logger.Info().Msg("🔌 Shutting down gRPC server...")
	s.grpcServer.GracefulStop()
}
