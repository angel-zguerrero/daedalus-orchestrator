package db_test

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testJWTKey = []byte("testsecretkey") // Use a consistent key for tests

// Helper to generate a JWT token for testing
func generateTestJWT(t *testing.T, username string, expiration time.Duration) string {
	expTime := time.Now().Add(expiration)
	claims := &jwt.RegisteredClaims{
		Subject:   username,
		ExpiresAt: jwt.NewNumericDate(expTime),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(testJWTKey)
	require.NoError(t, err, "Failed to sign test token")
	return tokenString
}

func setupSessionTestDB(t *testing.T) (*db.UnitOfWork, *db.SessionRepository, func()) {
	t.Helper()
	// Base path for test databases, e.g., under /tmp or similar
	baseTestPath := filepath.Join(os.TempDir(), "deadalus_orch_session_tests")
	dbPath := filepath.Join(baseTestPath, t.Name()) // Unique DB path per test

	// Ensure the directory exists
	err := os.MkdirAll(dbPath, 0755)
	require.NoError(t, err, "Failed to create test DB directory")

	// Assuming PebbleStore is the default or desired KVStore for these tests
	kvStore := newRocksdbStore(t)

	uow := db.NewUnitOfWork(kvStore, nil)
	idFactory := &db.DeterministicIDGeneratorFactory{}

	sessionRepo, err := db.NewSessionRepository(uow, idFactory, testJWTKey)
	require.NoError(t, err, "Failed to create SessionRepository")

	cleanup := func() {
		assert.NoError(t, err, "Failed to close KVStore")
		err = os.RemoveAll(baseTestPath) // Clean up the entire base test path
		assert.NoError(t, err, "Failed to remove test DB directory")
	}

	return uow, sessionRepo, cleanup
}

func TestSessionRepository_RegisterAndSessionExists(t *testing.T) {
	uow, repo, cleanup := setupSessionTestDB(t)
	defer cleanup()

	username := "testuser"
	tokenShortLived := generateTestJWT(t, username, 5*time.Minute)
	tokenLongLived := generateTestJWT(t, username, 1*time.Hour)

	now := time.Now()
	// 1. Register a new session
	err := repo.RegisterSession(tokenShortLived, now)
	require.NoError(t, err, "RegisterSession failed for new session")
	err = uow.Commit() // Commit the changes to the database
	require.NoError(t, err, "Failed to commit session registration")

	// 2. Check if session exists (it should)
	exists, err := repo.SessionExists(tokenShortLived, now) // Can use the same token or a new one for same user
	require.NoError(t, err, "SessionExists failed after registration")
	assert.True(t, exists, "Session should exist after registration")

	// 3. Verify stored data (optional, but good for sanity)
	// The GetSessionByUsername is a helper, assuming it's added to the repo
	// Alternatively, can use repo.repo.FindByField directly if GetSessionByUsername is not present

	storedSession, err := repo.GetSessionByUsername(username, now)
	require.NoError(t, err, "Failed to get session by username")
	require.NotNil(t, storedSession, "Stored session should not be nil")
	assert.Equal(t, username, storedSession.UserName)
	assert.Equal(t, tokenShortLived, storedSession.CurrentToken)

	parsedClaims, _ := repo.ParseToken(tokenShortLived)
	require.NotNil(t, parsedClaims)

	assert.Equal(t, int64(time.Until(parsedClaims.ExpiresAt.Time).Seconds()), storedSession.TTL, "TTL should match token expiry")

	// 4. Register another session for the same user (update)
	err = repo.RegisterSession(tokenLongLived, now)
	require.NoError(t, err, "RegisterSession failed for updating session")
	err = uow.Commit() // Commit the changes to the database
	require.NoError(t, err, "Failed to commit session registration")

	// 5. Check session exists (it should, with updated token details)
	exists, err = repo.SessionExists(tokenLongLived, now)
	require.NoError(t, err, "SessionExists failed after update")
	assert.True(t, exists, "Session should still exist after update")

	storedSessionUpdated, err := repo.GetSessionByUsername(username, now)
	require.NoError(t, err, "Failed to get updated session by username")
	require.NotNil(t, storedSessionUpdated, "Updated stored session should not be nil")
	assert.Equal(t, tokenLongLived, storedSessionUpdated.CurrentToken, "Token should be updated")

	parsedClaimsLong, _ := repo.ParseToken(tokenLongLived)
	require.NotNil(t, parsedClaimsLong)
	assert.NotEqual(t, storedSession.TTL, storedSessionUpdated.TTL, "TTL should have changed")
}

func TestSessionRepository_SessionExists_NotFound(t *testing.T) {
	_, repo, cleanup := setupSessionTestDB(t)
	defer cleanup()

	username := "nonexistentuser"
	token := generateTestJWT(t, username, 5*time.Minute)
	now := time.Now()
	exists, err := repo.SessionExists(token, now)
	require.NoError(t, err, "SessionExists failed for non-existent user")
	assert.False(t, exists, "Session should not exist for a user that never had a session")
}

func TestSessionRepository_SessionExists_ExpiredToken(t *testing.T) {
	uow, repo, cleanup := setupSessionTestDB(t)
	defer cleanup()

	username := "expireduser"
	// Token that expires very quickly, effectively "in the past" for the check
	// Note: Relies on test execution being fast enough relative to this very short expiry.
	// A more robust way would be to manually insert an expired session.
	tokenExpired := generateTestJWT(t, username, 2*time.Second) // Expired 1 second ago

	now := time.Now()
	// Register this "already expired" session for testing the retrieval logic
	// The TTL stored will be in the past.
	err := repo.RegisterSession(tokenExpired, now)
	require.NoError(t, err, "RegisterSession failed for expired token scenario")
	uow.Commit()

	time.Sleep(3 * time.Second) // Wait for the token to be definitely expired
	// Wait a tiny moment to ensure TTL is definitely in the past if there are clock sync issues
	// time.Sleep(10 * time.Millisecond)

	exists, err := repo.SessionExists(tokenExpired, now) // Use the same token
	require.NoError(t, err, "SessionExists failed for expired session")
	assert.False(t, exists, "Session should not exist if its stored TTL is in the past")
}

func TestSessionRepository_RegisterSession_InvalidToken(t *testing.T) {
	_, repo, cleanup := setupSessionTestDB(t)
	defer cleanup()
	now := time.Now()
	err := repo.RegisterSession("this.is.not.a.valid.jwt", now)
	require.Error(t, err, "RegisterSession should fail for an invalid token string")
	assert.Contains(t, err.Error(), "invalid token for session registration", "Error message should indicate invalid token")
}

func TestSessionRepository_SessionExists_InvalidToken(t *testing.T) {
	_, repo, cleanup := setupSessionTestDB(t)
	defer cleanup()
	now := time.Now()
	exists, err := repo.SessionExists("this.is.not.a.valid.jwt", now)
	require.Error(t, err, "SessionExists should fail for an invalid token string")
	assert.False(t, exists, "Session should not exist if token is invalid")
	assert.Contains(t, err.Error(), "invalid token for session check", "Error message should indicate invalid token")
}

func TestSessionRepository_RegisterSession_TokenMissingExpiry(t *testing.T) {
	_, repo, cleanup := setupSessionTestDB(t)
	defer cleanup()

	// Create a token without an 'exp' claim
	claimsNoExp := &jwt.RegisteredClaims{
		Subject:  "userNoExpiry",
		IssuedAt: jwt.NewNumericDate(time.Now()),
	}
	tokenNoExp := jwt.NewWithClaims(jwt.SigningMethodHS256, claimsNoExp)
	tokenStringNoExp, err := tokenNoExp.SignedString(testJWTKey)
	require.NoError(t, err)
	now := time.Now()
	err = repo.RegisterSession(tokenStringNoExp, now)
	require.Error(t, err, "RegisterSession should fail if token has no expiry")
	assert.Contains(t, err.Error(), "token has no expiration time", "Error message should indicate missing expiry")
}

func TestSessionRepository_RegisterSession_TokenMissingUsername(t *testing.T) {
	_, repo, cleanup := setupSessionTestDB(t)
	defer cleanup()

	// Create a token without a 'sub' (Subject/Username) claim
	claimsNoSub := &jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(5 * time.Minute)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}
	tokenNoSub := jwt.NewWithClaims(jwt.SigningMethodHS256, claimsNoSub)
	tokenStringNoSub, err := tokenNoSub.SignedString(testJWTKey)
	require.NoError(t, err)
	now := time.Now()
	err = repo.RegisterSession(tokenStringNoSub, now)
	require.Error(t, err, "RegisterSession should fail if token has no username/subject")
	assert.Contains(t, err.Error(), "username not found in token", "Error message should indicate missing username")
}
