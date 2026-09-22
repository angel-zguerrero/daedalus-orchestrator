package oauthapp

import (
	"net/http"
	"time"

	"deadalus-orch/server/internal/infrastructure/server/common"
	bo "deadalus-orch/server/internal/usecase/business-logic"
	"deadalus-orch/shared/models"

	"github.com/gin-gonic/gin"
)

// OAuthAppController provides CRUD management for OAuth 2.0 Service Accounts.
// All endpoints require an admin user session.
type OAuthAppController struct {
	Config *common.ServerConfing
}

func NewOAuthAppController(config *common.ServerConfing) *OAuthAppController {
	return &OAuthAppController{Config: config}
}

type createAppRequest struct {
	Name          string   `json:"name"          binding:"required"`
	Description   string   `json:"description"`
	AllowedScopes []string `json:"allowedScopes"`
}

func appResponseFromApp(app models.OAuthApp) models.OAuthAppResponse {
	return models.OAuthAppResponse{
		ID:            app.ID,
		ClientID:      app.ClientID,
		Name:          app.Name,
		TenantID:      app.TenantID,
		Description:   app.Description,
		AllowedScopes: app.AllowedScopes,
		CreatedAt:     app.CreatedAt,
		UpdatedAt:     app.UpdatedAt,
	}
}

// ListAppsHandler handles GET /rest-api/tenants/:code/oauth-apps
func (ctrl *OAuthAppController) ListAppsHandler(c *gin.Context) {
	tenantCtx, ok := common.GetTenantContext(c.Request.Context())
	if !ok || tenantCtx == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tenant context not found"})
		return
	}

	now := time.Now()
	oauthBO := bo.NewOAuthAppBO(ctrl.Config, 1*time.Hour)

	apps, err := oauthBO.ListApps(c.Request.Context(), tenantCtx.Tenant.ID, now)
	if err != nil {
		ctrl.Config.Logger.Error().Err(err).Str("tenantID", tenantCtx.Tenant.ID).Msg("Failed to list OAuth apps")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list Service Accounts: " + err.Error()})
		return
	}

	items := make([]models.OAuthAppResponse, 0, len(apps))
	for _, app := range apps {
		items = append(items, appResponseFromApp(app))
	}

	c.JSON(http.StatusOK, gin.H{
		"items":  items,
		"cursor": "",
	})
}

// ListGlobalAppsHandler handles GET /rest-api/oauth-apps
func (ctrl *OAuthAppController) ListGlobalAppsHandler(c *gin.Context) {
	now := time.Now()
	oauthBO := bo.NewOAuthAppBO(ctrl.Config, 1*time.Hour)

	apps, err := oauthBO.ListApps(c.Request.Context(), "", now)
	if err != nil {
		ctrl.Config.Logger.Error().Err(err).Msg("Failed to list global OAuth apps")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list Service Accounts: " + err.Error()})
		return
	}

	items := make([]models.OAuthAppResponse, 0, len(apps))
	for _, app := range apps {
		items = append(items, appResponseFromApp(app))
	}

	c.JSON(http.StatusOK, gin.H{
		"items":  items,
		"cursor": "",
	})
}

// GetAppHandler handles GET /rest-api/tenants/:code/oauth-apps/:id
func (ctrl *OAuthAppController) GetAppHandler(c *gin.Context) {
	appID := c.Param("id")
	if appID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "App ID is required"})
		return
	}

	now := time.Now()
	oauthBO := bo.NewOAuthAppBO(ctrl.Config, 1*time.Hour)

	app, err := oauthBO.GetApp(c.Request.Context(), appID, now)
	if err != nil {
		ctrl.Config.Logger.Error().Err(err).Str("appID", appID).Msg("Failed to get OAuth app")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get Service Account: " + err.Error()})
		return
	}
	if app == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Service Account not found"})
		return
	}

	c.JSON(http.StatusOK, appResponseFromApp(*app))
}

