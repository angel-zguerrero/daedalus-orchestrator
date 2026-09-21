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
	"deadalus-orch/server/internal/infrastructure/server/rest/oauthapp"
	"deadalus-orch/server/internal/infrastructure/server/rest/queue"
	"deadalus-orch/server/internal/infrastructure/server/rest/scheduledjob"
	"deadalus-orch/server/internal/infrastructure/server/rest/tenant"
	"deadalus-orch/server/internal/infrastructure/server/rest/user"
	"deadalus-orch/server/internal/infrastructure/server/rest/vnamespace"
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

	// OAuth controllers
	oauthTokenController := oauthapp.NewOAuthController(s.Config)
	oauthAppController := oauthapp.NewOAuthAppController(s.Config)

	// ── OAuth token endpoint — PUBLIC, rate-limited, NO auth middleware ────
	engine.POST("/rest-api/oauth/token",
		rateLimitMiddleware(s.Config.MasterNode, "ip", 1*time.Minute, 10),
		oauthTokenController.TokenHandler)

	// Crear el TenantBO para el middleware
	tenantBO := bo.NewTenantBO(s.Config)

	restAPIGroup := engine.Group("/rest-api")
	{

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
			tenantsGroup.GET("", tenantController.GetTenantsHandler)
			tenantsGroup.POST("", tenantController.CreateTenantHandler)
			tenantsGroup.POST("/bulk", tenantController.BulkCreateTenantHandler)
			tenantsGroup.GET("/:code", tenantController.GetTenantHandler)
			tenantsGroup.GET("/:code/summary", tenantController.GetTenantSummaryHandler)
			tenantsGroup.GET("/:code/metrics/tsdb", tsdbMetricsController.GetTSDBMetricsHandler)
			tenantsGroup.DELETE("/:code", tenantController.DeleteTenantHandler)
			{
				// ── Exchanges ────────────────────────────────────────────────────────
				tenantsGroup.POST("/:code/exchange", requireScope("exchanges:create", "exchanges:admin"), exchangeController.CreateExchangeHandler)
				tenantsGroup.POST("/:code/exchange/bulk", requireScope("exchanges:create", "exchanges:admin"), exchangeController.BulkCreateExchangeHandler)
				tenantsGroup.POST("/:code/exchange/publish-message", requireScope("exchanges:create", "exchanges:admin"), exchangeController.PublishMessageHandler)
				tenantsGroup.GET("/:code/exchange", requireScope("exchanges:list", "exchanges:admin"), exchangeController.GetExchangesHandler)
				tenantsGroup.GET("/:code/exchange/:exchangeCode/:vnamespace", requireScope("exchanges:list", "exchanges:admin"), exchangeController.GetExchangeHandler)
				tenantsGroup.DELETE("/:code/exchange/:exchangeCode/:vnamespace", requireScope("exchanges:delete", "exchanges:admin"), exchangeController.DeleteExchangeHandler)

				// ── Queues ───────────────────────────────────────────────────────────
				tenantsGroup.POST("/:code/queue", requireScope("queues:create", "queues:admin"), queueController.CreateQueueHandler)
				tenantsGroup.POST("/:code/queue/bulk", requireScope("queues:create", "queues:admin"), queueController.BulkCreateQueueHandler)
				tenantsGroup.POST("/:code/queue/:queueCode/:vnamespace/enqueue", requireScope("queues:create", "queues:admin"), queueController.EnqueueMessageHandler)
				tenantsGroup.GET("/:code/queue", requireScope("queues:list", "queues:admin"), queueController.GetQueuesHandler)
				tenantsGroup.GET("/:code/queue/:queueCode/:vnamespace", requireScope("queues:list", "queues:admin"), queueController.GetQueueHandler)
				tenantsGroup.GET("/:code/queue/:queueCode/:vnamespace/messages", requireScope("queues:list", "queues:admin"), queueController.GetQueueMessagesHandler)
				tenantsGroup.DELETE("/:code/queue/:queueCode/:vnamespace", requireScope("queues:delete", "queues:admin"), queueController.DeleteQueueHandler)

				// ── Bindings ─────────────────────────────────────────────────────────
				tenantsGroup.POST("/:code/binding", requireScope("bindings:create", "bindings:admin"), bindingController.CreateBindingHandler)
				tenantsGroup.GET("/:code/bindings", requireScope("bindings:list", "bindings:admin"), bindingController.GetBindingsHandler)
				tenantsGroup.GET("/:code/binding/:exchangeCode/:queueCode/:vnamespace", requireScope("bindings:list", "bindings:admin"), bindingController.GetBindingHandler)
				tenantsGroup.DELETE("/:code/binding/:bindingCode/:vnamespace", requireScope("bindings:delete", "bindings:admin"), bindingController.DeleteBindingHandler)

				tenantsGroup.GET("/:code/vnamespaces", vnamespaceController.GetVNamespacesHandler)

				// ── Scheduled Jobs ───────────────────────────────────────────────────
				tenantsGroup.POST("/:code/scheduled-job/one-off", requireScope("workflows:create", "workflows:admin"), scheduledJobController.CreateOneOffScheduledJobHandler)
				tenantsGroup.POST("/:code/scheduled-job/recurring", requireScope("workflows:create", "workflows:admin"), scheduledJobController.CreateRecurringScheduledJobHandler)
				tenantsGroup.GET("/:code/scheduled-jobs", requireScope("workflows:list", "workflows:admin"), scheduledJobController.GetScheduledJobsHandler)
				tenantsGroup.GET("/:code/scheduled-job/:id", requireScope("workflows:list", "workflows:admin"), scheduledJobController.GetScheduledJobHandler)
				tenantsGroup.DELETE("/:code/scheduled-job/:id", requireScope("workflows:delete", "workflows:admin"), scheduledJobController.DeleteScheduledJobHandler)

				// ── Env Groups (internal config — no OAuth scope, session only via sessionOnlyMiddleware) ──
				tenantsGroup.POST("/:code/env-groups", envConfigController.CreateTenantGroupHandler)
				tenantsGroup.GET("/:code/env-groups", envConfigController.ListTenantGroupsHandler)
				tenantsGroup.GET("/:code/env-groups/:groupId", envConfigController.GetTenantGroupHandler)
				tenantsGroup.PUT("/:code/env-groups/:groupId", envConfigController.UpdateTenantGroupHandler)
				tenantsGroup.DELETE("/:code/env-groups/:groupId", envConfigController.DeleteTenantGroupHandler)
				tenantsGroup.GET("/:code/env-groups/:groupId/vars", envConfigController.GetTenantVarsHandler)
				tenantsGroup.POST("/:code/env-groups/:groupId/vars", envConfigController.SaveTenantVarHandler)
				tenantsGroup.DELETE("/:code/env-groups/:groupId/vars/:varId", envConfigController.DeleteTenantVarHandler)
				tenantsGroup.PUT("/:code/env-groups/:groupId/vars/bulk", envConfigController.BulkSaveTenantVarsHandler)

				// ── OAuth App Management (session-only — OAuth tokens cannot manage other apps) ──
				tenantsGroup.GET("/:code/oauth-apps", sessionOnlyMiddleware(), oauthAppController.ListAppsHandler)
				tenantsGroup.POST("/:code/oauth-apps", sessionOnlyMiddleware(), oauthAppController.CreateAppHandler)
				tenantsGroup.GET("/:code/oauth-apps/:id", sessionOnlyMiddleware(), oauthAppController.GetAppHandler)
				tenantsGroup.DELETE("/:code/oauth-apps/:id", sessionOnlyMiddleware(), oauthAppController.DeleteAppHandler)
				tenantsGroup.POST("/:code/oauth-apps/:id/rotate-secret", sessionOnlyMiddleware(), oauthAppController.RotateSecretHandler)
			}
		}

		envGroupsGroup := restAPIGroup.Group("/env-groups")
		envGroupsGroup.Use(unifiedAuthMiddleware(s.Config.MasterNode, s.Config.Logger, s.Config.JwtKey))
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
		usersGroup.Use(unifiedAuthMiddleware(s.Config.MasterNode, s.Config.Logger, s.Config.JwtKey))
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
			jobWorkersGroup.GET("", jobWorkerController.GetJobWorkersHandler)
			jobWorkersGroup.GET("/:id", jobWorkerController.GetJobWorkerHandler)
		}

		// Cluster management endpoints
		apiV1Group := restAPIGroup.Group("/v1")
		apiV1Group.Use(unifiedAuthMiddleware(s.Config.MasterNode, s.Config.Logger, s.Config.JwtKey))
		apiV1Group.Use(rateLimitMiddleware(s.Config.MasterNode, "token", 1*time.Minute, 30))
		{
			clusterController.RegisterRoutes(apiV1Group)
			apiV1Group.GET("/cluster/metrics/tsdb", tsdbMetricsController.GetGlobalTSDBMetricsHandler)
		}

		// Dashboard endpoints
		dashboardGroup := restAPIGroup.Group("/dashboard")
		dashboardGroup.Use(unifiedAuthMiddleware(s.Config.MasterNode, s.Config.Logger, s.Config.JwtKey))
		dashboardGroup.Use(rateLimitMiddleware(s.Config.MasterNode, "token", 1*time.Minute, 300))
		{
			dashboardGroup.GET("/summary", dashboardController.GetDashboardSummaryHandler)
		}

	}

	metricsAPIGroup := engine.Group("/metrics")
	metricsAPIGroup.Use(unifiedAuthMiddleware(s.Config.MasterNode, s.Config.Logger, s.Config.JwtKey))
	metricsAPIGroup.Use(rateLimitMiddleware(s.Config.MasterNode, "token", 1*time.Minute, 300))
	metricsAPIGroup.GET("/", metricsController.GetSystemMetricsHandler)
}
