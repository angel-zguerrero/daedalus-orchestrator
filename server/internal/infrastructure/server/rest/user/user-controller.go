package user

import (
	"deadalus-orch/server/internal/infrastructure/server/common"
	bo "deadalus-orch/server/internal/usecase/business-logic"
	"deadalus-orch/shared/models"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type UserController struct {
	Config *common.ServerConfing
	UserBO *bo.UserBO
}

func NewUserController(Config *common.ServerConfing) *UserController {
	api := &UserController{
		Config: Config,
		UserBO: bo.NewUserBO(Config.MasterNode, &Config.Logger),
	}
	return api
}

func (ctrl *UserController) GetUsersHandler(c *gin.Context) {
	pageParam := c.Query("pageSize")
	page, err := strconv.Atoi(pageParam)
	if err != nil || page < 2 {
		page = 50
	} else if page > 1000 {
		page = 1000
	}

	findResult, err := ctrl.UserBO.GetUsers(c.Request.Context(), c.Query("q"), c.Query("cursor"), page)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Remove passwords from the response
	if findResult != nil && findResult.Entities != nil {
		for i := range findResult.Entities {
			findResult.Entities[i].PasswordHash = ""
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "User list",
		"result":  findResult,
	})
}

func (ctrl *UserController) CreateUserHandler(c *gin.Context) {
	var req models.CreateUser
	if err := c.ShouldBindJSON(&req); err != nil {
		ctrl.Config.Logger.Warn().Err(err).Msg("create user attempt with invalid payload")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	if req.Username == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username and Password are required"})
		return
	}

	userId, err := ctrl.UserBO.CreateUser(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "User created",
		"result":  userId,
	})
}

func (ctrl *UserController) UpdateUserHandler(c *gin.Context) {
	id := c.Param("id")
	var req models.UpdateUser
	if err := c.ShouldBindJSON(&req); err != nil {
		ctrl.Config.Logger.Warn().Err(err).Msg("update user attempt with invalid payload")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	req.ID = id

	success, err := ctrl.UserBO.UpdateUser(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "User updated",
		"result":  success,
	})
}

func (ctrl *UserController) DeleteUserHandler(c *gin.Context) {
	id := c.Param("id")

	success, err := ctrl.UserBO.DeleteUser(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "User deleted",
		"result":  success,
	})
}
