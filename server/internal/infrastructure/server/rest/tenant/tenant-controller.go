package tenant

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/infrastructure/server/common"
	bo "deadalus-orch/server/internal/usecase/business-logic"
	"deadalus-orch/shared/models"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type TenantController struct {
	Config          *common.ServerConfing
	TenantBO        *bo.TenantBO
	TenantSummaryBO *bo.TenantSummaryBO
}

func NewTenantController(Config *common.ServerConfing) *TenantController {
	api := &TenantController{
		Config:          Config,
		TenantBO:        bo.NewTenantBO(Config),
		TenantSummaryBO: bo.NewTenantSummaryBO(Config),
	}
	return api
}

type createTenantInMasterRequest struct {
	Code string `json:"code" binding:"required"`
	Name string `json:"name" binding:"required"`
}

type createBulkTenantInMasterRequest struct {
	Tenants []createTenantInMasterRequest `json:"tenants" binding:"required"`
}

// CreateTenantHandler handles POST /rest-api/tenants
func (ctrl *TenantController) CreateTenantHandler(c *gin.Context) {
	var req createTenantInMasterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ctrl.Config.Logger.Warn().Err(err).Msg("create tenant attempt with invalid payload")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	tenantInMaster, err := ctrl.TenantBO.CreateTenant(c.Request.Context(), req.Code, req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Tenant was asserted",
		"result":  tenantInMaster,
	})
}

func (ctrl *TenantController) BulkCreateTenantHandler(c *gin.Context) {
	var req createBulkTenantInMasterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ctrl.Config.Logger.Warn().Err(err).Msg("create tenant attempt with invalid payload")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}
	tenants := []*models.TenantInMaster{}
	for _, t := range req.Tenants {
		tenant := &models.TenantInMaster{
			Code: t.Code,
			Name: t.Name,
		}
		tenants = append(tenants, tenant)
	}
	tenantsInMaster, err := ctrl.TenantBO.BulkCreateTenant(c.Request.Context(), tenants)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Tenants were asserted",
		"result":  tenantsInMaster,
	})
}

// GetTenantHandler handles GET /rest-api/tenants/:code
func (ctrl *TenantController) GetTenantHandler(c *gin.Context) {
	tenantCode := c.Param("code")
	tenantInMaster, node, _, err := ctrl.TenantBO.GetTenant(c.Request.Context(), tenantCode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Tenant",
		"result":  tenantInMaster,
		"node": gin.H{
			"SelfMember": node.SelfMember,
			"ShardID":    node.ShardID,
			"Roles":      node.Roles,
		},
	})
}

// GetTenantSummaryHandler handles GET /rest-api/tenants/:code/summary
func (ctrl *TenantController) GetTenantSummaryHandler(c *gin.Context) {
	tenantCode := c.Param("code")

	// Get tenant info first to extract CF and CFS
	tenant, _, _, err := ctrl.TenantBO.GetTenant(c.Request.Context(), tenantCode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	cf := db.ColumnFamilyPrefix + strconv.Itoa(tenant.ColumnFamilyIndex)
	cfs := tenant.ID

	// For tenant summary, we still use the tenant node from the TenantBO result
	_, tenantNode, _, _ := ctrl.TenantBO.GetTenant(c.Request.Context(), tenantCode)

	tenantSummary, err := ctrl.TenantSummaryBO.GetTenantSummary(c.Request.Context(), tenant.ID, cf, cfs, &tenant, tenantNode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Tenant Summary",
		"result":  tenantSummary,
	})
}

func (ctrl *TenantController) DeleteTenantHandler(c *gin.Context) {
	tenantCode := c.Param("code")

	err := ctrl.TenantBO.DeleteTenant(c.Request.Context(), tenantCode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Tenant " + tenantCode + " was deleted",
	})
}

func (ctrl *TenantController) GetTenantsHandler(c *gin.Context) {
	pageParam := c.Query("pageSize")
	page, err := strconv.Atoi(pageParam)
	if err != nil || page < 2 {
		page = 50
	} else if page > 1000 {
		page = 1000
	}

	findResult, err := ctrl.TenantBO.GetTenants(c.Request.Context(), c.Query("q"), c.Query("cursor"), page)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if findResult.Entities == nil {
		findResult.Entities = []models.TenantInMaster{}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Tenant list",
		"result":  findResult,
	})
}
