package main

import (
	"dockerun/internal/config"
	"dockerun/internal/database"
	"dockerun/internal/docker"
	"dockerun/internal/router"
	"log"

	"github.com/gin-gonic/gin"
)

func main() {
	// 初始化配置
	cfg := config.Load()

	// 初始化数据库
	if err := database.Init(cfg.DatabasePath); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// 初始化Docker客户端
	if err := docker.Init(); err != nil {
		log.Printf("Warning: Failed to initialize Docker client: %v", err)
		log.Println("Docker functionality may not be available")
	}

	// 设置Gin模式
	if cfg.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	// 初始化路由
	r := router.Setup()

	// 启动服务器
	addr := cfg.ServerAddr
	log.Printf("Server starting on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

