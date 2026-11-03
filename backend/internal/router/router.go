package router

import (
	"dockerun/internal/builtin"
	"dockerun/internal/config"
	"dockerun/internal/handler"
	"dockerun/internal/middleware"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

func Setup() *gin.Engine {
	handler.CleanupTemplateImports()
	gin.SetMode(config.Load().Mode)
	r := gin.Default()
	r.SetTrustedProxies(nil)

	r.Use(func(c *gin.Context) {
		p := c.Request.URL.Path
		numeric := false
		for _, prefix := range []string{"/webhooks/", "/agent/tasks/", "/api/applications/", "/api/releases/", "/api/deployments/", "/api/bundles/", "/api/users/", "/api/registries/", "/api/targets/", "/api/deploy-scripts/", "/api/webhook-configs/", "/api/system/backups/"} {
			if strings.HasPrefix(p, prefix) {
				numeric = true
				break
			}
		}
		if id := c.Param("id"); numeric && id != "" {
			v, err := strconv.ParseUint(id, 10, 32)
			if err != nil || v == 0 {
				c.AbortWithStatusJSON(400, gin.H{"error": "无效的资源 ID"})
				return
			}
		}
		c.Next()
	})
	r.Any("/v2/*path", gin.WrapH(http.HandlerFunc(builtin.Serve)))
	r.GET("/registry/token", gin.WrapH(http.HandlerFunc(builtin.Token)))
	r.POST("/registry/token", gin.WrapH(http.HandlerFunc(builtin.Token)))

	// CORS中间件
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// 公开接口
	public := r.Group("/api")
	{
		// 控制台 WebSocket 仅接受由已认证管理员签发的一次性票据。
		public.GET("/containers/:id/console/ws", handler.ContainerConsoleSocket)
		// 认证相关
		public.POST("/auth/login", handler.Login)
		public.GET("/auth/check-setup", handler.CheckInitialSetup)
		public.POST("/auth/setup", handler.SetInitialPassword)
	}

	agents := r.Group("/agent")
	agents.Use(handler.AgentAuth())
	agents.GET("/poll", handler.AgentPoll)
	agents.POST("/tasks/:id/heartbeat", handler.AgentHeartbeat)
	agents.POST("/tasks/:id/logs", handler.AgentLog)
	agents.POST("/tasks/:id/complete", handler.AgentComplete)
	agents.GET("/tasks/:id/bundle", handler.AgentBundle)
	r.GET("/health", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })
	// Webhook 公开接口（不需要认证，Registry 调用）
	r.POST("/webhooks/:id", handler.ReceiveWebhook)

	// 需要认证的接口
	protected := r.Group("/api")
	protected.Use(middleware.AuthMiddleware())
	{
		// 用户相关
		protected.GET("/auth/me", handler.GetCurrentUser)
		protected.POST("/auth/change-password", handler.ChangePassword)

		protected.GET("/builtin/catalog", handler.BuiltinCatalog)
		protected.GET("/builtin/tags", handler.BuiltinTags)
		protected.GET("/auth/tokens", handler.ListTokens)
		protected.POST("/auth/tokens", handler.CreateToken)
		protected.DELETE("/auth/tokens/:id", handler.RevokeToken)
		protected.Use(middleware.RequireAdmin())
		protected.POST("/system/backups", handler.CreateBackup)
		protected.GET("/system/backups", handler.ListBackups)
		protected.GET("/system/backups/:id/download", handler.DownloadBackup)
		protected.GET("/targets", handler.ListTargets)
		protected.POST("/targets", handler.CreateTarget)
		protected.PATCH("/targets/:id", handler.UpdateTarget)
		protected.GET("/users", handler.ListUsers)
		protected.POST("/users", handler.CreateUser)
		protected.PATCH("/users/:id", handler.UpdateUser)
		protected.GET("/audit", handler.ListAudit)
		protected.GET("/overview", handler.Overview)
		protected.GET("/managed-templates", handler.ManagedTemplates)
		protected.POST("/managed-templates/imports", handler.PreviewTemplateImport)
		protected.POST("/managed-templates/imports/:import/commit", handler.CommitTemplateImport)
		protected.DELETE("/managed-templates/imports/:import", handler.CancelTemplateImport)
		protected.POST("/managed-templates/export", handler.ExportTemplatePackage)
		protected.GET("/managed-templates/:id/download", handler.ExportTemplatePackage)
		protected.DELETE("/managed-templates/:id", handler.DeleteManagedTemplate)
		protected.POST("/managed-templates/:id/applications", handler.CreateApplicationFromTemplate)
		protected.GET("/builtin/status", handler.BuiltinStatus)
		protected.POST("/builtin/maintenance", handler.RegistryMaintenance)
		protected.GET("/builtin/maintenance", handler.ListRegistryMaintenance)
		protected.DELETE("/builtin/manifests", handler.BuiltinDelete)
		protected.GET("/applications", handler.ListApplications)
		protected.POST("/applications", handler.CreateApplication)
		protected.GET("/applications/:id", handler.ApplicationDetail)
		protected.DELETE("/applications/:id", handler.DeleteApplication)
		protected.PUT("/applications/:id/scripts", handler.SaveApplicationScripts)
		protected.PUT("/applications/:id/deployment-config", handler.SaveApplicationDeployment)
		protected.POST("/applications/:id/template", handler.SaveApplicationTemplate)
		protected.GET("/applications/:id/sources", handler.ApplicationSources)
		protected.PUT("/applications/:id/environments", handler.SaveEnvironment)
		protected.PATCH("/applications/:id/environments/:environment_id/parameters", handler.SaveEnvironmentParameters)
		protected.PATCH("/applications/:id/environments/:environment_id/automation", handler.SaveEnvironmentAutomation)
		protected.DELETE("/applications/:id/environments/:environment_id", handler.DeleteEnvironment)
		protected.POST("/applications/:id/releases", handler.CreateRelease)
		protected.POST("/releases/:id/promote", handler.Promote)
		protected.DELETE("/releases/:id", handler.DeleteRelease)
		protected.DELETE("/releases/:id/retention", handler.ReleaseRetention)
		protected.POST("/deployments/preflight", handler.PreflightDeployment)
		protected.POST("/deployments", handler.EnqueueDeployment)
		protected.GET("/deployments", handler.ListDeployments)
		protected.DELETE("/deployments/:id", handler.DeleteDeployment)
		protected.GET("/deployments/:id", handler.DeploymentDetail)
		protected.GET("/deployments/:id/events", handler.DeploymentEvents)
		protected.GET("/bundles", handler.ListBundles)
		protected.DELETE("/bundles/:id", handler.DeleteBundle)
		protected.GET("/bundles/uploads/:upload", handler.UploadBundle)
		protected.PUT("/bundles/uploads/:upload", handler.UploadBundle)
		protected.POST("/bundles/uploads/:upload/complete", handler.FinishUpload)
		protected.POST("/bundles/import-directory", handler.ImportBundleDirectory)
		protected.POST("/bundles/:id/version", handler.BundleVersion)
		protected.POST("/bundles/:id/retry", handler.RetryBundle)
		protected.POST("/bundles/publish", handler.PublishBundle)
		protected.GET("/bundles/:id/download", handler.DownloadBundle)
		// 系统相关
		protected.GET("/system/docker-status", handler.CheckDockerStatus)
		protected.GET("/system/info", handler.GetSystemInfo)
		protected.GET("/system/config", handler.GetSystemConfig)
		protected.PUT("/system/config", handler.UpdateSystemConfig)

		// 容器管理
		protected.GET("/containers", handler.ListContainers)
		protected.GET("/containers/:id", handler.InspectContainer)
		protected.POST("/containers/:id/start", handler.StartContainer)
		protected.POST("/containers/:id/stop", handler.StopContainer)
		protected.POST("/containers/:id/restart", handler.RestartContainer)
		protected.POST("/containers/:id/kill", handler.KillContainer)
		protected.POST("/containers/:id/pause", handler.PauseContainer)
		protected.POST("/containers/:id/resume", handler.ResumeContainer)
		protected.POST("/containers/:id/rename", handler.RenameContainer)
		protected.PUT("/containers/:id/restart-policy", handler.UpdateContainerRestart)
		protected.POST("/containers/:id/commit", handler.CommitContainer)
		protected.POST("/containers/:id/console", handler.CreateContainerConsole)
		protected.DELETE("/containers/:id", handler.RemoveContainer)
		protected.GET("/containers/:id/logs", handler.GetContainerLogs)
		protected.GET("/containers/:id/stats", handler.GetContainerStats)
		protected.GET("/containers/:id/top", handler.GetContainerProcesses)

		// 镜像管理
		protected.GET("/images", handler.ListImages)
		protected.GET("/images/:id", handler.InspectImage)
		protected.GET("/images/:id/history", handler.GetImageHistory)
		protected.POST("/images/pull", handler.PullImage)
		protected.DELETE("/images/:id", handler.RemoveImage)
		protected.POST("/images/tag", handler.TagImage)
		protected.POST("/images/load", handler.LoadImage)
		protected.POST("/images/save", handler.SaveImage)
		protected.POST("/images/import", handler.ImportImage)
		protected.POST("/images/export", handler.ExportImages)
		protected.GET("/images/:id/export", handler.ExportImage)

		// 网络管理
		protected.GET("/networks", handler.ListNetworks)
		protected.GET("/networks/:id", handler.InspectNetwork)
		protected.POST("/networks", handler.CreateNetwork)
		protected.DELETE("/networks/:id", handler.RemoveNetwork)
		protected.POST("/networks/:id/connect", handler.ConnectNetwork)
		protected.POST("/networks/:id/disconnect", handler.DisconnectNetwork)

		// 卷管理
		protected.GET("/volumes", handler.ListVolumes)
		protected.GET("/volumes/:name", handler.InspectVolume)
		protected.POST("/volumes", handler.CreateVolume)
		protected.DELETE("/volumes/:name", handler.RemoveVolume)

		// 部署脚本管理
		protected.GET("/deploy-scripts", handler.ListDeployScripts)
		protected.GET("/deploy-scripts/:id", handler.GetDeployScript)
		protected.POST("/deploy-scripts", handler.CreateDeployScript)
		protected.PUT("/deploy-scripts/:id", handler.UpdateDeployScript)
		protected.DELETE("/deploy-scripts/:id", handler.DeleteDeployScript)
		protected.POST("/deploy-scripts/:id/execute", handler.ExecuteDeployScript)

		// 脚本模版
		protected.GET("/templates", handler.GetTemplates)
		protected.GET("/templates/:id", handler.GetTemplate)

		// Registry 管理
		protected.GET("/registries", handler.ListRegistries)
		protected.GET("/registries/:id", handler.GetRegistry)
		protected.POST("/registries", handler.CreateRegistry)
		protected.PUT("/registries/:id", handler.UpdateRegistry)
		protected.DELETE("/registries/:id", handler.DeleteRegistry)
		protected.POST("/registries/test", handler.TestRegistry)

		// Registry 镜像管理
		protected.GET("/registries/:id/images", handler.ListRegistryImages)
		protected.GET("/registries/:id/images/*name", handler.GetRegistryImageTags)
		protected.DELETE("/registries/:id/images/*name", handler.DeleteRegistryImage)

		// Webhook 管理
		protected.GET("/webhook-configs", handler.ListWebhookConfigs)
		protected.GET("/webhook-configs/:id", handler.GetWebhookConfig)
		protected.POST("/webhook-configs", handler.CreateWebhookConfig)
		protected.PUT("/webhook-configs/:id", handler.UpdateWebhookConfig)
		protected.DELETE("/webhook-configs/:id", handler.DeleteWebhookConfig)
		protected.GET("/webhook-configs/:id/url", handler.GetWebhookURL)
		protected.GET("/webhook-logs", handler.ListWebhookLogs)
	}

	// 静态文件服务（前端）
	setupStaticFiles(r)

	return r
}

