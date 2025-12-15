#!/usr/bin/env bash

###############################################################################
# Dockerun Build Script
# 用途：本地打包前后端，然后构建 Docker 镜像
###############################################################################

set -e  # 遇到错误立即退出

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 配置
IMAGE_NAME="ingot/dockerun"
IMAGE_TAG=0.1.0 # 默认 tag 为 latest，可通过参数指定
DOCKER_REGISTRY=docker-registry.ingotcloud.top  # 可选的镜像仓库地址
IMAGE_ALREADY_PUSHED=false  # 标记镜像是否已推送

# 日志函数
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查命令是否存在
check_command() {
    if ! command -v "$1" &> /dev/null; then
        log_error "$1 未安装，请先安装 $1"
        exit 1
    fi
}

# 打印横幅
print_banner() {
    echo ""
    echo "╔══════════════════════════════════════════════════════════════╗"
    echo "║                   Dockerun Build Script                      ║"
    echo "║                 构建前后端 + Docker 镜像                       ║"
    echo "╚══════════════════════════════════════════════════════════════╝"
    echo ""
}

# 检查依赖
check_dependencies() {
    log_info "检查构建依赖..."
    
    check_command "node"
    check_command "pnpm"
    check_command "go"
    check_command "docker"
    
    log_success "所有依赖检查通过"
}

# 清理旧的构建产物
clean_old_builds() {
    log_info "清理旧的构建产物..."
    
    # 清理前端构建产物
    if [ -d "frontend/dist" ]; then
        rm -rf frontend/dist
        log_success "已清理前端构建产物"
    fi
    
    # 清理后端二进制文件
    if [ -f "backend/main" ]; then
        rm -f backend/main
        log_success "已清理后端二进制文件"
    fi
}

# 构建前端
build_frontend() {
    log_info "开始构建前端..."
    
    cd frontend
    
    # 检查 node_modules
    if [ ! -d "node_modules" ]; then
        log_info "安装前端依赖..."
        pnpm install --frozen-lockfile
    fi
    
    # 构建
    log_info "执行前端构建（pnpm run build）..."
    pnpm run build
    
    cd ..
    
    # 检查构建产物
    if [ ! -d "frontend/dist" ]; then
        log_error "前端构建失败，未找到 dist 目录"
        exit 1
    fi
    
    log_success "前端构建完成 ✓"
}

# 构建后端（多架构）
build_backend() {
    log_info "开始构建后端（多架构：amd64 + arm64）..."
    
    cd backend
    
    # 下载依赖
    log_info "下载 Go 依赖..."
    go mod download
    
    # 构建 AMD64 版本
    log_info "构建 AMD64 版本..."
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags='-s -w' -o main-amd64 .
    
    if [ $? -ne 0 ]; then
        log_error "AMD64 构建失败"
        cd ..
        exit 1
    fi
    log_success "AMD64 版本构建完成 ✓"
    
    # 构建 ARM64 版本
    log_info "构建 ARM64 版本..."
    CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -a -installsuffix cgo -ldflags='-s -w' -o main-arm64 .
    
    if [ $? -ne 0 ]; then
        log_error "ARM64 构建失败"
        cd ..
        exit 1
    fi
    log_success "ARM64 版本构建完成 ✓"
    
    cd ..
    
    # 检查构建产物
    if [ ! -f "backend/main-amd64" ] || [ ! -f "backend/main-arm64" ]; then
        log_error "后端构建失败，未找到二进制文件"
        exit 1
    fi
    
    log_success "后端构建完成 ✓"
}

# 构建 Docker 镜像（多架构）
build_docker_image() {
    log_info "开始构建 Docker 多架构镜像..."
    
    # 构建完整镜像名称
    FULL_IMAGE_NAME="${IMAGE_NAME}:${IMAGE_TAG}"
    if [ -n "$DOCKER_REGISTRY" ]; then
        FULL_IMAGE_NAME="${DOCKER_REGISTRY}/${FULL_IMAGE_NAME}"
    fi
    
    log_info "镜像名称: ${FULL_IMAGE_NAME}"
    log_info "目标架构: linux/amd64, linux/arm64"
    
    # 确保 buildx 可用
    if ! docker buildx version &> /dev/null; then
        log_error "Docker Buildx 不可用，请升级 Docker"
        exit 1
    fi
    
    # 创建或使用 builder
    BUILDER_NAME="dockerun-builder"
    if ! docker buildx inspect "$BUILDER_NAME" &> /dev/null; then
        log_info "创建 buildx builder..."
        docker buildx create --name "$BUILDER_NAME" --use
    else
        docker buildx use "$BUILDER_NAME"
    fi
    
    # 构建并推送多架构镜像
    log_info "构建多架构镜像（这可能需要几分钟）..."
    docker buildx build \
        --platform linux/amd64,linux/arm64 \
        --tag "${FULL_IMAGE_NAME}" \
        --push \
        --no-cache \
        --file Dockerfile.multiarch \
        .
    
    if [ $? -eq 0 ]; then
        log_success "Docker 多架构镜像构建并推送完成 ✓"
        log_info "镜像: ${FULL_IMAGE_NAME}"
        log_info "支持架构: amd64, arm64"
        
        echo ""
        log_info "验证多架构镜像:"
        docker buildx imagetools inspect "${FULL_IMAGE_NAME}"
        
        # 标记镜像已推送（避免重复推送）
        IMAGE_ALREADY_PUSHED=true
    else
        log_error "Docker 镜像构建失败"
        exit 1
    fi
}

