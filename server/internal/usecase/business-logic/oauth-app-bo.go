package business_logic

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"deadalus-orch/server/internal/infrastructure/dragonboat"
	"deadalus-orch/server/internal/pkg/config"
	oauth_command "deadalus-orch/server/internal/usecase/command/oauth"
	"deadalus-orch/shared/models"

	"github.com/rs/zerolog"
)

type OAuthAppBO struct {
	MasterNode *dragonboat.RaftNode
	Logger     *zerolog.Logger
}

func NewOAuthAppBO(masterNode *dragonboat.RaftNode, logger *zerolog.Logger) *OAuthAppBO {
	return &OAuthAppBO{
		MasterNode: masterNode,
		Logger:     logger,
	}
}

// CreateApp registers a new OAuth application and returns the initial credentials
func (bo *OAuthAppBO) CreateApp(ctx context.Context, req models.CreateOAuthAppRequest, tenantID string) (*models.OAuthAppCredentialsResponse, error) {
	// Generate random ClientID
	clientIDBytes := make([]byte, 16)
	_, err := rand.Read(clientIDBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ClientID: %w", err)
	}
	clientID := hex.EncodeToString(clientIDBytes)

	// Generate random ClientSecret
	secretBytes := make([]byte, 32)
	_, err = rand.Read(secretBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ClientSecret: %w", err)
	}
	clientSecret := base64.RawURLEncoding.EncodeToString(secretBytes)

	// Hash ClientSecret
	hash, err := bcrypt.GenerateFromPassword([]byte(clientSecret), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash ClientSecret: %w", err)
	}

	newID := strings.ReplaceAll(uuid.New().String(), "-", "")

	app := models.OAuthApp{
		ID:               newID,
		ClientID:         clientID,
		ClientSecretHash: string(hash),
		Name:             req.Name,
		Description:      req.Description,
		TenantID:         tenantID,
		AllowedScopes:    req.AllowedScopes,
	}

	cmd := &oauth_command.CreateOAuthAppCommand{
		App: app,
	}

	_, err = dragonboat.ExecuteRepositoryCommand[interface{}](
		bo.MasterNode,
		ctx,
		cmd,
		config.GlobalConfiguration.ApiRaftTimeout,
		*bo.Logger,
		"create oauth app",
	)
	if err != nil {
		return nil, err
	}

	return &models.OAuthAppCredentialsResponse{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Name:         req.Name,
		Scopes:       req.AllowedScopes,
	}, nil
}

// RotateSecret generates a new secret for an existing app and updates it
func (bo *OAuthAppBO) RotateSecret(ctx context.Context, appID string) (*models.OAuthAppCredentialsResponse, error) {
	// Generate random ClientSecret
	secretBytes := make([]byte, 32)
	_, err := rand.Read(secretBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ClientSecret: %w", err)
	}
	clientSecret := base64.RawURLEncoding.EncodeToString(secretBytes)

	// Hash ClientSecret
	hash, err := bcrypt.GenerateFromPassword([]byte(clientSecret), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash ClientSecret: %w", err)
	}

	cmd := &oauth_command.RotateOAuthSecretCommand{
		AppID:         appID,
		NewSecretHash: string(hash),
	}

	_, err = dragonboat.ExecuteRepositoryCommand[interface{}](
		bo.MasterNode,
		ctx,
		cmd,
		config.GlobalConfiguration.ApiRaftTimeout,
		*bo.Logger,
		"rotate oauth app secret",
	)
	if err != nil {
		return nil, err
	}

	return &models.OAuthAppCredentialsResponse{
		ClientSecret: clientSecret,
	}, nil
}
