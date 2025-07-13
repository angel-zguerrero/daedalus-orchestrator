package business_logic

import (
	"bytes"
	"context"
	"encoding/gob"
	"errors"
	"strings"
	"time"

	"deadalus-orch/server/internal/infrastructure/dragonboat"
	"deadalus-orch/server/internal/pkg/config"
	"deadalus-orch/server/internal/pkg/utils"
	commands "deadalus-orch/server/internal/usecase/command"
	auth_command "deadalus-orch/server/internal/usecase/command/auth"
	general_command "deadalus-orch/server/internal/usecase/command/general"

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

	queryCommand := &general_command.Query_Command{
		Command: &general_command.Repository_Command{
			CMD: loginCmd,
		},
		Now: time.Now().UnixNano(),
	}

	raftCtx, cancel := context.WithTimeout(ctx, config.GlobalConfiguration.ApiRaftTimeout)
	defer cancel()
	result, err := bo.MasterNode.Read(raftCtx, *queryCommand)
	if err != nil {
		bo.Logger.Error().Err(err).Str("username", usernameOrEmail).Msg("Login command execution failed")
		return "", err
	}

	buf := bytes.NewBuffer(result.([]byte))
	dec := gob.NewDecoder(buf)
	parsedResult := &commands.CommandResult{}
	if err := dec.Decode(parsedResult); err != nil {
		bo.Logger.Error().Err(err).Str("username", usernameOrEmail).Msg("Login command returned unexpected result type (decode)")
		return "", err
	}

	if parsedResult.Error != "" {
		bo.Logger.Error().Str("username", usernameOrEmail).Str("error", parsedResult.Error).Msg("Login command returned an error")
		return "", errors.New(parsedResult.Error)
	}

	loggedIn, ok := parsedResult.Result.(bool)
	if !ok {
		bo.Logger.Error().Str("username", usernameOrEmail).Msg("Login command returned unexpected result type (bool assertion)")
		return "", errors.New("Login command returned unexpected result type (bool assertion)")
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

	fsmCmd := general_command.FSM_Command{
		Now:  utils.GetNowInInt(),
		Type: general_command.REPOSITORY_COMMAND,
		CMD:  registerSessionCmd,
	}

	writeCtx, writeCancel := context.WithTimeout(ctx, config.GlobalConfiguration.ApiRaftTimeout)
	defer writeCancel()

	_, err = bo.MasterNode.Write(writeCtx, fsmCmd)
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

	fsmCmd := general_command.FSM_Command{
		Now:  utils.GetNowInInt(),
		Type: general_command.REPOSITORY_COMMAND,
		CMD:  removeSessionCmd,
	}

	writeCtx, writeCancel := context.WithTimeout(ctx, config.GlobalConfiguration.ApiRaftTimeout)
	defer writeCancel()

	_, err := bo.MasterNode.Write(writeCtx, fsmCmd)
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
