# 为 Docker Registry 配置自签名 HTTPS 证书

## 问题说明

使用旧方法生成的自签名证书会报错：

```
tls: failed to verify certificate: x509: certificate relies on legacy Common Name field, use SANs instead
```

**原因**: Go 1.15+ 要求证书必须包含 SAN (Subject Alternative Name) 扩展字段。

---

## 方案 1: 生成包含 SAN 的自签名证书（推荐）⭐

### 步骤 1: 创建证书配置文件

创建 `openssl.cnf` 文件：

```bash
cat > openssl.cnf << 'EOF'
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
O  = My Company
OU = IT Department
CN = 192.168.0.170

[v3_req]
keyUsage = keyEncipherment, dataEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names

[alt_names]
DNS.1 = registry.local
DNS.2 = localhost
IP.1  = 192.168.0.170
IP.2  = 127.0.0.1
EOF
```

**重要**: 
- 修改 `CN = 192.168.0.170` 为你的实际 IP
- 在 `[alt_names]` 中添加所有可能访问的地址（IP 和域名）

### 步骤 2: 生成证书和私钥

```bash
# 生成证书和私钥（一步完成）
openssl req -x509 -nodes -days 365 \
  -newkey rsa:2048 \
  -keyout registry.key \
  -out registry.crt \
  -config openssl.cnf \
  -extensions v3_req

# 查看生成的证书（验证 SAN）
openssl x509 -in registry.crt -text -noout | grep -A 1 "Subject Alternative Name"
```

应该看到：

```
X509v3 Subject Alternative Name:
    DNS:registry.local, DNS:localhost, IP Address:192.168.0.170, IP Address:127.0.0.1
```

### 步骤 3: 在所有客户端机器上信任证书

**需要在以下机器上执行**：
- ✅ 运行 Dockerun 的服务器
- ✅ 需要 push/pull 镜像的开发机
- ✅ CI/CD 服务器

#### Linux 系统

```bash
# 复制证书到系统信任目录
sudo mkdir -p /etc/docker/certs.d/192.168.0.170:5000
sudo cp registry.crt /etc/docker/certs.d/192.168.0.170:5000/ca.crt

# 同时添加到系统证书库（可选，推荐）
sudo cp registry.crt /usr/local/share/ca-certificates/192.168.0.170.crt
sudo update-ca-certificates

# 重启 Docker
sudo systemctl restart docker
```

#### macOS 系统

```bash
# 创建 Docker 证书目录
mkdir -p ~/.docker/certs.d/192.168.0.170:5000
cp registry.crt ~/.docker/certs.d/192.168.0.170:5000/ca.crt

# 添加到系统钥匙串（可选）
sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain registry.crt

# 重启 Docker Desktop
# 通过界面重启或：
osascript -e 'quit app "Docker"'
open -a Docker
```

#### Windows 系统

```powershell
# 创建证书目录
New-Item -Path "$env:USERPROFILE\.docker\certs.d\192.168.0.170:5000" -ItemType Directory -Force

# 复制证书
Copy-Item registry.crt "$env:USERPROFILE\.docker\certs.d\192.168.0.170:5000\ca.crt"

# 导入证书到 Windows 证书存储（以管理员身份运行）
Import-Certificate -FilePath "registry.crt" -CertStoreLocation Cert:\LocalMachine\Root

# 重启 Docker Desktop
```

### 步骤 4: 配置 Registry 使用证书

#### 使用 Docker Compose

```yaml
version: '3.8'

services:
  registry:
    image: registry:2
    container_name: docker-registry
    restart: always
    ports:
      - "5000:5000"
    environment:
      REGISTRY_HTTP_TLS_CERTIFICATE: /certs/registry.crt
      REGISTRY_HTTP_TLS_KEY: /certs/registry.key
    volumes:
      - ./registry.crt:/certs/registry.crt:ro
      - ./registry.key:/certs/registry.key:ro
      - registry-data:/var/lib/registry

volumes:
  registry-data:
```

启动：

```bash
docker-compose up -d
```

#### 使用 Docker Run

```bash
docker run -d \
  --name docker-registry \
  --restart=always \
  -p 5000:5000 \
  -v $(pwd)/registry.crt:/certs/registry.crt:ro \
  -v $(pwd)/registry.key:/certs/registry.key:ro \
  -v registry-data:/var/lib/registry \
  -e REGISTRY_HTTP_TLS_CERTIFICATE=/certs/registry.crt \
  -e REGISTRY_HTTP_TLS_KEY=/certs/registry.key \
  registry:2
```

