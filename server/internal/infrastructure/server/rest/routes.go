package rest_server

import (
	rest_api_admin "deadalus-orch/server/internal/infrastructure/server/rest/admin"
	"deadalus-orch/server/internal/infrastructure/server/rest/metrics"
	"deadalus-orch/server/internal/infrastructure/server/rest/tenant"
	"time"

	"github.com/gin-gonic/gin"
)

func (s *RestServer) setupRoutes(engine *gin.Engine) {

	adminController := rest_api_admin.NewAdminController(s.Config)
	metricsController := metrics.NewMetricsController(s.Config)
	tenantController := tenant.NewTenantController(s.Config)

	adminAPIGroup := engine.Group("/admin-api")
	{

		adminAPIGroup.POST("/login", rateLimitMiddleware(s.Config.MasterNode, "ip", 1*time.Minute, 4), adminController.LoginHandler)

		tenantsGroup := adminAPIGroup.Group("/tenants")
		tenantsGroup.Use(authMiddleware(s.Config.MasterNode, s.Config.Logger, s.Config.JwtKey))
		tenantsGroup.Use(rateLimitMiddleware(s.Config.MasterNode, "token", 1*time.Minute, 20))
		{
			tenantsGroup.GET("", tenantController.GetTenantsHandler)
			tenantsGroup.POST("", tenantController.CreateTenantHandler)
			tenantsGroup.GET("/:id", tenantController.GetTenantHandler)
			tenantsGroup.DELETE("/:id", tenantController.DeleteTenantHandler)
		}
	}
	metricsAPIGroup := engine.Group("/metrics")
	metricsAPIGroup.Use(authMiddleware(s.Config.MasterNode, s.Config.Logger, s.Config.JwtKey))
	metricsAPIGroup.Use(rateLimitMiddleware(s.Config.MasterNode, "token", 1*time.Minute, 20))
	metricsAPIGroup.GET("/", metricsController.GetSystemMetricsHandler)
}
