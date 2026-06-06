package business_logic

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"deadalus-orch/server/internal/infrastructure/dragonboat"
	"deadalus-orch/server/internal/pkg/config"
	auth_command "deadalus-orch/server/internal/usecase/command/auth"

	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog"
)

type AuthBO struct {
	MasterNode  *dragonboat.RaftNode
	JwtKey      []byte
	JwtDuration time.Duration
	Logger      *zerolog.Logger
}

func NewAuthBO(masterNode *dragonboat.RaftNode, jwtKey []byte, jwtDuration time.Duration, logger *zerolog.Logger) *AuthBO {
	return &AuthBO{
		MasterNode:  masterNode,
		JwtKey:      jwtKey,
		JwtDuration: jwtDuration,
		Logger:      logger,
	}
}

func (bo *AuthBO) Login(ctx context.Context, usernameOrEmail, password string) (string, error) {
	loginCmd := &auth_command.LoginCommand{
		UsernameOrEmail: usernameOrEmail,
		Password:        password,
	}

	loggedIn, err := dragonboat.ExecuteRepositoryQuery[bool](
		bo.MasterNode,
		ctx,
		loginCmd,
		config.GlobalConfiguration.ApiRaftTimeout,
		*bo.Logger,
		"login",
	)
	if err != nil {
		return "", fmt.Errorf("login command execution failed: %w", err)
	}

	if !loggedIn {
		bo.Logger.Warn().Str("username", usernameOrEmail).Msg("Login attempt failed: invalid credentials")
		return "", errors.New("Login attempt failed: invalid credentials")
	}

	tokenString, err := bo.generateJWT(usernameOrEmail)
	if err != nil {
		bo.Logger.Error().Err(err).Str("username", usernameOrEmail).Msg("Failed to generate JWT token during login")
		return "", err
	}

	registerSessionCmd := &auth_command.RegisterSessionCommand{
		JWTToken: tokenString,
		JWTKey:   bo.JwtKey,
	}

	_, err = dragonboat.ExecuteRepositoryCommand[interface{}](
		bo.MasterNode,
		ctx,
		registerSessionCmd,
		config.GlobalConfiguration.ApiRaftTimeout,
		*bo.Logger,
		"register session after login",
	)
	if err != nil {
		bo.Logger.Error().Err(err).Str("username", usernameOrEmail).Msg("Failed to register session after login")
		return "", err
	}

	bo.Logger.Info().Str("username", usernameOrEmail).Msg("User logged in successfully and session registered")
	return tokenString, nil
}

func (bo *AuthBO) Logout(ctx context.Context, token string) error {
	token = strings.TrimPrefix(token, "Bearer ")
	if token == "" {
		return nil
	}

	removeSessionCmd := &auth_command.RemoveSessionCommand{
		JWTToken: token,
		JWTKey:   bo.JwtKey,
	}

	_, err := dragonboat.ExecuteRepositoryCommand[interface{}](
		bo.MasterNode,
		ctx,
		removeSessionCmd,
		config.GlobalConfiguration.ApiRaftTimeout,
		*bo.Logger,
		"remove session during logout",
	)
	if err != nil {
		bo.Logger.Error().Err(err).Msg("Failed removing current session during logout")
		return err
	}

	bo.Logger.Info().Msg("User logged out successfully")
	return nil
}

func (bo *AuthBO) generateJWT(username string) (string, error) {
	expirationTime := time.Now().Add(bo.JwtDuration)
	claims := &jwt.RegisteredClaims{
		Subject:   username,
		ExpiresAt: jwt.NewNumericDate(expirationTime),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(bo.JwtKey)
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

func (bo *AuthBO) CheckRootExists(ctx context.Context) (bool, error) {
	cmd := &auth_command.CheckRootUserExistsCommand{}
	exists, err := dragonboat.ExecuteRepositoryQuery[bool](
		bo.MasterNode,
		ctx,
		cmd,
		config.GlobalConfiguration.ApiRaftTimeout,
		*bo.Logger,
		"check root user exists",
	)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func (bo *AuthBO) SetupRootUser(ctx context.Context, username, password string) error {
	cmd := &auth_command.SetupRootUserCommand{
		Username: username,
		Password: password,
	}
	_, err := dragonboat.ExecuteRepositoryCommand[interface{}](
		bo.MasterNode,
		ctx,
		cmd,
		config.GlobalConfiguration.ApiRaftTimeout,
		*bo.Logger,
		"setup root user",
	)
	return err
}