### 步骤 5: 测试配置

```bash
# 测试 HTTPS 连接
curl https://192.168.0.170:5000/v2/_catalog

# 测试 Docker 登录
docker login 192.168.0.170:5000

# 测试推送
docker tag alpine:latest 192.168.0.170:5000/test/alpine:latest
docker push 192.168.0.170:5000/test/alpine:latest

# 测试拉取
docker pull 192.168.0.170:5000/test/alpine:latest
```

如果都成功，说明证书配置正确！✅

---

## 方案 2: 一键配置脚本

创建自动化配置脚本 `setup-registry-https.sh`：

```bash
#!/bin/bash

set -e

REGISTRY_IP="${1:-192.168.0.170}"
REGISTRY_PORT="${2:-5000}"
DAYS="${3:-365}"

echo "=== Docker Registry HTTPS 证书配置工具 ==="
echo "Registry: $REGISTRY_IP:$REGISTRY_PORT"
echo "有效期: $DAYS 天"
echo ""

# 创建工作目录
mkdir -p registry-certs
cd registry-certs

# 生成配置文件
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
OU = IT
CN = $REGISTRY_IP

[v3_req]
keyUsage = keyEncipherment, dataEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names

[alt_names]
DNS.1 = registry.local
DNS.2 = localhost
IP.1  = $REGISTRY_IP
IP.2  = 127.0.0.1
EOF

echo "✓ 生成配置文件"

# 生成证书
openssl req -x509 -nodes -days $DAYS \
  -newkey rsa:2048 \
  -keyout registry.key \
  -out registry.crt \
  -config openssl.cnf \
  -extensions v3_req

echo "✓ 生成证书和私钥"

# 验证证书
echo ""
echo "=== 证书信息 ==="
openssl x509 -in registry.crt -text -noout | grep -A 3 "Subject Alternative Name"

# 配置 Docker 信任证书
echo ""
echo "=== 配置 Docker 信任证书 ==="

if [[ "$OSTYPE" == "linux-gnu"* ]]; then
    echo "检测到 Linux 系统"
    
    # Docker 证书目录
    sudo mkdir -p /etc/docker/certs.d/$REGISTRY_IP:$REGISTRY_PORT
    sudo cp registry.crt /etc/docker/certs.d/$REGISTRY_IP:$REGISTRY_PORT/ca.crt
    echo "✓ 已配置 Docker 证书目录"
    
    # 系统证书库
    sudo cp registry.crt /usr/local/share/ca-certificates/$REGISTRY_IP.crt
    sudo update-ca-certificates
    echo "✓ 已添加到系统证书库"
    
    # 重启 Docker
    sudo systemctl restart docker
    echo "✓ 已重启 Docker 服务"
    
elif [[ "$OSTYPE" == "darwin"* ]]; then
    echo "检测到 macOS 系统"
    
    mkdir -p ~/.docker/certs.d/$REGISTRY_IP:$REGISTRY_PORT
    cp registry.crt ~/.docker/certs.d/$REGISTRY_IP:$REGISTRY_PORT/ca.crt
    echo "✓ 已配置 Docker 证书目录"
    
    echo "请重启 Docker Desktop"
fi

echo ""
echo "=== 配置完成 ==="
echo ""
echo "证书文件位置："
echo "  - 证书: $(pwd)/registry.crt"
echo "  - 私钥: $(pwd)/registry.key"
echo ""
echo "下一步："
echo "1. 使用这些证书配置你的 Registry"
echo "2. 测试连接: curl https://$REGISTRY_IP:$REGISTRY_PORT/v2/_catalog"
echo "3. 在其他机器上也安装此证书"
echo ""
```

使用方法：

```bash
chmod +x setup-registry-https.sh

# 默认配置
./setup-registry-https.sh

# 自定义 IP
./setup-registry-https.sh 192.168.1.100

# 自定义 IP 和端口
./setup-registry-https.sh 192.168.1.100 5000

# 自定义有效期（天数）
./setup-registry-https.sh 192.168.0.170 5000 730
```

---

## 方案 3: 继续使用 HTTP（最简单）

如果只是内网测试环境，建议继续使用 HTTP + insecure-registries 配置：

```bash
# 已经在前面配置过了
sudo ./scripts/setup-insecure-registry.sh 192.168.0.170:5000
```

