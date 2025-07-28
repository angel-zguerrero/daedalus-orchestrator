package exchange

import (
	"deadalus-orch/server/internal/infrastructure/server/common"
	bo "deadalus-orch/server/internal/usecase/business-logic"
	"deadalus-orch/shared/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

type ExchangeController struct {
	Config     *common.ServerConfing
	ExchangeBO *bo.ExchangeBO
}

func NewExchangeController(Config *common.ServerConfing) *ExchangeController {
	api := &ExchangeController{
		Config:     Config,
		ExchangeBO: bo.NewExchangeBO(Config),
	}
	return api
}

type createExchangeRequest struct {
	Name string `json:"name" binding:"required"`
	Type string `json:"type" binding:"required"`
}

type createBulkExchangeRequest struct {
	Exchanges []createExchangeRequest `json:"tenants" binding:"required"`
}

// AssertExchangeHandler handles POST /rest-api/tenants
func (ctrl *ExchangeController) AssertExchangeHandler(c *gin.Context) {
	var req createExchangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ctrl.Config.Logger.Warn().Err(err).Msg("create tenant attempt with invalid payload")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	tenantID := c.Param("id")

	tenant, err := ctrl.ExchangeBO.AssertExchange(c.Request.Context(), tenantID, req.Name, models.ExchangeType(req.Type))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Exchange was asserted",
		"result":  tenant,
	})
}

func (ctrl *ExchangeController) AssertExchangesHandler(c *gin.Context) {
	var req createBulkExchangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ctrl.Config.Logger.Warn().Err(err).Msg("create tenant attempt with invalid payload")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}
	tenantID := c.Param("id")
	exchanges := []*models.Exchange{}

	for _, t := range req.Exchanges {
		exchange := &models.Exchange{
			TenantID: tenantID,
			Name:     t.Name,
			Type:     models.ExchangeType(t.Type),
		}
		exchanges = append(exchanges, exchange)
	}
	exchangesResult, err := ctrl.ExchangeBO.AssertExchanges(c.Request.Context(), exchanges)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Exchanges were asserted",
		"result":  exchangesResult,
	})
}

/*
// GetExchangeHandler handles GET /rest-api/tenants/:id
func (ctrl *ExchangeController) GetExchangeHandler(c *gin.Context) {
	tenantID := c.Param("id")
	tenant, node, _, err := ctrl.ExchangeBO.GetExchange(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Exchange",
		"result":  tenant,
		"node": gin.H{
			"SelfMember": node.SelfMember,
			"ShardID":    node.ShardID,
			"Roles":      node.Roles,
		},
	})
}

func (ctrl *ExchangeController) DeleteExchangeHandler(c *gin.Context) {
	tenantID := c.Param("id")

	err := ctrl.ExchangeBO.DeleteExchange(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Exchange " + tenantID + " was deleted",
	})
}

func (ctrl *ExchangeController) GetExchangesHandler(c *gin.Context) {
	pageParam := c.Query("pageSize")
	page, err := strconv.Atoi(pageParam)
	if err != nil || page < 2 {
		page = 50
	} else if page > 1000 {
		page = 1000
	}

	findResult, err := ctrl.ExchangeBO.GetExchanges(c.Request.Context(), c.Query("q"), c.Query("cursor"), page)
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
*/
