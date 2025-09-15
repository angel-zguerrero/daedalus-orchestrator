package queue

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/infrastructure/server/common"
	bo "deadalus-orch/server/internal/usecase/business-logic"
	"deadalus-orch/shared/models"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type QueueController struct {
	Config   *common.ServerConfing
	QueueBO  *bo.QueueBO
	TenantBO *bo.TenantBO
}

func NewQueueController(Config *common.ServerConfing) *QueueController {
	api := &QueueController{
		Config:   Config,
		QueueBO:  bo.NewQueueBO(Config),
		TenantBO: bo.NewTenantBO(Config),
	}
	return api
}

type createQueueRequest struct {
	Code                      string            `json:"code" binding:"required"`
	Name                      string            `json:"name" binding:"required"`
	Type                      string            `json:"type" binding:"required"`
	State                     string            `json:"state"`
	VNamespace                string            `json:"vnamespace" binding:"required"`
	TTLQueue                  int               `json:"ttlQueue"`
	AllowDuplicated           bool              `json:"allowDuplicated"`
	MaxAttempts               int               `json:"maxAttempts"`
	DesiredPriorityThresholds map[int]int       `json:"desiredPriorityThresholds"`
	Headers                   map[string]string `json:"headers"`
}

type createBulkQueueRequest struct {
	Queues []createQueueRequest `json:"queues" binding:"required"`
}

// CreateQueueHandler handles POST /rest-api/tenants/:id/queue
func (ctrl *QueueController) CreateQueueHandler(c *gin.Context) {
	var req createQueueRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ctrl.Config.Logger.Warn().Err(err).Msg("create queue attempt with invalid payload")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	// Validate queue type
	if !isValidQueueType(req.Type) {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid queue type: %s. Valid types are: standard, delayed, dead-letter", req.Type)})
		return
	}

	tenantCode := c.Param("code")
	tenant, _, _, err := ctrl.TenantBO.GetTenant(c.Request.Context(), tenantCode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Set default state if not provided
	if req.State == "" {
		req.State = string(models.QueueActive)
	}

	// Set default values for new properties
	if req.MaxAttempts == 0 {
		req.MaxAttempts = 1
	}

	// Create queue with all properties
	queue := &models.Queue{
		Code:                      req.Code,
		VNamespace:                req.VNamespace,
		Name:                      req.Name,
		Type:                      models.QueueType(req.Type),
		State:                     models.QueueState(req.State),
		TTLQueue:                  req.TTLQueue,
		AllowDuplicated:           req.AllowDuplicated,
		MaxAttempts:               req.MaxAttempts,
		DesiredPriorityThresholds: req.DesiredPriorityThresholds,
		Headers:                   req.Headers, // Add headers support
	}

	queuesResult, err := ctrl.QueueBO.BulkCreateQueue(c.Request.Context(), []*models.Queue{queue}, db.ColumnFamilyPrefix+strconv.Itoa(tenant.ColumnFamilyIndex), tenant.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Queue was asserted",
		"result":  queuesResult[0],
	})
}

func (ctrl *QueueController) BulkCreateQueueHandler(c *gin.Context) {
	var req createBulkQueueRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ctrl.Config.Logger.Warn().Err(err).Msg("create queue attempt with invalid payload")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	// Validate queue types
	for _, queue := range req.Queues {
		if !isValidQueueType(queue.Type) {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid queue type: %s. Valid types are: standard, delayed, dead-letter", queue.Type)})
			return
		}
	}

	tenantCode := c.Param("code")

	tenant, _, _, err := ctrl.TenantBO.GetTenant(c.Request.Context(), tenantCode)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	queues := []*models.Queue{}

	for _, t := range req.Queues {
		// Set default state if not provided
		if t.State == "" {
			t.State = string(models.QueueActive)
		}
		// Set default values for new properties
		if t.MaxAttempts == 0 {
			t.MaxAttempts = 1
		}
		queue := &models.Queue{
			Code:                      t.Code,
			VNamespace:                t.VNamespace,
			Name:                      t.Name,
			Type:                      models.QueueType(t.Type),
			State:                     models.QueueState(t.State),
			TTLQueue:                  t.TTLQueue,
			AllowDuplicated:           t.AllowDuplicated,
			MaxAttempts:               t.MaxAttempts,
			DesiredPriorityThresholds: t.DesiredPriorityThresholds,
			Headers:                   t.Headers, // Add headers support
		}
		queues = append(queues, queue)
	}

	queuesResult, err := ctrl.QueueBO.BulkCreateQueue(c.Request.Context(), queues, db.ColumnFamilyPrefix+strconv.Itoa(tenant.ColumnFamilyIndex), tenant.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Queues were asserted",
		"result":  queuesResult,
	})
}

// GetQueueHandler handles GET /rest-api/tenants/:code/queue/:queueCode/:vnamespace
func (ctrl *QueueController) GetQueueHandler(c *gin.Context) {
	queueCode := c.Param("queueCode")
	vnamespace := c.Param("vnamespace")
	tenantCode := c.Param("code")

	tenant, _, _, err := ctrl.TenantBO.GetTenant(c.Request.Context(), tenantCode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	queue, err := ctrl.QueueBO.GetQueue(c.Request.Context(), queueCode, vnamespace, false, db.ColumnFamilyPrefix+strconv.Itoa(tenant.ColumnFamilyIndex), tenant.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Queue",
		"result":  queue,
	})
}

// DeleteQueueHandler handles DELETE /rest-api/tenants/:code/queue/:queueCode/:vnamespace
func (ctrl *QueueController) DeleteQueueHandler(c *gin.Context) {
	queueCode := c.Param("queueCode")
	vnamespace := c.Param("vnamespace")
	tenantCode := c.Param("code")

	tenant, _, _, err := ctrl.TenantBO.GetTenant(c.Request.Context(), tenantCode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	err = ctrl.QueueBO.DeleteQueue(c.Request.Context(), queueCode, vnamespace, db.ColumnFamilyPrefix+strconv.Itoa(tenant.ColumnFamilyIndex), tenant.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Queue " + queueCode + " in namespace " + vnamespace + " was deleted",
	})
}

func (ctrl *QueueController) GetQueuesHandler(c *gin.Context) {
	pageParam := c.Query("pageSize")
	tenantCode := c.Param("code")

	tenant, _, _, err := ctrl.TenantBO.GetTenant(c.Request.Context(), tenantCode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	page, err := strconv.Atoi(pageParam)
	if err != nil || page < 2 {
		page = 50
	} else if page > 1000 {
		page = 1000
	}

	// Check if headers should be included from query parameter
	includeHeaders := c.Query("includeHeaders") == "true"

	findResult, err := ctrl.QueueBO.GetQueues(c.Request.Context(), c.Query("q"), c.Query("cursor"), page, c.Query("vnamespace"), includeHeaders, db.ColumnFamilyPrefix+strconv.Itoa(tenant.ColumnFamilyIndex), tenant.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if findResult.Entities == nil {
		findResult.Entities = []models.Queue{}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Queue list",
		"result":  findResult,
	})
}

// isValidQueueType validates if the queue type is one of the allowed types
func isValidQueueType(queueType string) bool {
	switch queueType {
	case "standard", "delayed", "dead-letter":
		return true
	default:
		return false
	}
}