**优点**：
- ✅ 配置简单
- ✅ 无需管理证书
- ✅ 适合开发和测试环境

**缺点**：
- ❌ 不加密传输
- ❌ 不适合生产环境

---

## 方案对比

| 方案 | 安全性 | 配置难度 | 适用场景 |
|------|--------|----------|----------|
| HTTP + insecure | ⭐ | ⭐ | 内网开发/测试 |
| 自签名 HTTPS | ⭐⭐⭐ | ⭐⭐⭐ | 内网生产环境 |
| CA 签名 HTTPS | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ | 公网生产环境 |

---

## 在 Dockerun 中配置 HTTPS Registry

### Registry 配置

```json
{
  "name": "Secure Registry",
  "url": "https://192.168.0.170:5000",  ← 使用 https://
  "username": "admin",                   ← 如果配置了认证
  "password": "password",
  "insecure": false                      ← 改为 false
}
```

### Webhook 配置

保持不变：
- **Image Name**: `indohit/gateway`
- **Image Tag**: `0.1.0`
- **Registry**: 选择上面的 HTTPS Registry

---

## 故障排查

### 问题 1: 证书验证失败

```bash
# 检查证书 SAN
openssl x509 -in registry.crt -text -noout | grep -A 5 "Subject Alternative Name"

# 确认包含你的 IP 或域名
```

### 问题 2: Docker 仍然不信任证书

```bash
# 检查证书位置
ls -la /etc/docker/certs.d/192.168.0.170:5000/

# 应该有 ca.crt 文件
# 权限应该是可读的

# 重启 Docker
sudo systemctl restart docker
```

### 问题 3: 多台机器配置

**记住**: 每台需要访问 Registry 的机器都要安装证书！

```bash
# 方法 1: 手动复制证书
scp registry.crt user@other-machine:/tmp/
ssh user@other-machine "sudo mkdir -p /etc/docker/certs.d/192.168.0.170:5000 && \
  sudo cp /tmp/registry.crt /etc/docker/certs.d/192.168.0.170:5000/ca.crt && \
  sudo systemctl restart docker"

# 方法 2: 使用配置管理工具（Ansible 示例）
# ansible-playbook deploy-registry-cert.yml
```

---

## 添加认证（可选）

### 生成密码文件

```bash
mkdir auth
docker run --rm --entrypoint htpasswd \
  registry:2 -Bbn admin password123 > auth/htpasswd
```

### Docker Compose 配置

```yaml
version: '3.8'

services:
  registry:
    image: registry:2
    restart: always
    ports:
      - "5000:5000"
    environment:
      # HTTPS 配置
      REGISTRY_HTTP_TLS_CERTIFICATE: /certs/registry.crt
      REGISTRY_HTTP_TLS_KEY: /certs/registry.key
      # 认证配置
      REGISTRY_AUTH: htpasswd
      REGISTRY_AUTH_HTPASSWD_REALM: Registry Realm
      REGISTRY_AUTH_HTPASSWD_PATH: /auth/htpasswd
    volumes:
      - ./registry.crt:/certs/registry.crt:ro
      - ./registry.key:/certs/registry.key:ro
      - ./auth:/auth:ro
      - registry-data:/var/lib/registry

volumes:
  registry-data:
```

### 使用认证

```bash
# 登录
docker login 192.168.0.170:5000
Username: admin
Password: password123

# 现在可以正常 push/pull
docker push 192.168.0.170:5000/myimage:latest
```

---

## 推荐配置

### 开发环境

使用 **HTTP + insecure-registries**（最简单）

```bash
sudo ./scripts/setup-insecure-registry.sh 192.168.0.170:5000
```

### 生产环境

使用 **HTTPS + 自签名证书 + 认证**

```bash
# 1. 生成证书
./setup-registry-https.sh 192.168.0.170

# 2. 配置 Registry（HTTPS + 认证）
docker-compose up -d

# 3. 在所有客户端安装证书
# 4. 在 Dockerun 中配置 Registry（HTTPS）
```

---

## 需要帮助？

- 📖 [INSECURE_REGISTRY.md](./INSECURE_REGISTRY.md) - HTTP Registry 配置
- 📖 [QUICK_FIX_INSECURE_REGISTRY.md](./QUICK_FIX_INSECURE_REGISTRY.md) - 快速修复指南
- 📖 [REGISTRY_GUIDE.md](./REGISTRY_GUIDE.md) - Registry 管理指南

