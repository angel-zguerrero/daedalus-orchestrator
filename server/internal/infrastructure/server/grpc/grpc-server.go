package grpc_server

import (
	"fmt"
	"net"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"

	"deadalus-orch/server/internal/infrastructure/server/common"
	"deadalus-orch/server/internal/infrastructure/server/grpc/auth"     // Import new auth service
	"deadalus-orch/server/internal/infrastructure/server/grpc/exchange" // Import new exchange service
	healthmetrics "deadalus-orch/server/internal/infrastructure/server/grpc/metrics"
	"deadalus-orch/server/internal/infrastructure/server/grpc/nodescheduler" // Import nodescheduler service
	pb "deadalus-orch/server/internal/infrastructure/server/grpc/proto/health/metrics"
	pbAuth "deadalus-orch/server/internal/infrastructure/server/grpc/proto/pb/auth"                   // Import new auth pb
	pbExchange "deadalus-orch/server/internal/infrastructure/server/grpc/proto/pb/exchange"           // Import new exchange pb
	pbNodeScheduler "deadalus-orch/server/internal/infrastructure/server/grpc/proto/pb/nodescheduler" // Import nodescheduler pb
	pbQueue "deadalus-orch/server/internal/infrastructure/server/grpc/proto/pb/queue"                 // Import new queue pb
	pbT "deadalus-orch/server/internal/infrastructure/server/grpc/proto/pb/tenant"
	"deadalus-orch/server/internal/infrastructure/server/grpc/queue" // Import new queue service
	"deadalus-orch/server/internal/infrastructure/server/grpc/tenant"
	"deadalus-orch/server/internal/pkg/config"
	bo "deadalus-orch/server/internal/usecase/business-logic"
)

type GrpcServer struct {
	Config     *common.ServerConfing
	grpcServer *grpc.Server
	listener   net.Listener
}

func NewGrpcServer(cfg *common.ServerConfing) (*GrpcServer, error) {
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
	// Register new AuthService
	authBO := bo.NewAuthBO(cfg.MasterNode, cfg.JwtKey, cfg.JwtDuration, &cfg.Logger)
	authSvc := auth.NewAuthService(cfg, authBO)
	pbAuth.RegisterAuthServiceServer(server, authSvc)

	// Register new ExchangeService
	exchangeSvc := exchange.NewExchangeService(cfg)
	pbExchange.RegisterExchangeServiceServer(server, exchangeSvc)

	// Register new QueueService
	queueSvc := queue.NewQueueService(cfg)
	pbQueue.RegisterQueueServiceServer(server, queueSvc)

	// Register new NodeSchedulerService
	nodeSchedulerSvc := nodescheduler.NewNodeSchedulerService(cfg)
	pbNodeScheduler.RegisterNodeSchedulerServiceServer(server, nodeSchedulerSvc)

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
	s.Config.Logger.Info().Str("address", actualAddr).Msg("🚀 Starting gRPC server.")

	if err := s.grpcServer.Serve(s.listener); err != nil {
		s.Config.Logger.Error().Err(err).Msg("❌ Failed to start gRPC server")
		return err
	}
	s.Config.Logger.Info().Msg("✅ gRPC server shut down gracefully.")
	return nil
}

func (s *GrpcServer) Shutdown() {
	s.Config.Logger.Info().Msg("🔌 Shutting down gRPC server.")
	s.grpcServer.GracefulStop()
}
