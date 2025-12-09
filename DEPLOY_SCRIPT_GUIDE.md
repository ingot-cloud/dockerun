# Deploy Script 功能使用指南

## 概述

Deploy Script 功能现已完全重构，支持更复杂的部署场景：
- ✅ 多文件管理（主脚本、docker-compose、配置文件等）
- ✅ 环境变量配置
- ✅ 自定义工作目录
- ✅ 文件类型分类
- ✅ 完整的部署流程支持

## 使用场景示例

### 场景 1：部署一个带配置的 Docker Compose 应用

假设你要部署一个 Nginx 服务，需要以下文件：
1. `docker-compose.yml` - Docker Compose 配置
2. `nginx.conf` - Nginx 配置文件
3. `deploy.sh` - 部署脚本（创建目录、启动服务）

#### 步骤：

1. **创建 Deploy Script**
   - Name: `nginx-app`
   - Description: `Deploy Nginx with custom configuration`
   - Type: `docker-compose`
   - Work Directory: `/opt/nginx-app`

2. **编写主脚本** (`Main Script`)
```bash
#!/usr/bin/env bash

set -e

echo "==> Starting deployment..."

# 设置工作目录
cd "${WORK_DIR}"

# 创建必要的目录
echo "==> Creating directories..."
mkdir -p data logs config

# 复制配置文件到指定位置
echo "==> Setting up configuration..."
cp nginx.conf config/

# 设置权限
chmod 644 config/nginx.conf

# 启动服务
echo "==> Starting services..."
docker-compose up -d

echo "==> Deployment completed!"

# 输出容器ID（用于追踪）
docker-compose ps -q
```

3. **添加文件**

**文件 1: docker-compose.yml**
- File Name: `docker-compose.yml`
- File Path: `./docker-compose.yml`
- File Type: `compose`
- Description: `Docker Compose configuration`
- Content:
```yaml
version: '3.8'

services:
  nginx:
    image: nginx:latest
    container_name: nginx-app
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./config/nginx.conf:/etc/nginx/nginx.conf:ro
      - ./data:/usr/share/nginx/html
      - ./logs:/var/log/nginx
    restart: always
    environment:
      - TZ=${TIMEZONE}
```

**文件 2: nginx.conf**
- File Name: `nginx.conf`
- File Path: `./nginx.conf`
- File Type: `config`
- Description: `Nginx configuration file`
- Content:
```nginx
user nginx;
worker_processes auto;
error_log /var/log/nginx/error.log warn;
pid /var/run/nginx.pid;

events {
    worker_connections 1024;
}

http {
    include /etc/nginx/mime.types;
    default_type application/octet-stream;

    log_format main '$remote_addr - $remote_user [$time_local] "$request" '
                    '$status $body_bytes_sent "$http_referer" '
                    '"$http_user_agent" "$http_x_forwarded_for"';

    access_log /var/log/nginx/access.log main;

    sendfile on;
    keepalive_timeout 65;

    server {
        listen 80;
        server_name localhost;

        location / {
            root /usr/share/nginx/html;
            index index.html index.htm;
        }

        error_page 500 502 503 504 /50x.html;
        location = /50x.html {
            root /usr/share/nginx/html;
        }
    }
}
```

4. **添加环境变量**
- `TIMEZONE` = `Asia/Shanghai`
- `LOG_LEVEL` = `info`

5. **保存并执行**
   - 点击 "Save" 保存配置
   - 点击 "Execute" 执行部署

---

### 场景 2：部署需要数据初始化的应用

假设你要部署一个数据库应用，需要：
1. `docker-compose.yml` - Docker Compose 配置
2. `init.sql` - 初始化 SQL 脚本
3. `setup.sh` - 准备脚本（创建目录、设置权限）

#### 主脚本示例：

