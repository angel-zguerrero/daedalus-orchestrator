package oauth

import (
	"net/http"

	"deadalus-orch/server/internal/infrastructure/server/common"
	bo "deadalus-orch/server/internal/usecase/business-logic"
	"deadalus-orch/shared/models"

	"github.com/gin-gonic/gin"
)

type TokenController struct {
	Config *common.ServerConfing
}

func NewTokenController(Config *common.ServerConfing) *TokenController {
	return &TokenController{
		Config: Config,
	}
}

func (ctrl *TokenController) GenerateTokenHandler(c *gin.Context) {
	var req models.OAuthTokenRequest

	// Try JSON first, fallback to Form
	if err := c.ShouldBindJSON(&req); err != nil {
		if err := c.ShouldBind(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
			return
		}
	}

	tokenBO := bo.NewOAuthTokenBO(ctrl.Config.MasterNode, &ctrl.Config.Logger)

	resp, err := tokenBO.GenerateToken(c.Request.Context(), req)
	if err != nil {
		if err.Error() == "invalid client_id or client_secret" || err.Error() == "unsupported grant_type" || len(err.Error()) > 13 && err.Error()[:13] == "invalid scope" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		ctrl.Config.Logger.Error().Err(err).Msg("Failed to generate token")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}
