# Deploy Script Docker Manager 使用指南

## 概述

Deploy Script 功能现在默认集成了 `docker-manager.sh` 和 `docker-run.sh` 通用脚本,让容器部署变得更加简单和标准化。

## 特性

✅ **开箱即用** - 创建部署脚本时自动包含所有必需文件  
✅ **配置驱动** - 只需编辑配置文件,无需编写复杂的脚本  
✅ **功能完整** - 支持启动、停止、重启、日志查看等所有管理功能  
✅ **灵活强大** - 支持所有 Docker run 参数和配置选项  

## 工作原理

### 文件结构

创建新的 Deploy Script 时,系统会自动生成以下文件:

```
部署脚本/
├── deploy.sh           # 主部署脚本(自动执行)
├── docker-manager.sh   # 容器管理脚本(提供管理功能)
├── docker-run.sh       # 通用运行脚本(构建 docker run 命令)
└── app.env            # 配置文件(需要根据应用修改)
```

### 架构说明

1. **app.env** - 配置文件,定义所有环境变量和配置项
2. **docker-run.sh** - 通用执行脚本,根据配置文件构建并执行 docker run 命令
3. **docker-manager.sh** - 管理脚本,提供容器管理功能(启动、停止、重启等)
4. **deploy.sh** - 主部署脚本,调用 docker-manager.sh 启动容器

## 快速开始

### 1. 创建部署脚本

在 Web 界面中:
1. 点击 "Add script" 按钮
2. 填写基本信息:
   - **名称**: 例如 `deploy-nginx`
   - **描述**: 例如 `部署 Nginx 服务器`
   - **工作目录**: 例如 `/ingot-data/dockerun/nginx`

3. 系统会自动创建包含以下文件的部署脚本:
   - `deploy.sh` (主脚本)
   - `docker-manager.sh` (管理脚本)
   - `docker-run.sh` (运行脚本)
   - `app.env` (配置文件)

### 2. 编辑配置文件

在创建的部署脚本中,找到 `app.env` 文件并编辑:

```bash
# ==================== 基本配置 ====================
# 容器名称（必需）
CONTAINER_NAME="my-nginx"

# 镜像名称（必需）
IMAGE_NAME="nginx:latest"

# 重启策略
RESTART_POLICY="unless-stopped"

# ==================== 端口映射 ====================
PORTS="80:80 443:443"

# ==================== 环境变量 ====================
ENV_VARS="
TZ=Asia/Shanghai
"

# ==================== 卷挂载 ====================
VOLUMES="
/data/nginx/html:/usr/share/nginx/html
/data/nginx/conf:/etc/nginx/conf.d
"

# ==================== 资源限制 ====================
CPU_LIMIT="2.0"
MEMORY_LIMIT="1g"

# ==================== 健康检查 ====================
HEALTH_CHECK_ENABLED="true"
HEALTH_CHECK_CMD="curl -f http://localhost/ || exit 1"
```

### 3. 执行部署

点击 "Execute" 按钮执行部署脚本,系统会:
1. 创建工作目录
2. 写入所有脚本文件
3. 执行 `deploy.sh`
4. 调用 `docker-manager.sh app.env start`
5. 自动创建并启动容器

### 4. 管理容器

部署完成后,可以通过以下方式管理容器:

#### 方式 1: 通过 Web 界面重新执行

修改 `deploy.sh` 中的命令:
- 查看状态: `bash docker-manager.sh app.env status`
- 查看日志: `bash docker-manager.sh app.env logs -f`
- 停止容器: `bash docker-manager.sh app.env stop`
- 重启容器: `bash docker-manager.sh app.env restart`

#### 方式 2: 直接在服务器上执行

```bash
# 进入工作目录
cd /ingot-data/dockerun/nginx

# 查看容器状态
bash docker-manager.sh app.env status

# 实时查看日志
bash docker-manager.sh app.env logs -f

# 停止容器
bash docker-manager.sh app.env stop

# 重启容器
bash docker-manager.sh app.env restart

# 进入容器
bash docker-manager.sh app.env exec

# 删除容器
bash docker-manager.sh app.env remove
```

## 配置文件详解

### 必需配置

```bash
# 容器名称
CONTAINER_NAME="myapp"

# 镜像名称
IMAGE_NAME="nginx:latest"
```

### 网络配置

```bash
# 网络模式
NETWORK_MODE="bridge"  # bridge, host, none

# 或使用自定义网络
NETWORK_NAME="my-network"
CONTAINER_IP="172.20.0.10"
```

### 端口映射

```bash
# 格式: 主机端口:容器端口
PORTS="8080:80 8443:443"
```

### 环境变量

```bash
ENV_VARS="
DATABASE_HOST=mysql
DATABASE_PORT=3306
APP_ENV=production
TZ=Asia/Shanghai
"
```

### 卷挂载

```bash
# 主机路径挂载
VOLUMES="
/data/app:/app/data
/logs/app:/app/logs
/config/app:/app/config:ro
"

# 命名卷
NAMED_VOLUMES="
app-data:/app/data
app-cache:/app/cache
"
```

### 资源限制

```bash
# CPU 限制(核心数)
CPU_LIMIT="2.0"

# 内存限制
MEMORY_LIMIT="1g"
```

### 健康检查

