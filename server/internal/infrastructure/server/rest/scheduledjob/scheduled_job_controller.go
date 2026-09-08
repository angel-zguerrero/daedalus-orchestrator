package scheduledjob

import (
	"deadalus-orch/server/internal/infrastructure/server/common"
	bo "deadalus-orch/server/internal/usecase/business-logic"
	"deadalus-orch/shared/models"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type ScheduledJobController struct {
	Config         *common.ServerConfing
	ScheduledJobBO *bo.ScheduledJobBO
}

func NewScheduledJobController(config *common.ServerConfing) *ScheduledJobController {
	return &ScheduledJobController{
		Config:         config,
		ScheduledJobBO: bo.NewScheduledJobBO(config),
	}
}

type createOneOffScheduledJobRequest struct {
	TargetType  string            `json:"targetType" binding:"required"`
	TargetCode  string            `json:"targetCode" binding:"required"`
	VNamespace  string            `json:"vnamespace" binding:"required"`
	Content     string            `json:"content"`
	ContentType string            `json:"contentType"`
	Headers     map[string]string `json:"headers"`
	Handler     string            `json:"handler"`
	Parameters  map[string]string `json:"parameters"`
	Priority    int               `json:"priority"`
	RunAt       string            `json:"runAt"`
	RunAfter    string            `json:"runAfter"`
}

type createRecurringScheduledJobRequest struct {
	TargetType     string            `json:"targetType" binding:"required"`
	TargetCode     string            `json:"targetCode" binding:"required"`
	VNamespace     string            `json:"vnamespace" binding:"required"`
	Content        string            `json:"content"`
	ContentType    string            `json:"contentType"`
	Headers        map[string]string `json:"headers"`
	Handler        string            `json:"handler"`
	Parameters     map[string]string `json:"parameters"`
	Priority       int               `json:"priority"`
	Every          string            `json:"every"`
	CronExpression string            `json:"cronExpression"`
}

func (ctrl *ScheduledJobController) CreateOneOffScheduledJobHandler(c *gin.Context) {
	var req createOneOffScheduledJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ctrl.Config.Logger.Warn().Err(err).Msg("create one-off scheduled job attempt with invalid payload")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	tenantCode := c.Param("code")
	tenant, tenantNode, cf, cfs := common.MustGetTenantData(c.Request.Context())

	var runAt *time.Time
	if req.RunAt != "" {
		parsedRunAt, err := time.Parse(time.RFC3339, req.RunAt)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid runAt timestamp format (expected RFC3339): " + err.Error()})
			return
		}
		runAt = &parsedRunAt
	}

	job, err := ctrl.ScheduledJobBO.CreateOneOffScheduledJob(
		c.Request.Context(),
		tenantCode,
		req.TargetType,
		req.TargetCode,
		req.VNamespace,
		req.Content,
		req.ContentType,
		req.Headers,
		req.Handler,
		req.Parameters,
		req.Priority,
		runAt,
		req.RunAfter,
		cf, cfs,
		tenant,
		tenantNode,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "OneOff scheduled job created",
		"result":  job,
	})
}

func (ctrl *ScheduledJobController) CreateRecurringScheduledJobHandler(c *gin.Context) {
	var req createRecurringScheduledJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ctrl.Config.Logger.Warn().Err(err).Msg("create recurring scheduled job attempt with invalid payload")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	tenantCode := c.Param("code")
	tenant, tenantNode, cf, cfs := common.MustGetTenantData(c.Request.Context())

	job, err := ctrl.ScheduledJobBO.CreateRecurringScheduledJob(
		c.Request.Context(),
		tenantCode,
		req.TargetType,
		req.TargetCode,
		req.VNamespace,
		req.Content,
		req.ContentType,
		req.Headers,
		req.Handler,
		req.Parameters,
		req.Priority,
		req.Every,
		req.CronExpression,
		cf, cfs,
		tenant,
		tenantNode,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Recurring scheduled job created",
		"result":  job,
	})
}

func (ctrl *ScheduledJobController) GetScheduledJobHandler(c *gin.Context) {
	id := c.Param("id")
	_, tenantNode, cf, cfs := common.MustGetTenantData(c.Request.Context())

	job, err := ctrl.ScheduledJobBO.GetScheduledJob(c.Request.Context(), id, cf, cfs, tenantNode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Scheduled job",
		"result":  job,
	})
}

func (ctrl *ScheduledJobController) GetScheduledJobsHandler(c *gin.Context) {
	pageParam := c.Query("pageSize")
	pageSize, err := strconv.Atoi(pageParam)
	if err != nil || pageSize < 1 {
		pageSize = 50
	} else if pageSize > 1000 {
		pageSize = 1000
	}

	_, tenantNode, cf, cfs := common.MustGetTenantData(c.Request.Context())

	findResult, err := ctrl.ScheduledJobBO.ListScheduledJobs(
		c.Request.Context(),
		c.Query("cursor"),
		pageSize,
		c.Query("vnamespace"),
		cf, cfs,
		tenantNode,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if findResult.Entities == nil {
		findResult.Entities = []models.ScheduledJob{}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Scheduled jobs list",
		"result":  findResult,
	})
}

func (ctrl *ScheduledJobController) DeleteScheduledJobHandler(c *gin.Context) {
	id := c.Param("id")
	_, tenantNode, cf, cfs := common.MustGetTenantData(c.Request.Context())

	err := ctrl.ScheduledJobBO.DeleteScheduledJob(c.Request.Context(), id, cf, cfs, tenantNode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Scheduled job " + id + " was deleted",
	})
}
