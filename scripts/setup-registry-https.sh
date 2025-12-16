#!/bin/bash

# Docker Registry HTTPS 证书自动配置工具
# 生成包含 SAN 的自签名证书并配置 Docker 信任

set -e

REGISTRY_IP="${1:-192.168.0.170}"
REGISTRY_PORT="${2:-5000}"
DAYS="${3:-365}"
WORK_DIR="registry-certs-$(date +%Y%m%d-%H%M%S)"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_error() {
    echo -e "${RED}[错误]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[成功]${NC} $1"
}

print_info() {
    echo -e "${BLUE}[信息]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[警告]${NC} $1"
}

print_header() {
    echo ""
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}========================================${NC}"
}

# 检查必要的命令
check_requirements() {
    local missing=()
    
    if ! command -v openssl &> /dev/null; then
        missing+=("openssl")
    fi
    
    if [ ${#missing[@]} -ne 0 ]; then
        print_error "缺少必要的命令: ${missing[*]}"
        print_info "请安装后重试"
        exit 1
    fi
}

# 验证 IP 地址格式
validate_ip() {
    local ip=$1
    if [[ $ip =~ ^[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}$ ]]; then
        return 0
    else
        return 1
    fi
}

print_header "Docker Registry HTTPS 证书配置工具"
echo "Registry 地址: $REGISTRY_IP:$REGISTRY_PORT"
echo "证书有效期: $DAYS 天"
echo "工作目录: $WORK_DIR"
echo ""

# 验证 IP
if ! validate_ip "$REGISTRY_IP"; then
    print_warning "提供的地址看起来不是标准的 IPv4 地址"
    read -p "继续？(y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

# 检查依赖
check_requirements

# 创建工作目录
mkdir -p "$WORK_DIR"
cd "$WORK_DIR"
print_success "创建工作目录: $(pwd)"

# 生成 OpenSSL 配置文件
print_info "生成 OpenSSL 配置文件..."
cat > openssl.cnf << EOF
[req]
default_bits       = 2048
default_md         = sha256
distinguished_name = req_distinguished_name
x509_extensions    = v3_req
prompt             = no

[req_distinguished_name]
C  = CN
ST = Beijing
L  = Beijing
O  = Docker Registry
OU = IT Department
CN = $REGISTRY_IP

[v3_req]
keyUsage = critical, keyEncipherment, dataEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names

[alt_names]
DNS.1 = registry.local
DNS.2 = localhost
IP.1  = $REGISTRY_IP
IP.2  = 127.0.0.1
EOF

print_success "OpenSSL 配置文件已创建"

# 生成证书和私钥
print_info "生成自签名证书和私钥..."
openssl req -x509 -nodes -days "$DAYS" \
  -newkey rsa:2048 \
  -keyout registry.key \
  -out registry.crt \
  -config openssl.cnf \
  -extensions v3_req \
  2>/dev/null

if [ $? -eq 0 ]; then
    print_success "证书和私钥生成成功"
else
    print_error "证书生成失败"
    exit 1
fi

# 验证证书内容
print_header "证书信息"
echo "证书主题:"
openssl x509 -in registry.crt -noout -subject

echo ""
echo "证书有效期:"
openssl x509 -in registry.crt -noout -dates

echo ""
echo "Subject Alternative Names (SAN):"
openssl x509 -in registry.crt -text -noout | grep -A 3 "Subject Alternative Name" || print_warning "未找到 SAN 信息"

# 检测操作系统并配置
print_header "配置 Docker 信任证书"

DOCKER_CERT_DIR=""

if [[ "$OSTYPE" == "linux-gnu"* ]]; then
    print_info "检测到 Linux 系统"
    
    # Docker 证书目录
    DOCKER_CERT_DIR="/etc/docker/certs.d/$REGISTRY_IP:$REGISTRY_PORT"
    
    if [ "$EUID" -ne 0 ]; then
        print_warning "需要 root 权限来配置系统证书"
        print_info "请使用 sudo 运行此脚本，或手动执行以下命令："
        echo ""
        echo "  sudo mkdir -p $DOCKER_CERT_DIR"
        echo "  sudo cp $(pwd)/registry.crt $DOCKER_CERT_DIR/ca.crt"
        echo "  sudo cp $(pwd)/registry.crt /usr/local/share/ca-certificates/$REGISTRY_IP.crt"
        echo "  sudo update-ca-certificates"
        echo "  sudo systemctl restart docker"
        echo ""
    else
        # 配置 Docker 证书目录
        mkdir -p "$DOCKER_CERT_DIR"
        cp registry.crt "$DOCKER_CERT_DIR/ca.crt"
        print_success "已配置 Docker 证书目录: $DOCKER_CERT_DIR"
        
        # 添加到系统证书库
        if [ -d "/usr/local/share/ca-certificates" ]; then
            cp registry.crt "/usr/local/share/ca-certificates/$REGISTRY_IP.crt"
            update-ca-certificates > /dev/null 2>&1
            print_success "已添加到系统证书库"
        fi
        
        # 重启 Docker
        if command -v systemctl &> /dev/null; then
            systemctl restart docker
            print_success "Docker 服务已重启"
        else
            service docker restart
            print_success "Docker 服务已重启"
        fi
    fi
    
elif [[ "$OSTYPE" == "darwin"* ]]; then
    print_info "检测到 macOS 系统"
    
    # Docker Desktop 证书目录
    DOCKER_CERT_DIR="$HOME/.docker/certs.d/$REGISTRY_IP:$REGISTRY_PORT"
    mkdir -p "$DOCKER_CERT_DIR"
    cp registry.crt "$DOCKER_CERT_DIR/ca.crt"
    print_success "已配置 Docker 证书目录: $DOCKER_CERT_DIR"
    
    # 添加到系统钥匙串（需要密码）
    print_info "将证书添加到系统钥匙串..."
    if security add-trusted-cert -d -r trustRoot -k ~/Library/Keychains/login.keychain-db registry.crt 2>/dev/null; then
        print_success "证书已添加到系统钥匙串"
    else
        print_warning "添加到钥匙串失败，可能需要手动添加"
        print_info "你可以双击证书文件 registry.crt 手动导入"
    fi
    
    print_warning "请手动重启 Docker Desktop 使配置生效"
    
else
    print_warning "未识别的操作系统: $OSTYPE"
    print_info "请手动配置证书"
fi

# 生成 Docker Compose 配置示例
print_header "生成配置文件"

cat > docker-compose.yml << EOF
version: '3.8'

services:
  registry:
    image: registry:2
    container_name: docker-registry
    restart: always
    ports:
      - "$REGISTRY_PORT:5000"
    environment:
      # HTTPS 配置
      REGISTRY_HTTP_TLS_CERTIFICATE: /certs/registry.crt
      REGISTRY_HTTP_TLS_KEY: /certs/registry.key
      # 存储配置
      REGISTRY_STORAGE_FILESYSTEM_ROOTDIRECTORY: /var/lib/registry
      # 日志配置
      REGISTRY_LOG_LEVEL: info
    volumes:
      - \${PWD}/registry.crt:/certs/registry.crt:ro
      - \${PWD}/registry.key:/certs/registry.key:ro
      - registry-data:/var/lib/registry
    networks:
      - registry-net

networks:
  registry-net:
    driver: bridge

volumes:
  registry-data:
EOF

print_success "已生成 docker-compose.yml"

# 生成启动脚本
cat > start-registry.sh << 'EOF'
#!/bin/bash
echo "启动 Docker Registry (HTTPS)..."
docker-compose up -d
echo "等待服务启动..."
sleep 3
docker-compose ps
echo ""
echo "Registry 已启动在: https://$REGISTRY_IP:$REGISTRY_PORT"
echo "测试命令: curl https://$REGISTRY_IP:$REGISTRY_PORT/v2/_catalog"
EOF

chmod +x start-registry.sh
print_success "已生成启动脚本: start-registry.sh"

# 生成客户端配置脚本
cat > install-cert-client.sh << EOF
#!/bin/bash
# 在其他客户端机器上运行此脚本来安装证书

set -e

REGISTRY_IP="$REGISTRY_IP"
REGISTRY_PORT="$REGISTRY_PORT"
CERT_FILE="\$(dirname "\$0")/registry.crt"

if [ ! -f "\$CERT_FILE" ]; then
    echo "错误: 找不到证书文件 \$CERT_FILE"
    exit 1
fi

echo "配置客户端信任证书: \$REGISTRY_IP:\$REGISTRY_PORT"

if [[ "\$OSTYPE" == "linux-gnu"* ]]; then
    sudo mkdir -p /etc/docker/certs.d/\$REGISTRY_IP:\$REGISTRY_PORT
    sudo cp "\$CERT_FILE" /etc/docker/certs.d/\$REGISTRY_IP:\$REGISTRY_PORT/ca.crt
    sudo cp "\$CERT_FILE" /usr/local/share/ca-certificates/\$REGISTRY_IP.crt
    sudo update-ca-certificates
    sudo systemctl restart docker
    echo "配置完成！"
elif [[ "\$OSTYPE" == "darwin"* ]]; then
    mkdir -p ~/.docker/certs.d/\$REGISTRY_IP:\$REGISTRY_PORT
    cp "\$CERT_FILE" ~/.docker/certs.d/\$REGISTRY_IP:\$REGISTRY_PORT/ca.crt
    security add-trusted-cert -d -r trustRoot -k ~/Library/Keychains/login.keychain-db "\$CERT_FILE"
    echo "配置完成！请重启 Docker Desktop"
else
    echo "未知操作系统，请手动配置"
fi
EOF

chmod +x install-cert-client.sh
print_success "已生成客户端配置脚本: install-cert-client.sh"

# 最终输出
print_header "配置完成"

echo "📁 生成的文件:"
echo "  - registry.crt       (证书文件)"
echo "  - registry.key       (私钥文件)"
echo "  - openssl.cnf        (OpenSSL 配置)"
echo "  - docker-compose.yml (Registry 配置)"
echo "  - start-registry.sh  (启动脚本)"
echo "  - install-cert-client.sh (客户端配置脚本)"
echo ""

echo "📍 证书位置:"
echo "  $(pwd)/registry.crt"
echo ""

if [ -n "$DOCKER_CERT_DIR" ]; then
    echo "🔒 Docker 证书目录:"
    echo "  $DOCKER_CERT_DIR/ca.crt"
    echo ""
fi

echo "🚀 下一步操作:"
echo ""
echo "1️⃣  启动 Registry (HTTPS):"
echo "    cd $(pwd)"
echo "    ./start-registry.sh"
echo ""
echo "2️⃣  测试连接:"
echo "    curl https://$REGISTRY_IP:$REGISTRY_PORT/v2/_catalog"
echo ""
echo "3️⃣  在其他机器上安装证书:"
echo "    # 复制 registry.crt 和 install-cert-client.sh 到目标机器"
echo "    scp registry.crt install-cert-client.sh user@target-machine:/tmp/"
echo "    # 在目标机器上运行"
echo "    cd /tmp && sudo ./install-cert-client.sh"
echo ""
echo "4️⃣  在 Dockerun 中配置 Registry:"
echo "    URL: https://$REGISTRY_IP:$REGISTRY_PORT"
echo "    Insecure: false (取消勾选)"
echo ""

print_success "所有配置文件已生成在: $(pwd)"

exit 0

