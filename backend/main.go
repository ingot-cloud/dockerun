package main

import (
	"dockerun/internal/config"
	"dockerun/internal/database"
	"dockerun/internal/docker"
	"dockerun/internal/router"
	"dockerun/internal/webhook"
	"log"
	"os"
	"os/signal"
	"syscall"
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

	// 初始化 Webhook 任务队列
	webhook.InitQueue(3) // 3 个工作协程
	defer webhook.Shutdown()

	// 设置优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// 设置路由
	r := router.Setup()

	// 在后台启动服务器
	go func() {
		log.Printf("Server starting on %s", cfg.ServerAddr)
		if err := r.Run(cfg.ServerAddr); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// 等待退出信号
	<-quit
	log.Println("Shutting down server...")
	webhook.Shutdown()
	log.Println("Server gracefully stopped")
}

