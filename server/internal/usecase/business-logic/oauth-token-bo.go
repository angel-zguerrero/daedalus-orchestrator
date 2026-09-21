package business_logic

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
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

type OAuthTokenBO struct {
	MasterNode *dragonboat.RaftNode
	Logger     *zerolog.Logger
}

func NewOAuthTokenBO(masterNode *dragonboat.RaftNode, logger *zerolog.Logger) *OAuthTokenBO {
	return &OAuthTokenBO{
		MasterNode: masterNode,
		Logger:     logger,
	}
}

func (bo *OAuthTokenBO) GenerateToken(ctx context.Context, req models.OAuthTokenRequest) (*models.OAuthTokenResponse, error) {
	if req.GrantType != "client_credentials" {
		return nil, errors.New("unsupported grant_type")
	}

	// 1. Find App by ClientID
	findCmd := &oauth_command.FindOAuthAppCommand{
		ClientID: req.ClientID,
	}

	result, err := dragonboat.ExecuteRepositoryQuery[*models.OAuthApp](
		bo.MasterNode,
		ctx,
		findCmd,
		config.GlobalConfiguration.ApiRaftTimeout,
		*bo.Logger,
		"find oauth app",
	)
	if err != nil {
		return nil, fmt.Errorf("error finding app: %w", err)
	}
	if result == nil {
		return nil, errors.New("invalid client_id or client_secret")
	}
	app := result

	// 2. Verify Secret
	err = bcrypt.CompareHashAndPassword([]byte(app.ClientSecretHash), []byte(req.ClientSecret))
	if err != nil {
		bo.Logger.Warn().Str("clientID", req.ClientID).Msg("Invalid client_secret provided")
		return nil, errors.New("invalid client_id or client_secret")
	}

	// 3. Validate Scopes (if provided, they must be subset of AllowedScopes)
	var grantedScopes []string
	if req.Scope != "" {
		requestedScopes := strings.Split(req.Scope, " ")
		for _, rs := range requestedScopes {
			allowed := false
			for _, as := range app.AllowedScopes {
				if rs == as {
					allowed = true
					break
				}
			}
			if !allowed {
				return nil, fmt.Errorf("invalid scope requested: %s", rs)
			}
			grantedScopes = append(grantedScopes, rs)
		}
	} else {
		grantedScopes = app.AllowedScopes // default to all allowed
	}

	// 4. Generate opaque token
	rawToken := uuid.New().String() + "-" + uuid.New().String()

	// Hash it for storage using SHA-256
	hasher := sha256.New()
	hasher.Write([]byte(rawToken))
	tokenHash := hex.EncodeToString(hasher.Sum(nil))

	ttlSeconds := int64(3600) // 1 hour TTL
	newID := strings.ReplaceAll(uuid.New().String(), "-", "")

	tokenRecord := models.OAuthToken{
		ID:        newID,
		TokenHash: tokenHash,
		ClientID:  app.ClientID,
		Scopes:    grantedScopes,
		TTL:       ttlSeconds,
	}

	registerCmd := &oauth_command.RegisterOAuthTokenCommand{
		Token: tokenRecord,
	}

	_, err = dragonboat.ExecuteRepositoryCommand[interface{}](
		bo.MasterNode,
		ctx,
		registerCmd,
		config.GlobalConfiguration.ApiRaftTimeout,
		*bo.Logger,
		"register oauth token",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to save token: %w", err)
	}

	return &models.OAuthTokenResponse{
		AccessToken: rawToken,
		TokenType:   "Bearer",
		ExpiresIn:   ttlSeconds,
		Scope:       strings.Join(grantedScopes, " "),
	}, nil
}
