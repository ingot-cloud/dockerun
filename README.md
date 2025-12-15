# DockeRun - Docker管理平台

一个功能强大的Docker管理平台，提供类似Portainer的核心功能，同时支持自定义脚本部署和Docker环境自动安装。

## 主要功能

### 1. Docker资源管理
- **容器管理**
  - 查看所有容器列表
  - 启动、停止、重启容器
  - 查看容器日志
  - 删除容器
  - 进入容器命令行（支持WebSocket终端）
  
- **镜像管理**
  - 查看镜像列表
  - 拉取镜像
  - 删除镜像
  - 镜像打标签
  
- **网络管理**
  - 查看网络列表
  - 创建、删除网络
  - 连接/断开容器与网络
  
- **卷管理**
  - 查看卷列表
  - 创建、删除卷

### 2. 部署脚本管理
- 创建和管理部署脚本
- 支持Shell脚本和Docker Compose
- 脚本与容器关联
- 一键执行部署脚本
- 查看执行输出

### 3. Webhook 自动部署 🆕
- **Registry Webhook 集成**
  - 支持 Docker Hub、Harbor、GitLab Registry 等
  - 镜像推送时自动触发部署
  - 智能匹配镜像名称和标签
  
- **自动化流程**
  - 自动清理旧容器和旧镜像
  - 自动拉取最新镜像
  - 执行关联的部署脚本
  - 异步任务队列，不阻塞 Registry
  
- **安全可靠**
  - HMAC-SHA256 签名验证
  - 完整的执行日志记录
  - 支持通配符匹配
  - 可配置开关和清理策略

### 4. Docker环境检测与安装
- 自动检测系统信息（OS、架构、发行版）
- 检测Docker和Docker Compose是否已安装
- 支持多种安装方式：
  - 在线安装（Ubuntu/Debian、CentOS/RHEL、Fedora）
  - 离线RPM包安装
  - 离线DEB包安装
  - 离线二进制文件安装

### 5. 安全认证
- JWT Token认证
- 首次运行需设置管理员密码
- 支持修改密码
- 所有API接口都需要认证

## 技术栈

### 后端
- **语言**: Go 1.21
- **框架**: Gin
- **ORM**: GORM
- **数据库**: SQLite
- **Docker SDK**: Docker Go SDK
- **认证**: JWT

### 前端
- **框架**: Vue 3
- **语言**: TypeScript
- **状态管理**: Pinia
- **UI库**: Element Plus
- **构建工具**: Vite
- **路由**: Vue Router

## 快速开始

### 方式一：使用Docker运行（推荐）

1. **使用Docker Compose**

```bash
# 克隆仓库
git clone <repository-url>
cd dockerun

# 启动服务
docker-compose up -d

# 查看日志
docker-compose logs -f
```

2. **使用Docker命令**

```bash
# 构建镜像
docker build -t dockerun:latest .

# 运行容器
docker run -d \
  --name dockerun \
  --restart always \
  -p 8080:8080 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v $(pwd)/data:/app/data \
  -e JWT_SECRET=your-secret-key \
  dockerun:latest
```

3. **访问应用**

打开浏览器访问: `http://localhost:8080`

首次运行会提示设置管理员密码。

### 方式二：本地开发运行

#### 前置要求
- Go 1.21+
- Node.js 20+
- pnpm 8+
- Docker (用于管理Docker资源)

#### 安装依赖

```bash
# 安装pnpm（如果还没有安装）
npm install -g pnpm

# 安装所有依赖
make install-deps

# 或分别安装
cd frontend && pnpm install
cd backend && go mod download
```

#### 开发模式运行

**方式1：使用Makefile**

```bash
# 终端1: 运行后端
make dev-backend

# 终端2: 运行前端
make dev-frontend
```

**方式2：手动运行**

```bash
# 终端1: 运行后端
cd backend
go run main.go

# 终端2: 运行前端
cd frontend
pnpm run dev
```

访问: `http://localhost:5173`

#### 生产构建

```bash
# 构建整个项目
make build

# 运行
make run
```

## 配置说明

### 环境变量

| 变量名 | 说明 | 默认值 |
|--------|------|--------|
| SERVER_ADDR | 服务器监听地址 | :8080 |
| DATABASE_PATH | 数据库文件路径 | ./data/dockerun.db |
| JWT_SECRET | JWT密钥 | your-secret-key-change-in-production |
| GIN_MODE | Gin运行模式 | debug |

### Docker Compose配置

```yaml
environment:
  - GIN_MODE=release
  - SERVER_ADDR=:8080
  - DATABASE_PATH=/app/data/dockerun.db
  - JWT_SECRET=your-custom-secret-key

volumes:
  - /var/run/docker.sock:/var/run/docker.sock  # 必需
  - ./data:/app/data                           # 数据持久化
  - ./docker-packages:/app/docker-packages:ro  # 离线安装包（可选）
```

## 部署脚本示例

### Shell脚本示例

