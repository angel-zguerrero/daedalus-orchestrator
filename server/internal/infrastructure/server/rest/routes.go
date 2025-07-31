package rest_server

import (
	"deadalus-orch/server/internal/infrastructure/server/rest/auth"
	"deadalus-orch/server/internal/infrastructure/server/rest/exchange"
	"deadalus-orch/server/internal/infrastructure/server/rest/metrics"
	"deadalus-orch/server/internal/infrastructure/server/rest/tenant"
	"time"

	"github.com/gin-gonic/gin"
)

func (s *RestServer) setupRoutes(engine *gin.Engine) {

	adminController := auth.NewAdminController(s.Config)
	metricsController := metrics.NewMetricsController(s.Config)
	tenantController := tenant.NewTenantController(s.Config)
	exchangeController := exchange.NewExchangeController(s.Config)

	restAPIGroup := engine.Group("/rest-api")
	{

		restAPIGroup.POST("/login", rateLimitMiddleware(s.Config.MasterNode, "ip", 1*time.Minute, 4), adminController.LoginHandler)
		restAPIGroup.POST("/logout",
			authMiddleware(s.Config.MasterNode, s.Config.Logger, s.Config.JwtKey),
			rateLimitMiddleware(s.Config.MasterNode, "ip", 1*time.Minute, 4),
			adminController.LogoutHandler)

		tenantsGroup := restAPIGroup.Group("/tenants")
		tenantsGroup.Use(authMiddleware(s.Config.MasterNode, s.Config.Logger, s.Config.JwtKey))
		tenantsGroup.Use(rateLimitMiddleware(s.Config.MasterNode, "token", 1*time.Minute, 20))
		{
			tenantsGroup.GET("", tenantController.GetTenantsHandler)
			tenantsGroup.POST("", tenantController.CreateTenantHandler)
			tenantsGroup.POST("/bulk", tenantController.BulkCreateTenantHandler)
			tenantsGroup.GET("/:id", tenantController.GetTenantHandler)
			tenantsGroup.DELETE("/:id", tenantController.DeleteTenantHandler)
			{
				tenantsGroup.POST("/:id/assert/exchange", exchangeController.AssertExchangeHandler)
				tenantsGroup.POST("/:id/assert/exchanges", exchangeController.AssertExchangesHandler)
			}
		}
	}
	metricsAPIGroup := engine.Group("/metrics")
	metricsAPIGroup.Use(authMiddleware(s.Config.MasterNode, s.Config.Logger, s.Config.JwtKey))
	metricsAPIGroup.Use(rateLimitMiddleware(s.Config.MasterNode, "token", 1*time.Minute, 20))
	metricsAPIGroup.GET("/", metricsController.GetSystemMetricsHandler)
}
