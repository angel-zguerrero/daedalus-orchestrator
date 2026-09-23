package rest_server

import (
	"time"

	"deadalus-orch/server/internal/infrastructure/server/rest/activitytemplate"
	"deadalus-orch/server/internal/infrastructure/server/rest/auth"
	"deadalus-orch/server/internal/infrastructure/server/rest/binding"
	"deadalus-orch/server/internal/infrastructure/server/rest/cluster"
	"deadalus-orch/server/internal/infrastructure/server/rest/dashboard"
	"deadalus-orch/server/internal/infrastructure/server/rest/envconfig"
	"deadalus-orch/server/internal/infrastructure/server/rest/exchange"
	"deadalus-orch/server/internal/infrastructure/server/rest/jobworker"
	"deadalus-orch/server/internal/infrastructure/server/rest/metrics"
	"deadalus-orch/server/internal/infrastructure/server/rest/oauthapp"
	"deadalus-orch/server/internal/infrastructure/server/rest/queue"
	"deadalus-orch/server/internal/infrastructure/server/rest/scheduledjob"
	"deadalus-orch/server/internal/infrastructure/server/rest/tenant"
	"deadalus-orch/server/internal/infrastructure/server/rest/user"
	"deadalus-orch/server/internal/infrastructure/server/rest/vnamespace"
	"deadalus-orch/server/internal/infrastructure/server/rest/workflowdefinition"
	"deadalus-orch/server/internal/infrastructure/server/rest/workflowexecution"
	bo "deadalus-orch/server/internal/usecase/business-logic"

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
	activityTemplateController := activitytemplate.NewActivityTemplateController(s.Config)

	oauthController := oauthapp.NewOAuthController(s.Config)
	oauthAppController := oauthapp.NewOAuthAppController(s.Config)

	tenantBO := bo.NewTenantBO(s.Config)

	restAPIGroup := engine.Group("/rest-api")
	{
		// Public OAuth Token Endpoint
		restAPIGroup.POST("/oauth/token",
			rateLimitMiddleware(s.Config.MasterNode, "ip", 1*time.Minute, 10),
			oauthController.TokenHandler)

		restAPIGroup.GET("/auth/status", adminController.AuthStatusHandler)
		restAPIGroup.POST("/auth/setup", rateLimitMiddleware(s.Config.MasterNode, "ip", 1*time.Minute, 4), adminController.AuthSetupHandler)
		restAPIGroup.POST("/login", rateLimitMiddleware(s.Config.MasterNode, "ip", 1*time.Minute, 4), adminController.LoginHandler)
		restAPIGroup.POST("/logout",
			unifiedAuthMiddleware(s.Config.MasterNode, s.Config.Logger, s.Config.JwtKey),
			rateLimitMiddleware(s.Config.MasterNode, "ip", 1*time.Minute, 4),
			adminController.LogoutHandler)

		tenantsGroup := restAPIGroup.Group("/tenants")
		tenantsGroup.Use(unifiedAuthMiddleware(s.Config.MasterNode, s.Config.Logger, s.Config.JwtKey))
		tenantsGroup.Use(tenantContextMiddleware(tenantBO, s.Config, s.Config.Logger))
		tenantsGroup.Use(rateLimitMiddleware(s.Config.MasterNode, "token", 1*time.Minute, 300))
		{
			tenantsGroup.GET("", requireScope("tenants:list", "tenants:admin"), tenantController.GetTenantsHandler)
			tenantsGroup.POST("", requireScope("tenants:create", "tenants:admin"), tenantController.CreateTenantHandler)
			tenantsGroup.POST("/bulk", requireScope("tenants:create", "tenants:admin"), tenantController.BulkCreateTenantHandler)
			tenantsGroup.GET("/:code", requireScope("tenants:list", "tenants:admin"), tenantController.GetTenantHandler)
			tenantsGroup.GET("/:code/summary", requireScope("tenants:list", "tenants:admin"), tenantController.GetTenantSummaryHandler)
			tenantsGroup.GET("/:code/metrics/tsdb", requireScope("tenants:list", "tenants:admin"), tsdbMetricsController.GetTSDBMetricsHandler)
			tenantsGroup.DELETE("/:code", requireScope("tenants:delete", "tenants:admin"), tenantController.DeleteTenantHandler)

			// OAuth Service Account Management (Admin Session Only)
			tenantsGroup.GET("/:code/oauth-apps", sessionOnlyMiddleware(), oauthAppController.ListAppsHandler)
			tenantsGroup.POST("/:code/oauth-apps", sessionOnlyMiddleware(), oauthAppController.CreateAppHandler)
			tenantsGroup.GET("/:code/oauth-apps/:id", sessionOnlyMiddleware(), oauthAppController.GetAppHandler)
			tenantsGroup.DELETE("/:code/oauth-apps/:id", sessionOnlyMiddleware(), oauthAppController.DeleteAppHandler)
			tenantsGroup.POST("/:code/oauth-apps/:id/rotate-secret", sessionOnlyMiddleware(), oauthAppController.RotateSecretHandler)

			{
				// Exchanges
				tenantsGroup.POST("/:code/exchange", requireScope("exchanges:create", "exchanges:admin"), exchangeController.CreateExchangeHandler)
				tenantsGroup.POST("/:code/exchange/bulk", requireScope("exchanges:create", "exchanges:admin"), exchangeController.BulkCreateExchangeHandler)
				tenantsGroup.POST("/:code/exchange/publish-message", requireScope("exchanges:create", "exchanges:admin"), exchangeController.PublishMessageHandler)
				tenantsGroup.GET("/:code/exchange", requireScope("exchanges:list", "exchanges:admin"), exchangeController.GetExchangesHandler)
				tenantsGroup.GET("/:code/exchange/:exchangeCode/:vnamespace", requireScope("exchanges:list", "exchanges:admin"), exchangeController.GetExchangeHandler)
				tenantsGroup.DELETE("/:code/exchange/:exchangeCode/:vnamespace", requireScope("exchanges:delete", "exchanges:admin"), exchangeController.DeleteExchangeHandler)

				// Queues
				tenantsGroup.POST("/:code/queue", requireScope("queues:create", "queues:admin"), queueController.CreateQueueHandler)
				tenantsGroup.POST("/:code/queue/bulk", requireScope("queues:create", "queues:admin"), queueController.BulkCreateQueueHandler)
				tenantsGroup.POST("/:code/queue/:queueCode/:vnamespace/enqueue", requireScope("queues:create", "queues:admin"), queueController.EnqueueMessageHandler)
				tenantsGroup.GET("/:code/queue", requireScope("queues:list", "queues:admin"), queueController.GetQueuesHandler)
				tenantsGroup.GET("/:code/queue/:queueCode/:vnamespace", requireScope("queues:list", "queues:admin"), queueController.GetQueueHandler)
				tenantsGroup.GET("/:code/queue/:queueCode/:vnamespace/messages", requireScope("queues:list", "queues:admin"), queueController.GetQueueMessagesHandler)
				tenantsGroup.DELETE("/:code/queue/:queueCode/:vnamespace", requireScope("queues:delete", "queues:admin"), queueController.DeleteQueueHandler)

				// Bindings
				tenantsGroup.POST("/:code/binding", requireScope("bindings:create", "bindings:admin"), bindingController.CreateBindingHandler)
				tenantsGroup.GET("/:code/bindings", requireScope("bindings:list", "bindings:admin"), bindingController.GetBindingsHandler)
				tenantsGroup.GET("/:code/binding/:exchangeCode/:queueCode/:vnamespace", requireScope("bindings:list", "bindings:admin"), bindingController.GetBindingHandler)
				tenantsGroup.DELETE("/:code/binding/:bindingCode/:vnamespace", requireScope("bindings:delete", "bindings:admin"), bindingController.DeleteBindingHandler)

				tenantsGroup.GET("/:code/vnamespaces", requireScope("exchanges:list", "queues:list", "workflows:list", "tenants:list", "tenants:admin"), vnamespaceController.GetVNamespacesHandler)

				// Scheduled Jobs
				tenantsGroup.POST("/:code/scheduled-job/one-off", requireScope("workflows:create", "workflows:admin"), scheduledJobController.CreateOneOffScheduledJobHandler)
				tenantsGroup.POST("/:code/scheduled-job/recurring", requireScope("workflows:create", "workflows:admin"), scheduledJobController.CreateRecurringScheduledJobHandler)
				tenantsGroup.GET("/:code/scheduled-jobs", requireScope("workflows:list", "workflows:admin"), scheduledJobController.GetScheduledJobsHandler)
				tenantsGroup.GET("/:code/scheduled-job/:id", requireScope("workflows:list", "workflows:admin"), scheduledJobController.GetScheduledJobHandler)
				tenantsGroup.DELETE("/:code/scheduled-job/:id", requireScope("workflows:delete", "workflows:admin"), scheduledJobController.DeleteScheduledJobHandler)

				// Env Groups (admin only, no OAuth scope mapping)
				tenantsGroup.POST("/:code/env-groups", sessionOnlyMiddleware(), envConfigController.CreateTenantGroupHandler)
				tenantsGroup.GET("/:code/env-groups", sessionOnlyMiddleware(), envConfigController.ListTenantGroupsHandler)
				tenantsGroup.GET("/:code/env-groups/:groupId", sessionOnlyMiddleware(), envConfigController.GetTenantGroupHandler)
				tenantsGroup.PUT("/:code/env-groups/:groupId", sessionOnlyMiddleware(), envConfigController.UpdateTenantGroupHandler)
				tenantsGroup.DELETE("/:code/env-groups/:groupId", sessionOnlyMiddleware(), envConfigController.DeleteTenantGroupHandler)
				tenantsGroup.GET("/:code/env-groups/:groupId/vars", sessionOnlyMiddleware(), envConfigController.GetTenantVarsHandler)
				tenantsGroup.POST("/:code/env-groups/:groupId/vars", sessionOnlyMiddleware(), envConfigController.SaveTenantVarHandler)
				tenantsGroup.DELETE("/:code/env-groups/:groupId/vars/:varId", sessionOnlyMiddleware(), envConfigController.DeleteTenantVarHandler)
				tenantsGroup.PUT("/:code/env-groups/:groupId/vars/bulk", sessionOnlyMiddleware(), envConfigController.BulkSaveTenantVarsHandler)

				// Workflows
				tenantsGroup.POST("/:code/workflows", requireScope("workflows:create", "workflows:admin"), workflowDefinitionController.CreateTenantWorkflowHandler)
				tenantsGroup.GET("/:code/workflows", requireScope("workflows:list", "workflows:admin"), workflowDefinitionController.ListTenantWorkflowsHandler)
				tenantsGroup.GET("/:code/workflows/:id", requireScope("workflows:list", "workflows:admin"), workflowDefinitionController.GetTenantWorkflowHandler)
				tenantsGroup.PUT("/:code/workflows/:id", requireScope("workflows:edit", "workflows:admin"), workflowDefinitionController.UpdateTenantWorkflowHandler)
				tenantsGroup.DELETE("/:code/workflows/:id", requireScope("workflows:delete", "workflows:admin"), workflowDefinitionController.DeleteTenantWorkflowHandler)
				tenantsGroup.GET("/:code/workflows/:id/queues", requireScope("workflows:list", "workflows:admin"), workflowDefinitionController.GetTenantWorkflowQueuesHandler)
				tenantsGroup.GET("/:code/workflows/:id/versions", requireScope("workflows:list", "workflows:admin"), workflowDefinitionController.ListTenantWorkflowVersionsHandler)
				tenantsGroup.GET("/:code/workflows/:id/versions/:version", requireScope("workflows:list", "workflows:admin"), workflowDefinitionController.GetTenantWorkflowVersionHandler)

				tenantsGroup.POST("/:code/workflow-executions", requireScope("workflows:create", "workflows:admin"), workflowExecutionController.StartTenantExecutionHandler)
				tenantsGroup.GET("/:code/workflow-executions", requireScope("workflows:list", "workflows:admin"), workflowExecutionController.ListTenantExecutionsHandler)
				tenantsGroup.GET("/:code/workflow-executions/:id", requireScope("workflows:list", "workflows:admin"), workflowExecutionController.GetTenantExecutionHandler)

				// Activity Templates (Tenant)
				tenantsGroup.POST("/:code/activity-templates", requireScope("workflows:create", "workflows:admin"), activityTemplateController.CreateTenantActivityTemplateHandler)
				tenantsGroup.GET("/:code/activity-templates", requireScope("workflows:list", "workflows:admin"), activityTemplateController.ListTenantActivityTemplatesHandler)
				tenantsGroup.GET("/:code/activity-templates/for-designer", requireScope("workflows:list", "workflows:admin"), activityTemplateController.ListForDesignerHandler)
				tenantsGroup.GET("/:code/activity-templates/:id", requireScope("workflows:list", "workflows:admin"), activityTemplateController.GetTenantActivityTemplateHandler)
				tenantsGroup.PUT("/:code/activity-templates/:id", requireScope("workflows:edit", "workflows:admin"), activityTemplateController.UpdateTenantActivityTemplateHandler)
				tenantsGroup.DELETE("/:code/activity-templates/:id", requireScope("workflows:delete", "workflows:admin"), activityTemplateController.DeleteTenantActivityTemplateHandler)
			}
		}

		workflowsGroup := restAPIGroup.Group("/workflows")
		workflowsGroup.Use(unifiedAuthMiddleware(s.Config.MasterNode, s.Config.Logger, s.Config.JwtKey))
		workflowsGroup.Use(rateLimitMiddleware(s.Config.MasterNode, "token", 1*time.Minute, 300))
		{
			workflowsGroup.POST("", requireScope("workflows:create", "workflows:admin"), workflowDefinitionController.CreateGlobalWorkflowHandler)
			workflowsGroup.GET("", requireScope("workflows:list", "workflows:admin"), workflowDefinitionController.ListGlobalWorkflowsHandler)
			workflowsGroup.GET("/:id", requireScope("workflows:list", "workflows:admin"), workflowDefinitionController.GetGlobalWorkflowHandler)
			workflowsGroup.PUT("/:id", requireScope("workflows:edit", "workflows:admin"), workflowDefinitionController.UpdateGlobalWorkflowHandler)
			workflowsGroup.DELETE("/:id", requireScope("workflows:delete", "workflows:admin"), workflowDefinitionController.DeleteGlobalWorkflowHandler)
			workflowsGroup.GET("/:id/queues", requireScope("workflows:list", "workflows:admin"), workflowDefinitionController.GetGlobalWorkflowQueuesHandler)
			workflowsGroup.GET("/:id/versions", requireScope("workflows:list", "workflows:admin"), workflowDefinitionController.ListGlobalWorkflowVersionsHandler)
			workflowsGroup.GET("/:id/versions/:version", requireScope("workflows:list", "workflows:admin"), workflowDefinitionController.GetGlobalWorkflowVersionHandler)

			workflowsGroup.POST("/executions", requireScope("workflows:create", "workflows:admin"), workflowExecutionController.StartGlobalExecutionHandler)
			workflowsGroup.GET("/executions", requireScope("workflows:list", "workflows:admin"), workflowExecutionController.ListGlobalExecutionsHandler)
			workflowsGroup.GET("/executions/:id", requireScope("workflows:list", "workflows:admin"), workflowExecutionController.GetGlobalExecutionHandler)
			workflowsGroup.POST("/executions/jobs/:jobId/complete", requireScope("workflows:edit", "workflows:admin"), workflowExecutionController.CompleteGlobalJobHandler)
		}

		activityTemplatesGroup := restAPIGroup.Group("/activity-templates")
		activityTemplatesGroup.Use(unifiedAuthMiddleware(s.Config.MasterNode, s.Config.Logger, s.Config.JwtKey))
		activityTemplatesGroup.Use(rateLimitMiddleware(s.Config.MasterNode, "token", 1*time.Minute, 300))
		{
			activityTemplatesGroup.POST("", requireScope("workflows:create", "workflows:admin"), activityTemplateController.CreateGlobalActivityTemplateHandler)
			activityTemplatesGroup.GET("", requireScope("workflows:list", "workflows:admin"), activityTemplateController.ListGlobalActivityTemplatesHandler)
			activityTemplatesGroup.GET("/for-designer", requireScope("workflows:list", "workflows:admin"), activityTemplateController.ListForDesignerHandler)
			activityTemplatesGroup.GET("/:id", requireScope("workflows:list", "workflows:admin"), activityTemplateController.GetGlobalActivityTemplateHandler)
			activityTemplatesGroup.PUT("/:id", requireScope("workflows:edit", "workflows:admin"), activityTemplateController.UpdateGlobalActivityTemplateHandler)
			activityTemplatesGroup.DELETE("/:id", requireScope("workflows:delete", "workflows:admin"), activityTemplateController.DeleteGlobalActivityTemplateHandler)
		}

		envGroupsGroup := restAPIGroup.Group("/env-groups")
		envGroupsGroup.Use(unifiedAuthMiddleware(s.Config.MasterNode, s.Config.Logger, s.Config.JwtKey))
		envGroupsGroup.Use(sessionOnlyMiddleware())
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

		oauthAppsGroup := restAPIGroup.Group("/oauth-apps")
		oauthAppsGroup.Use(unifiedAuthMiddleware(s.Config.MasterNode, s.Config.Logger, s.Config.JwtKey))
		oauthAppsGroup.Use(sessionOnlyMiddleware())
		oauthAppsGroup.Use(rateLimitMiddleware(s.Config.MasterNode, "token", 1*time.Minute, 300))
		{
			oauthAppsGroup.GET("", oauthAppController.ListGlobalAppsHandler)
			oauthAppsGroup.POST("", oauthAppController.CreateGlobalAppHandler)
			oauthAppsGroup.GET("/:id", oauthAppController.GetAppHandler)
			oauthAppsGroup.DELETE("/:id", oauthAppController.DeleteAppHandler)
			oauthAppsGroup.POST("/:id/rotate-secret", oauthAppController.RotateSecretHandler)
		}

		usersGroup := restAPIGroup.Group("/users")
		usersGroup.Use(unifiedAuthMiddleware(s.Config.MasterNode, s.Config.Logger, s.Config.JwtKey))
		usersGroup.Use(sessionOnlyMiddleware())
		usersGroup.Use(rateLimitMiddleware(s.Config.MasterNode, "token", 1*time.Minute, 300))
		{
			usersGroup.GET("", userController.GetUsersHandler)
			usersGroup.POST("", userController.CreateUserHandler)
			usersGroup.PUT("/:id", userController.UpdateUserHandler)
			usersGroup.DELETE("/:id", userController.DeleteUserHandler)
		}

		jobWorkersGroup := restAPIGroup.Group("/job-workers")
		jobWorkersGroup.Use(unifiedAuthMiddleware(s.Config.MasterNode, s.Config.Logger, s.Config.JwtKey))
		jobWorkersGroup.Use(rateLimitMiddleware(s.Config.MasterNode, "token", 1*time.Minute, 300))
		{
			jobWorkersGroup.GET("", requireScope("workflows:list", "workflows:admin"), jobWorkerController.GetJobWorkersHandler)
			jobWorkersGroup.GET("/:id", requireScope("workflows:list", "workflows:admin"), jobWorkerController.GetJobWorkerHandler)
		}

		apiV1Group := restAPIGroup.Group("/v1")
		apiV1Group.Use(unifiedAuthMiddleware(s.Config.MasterNode, s.Config.Logger, s.Config.JwtKey))
		apiV1Group.Use(sessionOnlyMiddleware())
		apiV1Group.Use(rateLimitMiddleware(s.Config.MasterNode, "token", 1*time.Minute, 30))
		{
			clusterController.RegisterRoutes(apiV1Group)
			apiV1Group.GET("/cluster/metrics/tsdb", tsdbMetricsController.GetGlobalTSDBMetricsHandler)
		}

		dashboardGroup := restAPIGroup.Group("/dashboard")
		dashboardGroup.Use(unifiedAuthMiddleware(s.Config.MasterNode, s.Config.Logger, s.Config.JwtKey))
		dashboardGroup.Use(rateLimitMiddleware(s.Config.MasterNode, "token", 1*time.Minute, 300))
		{
			dashboardGroup.GET("/summary", requireScope("tenants:list", "tenants:admin"), dashboardController.GetDashboardSummaryHandler)
		}
	}

	metricsAPIGroup := engine.Group("/metrics")
	metricsAPIGroup.Use(unifiedAuthMiddleware(s.Config.MasterNode, s.Config.Logger, s.Config.JwtKey))
	metricsAPIGroup.Use(sessionOnlyMiddleware())
	metricsAPIGroup.Use(rateLimitMiddleware(s.Config.MasterNode, "token", 1*time.Minute, 300))
	metricsAPIGroup.GET("/", metricsController.GetSystemMetricsHandler)
}