```bash
#!/usr/bin/env bash

VERSION=latest
IMAGE_NAME=nginx
SERVICE_NAME=my-nginx

# 停止运行当前容器
docker ps -q --filter name="${SERVICE_NAME}" | xargs -r docker rm -f

# 拉取最新镜像
docker pull ${IMAGE_NAME}:${VERSION}

# 启动容器
docker run -d --name ${SERVICE_NAME} \
  --restart always \
  -p 8080:80 \
  ${IMAGE_NAME}:${VERSION}
```

### Docker Compose示例

**脚本内容：**
```bash
#!/usr/bin/env bash

SERVICE_NAME=my-app

# 停止并删除旧容器
docker-compose -f docker-compose.yml down

# 启动新容器
docker-compose -f docker-compose.yml up -d
```

**docker-compose.yml：**
```yaml
version: '3.8'

services:
  app:
    image: nginx:latest
    container_name: my-app
    restart: always
    ports:
      - "8080:80"
    volumes:
      - ./html:/usr/share/nginx/html
```

## Docker离线安装

### 准备离线安装包

1. **在线下载Docker包**

```bash
# x86_64架构
wget https://download.docker.com/linux/static/stable/x86_64/docker-24.0.7.tgz

# aarch64架构
wget https://download.docker.com/linux/static/stable/aarch64/docker-24.0.7.tgz
```

2. **下载RPM包（CentOS/RHEL）**

```bash
# 下载Docker RPM包
yumdownloader --resolve docker-ce docker-ce-cli containerd.io
```

3. **下载DEB包（Ubuntu/Debian）**

```bash
# 下载Docker DEB包
apt-get download docker-ce docker-ce-cli containerd.io
```

### 使用DockeRun安装Docker

1. 访问"系统设置"页面
2. 选择安装方式
3. 输入安装包路径（离线安装）
4. 点击"开始安装"

## 安全建议

1. **修改默认密码**：首次运行后立即设置强密码
2. **更改JWT密钥**：在生产环境中使用强随机JWT密钥
3. **使用HTTPS**：建议在生产环境中配置反向代理（Nginx）并启用HTTPS
4. **限制访问**：使用防火墙限制只有可信IP访问
5. **定期备份**：定期备份`data`目录中的数据库文件

## 常见问题

### 1. 无法连接到Docker

**问题**：页面显示Docker未安装或未运行

**解决方案**：
- 确保Docker已安装并运行：`systemctl status docker`
- 确保挂载了Docker socket：`-v /var/run/docker.sock:/var/run/docker.sock`
- 确保容器有足够的权限（可能需要`privileged: true`）

### 2. 前端无法访问后端API

**问题**：前端显示网络错误

**解决方案**：
- 检查后端是否正常运行
- 检查端口映射是否正确
- 检查防火墙设置

### 3. 部署脚本执行失败

**问题**：脚本执行返回错误

**解决方案**：
- 检查脚本语法是否正确
- 确保脚本有执行权限
- 查看执行输出中的错误信息
- 确保Docker命令可以在容器内执行

## 📚 文档

- **[Webhook 自动部署指南](./WEBHOOK_GUIDE.md)** - Webhook 功能详细配置和使用说明
- **[配置说明](./CONFIGURATION.md)** - 环境变量和系统配置
- **[网络功能指南](./NETWORK_GUIDE.md)** - Docker 网络自动创建和管理
- **[Registry 管理指南](./REGISTRY_GUIDE.md)** - 私有镜像仓库管理
- **[安全说明](./SECURITY.md)** - AES 加密和安全最佳实践

## 项目结构

```
dockerun/
├── backend/                 # 后端Go代码
│   ├── internal/
│   │   ├── config/         # 配置管理
│   │   ├── database/       # 数据库初始化
│   │   ├── docker/         # Docker操作
│   │   ├── handler/        # API处理器
│   │   ├── middleware/     # 中间件
│   │   ├── models/         # 数据模型
│   │   └── router/         # 路由配置
│   ├── go.mod
│   └── main.go
├── frontend/               # 前端Vue代码
│   ├── src/
│   │   ├── api/           # API接口
│   │   ├── layouts/       # 布局组件
│   │   ├── router/        # 路由配置
│   │   ├── stores/        # Pinia状态管理
│   │   ├── views/         # 页面组件
│   │   ├── App.vue
│   │   └── main.ts
│   ├── package.json
│   └── vite.config.ts
├── Dockerfile             # Docker镜像构建
├── docker-compose.yml     # Docker Compose配置
├── Makefile              # 构建脚本
└── README.md             # 项目文档
```

## 开发指南

### 添加新功能

1. **后端**：在`backend/internal/handler/`中添加新的处理器
2. **前端**：在`frontend/src/views/`中添加新的页面组件
3. **路由**：分别在后端和前端的路由配置中添加新路由
4. **API**：在`frontend/src/api/`中添加API接口定义

### 代码规范

- Go代码遵循Go标准规范
- 使用`go fmt`格式化代码
- 前端代码遵循Vue 3和TypeScript最佳实践

## 许可证

MIT License

## 贡献

欢迎提交Issue和Pull Request！

## 联系方式

如有问题或建议，请提交Issue。