```bash
#!/usr/bin/env bash

set -e

echo "==> MySQL Deployment Starting..."

# 环境变量
MYSQL_ROOT_PASSWORD="${MYSQL_ROOT_PASSWORD:-changeme}"
MYSQL_DATABASE="${MYSQL_DATABASE:-myapp}"

cd "${WORK_DIR}"

# 创建数据目录
echo "==> Creating directories..."
mkdir -p mysql-data mysql-logs mysql-init

# 复制初始化脚本
echo "==> Copying initialization scripts..."
cp init.sql mysql-init/

# 启动MySQL
echo "==> Starting MySQL..."
docker-compose up -d

# 等待MySQL启动
echo "==> Waiting for MySQL to be ready..."
sleep 10

# 检查MySQL状态
docker-compose exec -T mysql mysqladmin ping -h localhost --silent

echo "==> MySQL Deployment Completed!"
docker-compose ps -q
```

---

### 场景 3：多服务应用部署

对于包含多个服务的复杂应用：

#### 文件结构：
```
/opt/myapp/
├── docker-compose.yml       # 主 Compose 文件
├── backend/
│   └── .env                 # 后端环境配置
├── frontend/
│   └── nginx.conf          # 前端 Nginx 配置
├── database/
│   └── init.sql            # 数据库初始化
└── redis/
    └── redis.conf          # Redis 配置
```

#### 在 Deploy Script 中添加文件：

1. `docker-compose.yml` (file_path: `./docker-compose.yml`, type: `compose`)
2. `backend/.env` (file_path: `./backend/.env`, type: `config`)
3. `frontend/nginx.conf` (file_path: `./frontend/nginx.conf`, type: `config`)
4. `database/init.sql` (file_path: `./database/init.sql`, type: `script`)
5. `redis/redis.conf` (file_path: `./redis/redis.conf`, type: `config`)

#### 主脚本：
```bash
#!/usr/bin/env bash

set -e

echo "==> Multi-Service Application Deployment"

cd "${WORK_DIR}"

# 创建目录结构
mkdir -p backend frontend database redis
mkdir -p volumes/postgres volumes/redis

# 设置权限
chmod 600 backend/.env
chmod 644 frontend/nginx.conf database/init.sql redis/redis.conf

# 拉取镜像
echo "==> Pulling images..."
docker-compose pull

# 启动服务
echo "==> Starting services..."
docker-compose up -d

# 等待服务就绪
echo "==> Waiting for services..."
sleep 15

# 检查服务状态
docker-compose ps

echo "==> Deployment Completed!"
```

---

## 环境变量说明

系统会自动注入以下环境变量：
- `WORK_DIR` - 工作目录路径（自动设置）

你可以添加自定义环境变量，例如：
- `MYSQL_ROOT_PASSWORD=secret123`
- `API_BASE_URL=https://api.example.com`
- `REDIS_PASSWORD=redis_pass`
- `TIMEZONE=Asia/Shanghai`

---

## 文件类型说明

| 类型 | 说明 | 权限 | 示例 |
|------|------|------|------|
| `compose` | Docker Compose 文件 | 644 | docker-compose.yml |
| `config` | 配置文件 | 644 | nginx.conf, .env |
| `script` | 可执行脚本 | 755 | init.sh, backup.sh |
| `other` | 其他文件 | 644 | README.md, data.json |

---

## 最佳实践

### 1. 错误处理
在主脚本开头添加：
```bash
set -e  # 遇到错误立即退出
set -u  # 使用未定义变量时报错
set -o pipefail  # 管道命令失败时报错
```

### 2. 日志输出
使用清晰的日志输出：
```bash
echo "==> Step 1: Creating directories..."
echo "==> Step 2: Starting services..."
echo "==> Deployment completed!"
```

### 3. 目录管理
使用相对路径和 `$WORK_DIR`：
```bash
cd "${WORK_DIR}"
mkdir -p data logs config
```

### 4. 服务健康检查
部署后检查服务状态：
```bash
# 等待服务启动
sleep 10

# 检查容器状态
docker-compose ps

# 检查服务健康
curl -f http://localhost:8080/health || exit 1
```

### 5. 回滚机制
在脚本中添加回滚逻辑：
```bash
#!/usr/bin/env bash

set -e

# 备份当前配置
if [ -d "${WORK_DIR}/config" ]; then
    cp -r "${WORK_DIR}/config" "${WORK_DIR}/config.backup.$(date +%s)"
fi

# 部署新版本
docker-compose up -d

# 健康检查
if ! curl -f http://localhost:8080/health; then
    echo "Health check failed, rolling back..."
    docker-compose down
    # 恢复备份...
    exit 1
fi
```

