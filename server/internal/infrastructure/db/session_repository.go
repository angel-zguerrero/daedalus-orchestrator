package db

import (
	"fmt"
	"time"

	"deadalus-orch/shared/models"

	"github.com/golang-jwt/jwt/v5"
)

// MasterEventFC defines the column family for master event data, presumably including sessions.
// This should be defined alongside other column family constants like DefaultFC, AdminFC, MetaFC.
// For now, defining it here. Ideally, it would be in constants.go.
const MasterEventFC = "master_event_fc" // As per requirement

type SessionRepository struct {
	repo   *Repository[models.UserSession]
	jwtKey []byte // Key for verifying JWT tokens
}

// NewSessionRepository creates a new repository for UserSession.
// jwtKey is required to decode tokens.
func NewSessionRepository(uow *UnitOfWork, factory IDGeneratorFactory, jwtKey []byte) (*SessionRepository, error) {
	if uow == nil {
		return nil, fmt.Errorf("UnitOfWork is required for SessionRepository")
	}
	if len(jwtKey) == 0 {
		return nil, fmt.Errorf("jwtKey is required for SessionRepository")
	}

	// Use MasterEventFC as the column family as requested.
	// "session_schema" is an arbitrary schema name, adjust if there's a convention.
	repo, err := GetRepository[models.UserSession](uow, MasterEventFC, "session_schema", factory)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize generic repository for UserSession: %w", err)
	}
	return &SessionRepository{repo: repo, jwtKey: jwtKey}, nil
}

// parseToken decodes the JWT token and extracts claims.
func (r *SessionRepository) parseToken(tokenString string) (*jwt.RegisteredClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return r.jwtKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if claims, ok := token.Claims.(*jwt.RegisteredClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, fmt.Errorf("invalid token or claims type")
}

// SessionExists checks if a valid session exists for the user identified in the JWT token.
// It queries by UserName extracted from the token.
func (r *SessionRepository) SessionExists(jwtToken string) (bool, error) {
	claims, err := r.parseToken(jwtToken)
	if err != nil {
		// If token is invalid, no session can exist based on it.
		return false, fmt.Errorf("invalid token for session check: %w", err)
	}

	userName := claims.Subject
	if userName == "" {
		return false, fmt.Errorf("username not found in token claims")
	}

	now := time.Now()
	// FindByField expects the value to search for.
	// The generic repository handles searching by indexed fields.
	// UserName is marked as `orm:"unique"`, so it will have a unique index.
	existingSession, err := r.repo.FindByField("UserName", userName, now)
	if err != nil {
		return false, fmt.Errorf("error querying session for user %s: %w", userName, err)
	}

	if existingSession == nil {
		return false, nil // No session found
	}

	// Check if the stored session's TTL is still valid
	// The repository's Get operations should respect TTLs if the underlying KVStore does.
	// Here, existingSession.TTL is an absolute timestamp. The FindByField should ideally not return expired items.
	// If it does, we add an explicit check.
	// The custom repository's FindByField does not automatically filter by TTL for the main data key,
	// only for index entries if KVStore.Get respects TTL.
	// So, an explicit check here is safer.
	if existingSession.TTL < now.Unix() {
		// Session found but it's expired.
		// Optionally, one could delete the expired session here.
		return false, nil
	}

	return true, nil
}

// RegisterSession registers the current token as the active session.
// It decodes the JWT, extracts user identity and expiration,
// and updates or creates the UserSession record with the calculated TTL.
func (r *SessionRepository) RegisterSession(jwtToken string) error {
	claims, err := r.parseToken(jwtToken)
	if err != nil {
		return fmt.Errorf("invalid token for session registration: %w", err)
	}

	userName := claims.Subject
	if userName == "" {
		return fmt.Errorf("username not found in token for session registration")
	}

	expiresAt := claims.ExpiresAt
	if expiresAt == nil {
		return fmt.Errorf("token has no expiration time, cannot register session")
	}
	sessionTTL := expiresAt.Unix() // Absolute expiration time in seconds

	now := time.Now()

	// Check if a session for this user already exists
	existingSession, err := r.repo.FindByField("UserName", userName, now)
	if err != nil {
		// Distinguish between "not found" (which is not an error from FindByField, it returns nil)
		// and actual DB errors.
		return fmt.Errorf("error checking for existing session for user %s: %w", userName, err)
	}

	if existingSession != nil {
		// Update existing session
		existingSession.CurrentToken = jwtToken
		existingSession.TTL = sessionTTL // Update with the new token's expiry
		existingSession.UpdatedAt = now  // Assuming this field is for informational purposes

		// The Update method of the generic repository needs to be careful about TTL.
		// The `orm:"ttl"` tag on UserSession.TTL means the repository's Create/Update
		// methods will look for this field to pass to kvStore.PutTTL.
		// The value of UserSession.TTL (absolute time) is used by the repo to calculate duration.
		updated, err := r.repo.Update(existingSession, now)
		if err != nil {
			return fmt.Errorf("failed to update session for user %s: %w", userName, err)
		}
		if !updated {
			// This case might mean the record was deleted between Find and Update, or some other condition.
			return fmt.Errorf("session for user %s was not updated (possibly not found or no changes detected)", userName)
		}
	} else {
		// Create new session
		newSession := &models.UserSession{
			// ID will be auto-generated by r.repo.Create if empty
			UserName:     userName,
			CurrentToken: jwtToken,
			TTL:          sessionTTL, // Absolute expiration time
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		_, err := r.repo.Create(newSession, now)
		if err != nil {
			return fmt.Errorf("failed to create new session for user %s: %w", userName, err)
		}
	}

	return nil
}

// GetSessionByUsername is a helper if direct username access is needed (not part of original request but good practice)
func (r *SessionRepository) GetSessionByUsername(username string, now time.Time) (*models.UserSession, error) {
	session, err := r.repo.FindByField("UserName", username, now)
	if err != nil {
		return nil, fmt.Errorf("error getting session for user %s: %w", username, err)
	}
	if session != nil && session.TTL < now.Unix() {
		return nil, nil // Session expired
	}
	return session, nil
}
