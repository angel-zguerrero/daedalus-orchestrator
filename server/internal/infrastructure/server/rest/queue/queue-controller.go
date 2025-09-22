package queue

import (
	"deadalus-orch/server/internal/infrastructure/server/common"
	bo "deadalus-orch/server/internal/usecase/business-logic"
	"deadalus-orch/shared/models"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type QueueController struct {
	Config  *common.ServerConfing
	QueueBO *bo.QueueBO
}

func NewQueueController(Config *common.ServerConfing) *QueueController {
	api := &QueueController{
		Config:  Config,
		QueueBO: bo.NewQueueBO(Config),
	}
	return api
}

type createQueueRequest struct {
	Code                                  string            `json:"code" binding:"required"`
	Name                                  string            `json:"name" binding:"required"`
	Type                                  string            `json:"type" binding:"required"`
	State                                 string            `json:"state"`
	VNamespace                            string            `json:"vnamespace" binding:"required"`
	DefaultQueueMessageTTL                int               `json:"defaultQueueMessageTTL" binding:"min=0"`
	DefaultQueueMessageDelayTime          int               `json:"defaultQueueMessageDelayTime" binding:"min=0"`
	QueueExpires                          int               `json:"queueExpires" binding:"min=0"`
	AllowDuplicated                       bool              `json:"allowDuplicated"`
	MaxAttempts                           int               `json:"maxAttempts"`
	DesiredPriorityThresholds             map[int]int       `json:"desiredPriorityThresholds"`
	Headers                               map[string]string `json:"headers"`
	DeadLetterExchangeId                  string            `json:"deadLetterExchangeId"`
	DeadLetterExchangeRoutingKeyOrPattern string            `json:"deadLetterExchangeRoutingKeyOrPattern"`
}

type createBulkQueueRequest struct {
	Queues []createQueueRequest `json:"queues" binding:"required"`
}

type enqueueMessageRequest struct {
	Content     string            `json:"content"`
	ContentType string            `json:"contentType"`
	Headers     map[string]string `json:"headers"`
	Priority    int               `json:"priority"`
	Handler     string            `json:"handler" binding:"required"`
	Parameters  map[string]string `json:"parameters"`
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
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid queue type: %s. Valid types are: standard", req.Type)})
		return
	}

	tenant, tenantNode, cf, cfs := common.MustGetTenantData(c.Request.Context())

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
		Code:                                  req.Code,
		VNamespace:                            req.VNamespace,
		Name:                                  req.Name,
		Type:                                  models.QueueType(req.Type),
		State:                                 models.QueueState(req.State),
		DefaultQueueMessageTTL:                req.DefaultQueueMessageTTL,
		DefaultQueueMessageDelayTime:          req.DefaultQueueMessageDelayTime,
		QueueExpires:                          req.QueueExpires,
		AllowDuplicated:                       req.AllowDuplicated,
		MaxAttempts:                           req.MaxAttempts,
		DesiredPriorityThresholds:             req.DesiredPriorityThresholds,
		Headers:                               req.Headers,
		DeadLetterExchangeId:                  req.DeadLetterExchangeId,
		DeadLetterExchangeRoutingKeyOrPattern: req.DeadLetterExchangeRoutingKeyOrPattern,
	}

	queuesResult, err := ctrl.QueueBO.BulkCreateQueue(c.Request.Context(), []*models.Queue{queue}, cf, cfs, tenant, tenantNode)
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
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid queue type: %s. Valid types are: standard", queue.Type)})
			return
		}
	}

	tenant, tenantNode, cf, cfs := common.MustGetTenantData(c.Request.Context())

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
			Code:                                  t.Code,
			VNamespace:                            t.VNamespace,
			Name:                                  t.Name,
			Type:                                  models.QueueType(t.Type),
			State:                                 models.QueueState(t.State),
			DefaultQueueMessageTTL:                t.DefaultQueueMessageTTL,
			DefaultQueueMessageDelayTime:          t.DefaultQueueMessageDelayTime,
			QueueExpires:                          t.QueueExpires,
			AllowDuplicated:                       t.AllowDuplicated,
			MaxAttempts:                           t.MaxAttempts,
			DesiredPriorityThresholds:             t.DesiredPriorityThresholds,
			Headers:                               t.Headers,
			DeadLetterExchangeId:                  t.DeadLetterExchangeId,
			DeadLetterExchangeRoutingKeyOrPattern: t.DeadLetterExchangeRoutingKeyOrPattern,
		}
		queues = append(queues, queue)
	}

	queuesResult, err := ctrl.QueueBO.BulkCreateQueue(c.Request.Context(), queues, cf, cfs, tenant, tenantNode)
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

	tenant, tenantNode, cf, cfs := common.MustGetTenantData(c.Request.Context())

	queue, err := ctrl.QueueBO.GetQueue(c.Request.Context(), queueCode, vnamespace, false, cf, cfs, tenant, tenantNode)
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

	tenant, tenantNode, cf, cfs := common.MustGetTenantData(c.Request.Context())

	err := ctrl.QueueBO.DeleteQueue(c.Request.Context(), queueCode, vnamespace, cf, cfs, tenant, tenantNode)
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

	tenant, tenantNode, cf, cfs := common.MustGetTenantData(c.Request.Context())

	page, err := strconv.Atoi(pageParam)
	if err != nil || page < 2 {
		page = 50
	} else if page > 1000 {
		page = 1000
	}

	// Check if headers should be included from query parameter
	includeHeaders := c.Query("includeHeaders") == "true"

	findResult, err := ctrl.QueueBO.GetQueues(c.Request.Context(), c.Query("q"), c.Query("cursor"), page, c.Query("vnamespace"), includeHeaders, cf, cfs, tenant, tenantNode)
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

// EnqueueMessageHandler handles POST /rest-api/tenants/:code/queue/:queueCode/:vnamespace/enqueue
func (ctrl *QueueController) EnqueueMessageHandler(c *gin.Context) {
	var req enqueueMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ctrl.Config.Logger.Warn().Err(err).Msg("enqueue message attempt with invalid payload")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	queueCode := c.Param("queueCode")
	vnamespace := c.Param("vnamespace")

	tenant, tenantNode, cf, cfs := common.MustGetTenantData(c.Request.Context())

	// Create the message
	message := models.QueueMessage{
		Content:     []byte(req.Content),
		ContentType: req.ContentType,
		Headers:     req.Headers,
		Priority:    req.Priority,
		Handler:     req.Handler,
		Parameters:  req.Parameters,
		VNamespace:  vnamespace,
	}

	// Enqueue the message
	messageID, err := ctrl.QueueBO.EnqueueMessage(c.Request.Context(), queueCode, message, vnamespace, cf, cfs, tenant, tenantNode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "Message enqueued successfully",
		"messageId": messageID,
		"result": gin.H{
			queueCode: messageID,
		},
	})
}

// isValidQueueType validates if the queue type is one of the allowed types
func isValidQueueType(queueType string) bool {
	switch queueType {
	case "standard":
		return true
	default:
		return false
	}
}
