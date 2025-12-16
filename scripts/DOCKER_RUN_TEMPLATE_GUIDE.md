# Docker 容器运行模板脚本使用指南

## 简介

`docker-run-template.sh` 是一个通用的 Docker 容器运行脚本模板，支持灵活配置和多种操作方式。

## 特性

- ✅ **灵活配置**：支持配置文件、环境变量、交互式配置三种方式
- ✅ **完整功能**：启动、停止、重启、删除、查看日志、进入容器
- ✅ **资源管理**：CPU、内存限制配置
- ✅ **健康检查**：内置健康检查配置
- ✅ **网络配置**：支持自定义网络、端口映射
- ✅ **卷管理**：支持路径挂载和命名卷
- ✅ **日志控制**：可配置日志驱动和大小
- ✅ **安全增强**：支持以特定用户运行容器

---

## 快速开始

### 方式 1: 交互式配置（推荐新手）

```bash
./docker-run-template.sh interactive
```

按照提示输入配置，脚本会自动启动容器。

### 方式 2: 使用配置文件（推荐）

```bash
# 1. 生成示例配置
./docker-run-template.sh generate-config

# 2. 编辑配置文件
vi docker-run.example.conf

# 3. 复制并修改
cp docker-run.example.conf myapp.conf
vi myapp.conf

# 4. 使用配置启动
./docker-run-template.sh -c myapp.conf start
```

### 方式 3: 环境变量方式（推荐脚本调用）

```bash
CONTAINER_NAME=myapp \
IMAGE_NAME=nginx:latest \
PORTS="8080:80" \
./docker-run-template.sh start
```

---

## 命令说明

| 命令 | 说明 | 示例 |
|------|------|------|
| `start` | 启动容器 | `./docker-run-template.sh start` |
| `stop` | 停止容器 | `./docker-run-template.sh stop` |
| `restart` | 重启容器 | `./docker-run-template.sh restart` |
| `remove` / `rm` | 删除容器 | `./docker-run-template.sh rm` |
| `status` | 查看状态 | `./docker-run-template.sh status` |
| `logs` | 查看日志 | `./docker-run-template.sh logs` |
| `logs -f` | 实时日志 | `./docker-run-template.sh logs -f` |
| `exec` | 进入容器 | `./docker-run-template.sh exec` |
| `interactive` | 交互配置 | `./docker-run-template.sh interactive` |
| `generate-config` | 生成示例 | `./docker-run-template.sh generate-config` |

---

## 配置项详解

### 基本配置

```bash
# 容器名称（必需）
CONTAINER_NAME="myapp"

# 镜像名称（必需）
IMAGE_NAME="nginx:latest"

# 容器主机名（可选）
CONTAINER_HOSTNAME="myapp.local"

# 重启策略
# - no: 不自动重启
# - always: 总是重启
# - unless-stopped: 除非手动停止，否则重启
# - on-failure: 失败时重启
RESTART_POLICY="unless-stopped"
```

### 网络配置

```bash
# 网络模式
# - bridge: 桥接网络（默认）
# - host: 主机网络
# - none: 无网络
NETWORK_MODE="bridge"

# 自定义网络名称（优先级高于 NETWORK_MODE）
NETWORK_NAME="my-custom-network"

# 端口映射（多个用空格分隔）
PORTS="8080:80 8443:443 3306:3306"
```

### 环境变量

```bash
# 方式 1: 多行格式（推荐）
ENV_VARS="
DATABASE_HOST=mysql
DATABASE_PORT=3306
DATABASE_NAME=myapp
APP_ENV=production
TZ=Asia/Shanghai
"

# 方式 2: 单行格式
ENV_VARS="DATABASE_HOST=mysql DATABASE_PORT=3306"
```

### 挂载卷

```bash
# 路径挂载
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

# 挂载 Docker socket（用于容器管理容器）
MOUNT_DOCKER_SOCKET="true"
```

### 资源限制

```bash
# CPU 限制（核心数）
CPU_LIMIT="2.0"       # 2 核
CPU_LIMIT="0.5"       # 0.5 核

# 内存限制
MEMORY_LIMIT="512m"   # 512 MB
MEMORY_LIMIT="2g"     # 2 GB
```

