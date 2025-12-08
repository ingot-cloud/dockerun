#!/usr/bin/env bash

# DockeRun数据备份脚本

set -e

# 配置
BACKUP_DIR="${BACKUP_DIR:-/backup/dockerun}"
DATA_DIR="${DATA_DIR:-./data}"
DATE=$(date +%Y%m%d_%H%M%S)
RETENTION_DAYS="${RETENTION_DAYS:-30}"

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

# 创建备份目录
mkdir -p "${BACKUP_DIR}"

log_info "开始备份..."

# 备份数据库
if [ -f "${DATA_DIR}/dockerun.db" ]; then
    log_info "备份数据库..."
    cp "${DATA_DIR}/dockerun.db" "${BACKUP_DIR}/dockerun-${DATE}.db"
else
    log_warn "数据库文件不存在: ${DATA_DIR}/dockerun.db"
fi

# 创建完整备份
log_info "创建完整备份..."
tar czf "${BACKUP_DIR}/dockerun-backup-${DATE}.tar.gz" \
    -C "$(dirname ${DATA_DIR})" \
    "$(basename ${DATA_DIR})" \
    2>/dev/null || log_warn "备份过程中有些文件无法访问"

# 删除旧备份
log_info "清理旧备份（保留${RETENTION_DAYS}天）..."
find "${BACKUP_DIR}" -name "dockerun-backup-*.tar.gz" -mtime +${RETENTION_DAYS} -delete
find "${BACKUP_DIR}" -name "dockerun-*.db" -mtime +${RETENTION_DAYS} -delete

# 显示备份信息
BACKUP_SIZE=$(du -sh "${BACKUP_DIR}/dockerun-backup-${DATE}.tar.gz" | cut -f1)
log_info "备份完成"
log_info "备份文件: ${BACKUP_DIR}/dockerun-backup-${DATE}.tar.gz"
log_info "备份大小: ${BACKUP_SIZE}"
log_info "备份列表:"
ls -lh "${BACKUP_DIR}"/dockerun-backup-*.tar.gz | tail -5

