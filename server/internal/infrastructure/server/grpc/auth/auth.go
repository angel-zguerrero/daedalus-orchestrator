package auth

import (
	"bytes"
	"context"
	"encoding/gob"
	"strings"
	"time"

	"deadalus-orch/server/internal/infrastructure/dragonboat"
	"deadalus-orch/server/internal/infrastructure/server/common"
	pb "deadalus-orch/server/internal/infrastructure/server/grpc/proto/pb/auth"
	"deadalus-orch/server/internal/pkg/config"
	"deadalus-orch/server/internal/pkg/utils"
	commands "deadalus-orch/server/internal/usecase/command"

	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// AuthService implements the gRPC AuthService.
type AuthService struct {
	pb.UnimplementedAuthServiceServer
	MasterNode *dragonboat.RaftNode
	Config     *common.ServerConfing // Assuming ServerConfig provides JWT key and duration
}

// NewAuthService creates a new AuthService.
func NewAuthService(masterNode *dragonboat.RaftNode, cfg *common.ServerConfing) *AuthService {
	return &AuthService{
		MasterNode: masterNode,
		Config:     cfg,
	}
}

// Login handles the gRPC Login request.
func (s *AuthService) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	if req.UsernameOrEmail == "" || req.Password == "" {
		s.Config.Logger.Warn().Msg("Login attempt with empty username/email or password")
		return nil, status.Errorf(codes.InvalidArgument, "username/email and password are required")
	}

	loginCmd := &commands.LoginCommand{
		UsernameOrEmail: req.UsernameOrEmail,
		Password:        req.Password,
	}

	queryCommand := &commands.Query_Command{
		Command: &commands.Repository_Command{
			CMD: loginCmd,
		},
		Now: time.Now().UnixNano(),
	}

	raftCtx, cancel := context.WithTimeout(context.Background(), config.GlobalConfiguration.ApiRaftTimeout)
	defer cancel()
	result, err := s.MasterNode.Read(raftCtx, *queryCommand)
	if err != nil {
		s.Config.Logger.Error().Err(err).Str("username", req.UsernameOrEmail).Msg("Login command execution failed")
		return nil, status.Errorf(codes.Internal, "login failed: %v", err)
	}

	buf := bytes.NewBuffer(result.([]byte))
	dec := gob.NewDecoder(buf)
	parsedResult := &commands.CommandResult{}
	if err := dec.Decode(parsedResult); err != nil {
		s.Config.Logger.Error().Err(err).Str("username", req.UsernameOrEmail).Msg("Login command returned unexpected result type (decode)")
		return nil, status.Errorf(codes.Internal, "login failed due to an internal error (decode)")
	}

	if parsedResult.Error != "" {
		s.Config.Logger.Error().Str("username", req.UsernameOrEmail).Str("error", parsedResult.Error).Msg("Login command returned an error")
		// Distinguish between invalid credentials and other errors if possible, based on parsedResult.Error content.
		// For now, treating all errors from command as internal.
		return nil, status.Errorf(codes.Internal, "login failed: %s", parsedResult.Error)
	}

	loggedIn, ok := parsedResult.Result.(bool)
	if !ok {
		s.Config.Logger.Error().Str("username", req.UsernameOrEmail).Msg("Login command returned unexpected result type (bool assertion)")
		return nil, status.Errorf(codes.Internal, "login failed due to an internal error (type assertion)")
	}

	if !loggedIn {
		s.Config.Logger.Warn().Str("username", req.UsernameOrEmail).Msg("Login attempt failed: invalid credentials")
		return nil, status.Errorf(codes.Unauthenticated, "invalid username or password")
	}

	tokenString, err := s.generateJWT(req.UsernameOrEmail)
	if err != nil {
		s.Config.Logger.Error().Err(err).Str("username", req.UsernameOrEmail).Msg("Failed to generate JWT token during login")
		return nil, status.Errorf(codes.Internal, "login successful, but failed to generate token: %v", err)
	}

	registerSessionCmd := &commands.RegisterSessionCommand{
		JWTToken: tokenString,
		JWTKey:   s.Config.JwtKey,
	}

	fsmCmd := commands.FSM_Command{
		Now:  utils.GetNowInInt(),
		Type: commands.REPOSITORY_COMMAND,
		CMD:  registerSessionCmd,
	}

	writeCtx, writeCancel := context.WithTimeout(context.Background(), config.GlobalConfiguration.ApiRaftTimeout)
	defer writeCancel()

	_, err = s.MasterNode.Write(writeCtx, fsmCmd)
	if err != nil {
		s.Config.Logger.Error().Err(err).Str("username", req.UsernameOrEmail).Msg("Failed to register session after login")
		return nil, status.Errorf(codes.Internal, "failed to register session after login: %v", err)
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

	authHeader := authHeaders[0]
	const prefix = "Bearer "
	if !strings.HasPrefix(authHeader, prefix) {
		s.Config.Logger.Warn().Msg("Logout attempt with invalid authorization header format")
		return nil, status.Errorf(codes.Unauthenticated, "invalid authorization header format")
	}

	token := strings.TrimPrefix(authHeader, prefix)
	if token == "" {
		s.Config.Logger.Warn().Msg("Logout attempt with empty bearer token")
		return nil, status.Errorf(codes.Unauthenticated, "empty bearer token")
	}

	// Optional: Validate token structure or expiry locally before hitting Raft, though CheckSessionExistsCommand in middleware should handle expired/invalid.
	// For logout, the primary action is to invalidate the session.

	removeSessionCmd := &commands.RemoveSessionCommand{
		JWTToken: token,
		JWTKey:   s.Config.JwtKey, // Assuming JwtKey is accessible via s.Config
	}

	fsmCmd := commands.FSM_Command{
		Now:  utils.GetNowInInt(),
		Type: commands.REPOSITORY_COMMAND,
		CMD:  removeSessionCmd,
	}

	writeCtx, writeCancel := context.WithTimeout(context.Background(), config.GlobalConfiguration.ApiRaftTimeout)
	defer writeCancel()

	_, err := s.MasterNode.Write(writeCtx, fsmCmd)
	if err != nil {
		s.Config.Logger.Error().Err(err).Msg("Failed removing current session during gRPC logout")
		return nil, status.Errorf(codes.Internal, "failed removing current session: %v", err)
	}

	s.Config.Logger.Info().Msg("User logged out successfully via gRPC")
	return &pb.LogoutResponse{
		Message: "Logout successful",
	}, nil
}

// generateJWT generates a new JWT token.
func (s *AuthService) generateJWT(username string) (string, error) {
	expirationTime := time.Now().Add(s.Config.JwtDuration) // Assuming JwtDuration is accessible via s.Config
	claims := &jwt.RegisteredClaims{
		Subject:   username,
		ExpiresAt: jwt.NewNumericDate(expirationTime),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.Config.JwtKey) // Assuming JwtKey is accessible via s.Config
	if err != nil {
		return "", err
	}
	return tokenString, nil
}