// CreateAppHandler handles POST /rest-api/tenants/:code/oauth-apps
func (ctrl *OAuthAppController) CreateAppHandler(c *gin.Context) {
	var req createAppRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	tenantCtx, ok := common.GetTenantContext(c.Request.Context())
	if !ok || tenantCtx == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tenant context not found"})
		return
	}

	now := time.Now()
	oauthBO := bo.NewOAuthAppBO(ctrl.Config, 1*time.Hour)

	appResp, plainSecret, err := oauthBO.CreateApp(
		c.Request.Context(),
		tenantCtx.Tenant.ID,
		req.Name,
		req.Description,
		req.AllowedScopes,
		now,
	)
	if err != nil {
		ctrl.Config.Logger.Error().Err(err).Str("name", req.Name).Msg("Failed to create OAuth app")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create Service Account: " + err.Error()})
		return
	}

	ctrl.Config.Logger.Info().
		Str("appID", appResp.ID).
		Str("clientID", appResp.ClientID).
		Str("tenantID", tenantCtx.Tenant.ID).
		Msg("OAuth Service Account created successfully")

	c.JSON(http.StatusCreated, models.OAuthAppCreatedResponse{
		OAuthAppResponse: *appResp,
		ClientSecret:     plainSecret, // ONE-TIME — never returned again
	})
}

// CreateGlobalAppHandler handles POST /rest-api/oauth-apps
func (ctrl *OAuthAppController) CreateGlobalAppHandler(c *gin.Context) {
	var req createAppRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	now := time.Now()
	oauthBO := bo.NewOAuthAppBO(ctrl.Config, 1*time.Hour)

	appResp, plainSecret, err := oauthBO.CreateApp(
		c.Request.Context(),
		"", // empty TenantID for Global Service Account
		req.Name,
		req.Description,
		req.AllowedScopes,
		now,
	)
	if err != nil {
		ctrl.Config.Logger.Error().Err(err).Str("name", req.Name).Msg("Failed to create global OAuth app")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create Service Account: " + err.Error()})
		return
	}

	ctrl.Config.Logger.Info().
		Str("appID", appResp.ID).
		Str("clientID", appResp.ClientID).
		Msg("Global OAuth Service Account created successfully")

	c.JSON(http.StatusCreated, models.OAuthAppCreatedResponse{
		OAuthAppResponse: *appResp,
		ClientSecret:     plainSecret,
	})
}

// DeleteAppHandler handles DELETE /rest-api/tenants/:code/oauth-apps/:id
func (ctrl *OAuthAppController) DeleteAppHandler(c *gin.Context) {
	appID := c.Param("id")
	if appID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "App ID is required"})
		return
	}

	now := time.Now()
	oauthBO := bo.NewOAuthAppBO(ctrl.Config, 1*time.Hour)

	deleted, err := oauthBO.DeleteApp(c.Request.Context(), appID, now)
	if err != nil {
		ctrl.Config.Logger.Error().Err(err).Str("appID", appID).Msg("Failed to delete OAuth app")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete Service Account: " + err.Error()})
		return
	}
	if !deleted {
		c.JSON(http.StatusNotFound, gin.H{"error": "Service Account not found"})
		return
	}

	ctrl.Config.Logger.Info().Str("appID", appID).Msg("OAuth Service Account deleted")
	c.JSON(http.StatusOK, gin.H{"message": "Service Account deleted successfully"})
}

// RotateSecretHandler handles POST /rest-api/tenants/:code/oauth-apps/:id/rotate-secret
func (ctrl *OAuthAppController) RotateSecretHandler(c *gin.Context) {
	appID := c.Param("id")
	if appID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "App ID is required"})
		return
	}

	now := time.Now()
	oauthBO := bo.NewOAuthAppBO(ctrl.Config, 1*time.Hour)

	app, newSecret, err := oauthBO.RotateSecret(c.Request.Context(), appID, now)
	if err != nil {
		ctrl.Config.Logger.Error().Err(err).Str("appID", appID).Msg("Failed to rotate OAuth app secret")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to rotate secret: " + err.Error()})
		return
	}
	if app == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Service Account not found"})
		return
	}

	ctrl.Config.Logger.Info().Str("appID", appID).Msg("OAuth Service Account secret rotated")

	c.JSON(http.StatusOK, models.OAuthAppCreatedResponse{
		OAuthAppResponse: appResponseFromApp(*app),
		ClientSecret:     newSecret, // ONE-TIME NEW SECRET — never returned again
	})
}