### 健康检查

```bash
# 启用健康检查
HEALTH_CHECK_ENABLED="true"

# 健康检查命令
HEALTH_CHECK_CMD="curl -f http://localhost/health || exit 1"

# 检查间隔
HEALTH_CHECK_INTERVAL="30s"

# 超时时间
HEALTH_CHECK_TIMEOUT="10s"

# 重试次数
HEALTH_CHECK_RETRIES="3"
```

### 日志配置

```bash
# 日志驱动
LOG_DRIVER="json-file"

# 单个日志文件最大大小
LOG_MAX_SIZE="10m"

# 保留的日志文件数量
LOG_MAX_FILE="3"
```

### 安全配置

```bash
# 以特定用户运行
RUN_AS_USER="1000:1000"  # uid:gid

# 工作目录
WORKDIR="/app"
```

### 额外参数

```bash
# Docker run 的额外参数
EXTRA_DOCKER_ARGS="--privileged --cap-add=SYS_ADMIN"
```

---

## 实际应用示例

### 示例 1: Nginx Web 服务器

```bash
# nginx.conf
CONTAINER_NAME="my-nginx"
IMAGE_NAME="nginx:alpine"
RESTART_POLICY="unless-stopped"

PORTS="80:80 443:443"

VOLUMES="
/data/nginx/html:/usr/share/nginx/html:ro
/data/nginx/conf:/etc/nginx/conf.d:ro
/data/nginx/logs:/var/log/nginx
"

ENV_VARS="
TZ=Asia/Shanghai
"

HEALTH_CHECK_ENABLED="true"
HEALTH_CHECK_CMD="curl -f http://localhost/ || exit 1"
```

启动：
```bash
./docker-run-template.sh -c nginx.conf start
```

### 示例 2: MySQL 数据库

```bash
# mysql.conf
CONTAINER_NAME="my-mysql"
IMAGE_NAME="mysql:8.0"
RESTART_POLICY="unless-stopped"

PORTS="3306:3306"

NAMED_VOLUMES="
mysql-data:/var/lib/mysql
"

ENV_VARS="
MYSQL_ROOT_PASSWORD=root_password_123
MYSQL_DATABASE=myapp
MYSQL_USER=myapp
MYSQL_PASSWORD=myapp_password_123
TZ=Asia/Shanghai
"

MEMORY_LIMIT="2g"

HEALTH_CHECK_ENABLED="true"
HEALTH_CHECK_CMD="mysqladmin ping -h localhost -u root -proot_password_123"
```

### 示例 3: Redis 缓存

```bash
# redis.conf
CONTAINER_NAME="my-redis"
IMAGE_NAME="redis:alpine"
RESTART_POLICY="unless-stopped"

PORTS="6379:6379"

NAMED_VOLUMES="
redis-data:/data
"

VOLUMES="
/data/redis/redis.conf:/usr/local/etc/redis/redis.conf:ro
"

MEMORY_LIMIT="512m"

# Redis 使用配置文件启动
EXTRA_DOCKER_ARGS="redis-server /usr/local/etc/redis/redis.conf"
```

### 示例 4: Node.js 应用

```bash
# nodeapp.conf
CONTAINER_NAME="my-nodeapp"
IMAGE_NAME="node:18-alpine"
RESTART_POLICY="unless-stopped"
NETWORK_NAME="my-network"

PORTS="3000:3000"

VOLUMES="
/data/nodeapp:/app
/data/nodeapp/node_modules:/app/node_modules
"

ENV_VARS="
NODE_ENV=production
PORT=3000
DATABASE_URL=mysql://user:pass@mysql:3306/mydb
REDIS_URL=redis://redis:6379
TZ=Asia/Shanghai
"

WORKDIR="/app"

CPU_LIMIT="1.0"
MEMORY_LIMIT="1g"

HEALTH_CHECK_ENABLED="true"
HEALTH_CHECK_CMD="node healthcheck.js"

# 启动命令
EXTRA_DOCKER_ARGS="npm start"
```

### 示例 5: Python 应用（with Dockerfile）

