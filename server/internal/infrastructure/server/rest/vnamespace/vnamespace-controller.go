package vnamespace

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/infrastructure/server/common"
	bo "deadalus-orch/server/internal/usecase/business-logic"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type VNamespaceController struct {
	Config       *common.ServerConfing
	VNamespaceBO *bo.VNamespaceBO
	TenantBO     *bo.TenantBO
}

func NewVNamespaceController(Config *common.ServerConfing) *VNamespaceController {
	api := &VNamespaceController{
		Config:       Config,
		VNamespaceBO: bo.NewVNamespaceBO(Config),
		TenantBO:     bo.NewTenantBO(Config),
	}
	return api
}

// GetVNamespacesHandler handles GET /rest-api/tenants/:code/vnamespaces
func (ctrl *VNamespaceController) GetVNamespacesHandler(c *gin.Context) {
	tenantCode := c.Param("code")

	// Get query parameters
	query := c.DefaultQuery("q", "")
	cursor := c.DefaultQuery("cursor", "")
	pageSizeStr := c.DefaultQuery("pageSize", "20")

	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize <= 0 {
		pageSize = 20
	}

	// Limit maximum page size
	if pageSize > 100 {
		pageSize = 100
	}

	tenant, _, _, err := ctrl.TenantBO.GetTenant(c.Request.Context(), tenantCode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	cf := db.ColumnFamilyPrefix + strconv.Itoa(tenant.ColumnFamilyIndex)
	cfs := tenant.ID

	result, err := ctrl.VNamespaceBO.GetVNamespaces(c.Request.Context(), query, cursor, pageSize, cf, cfs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":    result.Entities,
		"cursor":  result.Cursor,
		"hasMore": result.Cursor != "",
	})
}
