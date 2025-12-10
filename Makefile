.PHONY: help build run dev clean docker-build docker-run install-deps

help: ## 显示帮助信息
	@echo "DockeRun - Docker管理平台"
	@echo ""
	@echo "可用命令："
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

install-deps: ## 安装依赖
	@echo "安装前端依赖..."
	cd frontend && pnpm install
	@echo "安装后端依赖..."
	cd backend && go mod download

build: ## 构建项目
	@echo "构建前端..."
	cd frontend && pnpm run build
	@echo "复制前端构建产物到后端..."
	rm -rf backend/static
	cp -r frontend/dist backend/static
	@echo "构建后端..."
	cd backend && go build -o ../bin/dockerun

run: ## 运行应用
	@echo "启动应用..."
	./bin/dockerun

dev-frontend: ## 开发模式运行前端
	cd frontend && pnpm run dev

dev-backend: ## 开发模式运行后端
	cd backend && TEMPLATE_DIR=../templates go run main.go

clean: ## 清理构建文件
	@echo "清理构建文件..."
	rm -rf bin/
	rm -rf frontend/dist/
	rm -rf data/

docker-build: ## 构建Docker镜像
	@echo "构建Docker镜像..."
	docker build -t dockerun:latest .

docker-run: ## 运行Docker容器
	@echo "运行Docker容器..."
	docker-compose up -d

docker-stop: ## 停止Docker容器
	@echo "停止Docker容器..."
	docker-compose down

docker-logs: ## 查看Docker容器日志
	docker-compose logs -f

test-backend: ## 运行后端测试
	cd backend && go test ./...

lint-backend: ## 后端代码检查
	cd backend && go fmt ./...
	cd backend && go vet ./...

