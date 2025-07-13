package auth

import (
	"context"
	"strings"

	"deadalus-orch/server/internal/infrastructure/server/common"
	pb "deadalus-orch/server/internal/infrastructure/server/grpc/proto/pb/auth"
	bo "deadalus-orch/server/internal/usecase/business-logic"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// AuthService implements the gRPC AuthService.
type AuthService struct {
	pb.UnimplementedAuthServiceServer
	Config *common.ServerConfing // Assuming ServerConfig provides JWT key and duration
	AuthBO *bo.AuthBO
}

// NewAuthService creates a new AuthService.
func NewAuthService(cfg *common.ServerConfing, authBO *bo.AuthBO) *AuthService {
	return &AuthService{
		Config: cfg,
		AuthBO: authBO,
	}
}

// Login handles the gRPC Login request.
func (s *AuthService) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	if req.UsernameOrEmail == "" || req.Password == "" {
		s.Config.Logger.Warn().Msg("Login attempt with empty username/email or password")
		return nil, status.Errorf(codes.InvalidArgument, "username/email and password are required")
	}

	tokenString, err := s.AuthBO.Login(ctx, req.UsernameOrEmail, req.Password)
	if err != nil {
		s.Config.Logger.Error().Err(err).Str("username", req.UsernameOrEmail).Msg("Login failed")
		return nil, status.Errorf(codes.Internal, "login failed: %v", err)
	}

	s.Config.Logger.Info().Str("username", req.UsernameOrEmail).Msg("User logged in successfully via gRPC and session registered")
	return &pb.LoginResponse{
		Message: "Login successful",
		Token:   tokenString,
	}, nil
}

// Logout handles the gRPC Logout request.
func (s *AuthService) Logout(ctx context.Context, req *pb.LogoutRequest) (*pb.LogoutResponse, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		s.Config.Logger.Warn().Msg("Logout attempt without metadata")
		return nil, status.Errorf(codes.Unauthenticated, "missing metadata")
	}

	authHeaders := md.Get("authorization")
	if len(authHeaders) == 0 {
		s.Config.Logger.Warn().Msg("Logout attempt without authorization header")
		return nil, status.Errorf(codes.Unauthenticated, "authorization header missing")
	}

	token := authHeaders[0]
	if err := s.AuthBO.Logout(ctx, token); err != nil {
		s.Config.Logger.Error().Err(err).Msg("Failed removing current session during gRPC logout")
		return nil, status.Errorf(codes.Internal, "failed removing current session: %v", err)
	}

	s.Config.Logger.Info().Msg("User logged out successfully via gRPC")
	return &pb.LogoutResponse{
		Message: "Logout successful",
	}, nil
}
