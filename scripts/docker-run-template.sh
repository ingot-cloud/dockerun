#!/usr/bin/env bash

###############################################################################
# Docker Container Run Template
# 通用的 Docker 容器运行脚本模板
# 
# 用法:
#   1. 使用配置文件: ./docker-run-template.sh -c myapp.conf start
#   2. 交互式配置: ./docker-run-template.sh interactive
#   3. 快速启动: ./docker-run-template.sh start
#   4. 查看帮助: ./docker-run-template.sh help
###############################################################################

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
MAGENTA='\033[0;35m'
NC='\033[0m'

# 默认配置文件
DEFAULT_CONFIG_FILE="docker-run.conf"
CONFIG_FILE=""

# 容器配置（可通过配置文件或环境变量覆盖）
CONTAINER_NAME="${CONTAINER_NAME:-myapp}"
IMAGE_NAME="${IMAGE_NAME:-nginx:latest}"
CONTAINER_HOSTNAME="${CONTAINER_HOSTNAME:-}"
RESTART_POLICY="${RESTART_POLICY:-unless-stopped}"

# 网络配置
NETWORK_MODE="${NETWORK_MODE:-bridge}"
NETWORK_NAME="${NETWORK_NAME:-}"

# 端口映射（格式：host_port:container_port）
# 可以定义多个，用空格分隔
PORTS="${PORTS:-}"
# 示例: PORTS="8080:80 8443:443"

# 环境变量（格式：KEY=VALUE）
# 可以定义多个，用换行分隔
ENV_VARS="${ENV_VARS:-}"
# 示例:
# ENV_VARS="
# DATABASE_HOST=mysql
# DATABASE_PORT=3306
# "

# 挂载卷（格式：host_path:container_path[:options]）
# 可以定义多个，用换行分隔
VOLUMES="${VOLUMES:-}"
# 示例:
# VOLUMES="
# /data/myapp:/app/data
# /logs/myapp:/app/logs:ro
# "

# 命名卷（格式：volume_name:container_path[:options]）
NAMED_VOLUMES="${NAMED_VOLUMES:-}"
# 示例:
# NAMED_VOLUMES="
# myapp-data:/app/data
# myapp-logs:/app/logs
# "

# Docker socket 挂载
MOUNT_DOCKER_SOCKET="${MOUNT_DOCKER_SOCKET:-false}"

# 额外的 Docker run 参数
EXTRA_DOCKER_ARGS="${EXTRA_DOCKER_ARGS:-}"

# 健康检查
HEALTH_CHECK_ENABLED="${HEALTH_CHECK_ENABLED:-false}"
HEALTH_CHECK_CMD="${HEALTH_CHECK_CMD:-}"
HEALTH_CHECK_INTERVAL="${HEALTH_CHECK_INTERVAL:-30s}"
HEALTH_CHECK_TIMEOUT="${HEALTH_CHECK_TIMEOUT:-10s}"
HEALTH_CHECK_RETRIES="${HEALTH_CHECK_RETRIES:-3}"

# 资源限制
CPU_LIMIT="${CPU_LIMIT:-}"
MEMORY_LIMIT="${MEMORY_LIMIT:-}"

# 用户配置
RUN_AS_USER="${RUN_AS_USER:-}"

# 工作目录
WORKDIR="${WORKDIR:-}"

# 日志配置
LOG_DRIVER="${LOG_DRIVER:-json-file}"
LOG_MAX_SIZE="${LOG_MAX_SIZE:-10m}"
LOG_MAX_FILE="${LOG_MAX_FILE:-3}"

###############################################################################
# 工具函数
###############################################################################

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[✓]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[⚠]${NC} $1"
}

log_error() {
    echo -e "${RED}[✗]${NC} $1"
}

log_step() {
    echo -e "${CYAN}[→]${NC} $1"
}

# 分隔线
print_separator() {
    echo -e "${BLUE}════════════════════════════════════════════════════════════════${NC}"
}

