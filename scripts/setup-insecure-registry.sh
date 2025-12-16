#!/bin/bash

# Docker Insecure Registry 配置脚本
# 用于配置 Docker daemon 允许不安全的 HTTP Registry

set -e

REGISTRY_URL="${1:-}"
SCRIPT_NAME=$(basename "$0")

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

print_error() {
    echo -e "${RED}[错误]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[成功]${NC} $1"
}

print_info() {
    echo -e "${YELLOW}[信息]${NC} $1"
}

print_usage() {
    cat << EOF
用法: $SCRIPT_NAME <registry-url>

示例:
  $SCRIPT_NAME 192.168.0.170:5000
  $SCRIPT_NAME localhost:5000
  $SCRIPT_NAME 192.168.0.0/24:5000

说明:
  此脚本会修改 /etc/docker/daemon.json 文件，添加 insecure-registries 配置，
  并重启 Docker 服务使配置生效。

要求:
  - 需要 root 权限（使用 sudo 运行）
  - 已安装 jq 命令（用于处理 JSON）

EOF
}

# 检查参数
if [ -z "$REGISTRY_URL" ]; then
    print_error "缺少 registry 地址参数"
    print_usage
    exit 1
fi

# 检查是否有 root 权限
if [ "$EUID" -ne 0 ]; then
    print_error "请使用 sudo 运行此脚本"
    exit 1
fi

print_info "准备配置 Docker 允许不安全的 Registry: $REGISTRY_URL"

# 检查 jq 是否安装
if ! command -v jq &> /dev/null; then
    print_error "未找到 jq 命令，正在尝试安装..."
    
    # 尝试安装 jq
    if command -v apt-get &> /dev/null; then
        apt-get update && apt-get install -y jq
    elif command -v yum &> /dev/null; then
        yum install -y jq
    elif command -v apk &> /dev/null; then
        apk add --no-cache jq
    else
        print_error "无法自动安装 jq，请手动安装后重试"
        exit 1
    fi
    
    print_success "jq 安装成功"
fi

# 配置文件路径
DAEMON_CONFIG="/etc/docker/daemon.json"
BACKUP_DIR="/etc/docker/backups"

# 创建备份目录
mkdir -p "$BACKUP_DIR"

# 备份原配置
if [ -f "$DAEMON_CONFIG" ]; then
    BACKUP_FILE="$BACKUP_DIR/daemon.json.backup.$(date +%Y%m%d_%H%M%S)"
    cp "$DAEMON_CONFIG" "$BACKUP_FILE"
    print_success "已备份原配置到: $BACKUP_FILE"
fi

# 创建或更新配置
if [ -f "$DAEMON_CONFIG" ]; then
    # 文件已存在，使用 jq 添加或更新配置
    print_info "更新现有配置文件..."
    
    # 读取现有的 insecure-registries 配置
    EXISTING_REGISTRIES=$(jq -r '.["insecure-registries"] // []' "$DAEMON_CONFIG")
    
    # 添加新的 registry（去重）
    NEW_CONFIG=$(jq \
        --arg registry "$REGISTRY_URL" \
        '.["insecure-registries"] = (.["insecure-registries"] // [] | . + [$registry] | unique)' \
        "$DAEMON_CONFIG")
    
    # 写入文件
    echo "$NEW_CONFIG" > "$DAEMON_CONFIG"
else
    # 创建新文件
    print_info "创建新配置文件..."
    cat > "$DAEMON_CONFIG" << EOF
{
  "insecure-registries": ["$REGISTRY_URL"]
}
EOF
fi

print_success "配置文件已更新"

# 验证 JSON 格式
if ! jq empty "$DAEMON_CONFIG" 2>/dev/null; then
    print_error "配置文件 JSON 格式错误，正在恢复备份..."
    if [ -f "$BACKUP_FILE" ]; then
        cp "$BACKUP_FILE" "$DAEMON_CONFIG"
        print_info "已恢复备份配置"
    fi
    exit 1
fi

# 显示当前配置
print_info "当前 insecure-registries 配置:"
jq '.["insecure-registries"]' "$DAEMON_CONFIG"

# 重启 Docker
print_info "重启 Docker 服务..."

if command -v systemctl &> /dev/null; then
    systemctl daemon-reload
    systemctl restart docker
    
    # 等待 Docker 启动
    sleep 3
    
    if systemctl is-active --quiet docker; then
        print_success "Docker 服务已成功重启"
    else
        print_error "Docker 服务重启失败，请检查日志: journalctl -u docker"
        exit 1
    fi
elif command -v service &> /dev/null; then
    service docker restart
    sleep 3
    print_success "Docker 服务已重启"
else
    print_error "无法自动重启 Docker，请手动重启"
    exit 1
fi

# 验证配置
print_info "验证配置..."
if docker info 2>/dev/null | grep -q "$REGISTRY_URL"; then
    print_success "配置验证成功！"
else
    print_error "配置验证失败，请检查 Docker 状态"
    exit 1
fi

# 显示 insecure registries 列表
echo ""
print_success "配置完成！当前允许的不安全 Registry:"
docker info 2>/dev/null | grep -A 10 "Insecure Registries" || echo "  $REGISTRY_URL"

echo ""
print_info "现在可以使用以下命令测试:"
echo "  docker pull $REGISTRY_URL/image:tag"

exit 0

