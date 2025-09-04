package binding

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

type BindingController struct {
	Config    *common.ServerConfing
	BindingBO *bo.BindingBO
	TenantBO  *bo.TenantBO
}

func NewBindingController(Config *common.ServerConfing) *BindingController {
	api := &BindingController{
		Config:    Config,
		BindingBO: bo.NewBindingBO(Config),
		TenantBO:  bo.NewTenantBO(Config),
	}
	return api
}

type createBindingRequest struct {
	ExchangeCode string            `json:"exchangeCode" binding:"required"`
	QueueCode    string            `json:"queueCode" binding:"required"`
	VNamespace   string            `json:"vnamespace" binding:"required"`
	RoutingKey   string            `json:"routingKey"`
	Pattern      string            `json:"pattern"`
	XMatch       string            `json:"xMatch"`
	BindingType  string            `json:"bindingType"`
	Headers      map[string]string `json:"headers"`
}

// CreateBindingHandler handles POST /rest-api/tenants/:id/binding
func (ctrl *BindingController) CreateBindingHandler(c *gin.Context) {
	var req createBindingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ctrl.Config.Logger.Warn().Err(err).Msg("create binding attempt with invalid payload")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	// Validate binding type
	if req.BindingType != "" && !isValidBindingType(req.BindingType) {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid binding type: %s. Valid types are: classic, dynamic", req.BindingType)})
		return
	}

	// Validate XMatch type
	if req.XMatch != "" && !isValidXMatchType(req.XMatch) {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid xMatch type: %s. Valid types are: all, any", req.XMatch)})
		return
	}

	tenantID := c.Param("id")
	tenant, _, _, err := ctrl.TenantBO.GetTenant(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
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
		req.QueueCode,
		req.ExchangeCode,
		req.VNamespace,
		req.RoutingKey,
		req.Pattern,
		xMatch,
		bindingType,
		req.Headers,
		db.ColumnFamilyPrefix+strconv.Itoa(tenant.ColumnFamilyIndex),
		tenant.ID,
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

// GetBindingHandler handles GET /rest-api/tenants/:id/binding/:exchangeCode/:queueCode/:vnamespace
func (ctrl *BindingController) GetBindingHandler(c *gin.Context) {
	exchangeCode := c.Param("exchangeCode")
	queueCode := c.Param("queueCode")
	vnamespace := c.Param("vnamespace")
	tenantID := c.Param("id")

	tenant, _, _, err := ctrl.TenantBO.GetTenant(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	binding, err := ctrl.BindingBO.GetBinding(
		c.Request.Context(),
		exchangeCode,
		queueCode,
		vnamespace,
		db.ColumnFamilyPrefix+strconv.Itoa(tenant.ColumnFamilyIndex),
		tenant.ID,
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

// DeleteBindingHandler handles DELETE /rest-api/tenants/:id/binding/:exchangeCode/:queueCode/:vnamespace
func (ctrl *BindingController) DeleteBindingHandler(c *gin.Context) {
	exchangeCode := c.Param("exchangeCode")
	queueCode := c.Param("queueCode")
	vnamespace := c.Param("vnamespace")
	tenantID := c.Param("id")

	tenant, _, _, err := ctrl.TenantBO.GetTenant(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	err = ctrl.BindingBO.DeleteBinding(
		c.Request.Context(),
		exchangeCode,
		queueCode,
		vnamespace,
		db.ColumnFamilyPrefix+strconv.Itoa(tenant.ColumnFamilyIndex),
		tenant.ID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Binding between exchange %s and queue %s in namespace %s was deleted", exchangeCode, queueCode, vnamespace),
	})
}

// GetBindingsHandler handles GET /rest-api/tenants/:id/bindings
func (ctrl *BindingController) GetBindingsHandler(c *gin.Context) {
	pageParam := c.Query("pageSize")
	tenantID := c.Param("id")

	tenant, _, _, err := ctrl.TenantBO.GetTenant(c.Request.Context(), tenantID)
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
		db.ColumnFamilyPrefix+strconv.Itoa(tenant.ColumnFamilyIndex),
		tenant.ID,
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

// isValidBindingType validates if the binding type is one of the allowed types
func isValidBindingType(bindingType string) bool {
	switch bindingType {
	case "classic", "dynamic":
		return true
	default:
		return false
	}
}

// isValidXMatchType validates if the XMatch type is one of the allowed types
func isValidXMatchType(xMatch string) bool {
	switch xMatch {
	case "all", "any":
		return true
	default:
		return false
	}
}
