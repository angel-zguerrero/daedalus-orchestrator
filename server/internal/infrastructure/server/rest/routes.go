package rest_server

import (
	"deadalus-orch/server/internal/infrastructure/server/rest/auth"
	"deadalus-orch/server/internal/infrastructure/server/rest/binding"
	"deadalus-orch/server/internal/infrastructure/server/rest/cluster"
	"deadalus-orch/server/internal/infrastructure/server/rest/exchange"
	"deadalus-orch/server/internal/infrastructure/server/rest/metrics"
	"deadalus-orch/server/internal/infrastructure/server/rest/nodescheduler"
	"deadalus-orch/server/internal/infrastructure/server/rest/queue"
	"deadalus-orch/server/internal/infrastructure/server/rest/tenant"
	"deadalus-orch/server/internal/infrastructure/server/rest/vnamespace"
	bo "deadalus-orch/server/internal/usecase/business-logic"
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
	clusterController := cluster.NewClusterController(s.Config)

	// Crear el TenantBO para el middleware
	tenantBO := bo.NewTenantBO(s.Config)

	restAPIGroup := engine.Group("/rest-api")
	{

		restAPIGroup.POST("/login", rateLimitMiddleware(s.Config.MasterNode, "ip", 1*time.Minute, 4), adminController.LoginHandler)
		restAPIGroup.POST("/logout",
			authMiddleware(s.Config.MasterNode, s.Config.Logger, s.Config.JwtKey),
			rateLimitMiddleware(s.Config.MasterNode, "ip", 1*time.Minute, 4),
			adminController.LogoutHandler)

		tenantsGroup := restAPIGroup.Group("/tenants")
		tenantsGroup.Use(authMiddleware(s.Config.MasterNode, s.Config.Logger, s.Config.JwtKey))
		tenantsGroup.Use(tenantContextMiddleware(tenantBO, s.Config, s.Config.Logger))
		tenantsGroup.Use(rateLimitMiddleware(s.Config.MasterNode, "token", 1*time.Minute, 300))
		{
			tenantsGroup.GET("", tenantController.GetTenantsHandler)
			tenantsGroup.POST("", tenantController.CreateTenantHandler)
			tenantsGroup.POST("/bulk", tenantController.BulkCreateTenantHandler)
			tenantsGroup.GET("/:code", tenantController.GetTenantHandler)
			tenantsGroup.GET("/:code/summary", tenantController.GetTenantSummaryHandler)
			tenantsGroup.DELETE("/:code", tenantController.DeleteTenantHandler)
			{
				tenantsGroup.POST("/:code/exchange", exchangeController.CreateExchangeHandler)
				tenantsGroup.POST("/:code/exchange/bulk", exchangeController.BulkCreateExchangeHandler)
				tenantsGroup.POST("/:code/exchange/publish-message", exchangeController.PublishMessageHandler)
				tenantsGroup.GET("/:code/exchange", exchangeController.GetExchangesHandler)
				tenantsGroup.GET("/:code/exchange/:exchangeCode/:vnamespace", exchangeController.GetExchangeHandler)
				tenantsGroup.DELETE("/:code/exchange/:exchangeCode/:vnamespace", exchangeController.DeleteExchangeHandler)

				tenantsGroup.POST("/:code/queue", queueController.CreateQueueHandler)
				tenantsGroup.POST("/:code/queue/bulk", queueController.BulkCreateQueueHandler)
				tenantsGroup.POST("/:code/queue/:queueCode/:vnamespace/enqueue", queueController.EnqueueMessageHandler)
				tenantsGroup.GET("/:code/queue", queueController.GetQueuesHandler)
				tenantsGroup.GET("/:code/queue/:queueCode/:vnamespace", queueController.GetQueueHandler)
				tenantsGroup.DELETE("/:code/queue/:queueCode/:vnamespace", queueController.DeleteQueueHandler)

				tenantsGroup.POST("/:code/binding", bindingController.CreateBindingHandler)
				tenantsGroup.GET("/:code/bindings", bindingController.GetBindingsHandler)
				tenantsGroup.GET("/:code/binding/:exchangeCode/:queueCode/:vnamespace", bindingController.GetBindingHandler)
				tenantsGroup.DELETE("/:code/binding/:bindingCode/:vnamespace", bindingController.DeleteBindingHandler)

				tenantsGroup.GET("/:code/vnamespaces", vnamespaceController.GetVNamespacesHandler)
			}
		}

		nodeSchedulersGroup := restAPIGroup.Group("/node-schedulers")
		nodeSchedulersGroup.Use(authMiddleware(s.Config.MasterNode, s.Config.Logger, s.Config.JwtKey))
		nodeSchedulersGroup.Use(rateLimitMiddleware(s.Config.MasterNode, "token", 1*time.Minute, 300))
		{
			nodeSchedulersGroup.GET("", nodeSchedulerController.GetNodeSchedulersHandler)
			nodeSchedulersGroup.GET("/:id", nodeSchedulerController.GetNodeSchedulerHandler)
		}

		// Cluster management endpoints
		apiV1Group := restAPIGroup.Group("/v1")
		apiV1Group.Use(authMiddleware(s.Config.MasterNode, s.Config.Logger, s.Config.JwtKey))
		apiV1Group.Use(rateLimitMiddleware(s.Config.MasterNode, "token", 1*time.Minute, 30))
		{
			clusterController.RegisterRoutes(apiV1Group)
		}

	}
	metricsAPIGroup := engine.Group("/metrics")
	metricsAPIGroup.Use(authMiddleware(s.Config.MasterNode, s.Config.Logger, s.Config.JwtKey))
	metricsAPIGroup.Use(rateLimitMiddleware(s.Config.MasterNode, "token", 1*time.Minute, 300))
	metricsAPIGroup.GET("/", metricsController.GetSystemMetricsHandler)
}
