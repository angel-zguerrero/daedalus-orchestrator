package oauth

import (
	"net/http"

	"deadalus-orch/server/internal/infrastructure/server/common"
	bo "deadalus-orch/server/internal/usecase/business-logic"
	"deadalus-orch/shared/models"

	"github.com/gin-gonic/gin"
)

type AppController struct {
	Config *common.ServerConfing
}

func NewAppController(Config *common.ServerConfing) *AppController {
	return &AppController{
		Config: Config,
	}
}

func (ctrl *AppController) CreateAppHandler(c *gin.Context) {
	tenantCtx, _ := common.GetTenantContext(c.Request.Context())
	if tenantCtx == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Tenant context not found"})
		return
	}

	var req models.CreateOAuthAppRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	appBO := bo.NewOAuthAppBO(ctrl.Config.MasterNode, &ctrl.Config.Logger)

	resp, err := appBO.CreateApp(c.Request.Context(), req, tenantCtx.Tenant.ID)
	if err != nil {
		ctrl.Config.Logger.Error().Err(err).Msg("Failed to create OAuth App")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create app: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, resp)
}

func (ctrl *AppController) RotateSecretHandler(c *gin.Context) {
	appID := c.Param("id")
	if appID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "App ID is required"})
		return
	}

	appBO := bo.NewOAuthAppBO(ctrl.Config.MasterNode, &ctrl.Config.Logger)

	resp, err := appBO.RotateSecret(c.Request.Context(), appID)
	if err != nil {
		ctrl.Config.Logger.Error().Err(err).Str("appID", appID).Msg("Failed to rotate OAuth App secret")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to rotate secret: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}
