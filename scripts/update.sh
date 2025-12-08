#!/usr/bin/env bash

# DockeRun更新脚本

set -e

# 颜色输出
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_info "=== DockeRun更新脚本 ==="

# 备份数据
log_info "备份当前数据..."
./scripts/backup.sh

# 拉取最新代码
log_info "拉取最新代码..."
git pull

# 停止服务
log_info "停止服务..."
docker-compose down

# 重新构建镜像
log_info "重新构建镜像..."
docker-compose build

# 启动服务
log_info "启动服务..."
docker-compose up -d

# 等待服务启动
log_info "等待服务启动..."
sleep 5

# 检查服务状态
log_info "检查服务状态..."
docker-compose ps

# 查看日志
log_info "查看最新日志..."
docker-compose logs --tail=20

log_info "=== 更新完成 ==="
log_info "请访问 http://localhost:8080 验证服务"

