# 快速修复：Docker Registry 连接错误

## 常见错误类型

### 错误 1: HTTP Registry 连接错误

```
Error response from daemon: Get "https://192.168.0.170:5000/v2/": 
http: server gave HTTP response to HTTPS client
```

**问题原因**: Registry 使用 HTTP 协议，但 Docker 默认要求 HTTPS。

### 错误 2: 自签名证书错误

```
tls: failed to verify certificate: x509: certificate relies on legacy Common Name field, use SANs instead
```

**问题原因**: 自签名证书缺少 SAN (Subject Alternative Name) 字段。

---

## 解决方案选择

根据你的需求选择：

| 场景 | 推荐方案 | 安全性 | 难度 |
|------|---------|--------|------|
| 开发/测试环境 | HTTP + insecure | ⭐ | ⭐ |
| 内网生产环境 | HTTPS + 自签名 | ⭐⭐⭐ | ⭐⭐⭐ |
| 公网环境 | HTTPS + CA 证书 | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ |

---

## 解决方案 A: 配置 HTTP Registry（最简单）⚡

### A1：使用自动配置脚本（推荐）

```bash
# 在运行 Dockerun 的服务器上执行
cd /path/to/dockerun
sudo ./scripts/setup-insecure-registry.sh 192.168.0.170:5000
```

脚本会自动：
- ✅ 备份现有配置
- ✅ 添加 insecure-registries 配置
- ✅ 验证 JSON 格式
- ✅ 重启 Docker 服务
- ✅ 验证配置是否生效

---

### A2：手动配置

#### Linux 系统

**1. 编辑 Docker daemon 配置**

```bash
sudo vim /etc/docker/daemon.json
```

**2. 添加以下内容**

如果文件不存在或为空：

```json
{
  "insecure-registries": ["192.168.0.170:5000"]
}
```

如果文件已有内容，只需添加 `insecure-registries` 字段：

```json
{
  "log-driver": "json-file",
  "insecure-registries": ["192.168.0.170:5000"]
}
```

**3. 重启 Docker**

```bash
sudo systemctl daemon-reload
sudo systemctl restart docker
```

**4. 验证配置**

```bash
docker info | grep -A 5 "Insecure Registries"
```

应该看到：

```
Insecure Registries:
  192.168.0.170:5000
```

#### macOS / Windows (Docker Desktop)

1. 打开 Docker Desktop
2. 进入 **Settings** → **Docker Engine**
3. 在 JSON 配置中添加：

```json
{
  "insecure-registries": [
    "192.168.0.170:5000"
  ]
}
```

4. 点击 **Apply & Restart**

---

## 解决方案 B: 配置 HTTPS Registry（推荐生产环境）🔒

如果你想使用 HTTPS 但遇到自签名证书错误，使用这个方案。

### B1：使用自动配置脚本（推荐）

```bash
# 在运行 Dockerun 的服务器上执行
cd /path/to/dockerun
sudo ./scripts/setup-registry-https.sh 192.168.0.170 5000

# 脚本会自动：
# ✅ 生成包含 SAN 的自签名证书
# ✅ 配置 Docker 信任证书
# ✅ 生成 Registry 启动配置
# ✅ 生成客户端配置脚本
```

### B2：手动配置（详细步骤）

#### 步骤 1: 创建证书配置文件

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
O  = Docker Registry
OU = IT
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

**重要**: 修改 `CN` 和 `IP.1` 为你的实际 IP 地址！

#### 步骤 2: 生成证书

```bash
openssl req -x509 -nodes -days 365 \
  -newkey rsa:2048 \
  -keyout registry.key \
  -out registry.crt \
  -config openssl.cnf \
  -extensions v3_req
```

#### 步骤 3: 验证证书包含 SAN

```bash
openssl x509 -in registry.crt -text -noout | grep -A 1 "Subject Alternative Name"
```

应该看到：
```
X509v3 Subject Alternative Name:
    DNS:registry.local, DNS:localhost, IP Address:192.168.0.170, ...
```

#### 步骤 4: 配置 Docker 信任证书

**Linux:**

```bash
sudo mkdir -p /etc/docker/certs.d/192.168.0.170:5000
sudo cp registry.crt /etc/docker/certs.d/192.168.0.170:5000/ca.crt
sudo systemctl restart docker
```

**macOS:**

```bash
mkdir -p ~/.docker/certs.d/192.168.0.170:5000
cp registry.crt ~/.docker/certs.d/192.168.0.170:5000/ca.crt
# 重启 Docker Desktop
```