# 检查 Docker
check_docker() {
    if ! command -v docker &> /dev/null; then
        log_error "Docker 未安装"
        echo "请安装 Docker: https://docs.docker.com/get-docker/"
        exit 1
    fi
    
    if ! docker info &> /dev/null; then
        log_error "Docker 守护进程未运行"
        exit 1
    fi
}

# 加载配置文件
load_config() {
    if [ -n "$CONFIG_FILE" ] && [ -f "$CONFIG_FILE" ]; then
        log_info "加载配置文件: $CONFIG_FILE"
        # shellcheck source=/dev/null
        source "$CONFIG_FILE"
        log_success "配置加载完成"
    elif [ -f "$DEFAULT_CONFIG_FILE" ]; then
        log_info "加载默认配置文件: $DEFAULT_CONFIG_FILE"
        # shellcheck source=/dev/null
        source "$DEFAULT_CONFIG_FILE"
        log_success "配置加载完成"
    fi
}

# 验证配置
validate_config() {
    local errors=0
    
    if [ -z "$CONTAINER_NAME" ]; then
        log_error "容器名称 (CONTAINER_NAME) 未设置"
        errors=$((errors + 1))
    fi
    
    if [ -z "$IMAGE_NAME" ]; then
        log_error "镜像名称 (IMAGE_NAME) 未设置"
        errors=$((errors + 1))
    fi
    
    return $errors
}

# 检查镜像
check_image() {
    log_step "检查镜像: $IMAGE_NAME"
    
    if ! docker image inspect "$IMAGE_NAME" &> /dev/null; then
        log_warning "镜像不存在，尝试拉取..."
        if docker pull "$IMAGE_NAME"; then
            log_success "镜像拉取成功"
        else
            log_error "镜像拉取失败"
            exit 1
        fi
    else
        log_success "镜像已存在"
    fi
}

# 构建 docker run 命令
build_docker_command() {
    local cmd="docker run -d"
    
    # 容器名称
    cmd="$cmd --name $CONTAINER_NAME"
    
    # 主机名
    if [ -n "$CONTAINER_HOSTNAME" ]; then
        cmd="$cmd --hostname $CONTAINER_HOSTNAME"
    fi
    
    # 重启策略
    cmd="$cmd --restart $RESTART_POLICY"
    
    # 网络配置
    if [ -n "$NETWORK_NAME" ]; then
        cmd="$cmd --network $NETWORK_NAME"
    elif [ -n "$NETWORK_MODE" ]; then
        cmd="$cmd --network $NETWORK_MODE"
    fi
    
    # 端口映射
    if [ -n "$PORTS" ]; then
        for port in $PORTS; do
            cmd="$cmd -p $port"
        done
    fi
    
    # 环境变量
    if [ -n "$ENV_VARS" ]; then
        while IFS= read -r env_var; do
            [ -z "$env_var" ] && continue
            cmd="$cmd -e \"$env_var\""
        done <<< "$ENV_VARS"
    fi
    
    # 挂载卷
    if [ -n "$VOLUMES" ]; then
        while IFS= read -r volume; do
            [ -z "$volume" ] && continue
            cmd="$cmd -v $volume"
        done <<< "$VOLUMES"
    fi
    
    # 命名卷
    if [ -n "$NAMED_VOLUMES" ]; then
        while IFS= read -r volume; do
            [ -z "$volume" ] && continue
            cmd="$cmd -v $volume"
        done <<< "$NAMED_VOLUMES"
    fi
    
    # Docker socket
    if [ "$MOUNT_DOCKER_SOCKET" = "true" ]; then
        cmd="$cmd -v /var/run/docker.sock:/var/run/docker.sock"
    fi
    
    # 健康检查
    if [ "$HEALTH_CHECK_ENABLED" = "true" ] && [ -n "$HEALTH_CHECK_CMD" ]; then
        cmd="$cmd --health-cmd=\"$HEALTH_CHECK_CMD\""
        cmd="$cmd --health-interval=$HEALTH_CHECK_INTERVAL"
        cmd="$cmd --health-timeout=$HEALTH_CHECK_TIMEOUT"
        cmd="$cmd --health-retries=$HEALTH_CHECK_RETRIES"
    fi
    
    # 资源限制
    if [ -n "$CPU_LIMIT" ]; then
        cmd="$cmd --cpus $CPU_LIMIT"
    fi
    
    if [ -n "$MEMORY_LIMIT" ]; then
        cmd="$cmd --memory $MEMORY_LIMIT"
    fi
    
    # 用户
    if [ -n "$RUN_AS_USER" ]; then
        cmd="$cmd --user $RUN_AS_USER"
    fi
    
    # 工作目录
    if [ -n "$WORKDIR" ]; then
        cmd="$cmd --workdir $WORKDIR"
    fi
    
    # 日志配置
    cmd="$cmd --log-driver $LOG_DRIVER"
    cmd="$cmd --log-opt max-size=$LOG_MAX_SIZE"
    cmd="$cmd --log-opt max-file=$LOG_MAX_FILE"
    
    # 额外参数
    if [ -n "$EXTRA_DOCKER_ARGS" ]; then
        cmd="$cmd $EXTRA_DOCKER_ARGS"
    fi
    
    # 镜像名称
    cmd="$cmd $IMAGE_NAME"
    
    echo "$cmd"
}

