package tenant

import (
	"deadalus-orch/server/internal/infrastructure/server/common"
	bo "deadalus-orch/server/internal/usecase/business-logic"
	"deadalus-orch/shared/models"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type TenantController struct {
	Config   *common.ServerConfing
	TenantBO *bo.TenantBO
}

// NewTenantController creates a new instance of RestAdminAPI.
func NewTenantController(Config *common.ServerConfing) *TenantController {
	api := &TenantController{
		Config:   Config,
		TenantBO: bo.NewTenantBO(Config),
	}
	return api
}

type createTenantInMasterRequest struct {
	Code string `json:"code" binding:"required"`
	Name string `json:"name" binding:"required"`
}

// CreateTenantHandler handles POST /admin-api/tenants
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
		"message": "Tenant was created",
		"result":  tenantInMaster,
	})
}

// GetTenantHandler handles GET /admin-api/tenants/:id
func (ctrl *TenantController) GetTenantHandler(c *gin.Context) {
	tenantID := c.Param("id")
	tenantInMaster, node, nodeHostInfo, err := ctrl.TenantBO.GetTenant(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if nodeHostInfo == nil {
		c.JSON(http.StatusOK, gin.H{
			"message": "Tenant",
			"result":  tenantInMaster,
		})
	} else {
		c.JSON(http.StatusOK, gin.H{
			"message": "Tenant",
			"result":  tenantInMaster,
			"node": gin.H{
				"SelfMember":   node.SelfMember,
				"ShardID":      node.ShardID,
				"Roles":        node.Roles,
				"NodeHostInfo": nodeHostInfo,
			},
		})
	}
}

func (ctrl *TenantController) DeleteTenantHandler(c *gin.Context) {
	tenantID := c.Param("id")

	err := ctrl.TenantBO.DeleteTenant(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Tenant " + tenantID + " was deleted",
	})
}

func (ctrl *TenantController) GetTenantsHandler(c *gin.Context) {
	pageParam := c.Query("page")
	page, err := strconv.Atoi(pageParam)
	if err != nil || page < 2 {
		page = 50
	} else if page > 1000 {
		page = 1000
	}

	findResult, err := ctrl.TenantBO.GetTenants(c.Request.Context(), c.Query("cursor"), page)
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
