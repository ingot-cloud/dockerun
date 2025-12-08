#!/usr/bin/env bash

# DockeRun数据恢复脚本

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查参数
if [ $# -ne 1 ]; then
    echo "用法: $0 <备份文件路径>"
    echo "示例: $0 /backup/dockerun/dockerun-backup-20231201_020000.tar.gz"
    exit 1
fi

BACKUP_FILE="$1"

# 检查备份文件是否存在
if [ ! -f "${BACKUP_FILE}" ]; then
    log_error "备份文件不存在: ${BACKUP_FILE}"
    exit 1
fi

log_info "准备恢复数据..."
log_info "备份文件: ${BACKUP_FILE}"

# 确认操作
read -p "是否继续恢复？这将覆盖现有数据 (y/N) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    log_info "取消恢复操作"
    exit 0
fi

# 停止服务
log_info "停止服务..."
if command -v docker-compose &> /dev/null; then
    docker-compose down || true
fi

# 备份当前数据（以防恢复失败）
if [ -d "./data" ]; then
    log_info "备份当前数据..."
    CURRENT_BACKUP="./data-backup-$(date +%Y%m%d_%H%M%S)"
    cp -r ./data "${CURRENT_BACKUP}"
    log_info "当前数据已备份到: ${CURRENT_BACKUP}"
fi

# 恢复数据
log_info "恢复数据..."
tar xzf "${BACKUP_FILE}" -C .

log_info "数据恢复完成"

# 启动服务
read -p "是否立即启动服务？(Y/n) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Nn]$ ]]; then
    log_info "启动服务..."
    if command -v docker-compose &> /dev/null; then
        docker-compose up -d
        log_info "服务已启动"
    else
        log_warn "未找到docker-compose，请手动启动服务"
    fi
fi

log_info "恢复完成"

