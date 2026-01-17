# 📚 Dockerun 模板指南

## 🎯 概述

Dockerun 使用基于文件系统的模板管理系统，每个模板都是一个独立的目录，包含元数据、部署脚本和相关文件。

## 📁 目录结构

```
templates/
├── nginx/              # Nginx Web 服务器模板
│   ├── meta.json       # 模板元数据
│   ├── deploy.sh       # 部署脚本
│   └── files/          # 附加文件
│       ├── nginx.conf
│       └── index.html
├── mysql/              # MySQL 数据库模板
│   ├── meta.json
│   └── deploy.sh
├── redis/              # Redis 缓存模板
│   ├── meta.json
│   └── deploy.sh
└── docker-compose/     # Docker Compose 模板
    ├── meta.json
    ├── deploy.sh
    └── files/
        └── docker-compose.yml
```

## 📝 meta.json 格式

每个模板必须包含一个 `meta.json` 文件，定义模板的元数据：

```json
{
  "id": "my-template",
  "name": "My Application",
  "description": "Deploy my application",
  "category": "Web Server",
  "icon": "🚀",
  "main_script": "deploy.sh",
  "work_dir": "/opt/my-app",
  "env_vars": {
    "APP_PORT": "8080",
    "APP_VERSION": "latest"
  },
  "files": [
    {
      "file_name": "config.yml",
      "file_path": "config/config.yml",
      "source": "files/config.yml",
      "file_type": "config",
      "description": "Application configuration"
    }
  ]
}
```

### 字段说明

| 字段 | 类型 | 必需 | 说明 |
|------|------|------|------|
| `id` | string | ✅ | 唯一标识符，小写字母和连字符 |
| `name` | string | ✅ | 显示名称 |
| `description` | string | ✅ | 模板描述 |
| `category` | string | ✅ | 分类（Web Server, Database, Cache等） |
| `icon` | string | ✅ | 图标（建议使用 emoji） |
| `main_script` | string | ✅ | 主脚本文件名（通常是 deploy.sh） |
| `work_dir` | string | ✅ | 工作目录路径 |
| `env_vars` | object | ❌ | 环境变量键值对 |
| `files` | array | ❌ | 附加文件列表 |

### 附加文件格式

```json
{
  "file_name": "文件名",
  "file_path": "部署时的相对路径",
  "source": "模板目录中的源文件路径",
  "file_type": "config|compose|script|other",
  "description": "文件说明"
}
```

## 🔨 部署脚本规范

### 基本要求

```bash
#!/usr/bin/env bash
set -e

echo "开始部署..."

# 1. 可以使用环境变量
echo "应用端口: ${APP_PORT}"
echo "工作目录: ${WORK_DIR}"

# 2. 创建必要的目录
mkdir -p ${WORK_DIR}/data

# 3. 清理旧容器（如果存在）
docker stop my-app 2>/dev/null || true
docker rm my-app 2>/dev/null || true

# 4. 运行新容器
CONTAINER_ID=$(docker run -d \
  --name my-app \
  -p ${APP_PORT}:80 \
  -v ${WORK_DIR}/data:/data \
  nginx:${APP_VERSION})

# 5. 输出容器ID（重要！）
echo "Container ID: $CONTAINER_ID"
```

### 关键点

1. **Shebang**: 必须使用 `#!/usr/bin/env bash`
2. **错误处理**: 使用 `set -e` 确保错误时停止
3. **环境变量**: 可以使用 meta.json 中定义的环境变量
4. **工作目录**: 通过 `${WORK_DIR}` 访问
5. **容器ID**: 必须输出，用于容器关联

### 输出容器ID的方式

```bash
# 方式1: 直接输出（推荐）
echo "Container ID: $CONTAINER_ID"

# 方式2: 包含在日志中
echo "Deployment successful! Container: $CONTAINER_ID"

# 方式3: 单独一行（12位或64位十六进制）
echo "$CONTAINER_ID"
```

## 🎨 创建自定义模板

### 步骤1: 创建目录结构

```bash
cd templates
mkdir -p my-app/files
```

### 步骤2: 创建 meta.json

```bash
cat > my-app/meta.json << 'EOF'
{
  "id": "my-app",
  "name": "My Application",
  "description": "My custom application template",
  "category": "Custom",
  "icon": "🎯",
  "main_script": "deploy.sh",
  "work_dir": "/opt/my-app",
  "env_vars": {
    "APP_PORT": "8080"
  },
  "files": []
}
EOF
```

### 步骤3: 创建部署脚本

