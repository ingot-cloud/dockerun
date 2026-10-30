ROOT_DIR := $(abspath $(dir $(lastword $(MAKEFILE_LIST))))
GO ?= go
PNPM ?= pnpm

# Keep the official checksum identity; only its transport uses the mirror.
# Override with environment variables or `make GOPROXY=... GOSUMDB=...`.
GOPROXY ?= https://goproxy.cn,direct
GOSUMDB ?= sum.golang.org https://goproxy.cn/sumdb/sum.golang.org
export GOPROXY GOSUMDB

DATA_DIR ?= $(ROOT_DIR)/data/local
TEMPLATE_DIR ?= $(ROOT_DIR)/templates
DATABASE_TYPE ?= sqlite
SERVER_ADDR ?= 127.0.0.1:8080
# Local Make targets default to Docker Desktop's host address; override for other environments.
PUBLIC_URL ?= http://host.docker.internal:8080
BACKEND_URL ?= http://127.0.0.1:8080
FRONTEND_PORT ?= 5999
BACKEND_ENV = DATA_DIR="$(abspath $(DATA_DIR))" TEMPLATE_DIR="$(abspath $(TEMPLATE_DIR))" DATABASE_TYPE="$(DATABASE_TYPE)" SERVER_ADDR="$(SERVER_ADDR)" PUBLIC_URL="$(PUBLIC_URL)"

.PHONY: help build run clean docker-build docker-run docker-stop docker-logs install-deps dev-frontend dev-backend test-backend lint-backend

help: ## 显示帮助信息
	@echo "DockeRun - Docker管理平台"
	@echo ""
	@echo "可用命令："
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

install-deps: ## 安装依赖
	@echo "安装前端依赖..."
	cd "$(ROOT_DIR)/frontend" && $(PNPM) install --frozen-lockfile
	@echo "安装后端依赖..."
	cd "$(ROOT_DIR)/backend" && $(GO) mod download

build: ## 构建项目
	@echo "构建前端..."
	cd "$(ROOT_DIR)/frontend" && $(PNPM) run build
	@echo "复制前端构建产物到后端..."
	rm -rf "$(ROOT_DIR)/backend/static"
	cp -r "$(ROOT_DIR)/frontend/dist" "$(ROOT_DIR)/backend/static"
	@echo "构建后端..."
	cd "$(ROOT_DIR)/backend" && $(GO) build -o ../bin/dockerun-server .
	cd "$(ROOT_DIR)/backend" && $(GO) build -o ../bin/dockerun ./cmd/dockerun

run: ## 运行应用
	@echo "启动应用..."
	cd "$(ROOT_DIR)" && $(BACKEND_ENV) ./bin/dockerun-server

dev-frontend: ## 开发模式运行前端（默认 http://localhost:5173）
	cd "$(ROOT_DIR)/frontend" && VITE_BACKEND_URL="$(BACKEND_URL)" $(PNPM) run dev --port "$(FRONTEND_PORT)" --strictPort

dev-backend: ## 开发模式运行后端（默认 SQLite，使用 data/local）
	cd "$(ROOT_DIR)/backend" && $(BACKEND_ENV) $(GO) run .

clean: ## 清理构建文件
	@echo "清理构建文件..."
	rm -rf bin/
	rm -rf frontend/dist/

docker-build: ## 构建Docker镜像
	@echo "构建Docker镜像..."
	docker build -t dockerun:latest .

docker-run: ## 运行Docker容器
	@echo "运行Docker容器..."
	docker compose up -d

docker-stop: ## 停止Docker容器
	@echo "停止Docker容器..."
	docker compose down

docker-logs: ## 查看Docker容器日志
	docker compose logs -f

test-backend: ## 运行后端测试
	cd "$(ROOT_DIR)/backend" && $(GO) test ./...

lint-backend: ## 后端代码检查
	cd "$(ROOT_DIR)/backend" && $(GO) fmt ./...
	cd "$(ROOT_DIR)/backend" && $(GO) vet ./...

