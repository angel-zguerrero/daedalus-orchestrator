package business_logic

import (
	"context"
	"fmt"
	"strings"
	"time"

	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	"deadalus-orch/server/internal/infrastructure/server/common"
	"deadalus-orch/server/internal/pkg/config"
	oauth_command "deadalus-orch/server/internal/usecase/command/oauth"
	models "deadalus-orch/shared/models"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// OAuthClaims extends RegisteredClaims with Daedalus-specific OAuth fields.
// TokenType is always "oauth" for machine-issued tokens.
// These tokens are validated by unifiedAuthMiddleware and are STATELESS
// — they are NOT registered in the KV session store.
type OAuthClaims struct {
	jwt.RegisteredClaims
	Scopes    []string `json:"scopes"`
	TenantID  string   `json:"tenant_id"`
	TokenType string   `json:"token_type"` // Always "oauth"
}

// OAuthAppBO provides business logic for managing OAuth 2.0 Service Accounts
// and issuing machine-to-machine access tokens.
type OAuthAppBO struct {
	Config   *common.ServerConfing
	TokenTTL time.Duration
}

func NewOAuthAppBO(config *common.ServerConfing, tokenTTL time.Duration) *OAuthAppBO {
	return &OAuthAppBO{Config: config, TokenTTL: tokenTTL}
}

// IssueToken validates the client credentials and returns a signed HS256 JWT access token.
//
// Security properties:
//   - Uses a generic error message for both wrong client ID and wrong secret
//     to prevent client enumeration.
//   - Does NOT write a session entry to the KV store. OAuth tokens are fully
//     stateless — validation is signature + expiry only.
func (bo *OAuthAppBO) IssueToken(ctx context.Context, clientID, clientSecret string, now time.Time) (string, error) {
	getCmd := &oauth_command.GetOAuthAppByClientIDCommand{ClientID: clientID}
	app, err := dragonboat.ExecuteRepositoryQuery[models.OAuthApp](
		bo.Config.MasterNode, ctx, getCmd,
		config.GlobalConfiguration.ApiRaftTimeout,
		bo.Config.Logger,
		"get oauth app by clientID",
	)
	if err != nil {
		return "", fmt.Errorf("invalid client credentials")
	}
	if app.ClientID == "" {
		return "", fmt.Errorf("invalid client credentials")
	}

	// bcrypt compare happens in-process (CPU-bound, no state change, no Raft round-trip needed)
	if !db.ValidateOAuthSecret(&app, clientSecret) {
		return "", fmt.Errorf("invalid client credentials")
	}

	claims := OAuthClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   app.ClientID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(bo.TokenTTL)),
			ID:        uuid.New().String(),
		},
		Scopes:    app.AllowedScopes,
		TenantID:  app.TenantID,
		TokenType: "oauth",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(bo.Config.JwtKey)
}

// CreateApp generates a new ClientID and ClientSecret, persists the app via Raft,
// and returns the plain-text secret exactly once.
//
// SECURITY CONTRACT: The second return value (plainTextSecret) must be placed
// in the HTTP response body exactly once and then discarded. Do NOT log it.
func (bo *OAuthAppBO) CreateApp(
	ctx context.Context,
	tenantID, name, description string,
	scopes []string,
	now time.Time,
) (*models.OAuthAppResponse, string, error) {
	clientID, err := db.GenerateClientID()
	if err != nil {
		return nil, "", err
	}
	clientSecret, err := db.GenerateClientSecret()
	if err != nil {
		return nil, "", err
	}

	newID := strings.ReplaceAll(uuid.New().String(), "-", "")
	cmd := &oauth_command.CreateOAuthAppCommand{
		ID:            newID,
		ClientID:      clientID,
		ClientSecret:  clientSecret, // plain text — hashed inside Execute() by the repo
		Name:          name,
		TenantID:      tenantID,
		Description:   description,
		AllowedScopes: scopes,
	}

	_, err = dragonboat.ExecuteRepositoryCommand[string](
		bo.Config.MasterNode, ctx, cmd,
		config.GlobalConfiguration.ApiRaftTimeout,
		bo.Config.Logger,
		"create oauth app",
	)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create Service Account: %w", err)
	}

	resp := &models.OAuthAppResponse{
		ID:            newID,
		ClientID:      clientID,
		Name:          name,
		TenantID:      tenantID,
		Description:   description,
		AllowedScopes: scopes,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	return resp, clientSecret, nil // clientSecret returned ONCE; discard after response
}

// ListApps returns all OAuthApps belonging to a tenant via the group index.
func (bo *OAuthAppBO) ListApps(ctx context.Context, tenantID string, now time.Time) ([]models.OAuthApp, error) {
	cmd := &oauth_command.GetOAuthAppsByTenantCommand{TenantID: tenantID}
	apps, err := dragonboat.ExecuteRepositoryQuery[[]models.OAuthApp](
		bo.Config.MasterNode, ctx, cmd,
		config.GlobalConfiguration.ApiRaftTimeout,
		bo.Config.Logger,
		"list oauth apps by tenant",
	)
	if err != nil {
		return nil, err
	}
	return apps, nil
}

// GetApp retrieves a single OAuthApp by its internal ID.
func (bo *OAuthAppBO) GetApp(ctx context.Context, appID string, now time.Time) (*models.OAuthApp, error) {
	cmd := &oauth_command.GetOAuthAppByIDCommand{ID: appID}
	app, err := dragonboat.ExecuteRepositoryQuery[models.OAuthApp](
		bo.Config.MasterNode, ctx, cmd,
		config.GlobalConfiguration.ApiRaftTimeout,
		bo.Config.Logger,
		"get oauth app by ID",
	)
	if err != nil {
		return nil, err
	}
	if app.ID == "" {
		return nil, nil // not found
	}
	return &app, nil
}

// RotateSecret generates and stores a new secret for an app via Raft.
// Returns the app (updated) and the plain-text new secret exactly once.
//
// SECURITY CONTRACT: plainTextSecret must be placed in the HTTP response
// exactly once and then discarded. Do NOT log it.
func (bo *OAuthAppBO) RotateSecret(ctx context.Context, appID string, now time.Time) (*models.OAuthApp, string, error) {
	rotateCmd := &oauth_command.RotateOAuthSecretCommand{AppID: appID}
	result, err := dragonboat.ExecuteRepositoryCommand[oauth_command.RotateOAuthSecretResult](
		bo.Config.MasterNode, ctx, rotateCmd,
		config.GlobalConfiguration.ApiRaftTimeout,
		bo.Config.Logger,
		"rotate oauth secret",
	)
	if err != nil {
		return nil, "", err
	}

	app, err := bo.GetApp(ctx, appID, now)
	return app, result.PlainSecret, err // result.PlainSecret returned ONCE; discard after response
}

// DeleteApp removes an OAuthApp by its internal ID.
func (bo *OAuthAppBO) DeleteApp(ctx context.Context, appID string, now time.Time) (bool, error) {
	cmd := &oauth_command.DeleteOAuthAppCommand{ID: appID}
	deleted, err := dragonboat.ExecuteRepositoryCommand[bool](
		bo.Config.MasterNode, ctx, cmd,
		config.GlobalConfiguration.ApiRaftTimeout,
		bo.Config.Logger,
		"delete oauth app",
	)
	return deleted, err
}

// ScopesToString converts a scope slice to a space-separated string (RFC 6749 format).
func ScopesToString(scopes []string) string {
	return strings.Join(scopes, " ")
}
