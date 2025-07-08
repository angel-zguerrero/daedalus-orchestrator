package rest_api_admin

import (
	"time"

	"github.com/gin-gonic/gin"
)

func setupRoutes(engine *gin.Engine, api *RestAdminAPI) {
	adminAPIGroup := engine.Group("/admin-api")
	{

		adminAPIGroup.POST("/login", api.rateLimitMiddleware("ip", 1*time.Minute, 4), api.loginHandler)

		tenantsGroup := adminAPIGroup.Group("/tenants")
		tenantsGroup.Use(api.authMiddleware())
		tenantsGroup.Use(api.rateLimitMiddleware("token", 1*time.Minute, 20))
		{
			tenantsGroup.GET("", api.getTenantsHandler)
			tenantsGroup.POST("", api.createTenantHandler)
			tenantsGroup.GET("/:id", api.getTenantHandler)
			tenantsGroup.DELETE("/:id", api.deleteTenantHandler)
		}
	}
}
