package rest_server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	bo "deadalus-orch/server/internal/usecase/business-logic"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

var testJWTKey = []byte("super-secret-jwt-key-32-bytes-long!")

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func issueTestOAuthToken(scopes []string, tenantID string) string {
	claims := bo.OAuthClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "app_client_123",
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		},
		Scopes:    scopes,
		TenantID:  tenantID,
		TokenType: "oauth",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	str, _ := token.SignedString(testJWTKey)
	return str
}

func TestUnifiedAuthMiddleware_OAuthToken_Success(t *testing.T) {
	router := setupTestRouter()
	router.Use(unifiedAuthMiddleware(nil, nilLogger(), testJWTKey))

	var capturedScopes []string
	var capturedClientID string

	router.GET("/api/v1/test", func(c *gin.Context) {
		if scopes, ok := c.Get("oauth_scopes"); ok {
			capturedScopes = scopes.([]string)
		}
		if clientID, ok := c.Get("oauth_client_id"); ok {
			capturedClientID = clientID.(string)
		}
		c.Status(http.StatusOK)
	})

	token := issueTestOAuthToken([]string{"queues:read", "queues:write"}, "tenant_alpha")

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "app_client_123", capturedClientID)
	assert.Equal(t, []string{"queues:read", "queues:write"}, capturedScopes)
}

func TestUnifiedAuthMiddleware_MissingHeader(t *testing.T) {
	router := setupTestRouter()
	router.Use(unifiedAuthMiddleware(nil, nilLogger(), testJWTKey))
	router.GET("/api/v1/test", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRequireScope_OAuthToken(t *testing.T) {
	router := setupTestRouter()
	router.Use(unifiedAuthMiddleware(nil, nilLogger(), testJWTKey))
	router.POST("/api/v1/queues", requireScope("queues:create", "queues:admin"), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	t.Run("Allowed Scope", func(t *testing.T) {
		token := issueTestOAuthToken([]string{"queues:create"}, "tenant_alpha")
		req := httptest.NewRequest("POST", "/api/v1/queues", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Forbidden Scope", func(t *testing.T) {
		token := issueTestOAuthToken([]string{"workflows:read"}, "tenant_alpha")
		req := httptest.NewRequest("POST", "/api/v1/queues", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}

func TestSessionOnlyMiddleware(t *testing.T) {
	router := setupTestRouter()
	router.Use(unifiedAuthMiddleware(nil, nilLogger(), testJWTKey))
	router.Use(sessionOnlyMiddleware())
	router.GET("/admin/users", func(c *gin.Context) { c.Status(http.StatusOK) })

	token := issueTestOAuthToken([]string{"queues:admin"}, "tenant_alpha")
	req := httptest.NewRequest("GET", "/admin/users", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func nilLogger() zerolog.Logger {
	return zerolog.Nop()
}
