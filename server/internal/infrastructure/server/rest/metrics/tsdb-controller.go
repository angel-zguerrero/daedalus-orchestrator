package metrics

import (
	"deadalus-orch/server/internal/infrastructure/server/common"
	bo "deadalus-orch/server/internal/usecase/business-logic"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type TSDBMetricsController struct {
	Config *common.ServerConfing
	TSDBBO *bo.TSDBMetricsBO
}

func NewTSDBMetricsController(Config *common.ServerConfing) *TSDBMetricsController {
	return &TSDBMetricsController{
		Config: Config,
		TSDBBO: bo.NewTSDBMetricsBO(Config),
	}
}

// GetTSDBMetricsHandler handles GET /rest-api/tenants/:code/metrics/tsdb
func (ctrl *TSDBMetricsController) GetTSDBMetricsHandler(c *gin.Context) {
	tenantCode := c.Param("code")
	queueCode := c.Query("queueCode")
	vnamespace := c.Query("vnamespace")

	// Parse resolution (default to 5 seconds if not specified)
	resolutionStr := c.Query("resolution")
	resolution := 5
	if r, err := strconv.Atoi(resolutionStr); err == nil && r > 0 {
		resolution = r
	}

	// Parse time ranges (default to last hour)
	now := time.Now().Unix()
	startTimeStr := c.Query("startTime")
	endTimeStr := c.Query("endTime")

	endTime := now
	if e, err := strconv.ParseInt(endTimeStr, 10, 64); err == nil && e > 0 {
		endTime = e
	}

	startTime := endTime - 3600 // 1 hour default
	if s, err := strconv.ParseInt(startTimeStr, 10, 64); err == nil && s > 0 {
		startTime = s
	}

	buckets, err := ctrl.TSDBBO.QueryMetrics(c.Request.Context(), tenantCode, queueCode, vnamespace, resolution, startTime, endTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "TSDB Metrics",
		"result":  buckets,
	})
}

// GetGlobalTSDBMetricsHandler handles GET /rest-api/cluster/metrics/tsdb
func (ctrl *TSDBMetricsController) GetGlobalTSDBMetricsHandler(c *gin.Context) {
	// Parse resolution (default to 5 seconds if not specified)
	resolutionStr := c.Query("resolution")
	resolution := 5
	if r, err := strconv.Atoi(resolutionStr); err == nil && r > 0 {
		resolution = r
	}

	// Parse time ranges (default to last 10 minutes)
	now := time.Now().Unix()
	startTimeStr := c.Query("startTime")
	endTimeStr := c.Query("endTime")

	endTime := now
	if e, err := strconv.ParseInt(endTimeStr, 10, 64); err == nil && e > 0 {
		endTime = e
	}

	startTime := endTime - 600 // 10 minutes default
	if s, err := strconv.ParseInt(startTimeStr, 10, 64); err == nil && s > 0 {
		startTime = s
	}

	buckets, err := ctrl.TSDBBO.QueryGlobalMetrics(c.Request.Context(), resolution, startTime, endTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Global TSDB Metrics",
		"result":  buckets,
	})
}
