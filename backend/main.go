package main

import (
	"dockerun/internal/config"
	"dockerun/internal/database"
	"dockerun/internal/docker"
	"dockerun/internal/router"
	"log"
)

func main() {
	// 加载配置
	cfg := config.Load()

	// 初始化数据库
	if err := database.Init(cfg.DatabasePath); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// 初始化 Docker 客户端
	if err := docker.Init(); err != nil {
		log.Fatalf("Failed to initialize Docker client: %v", err)
	}

	// 初始化 Docker 网络
	if err := docker.InitializeNetwork(); err != nil {
		log.Printf("Warning: Failed to initialize Docker network: %v", err)
	}

	// 设置路由
	r := router.Setup()

	// 启动服务器
	log.Printf("Server starting on %s", cfg.ServerAddr)
	if err := r.Run(cfg.ServerAddr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

