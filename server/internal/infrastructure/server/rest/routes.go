package rest_server

import (
	"deadalus-orch/server/internal/infrastructure/server/rest/auth"
	"deadalus-orch/server/internal/infrastructure/server/rest/binding"
	"deadalus-orch/server/internal/infrastructure/server/rest/exchange"
	"deadalus-orch/server/internal/infrastructure/server/rest/metrics"
	"deadalus-orch/server/internal/infrastructure/server/rest/nodescheduler"
	"deadalus-orch/server/internal/infrastructure/server/rest/queue"
	"deadalus-orch/server/internal/infrastructure/server/rest/tenant"
	"deadalus-orch/server/internal/infrastructure/server/rest/vnamespace"
	"time"

	"github.com/gin-gonic/gin"
)

func (s *RestServer) setupRoutes(engine *gin.Engine) {

	adminController := auth.NewAdminController(s.Config)
	metricsController := metrics.NewMetricsController(s.Config)
	tenantController := tenant.NewTenantController(s.Config)
	exchangeController := exchange.NewExchangeController(s.Config)
	queueController := queue.NewQueueController(s.Config)
	bindingController := binding.NewBindingController(s.Config)
	vnamespaceController := vnamespace.NewVNamespaceController(s.Config)
	nodeSchedulerController := nodescheduler.NewNodeSchedulerController(s.Config)

	restAPIGroup := engine.Group("/rest-api")
	{

		restAPIGroup.POST("/login", rateLimitMiddleware(s.Config.MasterNode, "ip", 1*time.Minute, 4), adminController.LoginHandler)
		restAPIGroup.POST("/logout",
			authMiddleware(s.Config.MasterNode, s.Config.Logger, s.Config.JwtKey),
			rateLimitMiddleware(s.Config.MasterNode, "ip", 1*time.Minute, 4),
			adminController.LogoutHandler)

		tenantsGroup := restAPIGroup.Group("/tenants")
		tenantsGroup.Use(authMiddleware(s.Config.MasterNode, s.Config.Logger, s.Config.JwtKey))
		tenantsGroup.Use(rateLimitMiddleware(s.Config.MasterNode, "token", 1*time.Minute, 300))
		{
			tenantsGroup.GET("", tenantController.GetTenantsHandler)
			tenantsGroup.POST("", tenantController.CreateTenantHandler)
			tenantsGroup.POST("/bulk", tenantController.BulkCreateTenantHandler)
			tenantsGroup.GET("/:id", tenantController.GetTenantHandler)
			tenantsGroup.GET("/:id/summary", tenantController.GetTenantSummaryHandler)
			tenantsGroup.DELETE("/:id", tenantController.DeleteTenantHandler)
			{
				tenantsGroup.POST("/:id/exchange", exchangeController.CreateExchangeHandler)
				tenantsGroup.POST("/:id/exchange/bulk", exchangeController.BulkCreateExchangeHandler)
				tenantsGroup.GET("/:id/exchange", exchangeController.GetExchangesHandler)
				tenantsGroup.GET("/:id/exchange/:code/:vnamespace", exchangeController.GetExchangeHandler)
				tenantsGroup.DELETE("/:id/exchange/:code/:vnamespace", exchangeController.DeleteExchangeHandler)

				tenantsGroup.POST("/:id/queue", queueController.CreateQueueHandler)
				tenantsGroup.POST("/:id/queue/bulk", queueController.BulkCreateQueueHandler)
				tenantsGroup.GET("/:id/queue", queueController.GetQueuesHandler)
				tenantsGroup.GET("/:id/queue/:code/:vnamespace", queueController.GetQueueHandler)
				tenantsGroup.DELETE("/:id/queue/:code/:vnamespace", queueController.DeleteQueueHandler)

				tenantsGroup.POST("/:id/binding", bindingController.CreateBindingHandler)
				tenantsGroup.GET("/:id/bindings", bindingController.GetBindingsHandler)
				tenantsGroup.GET("/:id/binding/:exchangeCode/:queueCode/:vnamespace", bindingController.GetBindingHandler)
				tenantsGroup.DELETE("/:id/binding/:exchangeCode/:queueCode/:vnamespace", bindingController.DeleteBindingHandler)

				tenantsGroup.GET("/:id/vnamespaces", vnamespaceController.GetVNamespacesHandler)
			}
		}

		nodeSchedulersGroup := restAPIGroup.Group("/node-schedulers")
		nodeSchedulersGroup.Use(authMiddleware(s.Config.MasterNode, s.Config.Logger, s.Config.JwtKey))
		nodeSchedulersGroup.Use(rateLimitMiddleware(s.Config.MasterNode, "token", 1*time.Minute, 300))
		{
			nodeSchedulersGroup.GET("", nodeSchedulerController.GetNodeSchedulersHandler)
			nodeSchedulersGroup.GET("/:id", nodeSchedulerController.GetNodeSchedulerHandler)
		}

	}
	metricsAPIGroup := engine.Group("/metrics")
	metricsAPIGroup.Use(authMiddleware(s.Config.MasterNode, s.Config.Logger, s.Config.JwtKey))
	metricsAPIGroup.Use(rateLimitMiddleware(s.Config.MasterNode, "token", 1*time.Minute, 300))
	metricsAPIGroup.GET("/", metricsController.GetSystemMetricsHandler)
}
