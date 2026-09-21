package rest_server

import (
	"deadalus-orch/server/internal/infrastructure/server/rest/auth"
	"deadalus-orch/server/internal/infrastructure/server/rest/binding"
	"deadalus-orch/server/internal/infrastructure/server/rest/cluster"
	"deadalus-orch/server/internal/infrastructure/server/rest/dashboard"
	"deadalus-orch/server/internal/infrastructure/server/rest/envconfig"
	"deadalus-orch/server/internal/infrastructure/server/rest/exchange"
	"deadalus-orch/server/internal/infrastructure/server/rest/jobworker"
	"deadalus-orch/server/internal/infrastructure/server/rest/metrics"
	"deadalus-orch/server/internal/infrastructure/server/rest/queue"
	"deadalus-orch/server/internal/infrastructure/server/rest/scheduledjob"
	"deadalus-orch/server/internal/infrastructure/server/rest/tenant"
	"deadalus-orch/server/internal/infrastructure/server/rest/user"
	"deadalus-orch/server/internal/infrastructure/server/rest/vnamespace"
	"deadalus-orch/server/internal/infrastructure/server/rest/workflowdefinition"
	"deadalus-orch/server/internal/infrastructure/server/rest/workflowexecution"
	bo "deadalus-orch/server/internal/usecase/business-logic"
	"time"

	"github.com/gin-gonic/gin"
)

