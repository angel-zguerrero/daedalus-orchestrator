package rest_api_admin

import (
	"net/http"

	"deadalus-orch/server/internal/infrastructure/server/common"
	bo "deadalus-orch/server/internal/usecase/business-logic"

	"github.com/gin-gonic/gin"
)

type loginRequest struct {
	UsernameOrEmail string `json:"usernameOrEmail" binding:"required"`
	Password        string `json:"password" binding:"required"`
}

type AdminController struct {
	Config *common.ServerConfing
}

// NewAdminController creates a new instance of RestAdminAPI.
func NewAdminController(Config *common.ServerConfing) *AdminController {

	api := &AdminController{
		Config: Config,
	}

	return api
}

// LoginHandler handles the /admin-api/login endpoint.
func (ctrl *AdminController) LoginHandler(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ctrl.Config.Logger.Warn().Err(err).Msg("Login attempt with invalid payload")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	authBO := bo.NewAuthBO(ctrl.Config.MasterNode, ctrl.Config.JwtKey, ctrl.Config.JwtDuration, &ctrl.Config.Logger)

	tokenString, err := authBO.Login(c.Request.Context(), req.UsernameOrEmail, req.Password)
	if err != nil {
		ctrl.Config.Logger.Error().Err(err).Str("username", req.UsernameOrEmail).Msg("Login failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Login failed: " + err.Error()})
		return
	}

	ctrl.Config.Logger.Info().Str("username", req.UsernameOrEmail).Msg("User logged in successfully and session registered")
	c.JSON(http.StatusOK, gin.H{
		"message": "Login successful",
		"token":   tokenString,
	})
}

func (ctrl *AdminController) LogoutHandler(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header missing"})
		return
	}

	authBO := bo.NewAuthBO(ctrl.Config.MasterNode, ctrl.Config.JwtKey, ctrl.Config.JwtDuration, &ctrl.Config.Logger)

	if err := authBO.Logout(c.Request.Context(), authHeader); err != nil {
		ctrl.Config.Logger.Error().Err(err).Msg("Failed removing current session")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed removing current session: " + err.Error()})
		return
	}

	ctrl.Config.Logger.Info().Msg("User logged out successfully")
	c.JSON(http.StatusOK, gin.H{
		"message": "Logout successful",
	})
}
