package db

import (
	"fmt"
	"strings"
	"time"

	"deadalus-orch/shared/models"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

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

	repo, err := GetRepository[models.UserSession](uow, MasterEventFC, MasterEventFCSelector, "session_schema", factory)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize generic repository for UserSession: %w", err)
	}
	return &SessionRepository{repo: repo, jwtKey: jwtKey}, nil
}

// ParseToken decodes the JWT token and extracts claims.
func (r *SessionRepository) ParseToken(tokenString string) (*jwt.RegisteredClaims, error) {
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
func (r *SessionRepository) SessionExists(jwtToken string, now time.Time) (bool, error) {
	claims, err := r.ParseToken(jwtToken)
	if err != nil {
		if strings.Contains(err.Error(), "token is expired") {
			return false, nil
		}
		return false, fmt.Errorf("invalid token for session check: %w", err)
	}

	userName := claims.Subject
	if userName == "" {
		return false, fmt.Errorf("username not found in token claims")
	}
	existingSession, err := r.repo.FindByField("UserName", userName, now)
	if err != nil {
		return false, fmt.Errorf("error querying session for user %s: %w", userName, err)
	}

	if existingSession == nil || existingSession.CurrentToken != jwtToken {
		return false, nil // No session found
	}

	return true, nil
}

// RegisterSession registers the current token as the active session.
// It decodes the JWT, extracts user identity and expiration,
// and updates or creates the UserSession record with the calculated TTL.
func (r *SessionRepository) RegisterSession(jwtToken string, now time.Time) error {
	claims, err := r.ParseToken(jwtToken)
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
	sessionTTL := int64(time.Until(expiresAt.Time).Seconds())

	// Check if a session for this user already exists
	existingSession, err := r.repo.FindByField("UserName", userName, now)
	if err != nil {
		// Distinguish between "not found" (which is not an error from FindByField, it returns nil)
		// and actual DB errors.
		return fmt.Errorf("error checking for existing session for user %s: %w", userName, err)
	}

	if existingSession != nil {
		existingSession.CurrentToken = jwtToken
		existingSession.TTL = sessionTTL // Update with the new token's expiry

		// The Update method of the generic repository needs to be careful about TTL.
		// The `orm:"ttl"` tag on UserSession.TTL means the repository's Create/Update
		// methods will look for this field to pass to kvStore.PutTTL.
		// The value of UserSession.TTL (absolute time) is used by the repo to calculate duration.
		_, err := r.repo.Update(existingSession, now)
		if err != nil {
			return fmt.Errorf("failed to update session for user %s: %w", userName, err)
		}

	} else {
		// Create new session
		newSession := &models.UserSession{
			ID:           uuid.NewSHA1(uuid.NameSpaceURL, []byte(userName)).String(),
			UserName:     userName,
			CurrentToken: jwtToken,
			TTL:          sessionTTL,
		}

		_, err := r.repo.Create(newSession, now)
		if err != nil {
			return fmt.Errorf("failed to create new session for user %s: %w", userName, err)
		}
	}

	return nil
}

func (r *SessionRepository) RemoveSession(jwtToken string, now time.Time) error {
	claims, err := r.ParseToken(jwtToken)
	if err != nil {
		return fmt.Errorf("invalid token for session registration: %w", err)
	}

	userName := claims.Subject
	if userName == "" {
		return fmt.Errorf("username not found in token for session registration")
	}

	// Check if a session for this user already exists
	existingSession, err := r.repo.FindByField("UserName", userName, now)
	if err != nil {
		// Distinguish between "not found" (which is not an error from FindByField, it returns nil)
		// and actual DB errors.
		return fmt.Errorf("error checking for existing session for user %s: %w", userName, err)
	}

	if existingSession != nil {

		_, err := r.repo.Delete(existingSession.ID, now)
		if err != nil {
			return fmt.Errorf("failed to delete session for user %s: %w", userName, err)
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

	return session, nil
}
