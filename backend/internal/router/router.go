package router

import (
	"dockerun/internal/handler"
	"dockerun/internal/middleware"

	"github.com/gin-gonic/gin"
)

func Setup() *gin.Engine {
	r := gin.Default()

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
		// 认证相关
		public.POST("/auth/login", handler.Login)
		public.GET("/auth/check-setup", handler.CheckInitialSetup)
		public.POST("/auth/setup", handler.SetInitialPassword)
	}

	// 需要认证的接口
	protected := r.Group("/api")
	protected.Use(middleware.AuthMiddleware())
	{
		// 用户相关
		protected.GET("/auth/me", handler.GetCurrentUser)
		protected.POST("/auth/change-password", handler.ChangePassword)

		// 系统相关
		protected.GET("/system/docker-status", handler.CheckDockerStatus)
		protected.GET("/system/info", handler.GetSystemInfo)
		protected.POST("/system/install-docker", handler.InstallDocker)

		// 容器管理
		protected.GET("/containers", handler.ListContainers)
		protected.GET("/containers/:id", handler.InspectContainer)
		protected.POST("/containers/:id/start", handler.StartContainer)
		protected.POST("/containers/:id/stop", handler.StopContainer)
		protected.POST("/containers/:id/restart", handler.RestartContainer)
		protected.DELETE("/containers/:id", handler.RemoveContainer)
		protected.GET("/containers/:id/logs", handler.GetContainerLogs)
		protected.GET("/containers/:id/stats", handler.GetContainerStats)

		// 镜像管理
		protected.GET("/images", handler.ListImages)
		protected.GET("/images/:id", handler.InspectImage)
		protected.POST("/images/pull", handler.PullImage)
		protected.DELETE("/images/:id", handler.RemoveImage)
		protected.POST("/images/tag", handler.TagImage)

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
	}

	return r
}

