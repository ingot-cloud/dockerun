# 配置 Docker 使用不安全的私有仓库（Insecure Registry）

## 问题描述

当使用 HTTP 协议的私有 Docker Registry 时，可能会遇到以下错误：

```
Error response from daemon: Get "https://192.168.0.170:5000/v2/": http: server gave HTTP response to HTTPS client
```

这是因为 Docker 默认只允许通过 HTTPS 连接到 Registry。

## 解决方案

### Linux 系统

#### 1. 编辑 Docker daemon 配置文件

```bash
sudo vim /etc/docker/daemon.json
```

#### 2. 添加 insecure-registries 配置

```json
{
  "insecure-registries": ["192.168.0.170:5000"]
}
```

如果文件已存在其他配置，只需添加 `insecure-registries` 字段：

```json
{
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "10m",
    "max-file": "3"
  },
  "insecure-registries": ["192.168.0.170:5000"]
}
```

**支持通配符和多个仓库**：

```json
{
  "insecure-registries": [
    "192.168.0.170:5000",
    "192.168.0.171:5000",
    "192.168.0.0/24:5000"
  ]
}
```

#### 3. 重启 Docker 服务

```bash
# systemd 系统（Ubuntu 16.04+, CentOS 7+, Debian 8+）
sudo systemctl daemon-reload
sudo systemctl restart docker

# 或者使用 service 命令
sudo service docker restart
```

#### 4. 验证配置

```bash
# 查看 Docker 配置
docker info | grep -A 5 "Insecure Registries"

# 测试拉取镜像
docker pull 192.168.0.170:5000/indohit/gateway:0.1.0
```

---

### macOS 系统（Docker Desktop）

#### 1. 打开 Docker Desktop

点击菜单栏的 Docker 图标

#### 2. 进入设置

**Preferences** → **Docker Engine**

#### 3. 编辑配置

在 JSON 配置中添加：

```json
{
  "builder": {
    "gc": {
      "defaultKeepStorage": "20GB",
      "enabled": true
    }
  },
  "experimental": false,
  "insecure-registries": [
    "192.168.0.170:5000"
  ]
}
```

#### 4. 应用并重启

点击 **Apply & Restart** 按钮

---

### Windows 系统（Docker Desktop）

#### 1. 打开 Docker Desktop

点击系统托盘的 Docker 图标

#### 2. 进入设置

**Settings** → **Docker Engine**

#### 3. 编辑配置

添加 insecure-registries 配置（同 macOS）

#### 4. 应用并重启

点击 **Apply & Restart**

---

## 使用 Docker Compose 的情况

如果 Dockerun 是通过 Docker Compose 运行的，需要在**宿主机**上配置 insecure-registries，而不是在容器内。

### 示例 docker-compose.yml

```yaml
version: '3.8'

services:
  # 私有仓库
  registry:
    image: registry:2
    ports:
      - "5000:5000"
    volumes:
      - registry-data:/var/lib/registry

  # Dockerun 应用
  dockerun:
    image: dockerun:latest
    ports:
      - "8080:8080"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock  # 挂载宿主机 Docker
    environment:
      - GIN_MODE=release
    depends_on:
      - registry

volumes:
  registry-data:
```

**重要**: 需要在运行 Docker Compose 的宿主机上配置 insecure-registries。

---

## 快速配置脚本

### Linux 一键配置

创建配置脚本 `setup-insecure-registry.sh`:

```bash
#!/bin/bash

REGISTRY_URL="$1"

if [ -z "$REGISTRY_URL" ]; then
    echo "Usage: $0 <registry-url>"
    echo "Example: $0 192.168.0.170:5000"
    exit 1
fi

echo "Configuring Docker to allow insecure registry: $REGISTRY_URL"

# 备份原配置
if [ -f /etc/docker/daemon.json ]; then
    sudo cp /etc/docker/daemon.json /etc/docker/daemon.json.backup.$(date +%s)
fi

# 创建或更新配置
if [ -f /etc/docker/daemon.json ]; then
    # 文件已存在，使用 jq 添加配置
    if command -v jq &> /dev/null; then
        sudo jq ". + {\"insecure-registries\": [\"$REGISTRY_URL\"] | unique}" /etc/docker/daemon.json > /tmp/daemon.json.tmp
        sudo mv /tmp/daemon.json.tmp /etc/docker/daemon.json
    else
        echo "jq not found. Please manually edit /etc/docker/daemon.json"
        echo "Add: \"insecure-registries\": [\"$REGISTRY_URL\"]"
        exit 1
    fi
else
    # 创建新文件
    echo "{\"insecure-registries\": [\"$REGISTRY_URL\"]}" | sudo tee /etc/docker/daemon.json
fi

# 重启 Docker
echo "Restarting Docker..."
sudo systemctl daemon-reload
sudo systemctl restart docker

# 验证
echo ""
echo "Configuration complete. Insecure registries:"
docker info | grep -A 5 "Insecure Registries"

echo ""
echo "You can now use: docker pull $REGISTRY_URL/image:tag"
```