---

## 故障排查

### 1. 查看执行输出
点击 Execute 后，会显示完整的脚本输出，包括：
- 标准输出（stdout）
- 标准错误（stderr）
- 退出码

### 2. 检查工作目录
确保工作目录具有正确的权限：
```bash
sudo mkdir -p /opt/myapp
sudo chown $(whoami):$(whoami) /opt/myapp
```

### 3. 验证文件内容
在执行前检查文件是否正确：
```bash
cd /opt/myapp
ls -la
cat docker-compose.yml
```

### 4. 测试脚本
可以手动执行脚本进行测试：
```bash
cd /opt/myapp
bash deploy.sh
```

---

## 示例：完整的生产部署脚本

```bash
#!/usr/bin/env bash

##############################################
# Production Deployment Script
# Application: My Web App
# Version: 1.0.0
##############################################

set -euo pipefail

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

# 环境变量检查
if [ -z "${WORK_DIR:-}" ]; then
    log_error "WORK_DIR is not set"
    exit 1
fi

# 切换到工作目录
cd "${WORK_DIR}"
log_info "Working directory: $(pwd)"

# 创建目录结构
log_info "Creating directory structure..."
mkdir -p {data,logs,config,backups}

# 备份旧版本
if [ -d "data" ] && [ "$(ls -A data)" ]; then
    BACKUP_DIR="backups/backup-$(date +%Y%m%d-%H%M%S)"
    log_info "Creating backup: $BACKUP_DIR"
    mkdir -p "$BACKUP_DIR"
    cp -r data "$BACKUP_DIR/"
fi

# 停止旧服务
if docker-compose ps | grep -q "Up"; then
    log_info "Stopping old services..."
    docker-compose down --timeout 30
fi

# 清理旧容器和镜像
log_info "Cleaning up..."
docker system prune -f --volumes

# 拉取最新镜像
log_info "Pulling latest images..."
docker-compose pull

# 启动新服务
log_info "Starting services..."
docker-compose up -d --remove-orphans

# 等待服务就绪
log_info "Waiting for services to be ready..."
RETRY_COUNT=0
MAX_RETRIES=30

while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
    if curl -sf http://localhost:8080/health > /dev/null; then
        log_info "Services are ready!"
        break
    fi
    RETRY_COUNT=$((RETRY_COUNT + 1))
    sleep 2
done

if [ $RETRY_COUNT -eq $MAX_RETRIES ]; then
    log_error "Services failed to start!"
    docker-compose logs --tail=50
    exit 1
fi

# 显示服务状态
log_info "Service status:"
docker-compose ps

# 清理旧备份（保留最近5个）
log_info "Cleaning old backups..."
cd backups
ls -t | tail -n +6 | xargs -r rm -rf
cd ..

log_info "Deployment completed successfully!"

# 输出容器ID
docker-compose ps -q
```

---

## API 说明

### 数据模型

#### DeployScript
```typescript
{
  id: number
  name: string                     // 脚本名称
  description: string              // 脚本描述
  script_type: 'shell' | 'docker-compose'
  main_script: string              // 主执行脚本
  work_dir: string                 // 工作目录
  env_vars: Record<string, string> // 环境变量
  files: DeployScriptFile[]        // 关联文件
  container_id: string             // 容器ID
  status: 'idle' | 'running' | 'stopped' | 'failed'
  created_at: string
  updated_at: string
}
```

#### DeployScriptFile
```typescript
{
  id: number
  deploy_script_id: number
  file_name: string                // 文件名
  file_path: string                // 相对路径
  file_type: 'compose' | 'config' | 'script' | 'other'
  content: string                  // 文件内容
  description: string              // 文件说明
  created_at: string
  updated_at: string
}
```

---

## 总结

新的 Deploy Script 功能提供了完整的部署解决方案：
- ✅ 支持复杂的多文件部署场景
- ✅ 环境变量灵活配置
- ✅ 自定义工作目录
- ✅ 完整的错误处理和输出
- ✅ 适用于各种部署场景（单服务、多服务、数据库、中间件等）

现在你可以轻松管理和执行各种复杂的 Docker 部署任务！🚀

