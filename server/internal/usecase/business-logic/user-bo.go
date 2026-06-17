package business_logic

import (
	"context"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/google/uuid"

	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	"deadalus-orch/server/internal/pkg/config"
	user_command "deadalus-orch/server/internal/usecase/command/user"
	"deadalus-orch/shared/models"

	"github.com/rs/zerolog"
)

type UserBO struct {
	MasterNode *dragonboat.RaftNode
	Logger     *zerolog.Logger
}

func NewUserBO(masterNode *dragonboat.RaftNode, logger *zerolog.Logger) *UserBO {
	return &UserBO{
		MasterNode: masterNode,
		Logger:     logger,
	}
}

func (bo *UserBO) GetUsers(ctx context.Context, filter string, cursor string, limit int) (*db.FindResult[models.User], error) {
	cmd := &user_command.GetUsersCommand{
		Filter: filter,
		Cursor: cursor,
		Limit:  limit,
	}

	result, err := dragonboat.ExecuteRepositoryQuery[db.FindResult[models.User]](
		bo.MasterNode,
		ctx,
		cmd,
		config.GlobalConfiguration.ApiRaftTimeout,
		*bo.Logger,
		"get users",
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get users: %w", err)
	}

	return &result, nil
}

func (bo *UserBO) CreateUser(ctx context.Context, input models.CreateUser) (string, error) {
	if input.ID == "" {
		input.ID = strings.ReplaceAll(uuid.New().String(), "-", "")
	}

	var hashStr string
	if input.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
		if err != nil {
			return "", fmt.Errorf("failed to hash password: %w", err)
		}
		hashStr = string(hash)
	}

	cmd := &user_command.CreateUserCommand{
		ID:         input.ID,
		Username:   input.Username,
		Email:      input.Email,
		Password:   hashStr,
		IsRootUser: input.IsRootUser,
	}

	result, err := dragonboat.ExecuteRepositoryCommand[string](
		bo.MasterNode,
		ctx,
		cmd,
		config.GlobalConfiguration.ApiRaftTimeout,
		*bo.Logger,
		"create user",
	)

	if err != nil {
		return "", fmt.Errorf("failed to create user: %w", err)
	}

	return result, nil
}

func (bo *UserBO) UpdateUser(ctx context.Context, input models.UpdateUser) (bool, error) {
	var hashStr string
	bo.Logger.Info().Msgf("UserBO.UpdateUser received input: Username=%s, Email=%s, PasswordLength=%d", input.Username, input.Email, len(input.Password))
	if input.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
		if err != nil {
			return false, fmt.Errorf("failed to hash password: %w", err)
		}
		hashStr = string(hash)
		bo.Logger.Info().Msgf("UserBO.UpdateUser hashed password, length: %d", len(hashStr))
	} else {
		bo.Logger.Info().Msg("UserBO.UpdateUser: input.Password is empty, no hash generated")
	}

	bo.Logger.Debug().Msgf("cmd new password: %+v", hashStr)

	cmd := &user_command.UpdateUserCommand{
		ID:       input.ID,
		Username: input.Username,
		Email:    input.Email,
		Password: hashStr,
	}

	result, err := dragonboat.ExecuteRepositoryCommand[bool](
		bo.MasterNode,
		ctx,
		cmd,
		config.GlobalConfiguration.ApiRaftTimeout,
		*bo.Logger,
		"update user",
	)

	if err != nil {
		return false, fmt.Errorf("failed to update user: %w", err)
	}

	return result, nil
}

func (bo *UserBO) DeleteUser(ctx context.Context, username string) (bool, error) {
	cmd := &user_command.DeleteUserCommand{
		Username: username,
	}

	result, err := dragonboat.ExecuteRepositoryCommand[bool](
		bo.MasterNode,
		ctx,
		cmd,
		config.GlobalConfiguration.ApiRaftTimeout,
		*bo.Logger,
		"delete user",
	)

	if err != nil {
		return false, fmt.Errorf("failed to delete user: %w", err)
	}

	return result, nil
}