```bash
HEALTH_CHECK_ENABLED="true"
HEALTH_CHECK_CMD="curl -f http://localhost/health || exit 1"
HEALTH_CHECK_INTERVAL="30s"
HEALTH_CHECK_TIMEOUT="10s"
HEALTH_CHECK_RETRIES="3"
HEALTH_CHECK_START_PERIOD="60s"
```

### 日志配置

```bash
LOG_DRIVER="json-file"
LOG_MAX_SIZE="10m"
LOG_MAX_FILE="3"
```

## 管理命令详解

### start - 启动容器

```bash
bash docker-manager.sh app.env start
```

- 如果容器不存在,创建并启动
- 如果容器已存在但未运行,启动它
- 如果容器已在运行,显示状态信息

### stop - 停止容器

```bash
bash docker-manager.sh app.env stop
```

### restart - 重启容器

```bash
bash docker-manager.sh app.env restart
```

### status/ps - 查看状态

```bash
bash docker-manager.sh app.env status
```

### logs - 查看日志

```bash
# 查看最后 100 行日志
bash docker-manager.sh app.env logs

# 查看最后 500 行日志
bash docker-manager.sh app.env logs false 500

# 实时查看日志
bash docker-manager.sh app.env logs -f

# 实时查看最后 200 行日志
bash docker-manager.sh app.env logs -f 200
```

### exec - 进入容器

```bash
# 使用 bash
bash docker-manager.sh app.env exec

# 使用 sh
bash docker-manager.sh app.env exec sh
```

### inspect - 查看详细信息

```bash
bash docker-manager.sh app.env inspect
```

### stats - 查看资源使用

```bash
bash docker-manager.sh app.env stats
```

### remove/rm - 删除容器

```bash
bash docker-manager.sh app.env remove
```

## 实际应用示例

### 示例 1: 部署 Nginx

```bash
# app.env
CONTAINER_NAME="my-nginx"
IMAGE_NAME="nginx:latest"
RESTART_POLICY="unless-stopped"
PORTS="80:80 443:443"
VOLUMES="
/data/nginx/html:/usr/share/nginx/html
/data/nginx/conf:/etc/nginx/conf.d
"
HEALTH_CHECK_ENABLED="true"
HEALTH_CHECK_CMD="curl -f http://localhost/ || exit 1"
```

### 示例 2: 部署 MySQL

```bash
# app.env
CONTAINER_NAME="my-mysql"
IMAGE_NAME="mysql:8.0"
RESTART_POLICY="unless-stopped"
PORTS="3306:3306"
ENV_VARS="
MYSQL_ROOT_PASSWORD=mypassword
MYSQL_DATABASE=mydb
TZ=Asia/Shanghai
"
VOLUMES="
/data/mysql:/var/lib/mysql
"
MEMORY_LIMIT="2g"
```

### 示例 3: 部署 Redis

```bash
# app.env
CONTAINER_NAME="my-redis"
IMAGE_NAME="redis:7-alpine"
RESTART_POLICY="unless-stopped"
PORTS="6379:6379"
VOLUMES="
/data/redis:/data
/config/redis/redis.conf:/etc/redis/redis.conf:ro
"
CONTAINER_CMD="redis-server /etc/redis/redis.conf"
MEMORY_LIMIT="512m"
```

### 示例 4: 使用自定义网络

```bash
# 先创建网络
docker network create --subnet=172.20.0.0/16 my-network

# app.env
CONTAINER_NAME="my-app"
IMAGE_NAME="myapp:latest"
RESTART_POLICY="unless-stopped"
NETWORK_NAME="my-network"
CONTAINER_IP="172.20.0.10"
PORTS="8080:8080"
```

## 优势

1. **极简使用** - 只需编辑配置文件,不需要编写复杂的脚本
2. **完全灵活** - 支持所有 docker run 参数,满足各种需求
3. **易于维护** - 配置和命令分离,清晰明了
4. **复用简单** - 只需复制和修改配置文件
5. **标准化** - 所有应用使用统一的部署方式
6. **功能完整** - 提供完整的容器管理功能

## 注意事项

1. **容器名称** - 必须唯一,不能与现有容器重名
2. **镜像拉取** - 首次运行会自动拉取镜像,需要确保网络连接正常
3. **目录权限** - 挂载的主机目录需要有适当的权限
4. **端口冲突** - 确保映射的端口未被占用
5. **资源限制** - 合理设置 CPU 和内存限制,避免影响系统性能
6. **工作目录** - 建议使用固定的工作目录,便于管理和维护

## 故障排查

### 容器启动失败

1. 查看详细日志:
   ```bash
   bash docker-manager.sh app.env logs
   ```

2. 检查配置文件中的必需参数是否正确

3. 检查镜像是否存在:
   ```bash
   docker images | grep <IMAGE_NAME>
   ```

### 端口冲突

```bash
# 检查端口占用
netstat -tulpn | grep <端口>

# 或使用 ss
ss -tulpn | grep <端口>
```

### 权限问题

```bash
# 检查挂载目录权限
ls -la /data/app

# 修改权限
chmod -R 755 /data/app
```

## 总结

通过集成 `docker-manager.sh` 和 `docker-run.sh`,Deploy Script 功能变得更加强大和易用。你只需要:

1. 创建部署脚本(自动生成所有必需文件)
2. 编辑 `app.env` 配置文件
3. 执行部署

就可以完成容器的部署和管理,大大简化了 Docker 容器的运维工作!

