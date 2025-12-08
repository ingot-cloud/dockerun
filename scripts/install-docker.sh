#!/usr/bin/env bash

# Docker自动安装脚本
# 支持Ubuntu/Debian/CentOS/RHEL/Fedora

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# 日志函数
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查是否为root用户
check_root() {
    if [ "$EUID" -ne 0 ]; then
        log_error "请使用root权限运行此脚本"
        exit 1
    fi
}

# 检测操作系统
detect_os() {
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        OS=$ID
        VER=$VERSION_ID
    else
        log_error "无法检测操作系统"
        exit 1
    fi
    
    log_info "检测到操作系统: $OS $VER"
}

# 检查Docker是否已安装
check_docker() {
    if command -v docker &> /dev/null; then
        log_warn "Docker已安装"
        docker --version
        return 0
    fi
    return 1
}

# 安装Docker - Ubuntu/Debian
install_docker_ubuntu() {
    log_info "开始在Ubuntu/Debian上安装Docker..."
    
    # 更新包索引
    apt-get update
    
    # 安装依赖
    apt-get install -y \
        ca-certificates \
        curl \
        gnupg \
        lsb-release
    
    # 添加Docker官方GPG密钥
    mkdir -p /etc/apt/keyrings
    curl -fsSL https://download.docker.com/linux/$OS/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
    
    # 设置仓库
    echo \
      "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/$OS \
      $(lsb_release -cs) stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null
    
    # 安装Docker Engine
    apt-get update
    apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
    
    log_info "Docker安装完成"
}

# 安装Docker - CentOS/RHEL/Fedora
install_docker_centos() {
    log_info "开始在CentOS/RHEL/Fedora上安装Docker..."
    
    # 安装依赖
    yum install -y yum-utils
    
    # 添加Docker仓库
    yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo
    
    # 安装Docker Engine
    yum install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
    
    log_info "Docker安装完成"
}

# 启动Docker服务
start_docker() {
    log_info "启动Docker服务..."
    
    systemctl start docker
    systemctl enable docker
    
    # 验证安装
    if docker run --rm hello-world &> /dev/null; then
        log_info "Docker安装验证成功"
    else
        log_warn "Docker安装验证失败，请检查配置"
    fi
}

# 配置Docker（可选）
configure_docker() {
    log_info "配置Docker..."
    
    # 创建Docker配置目录
    mkdir -p /etc/docker
    
    # 配置镜像加速（可选）
    cat > /etc/docker/daemon.json <<EOF
{
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "10m",
    "max-file": "3"
  }
}
EOF
    
    # 重启Docker以应用配置
    systemctl restart docker
    
    log_info "Docker配置完成"
}

# 添加用户到docker组（可选）
add_user_to_docker_group() {
    if [ -n "$SUDO_USER" ]; then
        log_info "将用户 $SUDO_USER 添加到docker组..."
        usermod -aG docker $SUDO_USER
        log_info "请重新登录以使组权限生效"
    fi
}

# 主函数
main() {
    log_info "=== Docker自动安装脚本 ==="
    
    # 检查root权限
    check_root
    
    # 检查Docker是否已安装
    if check_docker; then
        read -p "Docker已安装，是否重新安装？(y/N) " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log_info "退出安装"
            exit 0
        fi
    fi
    
    # 检测操作系统
    detect_os
    
    # 根据操作系统安装Docker
    case $OS in
        ubuntu|debian)
            install_docker_ubuntu
            ;;
        centos|rhel|fedora)
            install_docker_centos
            ;;
        *)
            log_error "不支持的操作系统: $OS"
            exit 1
            ;;
    esac
    
    # 启动Docker服务
    start_docker
    
    # 配置Docker
    configure_docker
    
    # 添加用户到docker组
    add_user_to_docker_group
    
    log_info "=== Docker安装完成 ==="
    docker --version
    docker compose version
}

# 运行主函数
main "$@"

