# ⚙️ Dockerun 配置指南

## 环境变量

Dockerun 通过环境变量进行配置，支持以下配置项：

### 服务器配置

| 环境变量 | 说明 | 默认值 | 示例 |
|---------|------|--------|------|
| `SERVER_ADDR` | 服务器监听地址 | `:8080` | `:8080`, `0.0.0.0:8080` |
| `GIN_MODE` | 运行模式 | `debug` | `debug`, `release` |

### 数据库配置

| 环境变量 | 说明 | 默认值 | 示例 |
|---------|------|--------|------|
| `DATABASE_PATH` | SQLite 数据库文件路径 | `./data/dockerun.db` | `/app/data/dockerun.db` |

### 安全配置

| 环境变量 | 说明 | 默认值 | 示例 |
|---------|------|--------|------|
| `JWT_SECRET` | JWT 签名密钥 | `your-secret-key-change-in-production` | 任意长字符串 |

⚠️ **重要**: 生产环境必须修改 `JWT_SECRET` 为强密码！

### 模板配置

| 环境变量 | 说明 | 默认值 | 示例 |
|---------|------|--------|------|
| `TEMPLATE_DIR` | 模板目录路径 | `./templates` | `../templates`, `/app/templates` |

#### 模板目录说明

不同环境下的模板目录配置：

**本地开发环境：**
```bash
# 在 backend/ 目录运行时
TEMPLATE_DIR=../templates

# 在项目根目录运行时
TEMPLATE_DIR=./templates
```

**Docker 容器：**
```bash
# Dockerfile 中已配置
TEMPLATE_DIR=/app/templates
```

### Docker Network 配置

应用启动时会自动创建一个 Docker 网络，用于容器间通信。

| 环境变量 | 说明 | 默认值 | 示例 |
|---------|------|--------|------|
| `DOCKER_NETWORK_NAME` | 网络名称 | `dockerun-net` | `my-app-net` |
| `DOCKER_NETWORK_DRIVER` | 网络驱动 | `bridge` | `bridge`, `overlay` |
| `DOCKER_NETWORK_SUBNET` | 子网 | `172.20.0.0/16` | `10.10.0.0/16` |
| `DOCKER_NETWORK_GATEWAY` | 网关 | `172.20.0.1` | `10.10.0.1` |

#### 功能说明

- ✅ 启动时自动检查网络是否存在
- ✅ 如果不存在则自动创建
- ✅ 如果已存在则跳过创建
- ✅ 支持自定义子网和网关
- ✅ 部署的容器可以连接到此网络

---

## 配置方式

### 方式1: 环境变量（推荐）

```bash
# Linux/macOS
export SERVER_ADDR=:8080
export TEMPLATE_DIR=../templates
go run main.go

# Windows CMD
set SERVER_ADDR=:8080
set TEMPLATE_DIR=../templates
go run main.go

# Windows PowerShell
$env:SERVER_ADDR=":8080"
$env:TEMPLATE_DIR="../templates"
go run main.go
```

### 方式2: 使用 Makefile（推荐）

项目已在 Makefile 中配置好环境变量：

```bash
# 开发模式（自动设置 TEMPLATE_DIR=../templates）
cd backend
make dev

# 或在项目根目录
make dev-backend
```

### 方式3: .env 文件

创建 `.env` 文件（已在 .gitignore 中）：

```bash
SERVER_ADDR=:8080
DATABASE_PATH=./data/dockerun.db
JWT_SECRET=my-super-secret-key
TEMPLATE_DIR=../templates
GIN_MODE=debug
```

然后使用 `godotenv` 或类似工具加载。

### 方式4: Docker Compose

在 `docker-compose.yml` 中配置（已配置）：

```yaml
services:
  dockerun:
    environment:
      - SERVER_ADDR=:8080
      - DATABASE_PATH=/app/data/dockerun.db
      - TEMPLATE_DIR=/app/templates
      - JWT_SECRET=${JWT_SECRET:-change-this-secret-in-production}
```

---

## 不同环境的配置

### 本地开发

```bash
# backend/Makefile 已配置
cd backend
make dev
```

等价于：

```bash
cd backend
TEMPLATE_DIR=../templates go run main.go
```

### Docker 开发

```bash
docker-compose up
```

环境变量在 `docker-compose.yml` 中已配置。

### 生产部署

#### Docker 方式

```bash
docker run -d \
  --name dockerun \
  -p 8080:8080 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v $(pwd)/data:/app/data \
  -e GIN_MODE=release \
  -e JWT_SECRET=your-production-secret \
  -e TEMPLATE_DIR=/app/templates \
  dockerun:latest
```

#### 二进制方式

```bash
export GIN_MODE=release
export SERVER_ADDR=:8080
export DATABASE_PATH=/var/lib/dockerun/dockerun.db
export TEMPLATE_DIR=/opt/dockerun/templates
export JWT_SECRET=your-production-secret

./dockerun
```

---

## 验证配置

### 检查模板目录

启动应用时会输出日志：

```
Loading templates from: ../templates
```

如果看到此日志，说明模板目录配置正确。

### 检查配置是否生效

```bash
# 查看应用日志
docker logs dockerun

# 或直接运行查看输出
cd backend
make dev
```

---

## 故障排查

### 模板未加载

**问题**: 看到 `Template directory not found` 错误

**解决方案**:
1. 检查 `TEMPLATE_DIR` 环境变量是否设置
2. 确认目录路径是否正确
3. 检查目录权限

```bash
# 检查环境变量
echo $TEMPLATE_DIR

# 检查目录是否存在
ls -la $TEMPLATE_DIR

# 本地开发确保使用相对路径
cd backend
TEMPLATE_DIR=../templates make dev
```

### 数据库文件未找到

**问题**: 看到 `unable to open database file` 错误

**解决方案**:
1. 确认 `DATABASE_PATH` 目录存在
2. 创建数据目录

```bash
# 创建数据目录
mkdir -p data

# 或在 Docker 中
docker exec dockerun mkdir -p /app/data
```

### JWT 验证失败

**问题**: 登录后提示 `认证失败`

**解决方案**:
1. 确保 `JWT_SECRET` 一致
2. 检查是否修改了密钥
3. 清除浏览器缓存或重新登录

---

## 最佳实践

### 开发环境

1. ✅ 使用 `make dev` 命令（已配置好环境变量）
2. ✅ 使用 `debug` 模式查看详细日志
3. ✅ 模板目录使用相对路径 `../templates`

### 生产环境

1. ✅ 使用 `release` 模式
2. ✅ 修改 `JWT_SECRET` 为强密码
3. ✅ 使用绝对路径配置
4. ✅ 定期备份数据库文件
5. ✅ 使用 Docker volume 持久化数据

### 安全建议

1. 🔒 不要在代码中硬编码敏感信息
2. 🔒 生产环境使用强 JWT 密钥
3. 🔒 定期更新密钥
4. 🔒 限制数据库文件权限
5. 🔒 使用 HTTPS（配置反向代理）

---

## 示例配置

### 开发环境完整配置

```bash
# backend/.env.dev
SERVER_ADDR=:8080
DATABASE_PATH=./data/dockerun.db
TEMPLATE_DIR=../templates
GIN_MODE=debug
JWT_SECRET=dev-secret-key
```

### 生产环境完整配置

```bash
# .env.production
SERVER_ADDR=:8080
DATABASE_PATH=/app/data/dockerun.db
TEMPLATE_DIR=/app/templates
GIN_MODE=release
JWT_SECRET=xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

---

更多信息请参考 [README.md](README.md)

