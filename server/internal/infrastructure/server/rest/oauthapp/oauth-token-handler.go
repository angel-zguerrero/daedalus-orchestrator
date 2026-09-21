package oauthapp

import (
	"net/http"
	"strings"
	"time"

	"deadalus-orch/server/internal/infrastructure/server/common"
	bo "deadalus-orch/server/internal/usecase/business-logic"

	"github.com/gin-gonic/gin"
)

// OAuthController handles the token-issuance endpoint.
// This endpoint is PUBLIC — it has no authMiddleware applied.
// Rate-limiting is applied at the route level in routes.go.
type OAuthController struct {
	Config *common.ServerConfing
}

func NewOAuthController(config *common.ServerConfing) *OAuthController {
	return &OAuthController{Config: config}
}

// tokenRequest holds the parsed fields from the application/x-www-form-urlencoded body.
type tokenRequest struct {
	GrantType    string
	ClientID     string
	ClientSecret string
}

// TokenHandler implements POST /rest-api/oauth/token
//
// Implements RFC 6749 §4.4 — Client Credentials Grant.
// Request Content-Type: application/x-www-form-urlencoded
// Required fields: grant_type=client_credentials, client_id, client_secret
//
// Success (200):
//
//	{ "access_token": "<jwt>", "token_type": "bearer", "expires_in": 3600, "scope": "queues:list" }
//
// Error (400): unsupported_grant_type
// Error (401): invalid_client
func (ctrl *OAuthController) TokenHandler(c *gin.Context) {
	req := tokenRequest{
		GrantType:    c.PostForm("grant_type"),
		ClientID:     c.PostForm("client_id"),
		ClientSecret: c.PostForm("client_secret"),
	}

	if req.GrantType != "client_credentials" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":             "unsupported_grant_type",
			"error_description": "Only the client_credentials grant type is supported",
		})
		return
	}

	if req.ClientID == "" || req.ClientSecret == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":             "invalid_client",
			"error_description": "client_id and client_secret are required",
		})
		return
	}

	now := time.Now()
	oauthBO := bo.NewOAuthAppBO(ctrl.Config, 1*time.Hour)

	tokenString, err := oauthBO.IssueToken(c.Request.Context(), req.ClientID, req.ClientSecret, now)
	if err != nil {
		ctrl.Config.Logger.Warn().Err(err).Str("client_id", req.ClientID).Msg("OAuth token request failed")
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":             "invalid_client",
			"error_description": "Invalid client credentials",
		})
		return
	}

	// Fetch app to populate the scope field in the response
	app, err := oauthBO.GetApp(c.Request.Context(), req.ClientID, now)

	scope := ""
	if err == nil && app != nil {
		scope = strings.Join(app.AllowedScopes, " ")
	}

	ctrl.Config.Logger.Info().Str("client_id", req.ClientID).Msg("OAuth token issued successfully")
	c.JSON(http.StatusOK, gin.H{
		"access_token": tokenString,
		"token_type":   "bearer",
		"expires_in":   3600,
		"scope":        scope,
	})
}
