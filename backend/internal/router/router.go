package router

import (
	"dockerun/internal/handler"
	"dockerun/internal/middleware"
	"net/http"
	"os"
	"path/filepath"
	"strings"

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

	// 静态文件服务（前端）
	setupStaticFiles(r)

	return r
}

// setupStaticFiles 设置静态文件服务
func setupStaticFiles(r *gin.Engine) {
	// 尝试多个可能的静态文件路径
	staticPaths := []string{
		"./static",           // Docker 环境
		"../frontend/dist",   // 开发环境
		"./frontend/dist",    // 根目录运行
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
	r.StaticFile("/favicon.ico", filepath.Join(staticPath, "favicon.ico"))

	// SPA 路由处理 - 所有非 API 和非静态资源请求都返回 index.html
	r.NoRoute(func(c *gin.Context) {
		// 如果是 API 请求，返回 404
		if strings.HasPrefix(c.Request.URL.Path, "/api") {
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