```bash
# pythonapp.conf
CONTAINER_NAME="my-pythonapp"
IMAGE_NAME="python:3.11-slim"
RESTART_POLICY="unless-stopped"

PORTS="5000:5000"

VOLUMES="
/data/pythonapp:/app
"

ENV_VARS="
PYTHONUNBUFFERED=1
FLASK_APP=app.py
FLASK_ENV=production
DATABASE_URL=postgresql://user:pass@postgres:5432/mydb
"

WORKDIR="/app"
RUN_AS_USER="1000:1000"

CPU_LIMIT="2.0"
MEMORY_LIMIT="1g"

# 安装依赖并启动
EXTRA_DOCKER_ARGS="bash -c 'pip install -r requirements.txt && python app.py'"
```

---

## 高级用法

### 1. 创建自定义网络

```bash
# 创建网络
docker network create my-network

# 使用网络
NETWORK_NAME="my-network" ./docker-run-template.sh -c app.conf start
```

### 2. 多容器协同

```bash
# 启动数据库
./docker-run-template.sh -c mysql.conf start

# 启动应用（连接到同一网络）
./docker-run-template.sh -c app.conf start
```

### 3. 批量管理

```bash
# 批量启动
for conf in *.conf; do
    ./docker-run-template.sh -c "$conf" start
done

# 批量停止
for conf in *.conf; do
    ./docker-run-template.sh -c "$conf" stop
done
```

### 4. 在 Dockerfile 中使用

```dockerfile
FROM alpine:latest

COPY docker-run-template.sh /usr/local/bin/
COPY myapp.conf /etc/docker-run/

CMD ["/usr/local/bin/docker-run-template.sh", "-c", "/etc/docker-run/myapp.conf", "start"]
```

---

## 常见问题

### Q1: 如何查看容器实时日志？

```bash
./docker-run-template.sh logs -f
```

### Q2: 如何进入容器调试？

```bash
# 使用 bash
./docker-run-template.sh exec

# 使用 sh（Alpine 镜像）
./docker-run-template.sh exec sh
```

### Q3: 如何更新容器？

```bash
# 1. 拉取新镜像
docker pull new-image:tag

# 2. 更新配置文件中的镜像名
vi myapp.conf

# 3. 重新创建容器
./docker-run-template.sh stop
./docker-run-template.sh rm
./docker-run-template.sh start
```

### Q4: 配置文件放在哪里？

建议目录结构：
```
project/
├── docker-run-template.sh
├── configs/
│   ├── app1.conf
│   ├── app2.conf
│   └── db.conf
└── data/
    ├── app1/
    └── db/
```

### Q5: 如何设置时区？

在环境变量中添加：
```bash
ENV_VARS="
TZ=Asia/Shanghai
"
```

或挂载时区文件：
```bash
VOLUMES="
/etc/localtime:/etc/localtime:ro
/etc/timezone:/etc/timezone:ro
"
```

---

## 最佳实践

1. **使用配置文件**：方便版本控制和复用
2. **命名规范**：使用有意义的容器名和配置文件名
3. **健康检查**：为生产环境的容器配置健康检查
4. **资源限制**：合理设置 CPU 和内存限制
5. **日志管理**：配置日志大小限制，避免磁盘占满
6. **网络隔离**：使用自定义网络隔离不同的应用
7. **数据持久化**：重要数据使用命名卷或挂载路径
8. **安全配置**：避免使用 root 用户运行容器

---

## 故障排查

### 容器无法启动

```bash
# 查看详细错误
./docker-run-template.sh logs

# 检查配置
./docker-run-template.sh status

# 尝试手动运行
docker run --rm -it IMAGE_NAME /bin/sh
```

### 端口冲突

```bash
# 检查端口占用
netstat -tlnp | grep PORT
lsof -i:PORT

# 修改配置文件中的端口映射
```

### 权限问题

```bash
# 检查目录权限
ls -la /data/myapp

# 修改目录所有者
sudo chown -R 1000:1000 /data/myapp

# 或在配置中指定用户
RUN_AS_USER="1000:1000"
```

---

## 参考资料

- [Docker 官方文档](https://docs.docker.com/)
- [Docker Run 参考](https://docs.docker.com/engine/reference/run/)
- [Docker Compose](https://docs.docker.com/compose/)