###############################################################################
# 容器操作
###############################################################################

# 启动容器
start_container() {
    log_step "启动容器: $CONTAINER_NAME"
    
    # 检查容器是否已存在
    if docker ps -a --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
        if docker ps --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
            log_warning "容器已在运行中"
            show_status
            return 0
        else
            log_info "启动已存在的容器..."
            docker start "$CONTAINER_NAME"
            log_success "容器启动成功"
            show_status
            return 0
        fi
    fi
    
    # 检查镜像
    check_image
    
    # 创建必要的目录
    create_directories
    
    # 构建并执行命令
    local docker_cmd
    docker_cmd=$(build_docker_command)
    
    log_info "执行命令:"
    echo -e "${CYAN}$docker_cmd${NC}"
    echo ""
    
    # 执行命令
    if eval "$docker_cmd"; then
        log_success "容器创建并启动成功"
        echo ""
        show_status
        return 0
    else
        log_error "容器启动失败"
        return 1
    fi
}

# 停止容器
stop_container() {
    log_step "停止容器: $CONTAINER_NAME"
    
    if ! docker ps --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
        log_warning "容器未运行"
        return 0
    fi
    
    docker stop "$CONTAINER_NAME"
    log_success "容器已停止"
}

# 重启容器
restart_container() {
    log_step "重启容器: $CONTAINER_NAME"
    
    if docker ps -a --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
        docker restart "$CONTAINER_NAME"
        log_success "容器已重启"
        show_status
    else
        log_error "容器不存在"
        return 1
    fi
}

# 删除容器
remove_container() {
    log_step "删除容器: $CONTAINER_NAME"
    
    if ! docker ps -a --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
        log_warning "容器不存在"
        return 0
    fi
    
    # 停止容器（如果正在运行）
    if docker ps --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
        log_info "停止容器..."
        docker stop "$CONTAINER_NAME"
    fi
    
    docker rm "$CONTAINER_NAME"
    log_success "容器已删除"
}

# 删除镜像
remove_image() {
    log_info "删除 Docker 镜像:${IMAGE_NAME}"

    # 先删除容器
    if docker ps -a --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
        docker stop "${CONTAINER_NAME}"
        docker rm "${CONTAINER_NAME}"
    fi

    docker rmi "${IMAGE_NAME}"
    log_success "镜像已删除"
}

