package binding

import (
	"deadalus-orch/server/internal/infrastructure/server/common"
	bo "deadalus-orch/server/internal/usecase/business-logic"
	"deadalus-orch/shared/models"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type BindingController struct {
	Config    *common.ServerConfing
	BindingBO *bo.BindingBO
}

func NewBindingController(Config *common.ServerConfing) *BindingController {
	api := &BindingController{
		Config:    Config,
		BindingBO: bo.NewBindingBO(Config),
	}
	return api
}

type createBindingRequest struct {
	Code                  string            `json:"code" binding:"required"`
	ExchangeCode          string            `json:"exchangeCode" binding:"required"`
	QueueCode             string            `json:"queueCode"`
	TargetExchangeCode    string            `json:"targetExchangeCode"`
	AlternateExchangeCode string            `json:"alternateExchangeCode"`
	VNamespace            string            `json:"vnamespace" binding:"required"`
	RoutingKey            string            `json:"routingKey"`
	Pattern               string            `json:"pattern"`
	XMatch                string            `json:"xMatch"`
	BindingType           string            `json:"bindingType"`
	TargetExchangeType    string            `json:"targetExchangeType"`
	Headers               map[string]string `json:"headers"`
}

// CreateBindingHandler handles POST /rest-api/tenants/:id/binding
func (ctrl *BindingController) CreateBindingHandler(c *gin.Context) {
	var req createBindingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ctrl.Config.Logger.Warn().Err(err).Msg("create binding attempt with invalid payload")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	tenant, tenantNode, cf, cfs := common.MustGetTenantData(c.Request.Context())

	// Set default target exchange type if not provided
	targetExchangeType := models.TargetExchangeTypeQueue
	if req.TargetExchangeType != "" {
		targetExchangeType = models.TargetExchangeType(req.TargetExchangeType)
	}

	// Set default binding type if not provided
	bindingType := models.BindingTypeClassic
	if req.BindingType != "" {
		bindingType = models.BindingType(req.BindingType)
	}

	// Set default XMatch if not provided
	xMatch := models.XMatchTypeAll
	if req.XMatch != "" {
		xMatch = models.XMatchType(req.XMatch)
	}

	binding, err := ctrl.BindingBO.CreateBinding(
		c.Request.Context(),
		req.Code,
		req.QueueCode,
		req.ExchangeCode,
		req.TargetExchangeCode,
		req.AlternateExchangeCode,
		req.VNamespace,
		req.RoutingKey,
		req.Pattern,
		xMatch,
		bindingType,
		targetExchangeType,
		req.Headers,
		cf,
		cfs,
		tenant,
		tenantNode,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Binding was created",
		"result":  binding,
	})
}

// GetBindingHandler handles GET /rest-api/tenants/:code/binding/:exchangeCode/:queueCode/:vnamespace
func (ctrl *BindingController) GetBindingHandler(c *gin.Context) {
	exchangeCode := c.Param("exchangeCode")
	queueCode := c.Param("queueCode")
	vnamespace := c.Param("vnamespace")

	tenant, tenantNode, cf, cfs := common.MustGetTenantData(c.Request.Context())

	binding, err := ctrl.BindingBO.GetBinding(
		c.Request.Context(),
		exchangeCode,
		queueCode,
		vnamespace,
		cf,
		cfs,
		tenant,
		tenantNode,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Binding",
		"result":  binding,
	})
}

// DeleteBindingHandler handles DELETE /rest-api/tenants/:code/binding/:bindingCode/:vnamespace
func (ctrl *BindingController) DeleteBindingHandler(c *gin.Context) {
	bindingCode := c.Param("bindingCode")
	vnamespace := c.Param("vnamespace")

	tenant, tenantNode, cf, cfs := common.MustGetTenantData(c.Request.Context())

	err := ctrl.BindingBO.DeleteBinding(
		c.Request.Context(),
		bindingCode,
		vnamespace,
		cf,
		cfs,
		tenant,
		tenantNode,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Binding with code %s in namespace %s was deleted", bindingCode, vnamespace),
	})
}

// GetBindingsHandler handles GET /rest-api/tenants/:id/bindings
func (ctrl *BindingController) GetBindingsHandler(c *gin.Context) {
	pageParam := c.Query("pageSize")

	tenant, tenantNode, cf, cfs := common.MustGetTenantData(c.Request.Context())

	page, err := strconv.Atoi(pageParam)
	if err != nil || page < 2 {
		page = 50
	} else if page > 1000 {
		page = 1000
	}

	// Parse includeObjects parameter
	includeObjects := false
	if includeObjectsParam := c.Query("includeObjects"); includeObjectsParam != "" {
		includeObjects, _ = strconv.ParseBool(includeObjectsParam)
	}

	findResult, err := ctrl.BindingBO.GetBindings(
		c.Request.Context(),
		c.Query("q"),
		c.Query("cursor"),
		page,
		c.Query("vnamespace"),
		includeObjects,
		cf,
		cfs,
		tenant,
		tenantNode,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Binding list",
		"result":  findResult,
	})
}
