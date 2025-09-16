package exchange

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/infrastructure/server/common"
	bo "deadalus-orch/server/internal/usecase/business-logic"
	"deadalus-orch/shared/models"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type ExchangeController struct {
	Config     *common.ServerConfing
	ExchangeBO *bo.ExchangeBO
	TenantBO   *bo.TenantBO
}

func NewExchangeController(Config *common.ServerConfing) *ExchangeController {
	api := &ExchangeController{
		Config:     Config,
		ExchangeBO: bo.NewExchangeBO(Config),
		TenantBO:   bo.NewTenantBO(Config),
	}
	return api
}

type createExchangeRequest struct {
	Code       string            `json:"code" binding:"required"`
	Name       string            `json:"name" binding:"required"`
	Type       string            `json:"type" binding:"required"`
	VNamespace string            `json:"vnamespace" binding:"required"`
	Headers    map[string]string `json:"headers"`
}

type createBulkExchangeRequest struct {
	Exchanges []createExchangeRequest `json:"exchanges" binding:"required"`
}

type publishMessageRequest struct {
	ExchangeCode                   string           `json:"exchangeCode" binding:"required"`
	RoutingKeyOrPatternOrQueueCode string           `json:"routingKeyOrPatternOrQueueCode"`
	VNamespace                     string           `json:"vnamespace" binding:"required"`
	Message                        queueMessageData `json:"message" binding:"required"`
}

type queueMessageData struct {
	MessageID   string            `json:"messageId"`
	Handler     string            `json:"handler" binding:"required"`
	Priority    int               `json:"priority"`
	Parameters  map[string]string `json:"parameters"`
	Headers     map[string]string `json:"headers"`
	ContentType string            `json:"contentType"`
	Content     []byte            `json:"content"`
}

// CreateExchangeHandler handles POST /rest-api/tenants
func (ctrl *ExchangeController) CreateExchangeHandler(c *gin.Context) {
	var req createExchangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ctrl.Config.Logger.Warn().Err(err).Msg("create tenant attempt with invalid payload")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	tenantCode := c.Param("code")
	tenant, _, _, err := ctrl.TenantBO.GetTenant(c.Request.Context(), tenantCode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	exchange, err := ctrl.ExchangeBO.CreateExchange(c.Request.Context(), req.Code, req.VNamespace, req.Name, models.ExchangeType(req.Type), req.Headers, db.ColumnFamilyPrefix+strconv.Itoa(tenant.ColumnFamilyIndex), tenant.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Exchange was asserted",
		"result":  exchange,
	})
}

func (ctrl *ExchangeController) BulkCreateExchangeHandler(c *gin.Context) {
	var req createBulkExchangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ctrl.Config.Logger.Warn().Err(err).Msg("create tenant attempt with invalid payload")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}
	tenantCode := c.Param("code")

	tenant, _, _, err := ctrl.TenantBO.GetTenant(c.Request.Context(), tenantCode)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	exchanges := []*models.Exchange{}

	for _, t := range req.Exchanges {
		exchange := &models.Exchange{
			Code:       t.Code,
			VNamespace: t.VNamespace,
			Name:       t.Name,
			Type:       models.ExchangeType(t.Type),
			Headers:    t.Headers,
		}
		exchanges = append(exchanges, exchange)
	}
	exchangesResult, err := ctrl.ExchangeBO.BulkCreateExchange(c.Request.Context(), exchanges, db.ColumnFamilyPrefix+strconv.Itoa(tenant.ColumnFamilyIndex), tenant.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Exchanges were asserted",
		"result":  exchangesResult,
	})
}

// GetExchangeHandler handles GET /rest-api/tenants/:code/exchange/:exchangeCode/:vnamespace
func (ctrl *ExchangeController) GetExchangeHandler(c *gin.Context) {
	exchangeCode := c.Param("exchangeCode")
	vnamespace := c.Param("vnamespace")
	tenantCode := c.Param("code")

	tenant, _, _, err := ctrl.TenantBO.GetTenant(c.Request.Context(), tenantCode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	exchange, err := ctrl.ExchangeBO.GetExchange(c.Request.Context(), exchangeCode, vnamespace, db.ColumnFamilyPrefix+strconv.Itoa(tenant.ColumnFamilyIndex), tenant.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Exchange",
		"result":  exchange,
	})
}

// DeleteExchangeHandler handles DELETE /rest-api/tenants/:code/exchange/:exchangeCode/:vnamespace
func (ctrl *ExchangeController) DeleteExchangeHandler(c *gin.Context) {
	exchangeCode := c.Param("exchangeCode")
	vnamespace := c.Param("vnamespace")
	tenantCode := c.Param("code")

	tenant, _, _, err := ctrl.TenantBO.GetTenant(c.Request.Context(), tenantCode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	err = ctrl.ExchangeBO.DeleteExchange(c.Request.Context(), exchangeCode, vnamespace, db.ColumnFamilyPrefix+strconv.Itoa(tenant.ColumnFamilyIndex), tenant.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Exchange " + exchangeCode + " in namespace " + vnamespace + " was deleted",
	})
}

func (ctrl *ExchangeController) GetExchangesHandler(c *gin.Context) {
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

	findResult, err := ctrl.ExchangeBO.GetExchanges(c.Request.Context(), c.Query("q"), c.Query("cursor"), page, c.Query("vnamespace"), db.ColumnFamilyPrefix+strconv.Itoa(tenant.ColumnFamilyIndex), tenant.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if findResult.Entities == nil {
		findResult.Entities = []models.Exchange{}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Exchange list",
		"result":  findResult,
	})
}

// PublishMessageHandler handles POST /rest-api/tenants/:code/exchange/publish-message
func (ctrl *ExchangeController) PublishMessageHandler(c *gin.Context) {
	var req publishMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ctrl.Config.Logger.Warn().Err(err).Msg("publish message attempt with invalid payload")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	tenantCode := c.Param("code")
	tenant, _, _, err := ctrl.TenantBO.GetTenant(c.Request.Context(), tenantCode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Convert request to models.QueueMessage
	message := models.QueueMessage{
		MessageID:   req.Message.MessageID,
		Handler:     req.Message.Handler,
		Priority:    req.Message.Priority,
		Parameters:  req.Message.Parameters,
		Headers:     req.Message.Headers,
		ContentType: req.Message.ContentType,
		Content:     req.Message.Content,
	}

	queueCodes, err := ctrl.ExchangeBO.PublishMessage(
		c.Request.Context(),
		req.ExchangeCode,
		req.RoutingKeyOrPatternOrQueueCode,
		req.VNamespace,
		message,
		db.ColumnFamilyPrefix+strconv.Itoa(tenant.ColumnFamilyIndex),
		tenant.ID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "Message published successfully",
		"queueCodes": queueCodes,
	})
}