// setupStaticFiles 设置静态文件服务
func setupStaticFiles(r *gin.Engine) {
	// 尝试多个可能的静态文件路径
	staticPaths := []string{
		"./static",         // Docker 环境
		"../frontend/dist", // 开发环境
		"./frontend/dist",  // 根目录运行
	}

	var staticPath string
	for _, path := range staticPaths {
		if _, err := os.Stat(path); err == nil {
			staticPath = path
			break
		}
	}

	if staticPath == "" {
		// 如果没有找到静态文件目录，只提供 API 服务
		r.NoRoute(func(c *gin.Context) {
			if !strings.HasPrefix(c.Request.URL.Path, "/api") {
				c.JSON(http.StatusNotFound, gin.H{
					"error": "Static files not found. Please build the frontend first or access the API at /api/*",
				})
			}
		})
		return
	}

	// 服务静态文件
	r.Static("/assets", filepath.Join(staticPath, "assets"))
	r.StaticFile("/favicon.svg", filepath.Join(staticPath, "favicon.svg"))

	// SPA 路由处理 - 所有非 API 和非静态资源请求都返回 index.html
	r.NoRoute(func(c *gin.Context) {
		// 如果是 API 请求，返回 404
		if strings.HasPrefix(c.Request.URL.Path, "/api") || strings.HasPrefix(c.Request.URL.Path, "/agent/") || strings.HasPrefix(c.Request.URL.Path, "/v2/") || c.Request.Method != "GET" {
			c.JSON(http.StatusNotFound, gin.H{"error": "API endpoint not found"})
			return
		}

		// 如果是静态资源请求（带扩展名），返回 404
		if strings.Contains(c.Request.URL.Path, ".") {
			c.Status(http.StatusNotFound)
			return
		}

		// 其他请求返回 index.html（前端路由）
		indexPath := filepath.Join(staticPath, "index.html")
		if _, err := os.Stat(indexPath); err == nil {
			c.File(indexPath)
		} else {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "index.html not found",
			})
		}
	})
}