使用方法：

```bash
chmod +x setup-insecure-registry.sh
sudo ./setup-insecure-registry.sh 192.168.0.170:5000
```

---

## 安全建议

### 1. 仅在内网使用

不安全的 Registry 应该只在内网环境中使用，**不要暴露到公网**。

### 2. 使用 HTTPS（推荐）

在生产环境，建议为 Registry 配置 HTTPS：

```bash
# 生成自签名证书
openssl req -newkey rsa:4096 -nodes -sha256 -keyout registry.key \
  -x509 -days 365 -out registry.crt \
  -subj "/CN=192.168.0.170"

# 启动带 TLS 的 Registry
docker run -d -p 5000:5000 \
  -v $(pwd)/registry.crt:/certs/registry.crt \
  -v $(pwd)/registry.key:/certs/registry.key \
  -e REGISTRY_HTTP_TLS_CERTIFICATE=/certs/registry.crt \
  -e REGISTRY_HTTP_TLS_KEY=/certs/registry.key \
  registry:2
```

### 3. 添加认证

配置基本认证：

```bash
# 创建密码文件
mkdir auth
docker run --rm --entrypoint htpasswd registry:2 -Bbn testuser testpassword > auth/htpasswd

# 启动带认证的 Registry
docker run -d -p 5000:5000 \
  -v $(pwd)/auth:/auth \
  -e "REGISTRY_AUTH=htpasswd" \
  -e "REGISTRY_AUTH_HTPASSWD_REALM=Registry Realm" \
  -e "REGISTRY_AUTH_HTPASSWD_PATH=/auth/htpasswd" \
  registry:2
```

---

## 故障排查

### 1. 检查 Docker daemon 配置

```bash
# 查看完整配置
cat /etc/docker/daemon.json

# 查看 Docker 信息
docker info

# 查看 insecure-registries 配置
docker info | grep -i insecure
```

### 2. 检查 Docker 服务状态

```bash
sudo systemctl status docker

# 查看日志
sudo journalctl -u docker -f
```

### 3. 测试连接

```bash
# 测试 Registry 是否可访问
curl http://192.168.0.170:5000/v2/_catalog

# 测试登录
docker login 192.168.0.170:5000

# 测试拉取
docker pull 192.168.0.170:5000/indohit/gateway:0.1.0
```

### 4. 防火墙检查

```bash
# 检查端口是否开放
telnet 192.168.0.170 5000

# 或使用 nc
nc -zv 192.168.0.170 5000
```

---

## 常见错误

### 错误 1: daemon.json 格式错误

```
Error response from daemon: error parsing daemon config file: invalid character
```

**解决**: 使用 JSON 验证工具检查语法：

```bash
cat /etc/docker/daemon.json | python -m json.tool
```

### 错误 2: Docker 无法重启

**解决**: 检查配置文件语法，删除无效配置：

```bash
sudo mv /etc/docker/daemon.json /etc/docker/daemon.json.broken
sudo systemctl restart docker
```

---

## 在 Dockerun 中的配置

在 Dockerun 的 Registry 配置中，确保设置了正确的 URL：

```json
{
  "name": "Local Registry",
  "url": "http://192.168.0.170:5000",
  "username": "",
  "password": "",
  "insecure": true
}
```

**注意**: `url` 字段使用 `http://` 前缀，而不是 `https://`。

---

## 参考资料

- [Docker Official: Insecure Registry](https://docs.docker.com/registry/insecure/)
- [Docker Official: Registry Configuration](https://docs.docker.com/registry/configuration/)

