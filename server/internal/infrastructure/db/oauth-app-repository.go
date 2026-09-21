package db

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	models "deadalus-orch/shared/models"

	"golang.org/x/crypto/bcrypt"
)

// bcryptCost is the work factor for hashing client secrets.
// 12 is the OWASP-recommended minimum for bcrypt.
const oauthBcryptCost = 12

// OAuthAppRepository provides CRUD operations for OAuth 2.0 Service Account applications.
// Stored in AdminFC ("admin") / AdminFCSector ("admin-sector") / schema "admin_schema",
// the same column family as User and TenantInMaster.
type OAuthAppRepository struct {
	repo *Repository[models.OAuthApp]
}

// NewOAuthAppRepository constructs an OAuthAppRepository within the given UnitOfWork.
func NewOAuthAppRepository(uow *UnitOfWork, factory IDGeneratorFactory) (*OAuthAppRepository, error) {
	if uow == nil {
		return nil, fmt.Errorf("UnitOfWork is required")
	}
	repo, err := GetRepository[models.OAuthApp](uow, AdminFC, AdminFCSector, "admin_schema", factory)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize OAuthApp repository: %w", err)
	}
	return &OAuthAppRepository{repo: repo}, nil
}

// GenerateClientID returns a cryptographically secure 32-byte hex string (64 hex chars).
// Used as the public OAuth client identifier.
func GenerateClientID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate client ID: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// GenerateClientSecret returns a cryptographically secure 48-byte hex string (96 hex chars).
//
// SECURITY CONTRACT: The caller MUST return this value exactly once in the HTTP response
// and then discard it. It must never be logged, stored, or re-transmitted.
func GenerateClientSecret() (string, error) {
	b := make([]byte, 48)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate client secret: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// HashClientSecret returns a bcrypt hash (cost=12) of the plain-text secret.
// The hash is what gets stored; the plain-text is discarded after this call.
func HashClientSecret(secret string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(secret), oauthBcryptCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash client secret: %w", err)
	}
	return string(hash), nil
}

// CreateApp hashes the ClientSecret in input, then persists a new OAuthApp.
// Returns the newly assigned entity ID.
//
// SECURITY: input.ClientSecret must be the plain-text secret. It is hashed here
// and never stored in its original form.
func (r *OAuthAppRepository) CreateApp(input models.CreateOAuthApp) (string, error) {
	hash, err := HashClientSecret(input.ClientSecret)
	if err != nil {
		return "", err
	}
	app := &models.OAuthApp{
		ID:               input.ID,
		ClientID:         input.ClientID,
		ClientSecretHash: hash,
		Name:             input.Name,
		TenantID:         input.TenantID,
		Description:      input.Description,
		AllowedScopes:    input.AllowedScopes,
		CreatedAt:        input.Now,
		UpdatedAt:        input.Now,
	}
	return r.repo.Create(app, input.Now)
}

// GetAppByClientID looks up a single OAuthApp by its public ClientID.
// Returns nil, nil when no app is found (not an error).
func (r *OAuthAppRepository) GetAppByClientID(clientID string, now time.Time) (*models.OAuthApp, error) {
	return r.repo.FindByField("ClientID", clientID, now)
}

// GetAppsByTenant returns all OAuthApps belonging to a given tenant using the
// group index on TenantID for O(1) ID lookup.
func (r *OAuthAppRepository) GetAppsByTenant(tenantID string, now time.Time) ([]models.OAuthApp, error) {
	return r.repo.FindByGroup("TenantID", tenantID, now)
}

// GetAppByID retrieves a single OAuthApp by its internal primary-key ID.
// Returns nil, nil when no app is found.
func (r *OAuthAppRepository) GetAppByID(id string, now time.Time) (*models.OAuthApp, error) {
	return r.repo.FindByField("ID", id, now)
}

// RotateSecret generates a new client secret, hashes it, persists the updated app,
// and returns the plain-text new secret exactly once.
//
// SECURITY CONTRACT: The caller MUST return the returned plain-text secret exactly once
// in the HTTP response and then discard it.
func (r *OAuthAppRepository) RotateSecret(id string, now time.Time) (string, error) {
	app, err := r.GetAppByID(id, now)
	if err != nil {
		return "", fmt.Errorf("error retrieving app for secret rotation: %w", err)
	}
	if app == nil {
		return "", fmt.Errorf("app not found: %s", id)
	}

	newSecret, err := GenerateClientSecret()
	if err != nil {
		return "", err
	}
	newHash, err := HashClientSecret(newSecret)
	if err != nil {
		return "", err
	}

	app.ClientSecretHash = newHash
	app.UpdatedAt = now

	if _, err := r.repo.Update(app, now); err != nil {
		return "", fmt.Errorf("failed to persist rotated secret: %w", err)
	}
	return newSecret, nil
}

// ValidateSecret returns true if the provided plain-text secret matches the stored bcrypt hash.
// It is constant-time to prevent timing attacks.
func (r *OAuthAppRepository) ValidateSecret(app *models.OAuthApp, plainSecret string) bool {
	return bcrypt.CompareHashAndPassword([]byte(app.ClientSecretHash), []byte(plainSecret)) == nil
}

// ValidateOAuthSecret is a package-level helper for validating a client secret in-process
// without requiring a repository instance or a Raft round-trip.
// Used by OAuthAppBO.IssueToken to perform the bcrypt compare after fetching the app via Raft.
func ValidateOAuthSecret(app *models.OAuthApp, plainSecret string) bool {
	return bcrypt.CompareHashAndPassword([]byte(app.ClientSecretHash), []byte(plainSecret)) == nil
}

// DeleteApp removes an OAuthApp by its internal primary-key ID.
// Returns (true, nil) on success, (false, nil) when the app does not exist.
func (r *OAuthAppRepository) DeleteApp(id string, now time.Time) (bool, error) {
	return r.repo.Delete(id, now)
}
