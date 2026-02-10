package nodescheduler

import (
	"deadalus-orch/server/internal/infrastructure/server/common"
	bo "deadalus-orch/server/internal/usecase/business-logic"
	"deadalus-orch/shared/models"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type NodeSchedulerController struct {
	Config          *common.ServerConfing
	NodeSchedulerBO *bo.NodeSchedulerBO
}

func NewNodeSchedulerController(Config *common.ServerConfing) *NodeSchedulerController {
	api := &NodeSchedulerController{
		Config:          Config,
		NodeSchedulerBO: bo.NewNodeSchedulerBO(Config),
	}
	return api
}

// GetNodeSchedulerHandler handles GET /rest-api/node-schedulers/:id
func (ctrl *NodeSchedulerController) GetNodeSchedulerHandler(c *gin.Context) {
	nodeSchedulerID := c.Param("id")
	nodeScheduler, err := ctrl.NodeSchedulerBO.GetNodeScheduler(c.Request.Context(), nodeSchedulerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "NodeScheduler",
		"result":  nodeScheduler,
	})
}

// GetNodeSchedulersHandler handles GET /rest-api/node-schedulers
func (ctrl *NodeSchedulerController) GetNodeSchedulersHandler(c *gin.Context) {
	pageParam := c.Query("pageSize")
	page, err := strconv.Atoi(pageParam)
	if err != nil || page < 2 {
		page = 50
	} else if page > 1000 {
		page = 1000
	}

	findResult, err := ctrl.NodeSchedulerBO.GetNodeSchedulers(c.Request.Context(), c.Query("q"), "", "", -1, c.Query("cursor"), page)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if findResult.Entities == nil {
		findResult.Entities = []models.NodeScheduler{}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "NodeScheduler list",
		"result":  findResult,
	})
}
