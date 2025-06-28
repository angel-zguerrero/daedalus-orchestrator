package rest_api_admin

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// getTenantsHandler handles GET /admin-api/tenants
func (api *RestAdminAPI) getTenantsHandler(c *gin.Context) {

	c.JSON(http.StatusOK, gin.H{"message": "Listing all tenants (dummy)", "tenants": []string{"tenantA", "tenantttB", "tenantC"}})
}

// createTenantHandler handles POST /admin-api/tenants
func (api *RestAdminAPI) createTenantHandler(c *gin.Context) {
	c.JSON(http.StatusCreated, gin.H{"message": "Tenant created (dummy)", "id": "newTenantID123"})
}

// getTenantHandler handles GET /admin-api/tenants/:id
func (api *RestAdminAPI) getTenantHandler(c *gin.Context) {
	tenantID := c.Param("id")

	c.JSON(http.StatusOK, gin.H{"message": "Details for tenant " + tenantID + " (dummy)", "id": tenantID, "name": "Dummy Tenant " + tenantID})
}

// updateTenantHandler handles PUT /admin-api/tenants/:id
func (api *RestAdminAPI) updateTenantHandler(c *gin.Context) {
	tenantID := c.Param("id")

	c.JSON(http.StatusOK, gin.H{"message": "Tenant " + tenantID + " updated (dummy)", "id": tenantID})
}

// deleteTenantHandler handles DELETE /admin-api/tenants/:id
func (api *RestAdminAPI) deleteTenantHandler(c *gin.Context) {
	tenantID := c.Param("id")

	c.JSON(http.StatusOK, gin.H{"message": "Tenant " + tenantID + " deleted (dummy)"})
}
