package oauthapp

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"deadalus-orch/server/internal/infrastructure/server/common"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func TestTokenHandler_UnsupportedGrantType(t *testing.T) {
	router := setupTestRouter()
	cfg := &common.ServerConfing{}
	ctrl := NewOAuthController(cfg)

	router.POST("/rest-api/oauth/token", ctrl.TokenHandler)

	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("client_id", "app_123")
	form.Set("client_secret", "sec_456")

	req := httptest.NewRequest("POST", "/rest-api/oauth/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "unsupported_grant_type")
}

func TestTokenHandler_MissingCredentials(t *testing.T) {
	router := setupTestRouter()
	cfg := &common.ServerConfing{}
	ctrl := NewOAuthController(cfg)

	router.POST("/rest-api/oauth/token", ctrl.TokenHandler)

	form := url.Values{}
	form.Set("grant_type", "client_credentials")

	req := httptest.NewRequest("POST", "/rest-api/oauth/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "invalid_client")
}