# 查看状态
show_status() {
    print_separator
    echo -e "${CYAN}容器状态${NC}"
    print_separator
    
    if docker ps -a --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
        docker ps -a --filter "name=^${CONTAINER_NAME}$" \
            --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
    else
        log_warning "容器不存在"
    fi
    
    print_separator
}

# 查看日志
show_logs() {
    local follow="${1:-false}"
    local lines="${2:-100}"
    
    if ! docker ps -a --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
        log_error "容器不存在"
        return 1
    fi
    
    if [ "$follow" = "true" ] || [ "$follow" = "-f" ]; then
        log_info "实时查看日志（Ctrl+C 退出）..."
        docker logs -f --tail "$lines" "$CONTAINER_NAME"
    else
        docker logs --tail "$lines" "$CONTAINER_NAME"
    fi
}

# 进入容器
exec_shell() {
    local shell="${1:-bash}"
    
    if ! docker ps --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
        log_error "容器未运行"
        return 1
    fi
    
    log_info "进入容器 (shell: $shell)..."
    docker exec -it "$CONTAINER_NAME" "$shell" || \
    docker exec -it "$CONTAINER_NAME" sh
}

# 创建必要的目录
create_directories() {
    if [ -n "$VOLUMES" ]; then
        while IFS= read -r volume; do
            [ -z "$volume" ] && continue
            
            # 提取主机路径
            local host_path
            host_path=$(echo "$volume" | cut -d':' -f1)
            
            # 如果是绝对路径且不存在，则创建
            if [[ "$host_path" == /* ]] && [ ! -e "$host_path" ]; then
                log_info "创建目录: $host_path"
                mkdir -p "$host_path"
            fi
        done <<< "$VOLUMES"
    fi
}

###############################################################################
# 配置生成
###############################################################################

# 交互式配置
interactive_config() {
    print_separator
    echo -e "${MAGENTA}Docker 容器交互式配置${NC}"
    print_separator
    echo ""
    
    # 基本配置
    read -p "容器名称 [$CONTAINER_NAME]: " input
    CONTAINER_NAME="${input:-$CONTAINER_NAME}"
    
    read -p "镜像名称 [$IMAGE_NAME]: " input
    IMAGE_NAME="${input:-$IMAGE_NAME}"
    
    read -p "重启策略 (no/always/unless-stopped/on-failure) [$RESTART_POLICY]: " input
    RESTART_POLICY="${input:-$RESTART_POLICY}"
    
    # 网络配置
    echo ""
    log_info "网络配置"
    read -p "网络模式 (bridge/host/none) [$NETWORK_MODE]: " input
    NETWORK_MODE="${input:-$NETWORK_MODE}"
    
    read -p "自定义网络名称 (可选): " input
    NETWORK_NAME="${input:-$NETWORK_NAME}"
    
    # 端口映射
    echo ""
    log_info "端口映射（格式: host_port:container_port，多个用空格分隔）"
    read -p "端口映射: " input
    PORTS="${input:-$PORTS}"
    
    # 环境变量
    echo ""
    log_info "环境变量（输入 'done' 完成）"
    ENV_VARS=""
    while true; do
        read -p "环境变量 (KEY=VALUE): " input
        [ "$input" = "done" ] && break
        [ -n "$input" ] && ENV_VARS="${ENV_VARS}${input}"$'\n'
    done
    
    # 挂载卷
    echo ""
    log_info "挂载卷（输入 'done' 完成）"
    VOLUMES=""
    while true; do
        read -p "挂载卷 (host_path:container_path): " input
        [ "$input" = "done" ] && break
        [ -n "$input" ] && VOLUMES="${VOLUMES}${input}"$'\n'
    done
    
    # Docker socket
    echo ""
    read -p "挂载 Docker socket? (y/N): " input
    if [[ "$input" =~ ^[Yy]$ ]]; then
        MOUNT_DOCKER_SOCKET="true"
    fi
    
    # 生成配置文件
    echo ""
    read -p "保存配置到文件? (y/N): " input
    if [[ "$input" =~ ^[Yy]$ ]]; then
        read -p "配置文件名 [$DEFAULT_CONFIG_FILE]: " input
        local config_file="${input:-$DEFAULT_CONFIG_FILE}"
        generate_config_file "$config_file"
    fi
    
    echo ""
    read -p "现在启动容器? (Y/n): " input
    if [[ ! "$input" =~ ^[Nn]$ ]]; then
        start_container
    fi
}

# 生成配置文件
generate_config_file() {
    local config_file="${1:-$DEFAULT_CONFIG_FILE}"
    
    cat > "$config_file" << EOF
# Docker 容器运行配置
# 生成时间: $(date)

# 基本配置
CONTAINER_NAME="$CONTAINER_NAME"
IMAGE_NAME="$IMAGE_NAME"
RESTART_POLICY="$RESTART_POLICY"

# 网络配置
NETWORK_MODE="$NETWORK_MODE"
NETWORK_NAME="$NETWORK_NAME"

# 端口映射
PORTS="$PORTS"

# 环境变量
ENV_VARS="$(echo "$ENV_VARS" | sed 's/^/"/;s/$/"/' | tr '\n' ' ')"

# 挂载卷
VOLUMES="$(echo "$VOLUMES" | sed 's/^/"/;s/$/"/' | tr '\n' ' ')"

# Docker socket
MOUNT_DOCKER_SOCKET="$MOUNT_DOCKER_SOCKET"

# 资源限制
# CPU_LIMIT="1.0"
# MEMORY_LIMIT="512m"

# 健康检查
# HEALTH_CHECK_ENABLED="true"
# HEALTH_CHECK_CMD="curl -f http://localhost/ || exit 1"
# HEALTH_CHECK_INTERVAL="30s"

# 日志配置
LOG_DRIVER="$LOG_DRIVER"
LOG_MAX_SIZE="$LOG_MAX_SIZE"
LOG_MAX_FILE="$LOG_MAX_FILE"
EOF

    log_success "配置已保存到: $config_file"
}

# 生成示例配置
generate_example_config() {
    local example_file="${1:-docker-run.example.conf}"
    
    cat > "$example_file" << 'EOF'
# Docker 容器运行配置示例

# ==================== 基本配置 ====================
CONTAINER_NAME="myapp"
IMAGE_NAME="nginx:latest"
CONTAINER_HOSTNAME="myapp.local"
RESTART_POLICY="unless-stopped"

# ==================== 网络配置 ====================
# 网络模式: bridge, host, none
NETWORK_MODE="bridge"

# 自定义网络（如果使用）
NETWORK_NAME="my-network"

# ==================== 端口映射 ====================
# 格式: host_port:container_port
# 多个端口用空格分隔
PORTS="8080:80 8443:443"

# ==================== 环境变量 ====================
# 每行一个环境变量
ENV_VARS="
DATABASE_HOST=mysql
DATABASE_PORT=3306
DATABASE_NAME=myapp
DATABASE_USER=root
DATABASE_PASSWORD=password123
APP_ENV=production
TZ=Asia/Shanghai
"

# ==================== 挂载卷 ====================
# 格式: host_path:container_path[:options]
# options: ro (只读), rw (读写，默认)
VOLUMES="
/data/myapp:/app/data
/logs/myapp:/app/logs
/config/myapp:/app/config:ro
"

# 命名卷
NAMED_VOLUMES="
myapp-data:/app/data
myapp-cache:/app/cache
"

# ==================== Docker Socket ====================
# 如果容器需要管理其他容器（如 Dockerun, Portainer）
MOUNT_DOCKER_SOCKET="false"

# ==================== 资源限制 ====================
# CPU 限制（核心数）
CPU_LIMIT="2.0"

# 内存限制
MEMORY_LIMIT="1g"

# ==================== 健康检查 ====================
HEALTH_CHECK_ENABLED="true"
HEALTH_CHECK_CMD="curl -f http://localhost/health || exit 1"
HEALTH_CHECK_INTERVAL="30s"
HEALTH_CHECK_TIMEOUT="10s"
HEALTH_CHECK_RETRIES="3"

# ==================== 用户配置 ====================
# 以特定用户运行容器
# RUN_AS_USER="1000:1000"

# ==================== 工作目录 ====================
# WORKDIR="/app"

# ==================== 日志配置 ====================
LOG_DRIVER="json-file"
LOG_MAX_SIZE="10m"
LOG_MAX_FILE="3"

# ==================== 额外参数 ====================
# Docker run 的额外参数
# EXTRA_DOCKER_ARGS="--privileged --cap-add=SYS_ADMIN"
EOF

    log_success "示例配置已生成: $example_file"
    echo ""
    echo "编辑配置文件: vi $example_file"
    echo "使用配置文件: ./$(basename "$0") -c $example_file start"
}

###############################################################################
# 帮助信息
###############################################################################

show_help() {
    cat << EOF
${CYAN}Docker 容器运行模板脚本${NC}

${YELLOW}用法:${NC}
  $(basename "$0") [选项] <命令>

${YELLOW}选项:${NC}
  -c, --config FILE    使用指定的配置文件
  -h, --help           显示帮助信息

${YELLOW}命令:${NC}
  start                启动容器
  stop                 停止容器
  restart              重启容器
  remove, rm           删除容器
  remove-image, rmi    删除镜像
  status               查看容器状态
  logs [lines]         查看日志（默认最后 100 行）
  logs -f [lines]      实时查看日志
  exec [shell]         进入容器（默认 bash）
  interactive, config  交互式配置
  generate-config      生成示例配置文件
  help                 显示帮助信息

${YELLOW}示例:${NC}
  # 交互式配置并启动
  $(basename "$0") interactive

  # 生成示例配置文件
  $(basename "$0") generate-config

  # 使用配置文件启动
  $(basename "$0") -c myapp.conf start

  # 查看实时日志
  $(basename "$0") logs -f

  # 进入容器
  $(basename "$0") exec

  # 查看状态
  $(basename "$0") status

${YELLOW}配置文件格式:${NC}
  配置文件是 shell 脚本格式，定义变量即可。
  运行 'generate-config' 命令查看完整示例。

${YELLOW}环境变量:${NC}
  所有配置项都可以通过环境变量设置，例如:
  
  CONTAINER_NAME=myapp IMAGE_NAME=nginx:latest \\
  PORTS="8080:80" $(basename "$0") start

EOF
}

###############################################################################
# 主程序
###############################################################################

main() {
    # 解析选项
    while [[ $# -gt 0 ]]; do
        case $1 in
            -c|--config)
                CONFIG_FILE="$2"
                shift 2
                ;;
            -h|--help)
                show_help
                exit 0
                ;;
            -*)
                log_error "未知选项: $1"
                show_help
                exit 1
                ;;
            *)
                break
                ;;
        esac
    done
    
    # 检查 Docker
    check_docker
    
    # 加载配置
    load_config
    
    # 获取命令
    local command="${1:-help}"
    shift || true
    
    # 执行命令
    case "$command" in
        start)
            validate_config || exit 1
            start_container
            ;;
        stop)
            stop_container
            ;;
        restart)
            restart_container
            ;;
        remove|rm)
            remove_container
            ;;
        remove-image|rmi)
            remove_image
            ;;
        status)
            show_status
            ;;
        logs)
            show_logs "$@"
            ;;
        exec)
            exec_shell "$@"
            ;;
        interactive|config)
            interactive_config
            ;;
        generate-config|gen-config|example)
            generate_example_config "$@"
            ;;
        help|--help|-h)
            show_help
            ;;
        *)
            log_error "未知命令: $command"
            echo ""
            show_help
            exit 1
            ;;
    esac
}

# 运行主程序
main "$@"

