package grpc_server

import (
	"fmt"
	"net"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"

	"deadalus-orch/server/internal/pkg/config"
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

func NewGrpcServer(cfg *common.RestServerConfing) (*GrpcServer, error) {
	if cfg.MasterNode == nil {
		cfg.Logger.Fatal().Msg("gRPC: Raft node cannot be nil")
	}

	listenAddr := fmt.Sprintf("%s:%d", config.GlobalConfiguration.GrpcServerListenAddrHost, config.GlobalConfiguration.GrpcServerListenAddrPort)
	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return nil, fmt.Errorf("gRPC: failed to listen on %s: %w", listenAddr, err)
	}

	otelHandler := otelgrpc.NewServerHandler()
	authInterceptor := UnaryAuthInterceptor(cfg.MasterNode, cfg.Logger, cfg.JwtKey)
	rateLimitInterceptor := UnaryRateLimitInterceptor(cfg.MasterNode, cfg.Logger, "token", time.Minute, 20)

	server := grpc.NewServer(
		grpc.StatsHandler(otelHandler),
		grpc.ChainUnaryInterceptor(authInterceptor, rateLimitInterceptor),
	)

	// Registrar servicios
	pb.RegisterMetricsServiceServer(server, healthmetrics.NewMetricsServer())
	pbT.RegisterTenantServiceServer(server, tenant.NewTenantService(cfg))

	return &GrpcServer{
		Config:     cfg,
		grpcServer: server,
		listener:   lis,
	}, nil
}

func (s *GrpcServer) Start() error {
	// The address is now determined in NewGrpcServer using GlobalConfiguration
	// and the listener is already created with this address.
	// We log the address the listener is actually using.
	actualAddr := s.listener.Addr().String()
	s.Config.Logger.Info().Str("address", actualAddr).Msg("🚀 Starting gRPC server...")

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