# 询问是否推送镜像
ask_push_image() {
    # 如果镜像已经通过 buildx 推送，跳过
    if [ "$IMAGE_ALREADY_PUSHED" = true ]; then
        log_info "镜像已通过 buildx 推送到仓库 ✓"
        return
    fi
    
    if [ -z "$DOCKER_REGISTRY" ]; then
        return
    fi
    
    echo ""
    read -p "是否推送镜像到仓库？(y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        push_docker_image
    fi
}

# 推送 Docker 镜像
push_docker_image() {
    FULL_IMAGE_NAME="${DOCKER_REGISTRY}/${IMAGE_NAME}:${IMAGE_TAG}"
    
    log_info "推送镜像到仓库..."
    docker push "${FULL_IMAGE_NAME}"
    
    if [ $? -eq 0 ]; then
        log_success "镜像推送完成 ✓"
    else
        log_error "镜像推送失败"
        exit 1
    fi
}

# 显示构建摘要
print_summary() {
    echo ""
    echo "╔══════════════════════════════════════════════════════════════╗"
    echo "║                      构建完成摘要                              ║"
    echo "╠══════════════════════════════════════════════════════════════╣"
    echo "║  ✓ 前端构建: frontend/dist                                    ║"
    echo "║  ✓ 后端构建: backend/main                                     ║"
    echo "║  ✓ Docker 镜像: ${IMAGE_NAME}:${IMAGE_TAG}"
    echo "╠══════════════════════════════════════════════════════════════╣"
    echo "║  运行镜像:                                                     ║"
    echo "║  docker run -d -p 8080:8080 \\                                ║"
    echo "║    -v /var/run/docker.sock:/var/run/docker.sock \\            ║"
    echo "║    -v dockerun-data:/app/data \\                              ║"
    echo "║    --name dockerun \\                                         ║"
    echo "║    ${IMAGE_NAME}:${IMAGE_TAG}"
    echo "╚══════════════════════════════════════════════════════════════╝"
    echo ""
}

# 显示使用帮助
show_usage() {
    echo "用法: ./build.sh [IMAGE_TAG]"
    echo ""
    echo "参数:"
    echo "  IMAGE_TAG    Docker 镜像标签（默认: latest）"
    echo ""
    echo "环境变量:"
    echo "  DOCKER_REGISTRY    Docker 镜像仓库地址（可选）"
    echo ""
    echo "示例:"
    echo "  ./build.sh              # 构建 dockerun:latest"
    echo "  ./build.sh v1.0.0       # 构建 dockerun:v1.0.0"
    echo "  DOCKER_REGISTRY=docker.io/myuser ./build.sh v1.0.0"
    echo ""
}

# 主函数
main() {
    # 检查参数
    if [ "$1" == "-h" ] || [ "$1" == "--help" ]; then
        show_usage
        exit 0
    fi
    
    # 记录开始时间
    START_TIME=$(date +%s)
    
    print_banner
    
    log_info "镜像标签: ${IMAGE_TAG}"
    if [ -n "$DOCKER_REGISTRY" ]; then
        log_info "镜像仓库: ${DOCKER_REGISTRY}"
    fi
    echo ""
    
    # 执行构建步骤
    check_dependencies
    clean_old_builds
    
    echo ""
    build_frontend
    
    echo ""
    build_backend
    
    echo ""
    build_docker_image
    
    # 询问是否推送
    ask_push_image
    
    # 计算耗时
    END_TIME=$(date +%s)
    DURATION=$((END_TIME - START_TIME))
    
    echo ""
    log_success "所有构建步骤完成！耗时: ${DURATION}s"
    
    # 显示摘要
    print_summary
}

# 执行主函数
main "$@"