func (s *RestServer) setupRoutes(engine *gin.Engine) {

	adminController := auth.NewAdminController(s.Config)
	metricsController := metrics.NewMetricsController(s.Config)
	tsdbMetricsController := metrics.NewTSDBMetricsController(s.Config)
	tenantController := tenant.NewTenantController(s.Config)
	exchangeController := exchange.NewExchangeController(s.Config)
	queueController := queue.NewQueueController(s.Config)
	bindingController := binding.NewBindingController(s.Config)
	vnamespaceController := vnamespace.NewVNamespaceController(s.Config)
	jobWorkerController := jobworker.NewJobWorkerController(s.Config)
	clusterController := cluster.NewClusterController(s.Config)
	dashboardController := dashboard.NewDashboardController(s.Config)
	userController := user.NewUserController(s.Config)
	scheduledJobController := scheduledjob.NewScheduledJobController(s.Config)
	envConfigController := envconfig.NewEnvConfigController(s.Config)
	workflowDefinitionController := workflowdefinition.NewWorkflowDefinitionController(s.Config)
	workflowExecutionController := workflowexecution.NewWorkflowExecutionController(s.Config)

	// Crear el TenantBO para el middleware
	tenantBO := bo.NewTenantBO(s.Config)

	restAPIGroup := engine.Group("/rest-api")
	{

		restAPIGroup.GET("/auth/status", adminController.AuthStatusHandler)
		restAPIGroup.POST("/auth/setup", rateLimitMiddleware(s.Config.MasterNode, "ip", 1*time.Minute, 4), adminController.AuthSetupHandler)
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
			tenantsGroup.GET("/:code/metrics/tsdb", tsdbMetricsController.GetTSDBMetricsHandler)
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
				tenantsGroup.GET("/:code/queue/:queueCode/:vnamespace/messages", queueController.GetQueueMessagesHandler)
				tenantsGroup.DELETE("/:code/queue/:queueCode/:vnamespace", queueController.DeleteQueueHandler)

				tenantsGroup.POST("/:code/binding", bindingController.CreateBindingHandler)
				tenantsGroup.GET("/:code/bindings", bindingController.GetBindingsHandler)
				tenantsGroup.GET("/:code/binding/:exchangeCode/:queueCode/:vnamespace", bindingController.GetBindingHandler)
				tenantsGroup.DELETE("/:code/binding/:bindingCode/:vnamespace", bindingController.DeleteBindingHandler)

				tenantsGroup.GET("/:code/vnamespaces", vnamespaceController.GetVNamespacesHandler)

				tenantsGroup.POST("/:code/scheduled-job/one-off", scheduledJobController.CreateOneOffScheduledJobHandler)
				tenantsGroup.POST("/:code/scheduled-job/recurring", scheduledJobController.CreateRecurringScheduledJobHandler)
				tenantsGroup.GET("/:code/scheduled-jobs", scheduledJobController.GetScheduledJobsHandler)
				tenantsGroup.GET("/:code/scheduled-job/:id", scheduledJobController.GetScheduledJobHandler)
				tenantsGroup.DELETE("/:code/scheduled-job/:id", scheduledJobController.DeleteScheduledJobHandler)

				tenantsGroup.POST("/:code/env-groups", envConfigController.CreateTenantGroupHandler)
				tenantsGroup.GET("/:code/env-groups", envConfigController.ListTenantGroupsHandler)
				tenantsGroup.GET("/:code/env-groups/:groupId", envConfigController.GetTenantGroupHandler)
				tenantsGroup.PUT("/:code/env-groups/:groupId", envConfigController.UpdateTenantGroupHandler)
				tenantsGroup.DELETE("/:code/env-groups/:groupId", envConfigController.DeleteTenantGroupHandler)
				tenantsGroup.GET("/:code/env-groups/:groupId/vars", envConfigController.GetTenantVarsHandler)
				tenantsGroup.POST("/:code/env-groups/:groupId/vars", envConfigController.SaveTenantVarHandler)
				tenantsGroup.DELETE("/:code/env-groups/:groupId/vars/:varId", envConfigController.DeleteTenantVarHandler)
				tenantsGroup.PUT("/:code/env-groups/:groupId/vars/bulk", envConfigController.BulkSaveTenantVarsHandler)

				tenantsGroup.POST("/:code/workflows", workflowDefinitionController.CreateTenantWorkflowHandler)
				tenantsGroup.GET("/:code/workflows", workflowDefinitionController.ListTenantWorkflowsHandler)
				tenantsGroup.GET("/:code/workflows/:id", workflowDefinitionController.GetTenantWorkflowHandler)
				tenantsGroup.PUT("/:code/workflows/:id", workflowDefinitionController.UpdateTenantWorkflowHandler)
				tenantsGroup.DELETE("/:code/workflows/:id", workflowDefinitionController.DeleteTenantWorkflowHandler)
				tenantsGroup.GET("/:code/workflows/:id/queues", workflowDefinitionController.GetTenantWorkflowQueuesHandler)
				tenantsGroup.GET("/:code/workflows/:id/versions", workflowDefinitionController.ListTenantWorkflowVersionsHandler)
				tenantsGroup.GET("/:code/workflows/:id/versions/:version", workflowDefinitionController.GetTenantWorkflowVersionHandler)

				tenantsGroup.POST("/:code/workflow-executions", workflowExecutionController.StartTenantExecutionHandler)
				tenantsGroup.GET("/:code/workflow-executions", workflowExecutionController.ListTenantExecutionsHandler)
				tenantsGroup.GET("/:code/workflow-executions/:id", workflowExecutionController.GetTenantExecutionHandler)
			}
		}

		workflowsGroup := restAPIGroup.Group("/workflows")
		workflowsGroup.Use(authMiddleware(s.Config.MasterNode, s.Config.Logger, s.Config.JwtKey))
		workflowsGroup.Use(rateLimitMiddleware(s.Config.MasterNode, "token", 1*time.Minute, 300))
		{
			workflowsGroup.POST("", workflowDefinitionController.CreateGlobalWorkflowHandler)
			workflowsGroup.GET("", workflowDefinitionController.ListGlobalWorkflowsHandler)
			workflowsGroup.GET("/:id", workflowDefinitionController.GetGlobalWorkflowHandler)
			workflowsGroup.PUT("/:id", workflowDefinitionController.UpdateGlobalWorkflowHandler)
			workflowsGroup.DELETE("/:id", workflowDefinitionController.DeleteGlobalWorkflowHandler)
			workflowsGroup.GET("/:id/queues", workflowDefinitionController.GetGlobalWorkflowQueuesHandler)
			workflowsGroup.GET("/:id/versions", workflowDefinitionController.ListGlobalWorkflowVersionsHandler)
			workflowsGroup.GET("/:id/versions/:version", workflowDefinitionController.GetGlobalWorkflowVersionHandler)

			workflowsGroup.POST("/executions", workflowExecutionController.StartGlobalExecutionHandler)
			workflowsGroup.GET("/executions", workflowExecutionController.ListGlobalExecutionsHandler)
			workflowsGroup.GET("/executions/:id", workflowExecutionController.GetGlobalExecutionHandler)
			workflowsGroup.POST("/executions/jobs/:jobId/complete", workflowExecutionController.CompleteGlobalJobHandler)
		}

		envGroupsGroup := restAPIGroup.Group("/env-groups")
		envGroupsGroup.Use(authMiddleware(s.Config.MasterNode, s.Config.Logger, s.Config.JwtKey))
		envGroupsGroup.Use(rateLimitMiddleware(s.Config.MasterNode, "token", 1*time.Minute, 300))
		{
			envGroupsGroup.POST("", envConfigController.CreateGlobalGroupHandler)
			envGroupsGroup.GET("", envConfigController.ListGlobalGroupsHandler)
			envGroupsGroup.GET("/:groupId", envConfigController.GetGlobalGroupHandler)
			envGroupsGroup.PUT("/:groupId", envConfigController.UpdateGlobalGroupHandler)
			envGroupsGroup.DELETE("/:groupId", envConfigController.DeleteGlobalGroupHandler)
			envGroupsGroup.GET("/:groupId/vars", envConfigController.GetGlobalVarsHandler)
			envGroupsGroup.POST("/:groupId/vars", envConfigController.SaveGlobalVarHandler)
			envGroupsGroup.DELETE("/:groupId/vars/:varId", envConfigController.DeleteGlobalVarHandler)
			envGroupsGroup.PUT("/:groupId/vars/bulk", envConfigController.BulkSaveGlobalVarsHandler)
		}

		usersGroup := restAPIGroup.Group("/users")
		usersGroup.Use(authMiddleware(s.Config.MasterNode, s.Config.Logger, s.Config.JwtKey))
		usersGroup.Use(rateLimitMiddleware(s.Config.MasterNode, "token", 1*time.Minute, 300))
		{
			usersGroup.GET("", userController.GetUsersHandler)
			usersGroup.POST("", userController.CreateUserHandler)
			usersGroup.PUT("/:id", userController.UpdateUserHandler)
			usersGroup.DELETE("/:id", userController.DeleteUserHandler)
		}

		jobWorkersGroup := restAPIGroup.Group("/job-workers")
		jobWorkersGroup.Use(authMiddleware(s.Config.MasterNode, s.Config.Logger, s.Config.JwtKey))
		jobWorkersGroup.Use(rateLimitMiddleware(s.Config.MasterNode, "token", 1*time.Minute, 300))
		{
			jobWorkersGroup.GET("", jobWorkerController.GetJobWorkersHandler)
			jobWorkersGroup.GET("/:id", jobWorkerController.GetJobWorkerHandler)
		}

		// Cluster management endpoints
		apiV1Group := restAPIGroup.Group("/v1")
		apiV1Group.Use(authMiddleware(s.Config.MasterNode, s.Config.Logger, s.Config.JwtKey))
		apiV1Group.Use(rateLimitMiddleware(s.Config.MasterNode, "token", 1*time.Minute, 30))
		{
			clusterController.RegisterRoutes(apiV1Group)
			apiV1Group.GET("/cluster/metrics/tsdb", tsdbMetricsController.GetGlobalTSDBMetricsHandler)
		}

		// Dashboard endpoints
		dashboardGroup := restAPIGroup.Group("/dashboard")
		dashboardGroup.Use(authMiddleware(s.Config.MasterNode, s.Config.Logger, s.Config.JwtKey))
		dashboardGroup.Use(rateLimitMiddleware(s.Config.MasterNode, "token", 1*time.Minute, 300))
		{
			dashboardGroup.GET("/summary", dashboardController.GetDashboardSummaryHandler)
		}

	}

	metricsAPIGroup := engine.Group("/metrics")
	metricsAPIGroup.Use(authMiddleware(s.Config.MasterNode, s.Config.Logger, s.Config.JwtKey))
	metricsAPIGroup.Use(rateLimitMiddleware(s.Config.MasterNode, "token", 1*time.Minute, 300))
	metricsAPIGroup.GET("/", metricsController.GetSystemMetricsHandler)
}
