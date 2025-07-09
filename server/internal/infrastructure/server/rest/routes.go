package rest_server

import (
	rest_api_admin "deadalus-orch/server/internal/infrastructure/server/rest/admin"
	"time"

	"github.com/gin-gonic/gin"
)

func (s *RestServer) setupRoutes(engine *gin.Engine) {

	adminController := rest_api_admin.NewAdminController(s.Config)

	adminAPIGroup := engine.Group("/admin-api")
	{

		adminAPIGroup.POST("/login", rateLimitMiddleware(s.Config.MasterNode, "ip", 1*time.Minute, 4), adminController.LoginHandler)

		tenantsGroup := adminAPIGroup.Group("/tenants")
		tenantsGroup.Use(authMiddleware(s.Config.MasterNode, s.Config.Logger, s.Config.JwtKey))
		tenantsGroup.Use(rateLimitMiddleware(s.Config.MasterNode, "token", 1*time.Minute, 20))
		{
			tenantsGroup.GET("", adminController.GetTenantsHandler)
			tenantsGroup.POST("", adminController.CreateTenantHandler)
			tenantsGroup.GET("/:id", adminController.GetTenantHandler)
			tenantsGroup.DELETE("/:id", adminController.DeleteTenantHandler)
		}
	}
}
