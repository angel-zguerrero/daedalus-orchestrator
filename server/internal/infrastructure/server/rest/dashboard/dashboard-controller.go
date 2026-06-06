package dashboard

import (
	"deadalus-orch/server/internal/infrastructure/server/common"
	bo "deadalus-orch/server/internal/usecase/business-logic"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// DashboardController handles the dashboard-related REST API endpoints.
type DashboardController struct {
	Config              *common.ServerConfing
	DashboardSummaryBO  *bo.DashboardSummaryBO
}

func NewDashboardController(config *common.ServerConfing) *DashboardController {
	return &DashboardController{
		Config:             config,
		DashboardSummaryBO: bo.NewDashboardSummaryBO(config),
	}
}

// GetDashboardSummaryHandler handles GET /rest-api/dashboard/summary
func (ctrl *DashboardController) GetDashboardSummaryHandler(c *gin.Context) {
	summary, err := ctrl.DashboardSummaryBO.GetDashboardSummary(c.Request.Context())
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Dashboard summary not available yet"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Dashboard Summary",
		"result":  summary,
	})
}