```bash
cat > my-app/deploy.sh << 'EOF'
#!/usr/bin/env bash
set -e

echo "Deploying My Application..."

mkdir -p ${WORK_DIR}/data

docker stop my-app 2>/dev/null || true
docker rm my-app 2>/dev/null || true

CONTAINER_ID=$(docker run -d \
  --name my-app \
  -p ${APP_PORT}:80 \
  nginx:latest)

echo "Container ID: $CONTAINER_ID"
EOF

chmod +x my-app/deploy.sh
```

### 步骤4: 添加配置文件（可选）

```bash
cat > my-app/files/config.yml << 'EOF'
app:
  name: My Application
  port: 8080
EOF
```

然后更新 meta.json：

```json
{
  "files": [
    {
      "file_name": "config.yml",
      "file_path": "config/config.yml",
      "source": "files/config.yml",
      "file_type": "config",
      "description": "Application configuration"
    }
  ]
}
```

### 步骤5: 重启应用

```bash
# 开发环境
cd backend && make dev

# 生产环境（Docker）
docker restart dockerun
```

## 📚 模板示例

### 简单 Web 服务器

```json
{
  "id": "simple-web",
  "name": "Simple Web Server",
  "description": "Deploy a simple Nginx web server",
  "category": "Web Server",
  "icon": "🌐",
  "main_script": "deploy.sh",
  "work_dir": "/opt/simple-web",
  "env_vars": {
    "PORT": "80"
  },
  "files": []
}
```

```bash
#!/usr/bin/env bash
set -e

docker run -d \
  --name simple-web \
  -p ${PORT}:80 \
  nginx:alpine

echo "Container ID: $(docker ps -qf name=simple-web)"
```

### 带配置的应用

```json
{
  "id": "app-with-config",
  "name": "Application with Config",
  "description": "Deploy application with configuration files",
  "category": "Application",
  "icon": "⚙️",
  "main_script": "deploy.sh",
  "work_dir": "/opt/my-app",
  "env_vars": {
    "APP_ENV": "production"
  },
  "files": [
    {
      "file_name": "app.conf",
      "file_path": "conf/app.conf",
      "source": "files/app.conf",
      "file_type": "config",
      "description": "Main configuration"
    },
    {
      "file_name": "init.sh",
      "file_path": "scripts/init.sh",
      "source": "files/init.sh",
      "file_type": "script",
      "description": "Initialization script"
    }
  ]
}
```

## 🔍 最佳实践

### 1. 命名规范
- **模板ID**: 小写字母、数字、连字符，如 `nginx-php-mysql`
- **容器名称**: 使用有意义的名称，避免冲突
- **工作目录**: 使用 `/opt/app-name` 或 `/ingot-data/dockerun/app-name`

### 2. 环境变量
- 提供合理的默认值
- 使用描述性的变量名
- 避免硬编码敏感信息

### 3. 数据持久化
```bash
# 使用卷挂载保存数据
-v ${WORK_DIR}/data:/var/lib/mysql
-v ${WORK_DIR}/config:/etc/app/config:ro
```

### 4. 错误处理
```bash
# 清理旧容器，忽略错误
docker stop my-app 2>/dev/null || true
docker rm my-app 2>/dev/null || true

# 检查命令是否成功
if ! docker ps -q --filter name=my-app; then
  echo "Failed to start container"
  exit 1
fi
```

### 5. 输出信息
```bash
echo "Deployment completed!"
echo "Container ID: $CONTAINER_ID"
echo "Access URL: http://localhost:${APP_PORT}"
echo "Work Directory: ${WORK_DIR}"
```

## 🎯 分类建议

| 分类 | 说明 | 示例 |
|------|------|------|
| Web Server | Web 服务器 | Nginx, Apache, Caddy |
| Database | 数据库 | MySQL, PostgreSQL, MongoDB |
| Cache | 缓存 | Redis, Memcached |
| Message Queue | 消息队列 | RabbitMQ, Kafka, NATS |
| Multi-Container | 多容器应用 | Docker Compose 应用 |
| Monitoring | 监控 | Prometheus, Grafana |
| Development | 开发环境 | Node.js, Python, Go |
| Other | 其他 | 自定义应用 |

## 🐛 故障排查

### 模板不显示

1. 检查目录结构是否正确
2. 验证 meta.json 格式
3. 查看后端日志

### 部署失败

1. 检查脚本权限（需要可执行）
2. 验证环境变量
3. 查看脚本输出

### 文件未加载

1. 检查 files 中的 source 路径
2. 确认文件存在
3. 验证文件权限

## 📖 参考资料

- [Docker 官方文档](https://docs.docker.com/)
- [Docker Compose 文档](https://docs.docker.com/compose/)
- [Bash 脚本指南](https://www.gnu.org/software/bash/manual/)

---

Happy Templating! 🎉