#### 步骤 5: 配置 Registry 使用证书

编辑 `docker-compose.yml`:

```yaml
version: '3.8'

services:
  registry:
    image: registry:2
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

### B3：在 Dockerun 中配置

```json
{
  "name": "Secure Registry",
  "url": "https://192.168.0.170:5000",  ← 使用 https://
  "username": "",
  "password": "",
  "insecure": false                      ← 取消勾选
}
```

---

## 验证修复

### 测试 1：检查配置

```bash
docker info | grep -i insecure
```

### 测试 2：测试连接

**如果使用 HTTP:**
```bash
curl http://192.168.0.170:5000/v2/_catalog
```

**如果使用 HTTPS:**
```bash
curl https://192.168.0.170:5000/v2/_catalog
```

应该返回 JSON 格式的镜像列表：
```json
{"repositories":["indohit/gateway","test/alpine"]}
```

### 测试 3：手动拉取镜像

```bash
# HTTP Registry
docker pull 192.168.0.170:5000/indohit/gateway:0.1.0

# HTTPS Registry
docker pull 192.168.0.170:5000/indohit/gateway:0.1.0
```

如果成功，说明配置生效！✅

---

## 在 Dockerun 中配置

修复 Docker daemon 配置后，在 Dockerun 中确保 Registry 设置正确：

### 1. 进入 Registry 管理页面

导航到 **Registry** 页面

### 2. 编辑或创建 Registry

```json
{
  "name": "Local Registry",
  "url": "http://192.168.0.170:5000",  ← 注意使用 http://
  "username": "",                       ← 如果有认证则填写
  "password": "",                       ← 如果有认证则填写
  "insecure": true                      ← 标记为不安全
}
```

**重要**: 
- ✅ URL 使用 `http://` 前缀
- ✅ 勾选 "Insecure" 选项

### 3. 在 Webhook 配置中关联 Registry

创建或编辑 Webhook 配置时：

1. **Image Name**: `indohit/gateway` （不需要包含 registry 地址）
2. **Image Tag**: `0.1.0` 或留空匹配所有标签
3. **Registry**: 选择上面创建的 Registry
4. **Deploy Script**: 选择对应的部署脚本

---

## 常见问题

### Q1: 修改配置后 Docker 无法启动

**A**: 检查 JSON 语法是否正确

```bash
# 验证 JSON 格式
cat /etc/docker/daemon.json | python -m json.tool

# 或使用 jq
cat /etc/docker/daemon.json | jq .
```

如果有语法错误，恢复备份：

```bash
sudo mv /etc/docker/daemon.json /etc/docker/daemon.json.broken
sudo systemctl restart docker
```

### Q2: 配置后仍然报同样的错误

**A**: 确认以下几点：

1. Docker 服务已重启
2. insecure-registries 配置正确
3. 使用 `docker info` 验证配置

### Q3: 可以配置多个 insecure registries 吗？

**A**: 可以，使用数组格式：

```json
{
  "insecure-registries": [
    "192.168.0.170:5000",
    "192.168.0.171:5000",
    "localhost:5000"
  ]
}
```

### Q4: 是否可以使用通配符？

**A**: 可以使用网段：

```json
{
  "insecure-registries": [
    "192.168.0.0/24:5000"
  ]
}
```

---

## 安全警告 ⚠️

**不安全的 Registry 只应在以下场景使用：**

- ✅ 内网开发环境
- ✅ 测试环境
- ✅ 受保护的私有网络

**不要在以下场景使用：**

- ❌ 生产环境
- ❌ 公网可访问的环境
- ❌ 包含敏感数据的镜像

**生产环境建议：**

1. 为 Registry 配置 HTTPS（使用有效证书）
2. 启用基本认证或 Token 认证
3. 配置防火墙规则限制访问

---

## 下一步

配置完成后：

1. ✅ 重启 Dockerun 应用（如果正在运行）
2. ✅ 测试 Webhook 推送
3. ✅ 检查 Webhook 日志确认部署成功

## 需要帮助？

详细文档：
- 📖 [INSECURE_REGISTRY.md](./INSECURE_REGISTRY.md) - 完整配置指南
- 📖 [WEBHOOK_GUIDE.md](./WEBHOOK_GUIDE.md) - Webhook 配置指南
- 📖 [REGISTRY_GUIDE.md](./REGISTRY_GUIDE.md) - Registry 管理指南

---

**配置完成后，你的 Webhook 自动部署就可以正常工作了！** 🎉

