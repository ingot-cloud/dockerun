# Dockerun 构建和部署指南

本文档介绍如何在本地构建和部署 Dockerun 应用。

---

## 📋 目录

- [环境要求](#环境要求)
- [快速开始](#快速开始)
- [构建选项](#构建选项)
- [运行容器](#运行容器)
- [高级用法](#高级用法)
- [故障排查](#故障排查)

---

## 🔧 环境要求

### 本地构建需要

- **Node.js** >= 20.0.0
- **pnpm** >= 8.0.0
- **Go** >= 1.21
- **Docker** >= 20.10

### 验证安装

```bash
node --version    # v20.x.x
pnpm --version    # 8.x.x
go version        # go1.21.x
docker --version  # Docker version 20.10.x
```

---

## 🚀 快速开始

### 1. 一键构建 + 启动

```bash
# 构建镜像
./build.sh

# 启动容器
./run.sh start

# 访问应用
open http://localhost:8080
```

就这么简单！✨

---

## 🏗️ 构建选项

### 方式一：本地构建（推荐）

**适用场景**：本地开发、高配置机器

```bash
# 构建 latest 版本
./build.sh

# 构建指定版本
./build.sh v1.0.0

# 构建并推送到私有仓库
DOCKER_REGISTRY=docker.io/myuser ./build.sh v1.0.0
```

**构建流程**：
```
1. 检查依赖（node, pnpm, go, docker）
2. 清理旧构建产物
3. 构建前端（pnpm build）
4. 构建后端（go build）
5. 构建 Docker 镜像
```

**优势**：
- ✅ 速度快（利用本地缓存）
- ✅ 占用 Docker 资源少
- ✅ 适合频繁构建

---

### 方式二：Docker 多阶段构建

**适用场景**：CI/CD、构建服务器

```bash
# 使用多阶段 Dockerfile
docker build -f Dockerfile.multistage -t dockerun:latest .
```

**优势**：
- ✅ 环境一致性好
- ✅ 不依赖本地工具链
- ✅ 适合 CI/CD 流水线

**劣势**：
- ❌ 构建时间长
- ❌ 占用 Docker 资源多

---

## 🐳 运行容器

### 使用 run.sh 脚本（推荐）

```bash
# 启动容器
./run.sh start

# 停止容器
./run.sh stop

# 重启容器
./run.sh restart

# 查看日志
./run.sh logs

# 查看状态
./run.sh status

# 进入容器 shell
./run.sh shell

# 删除容器
./run.sh remove
```

---

### 手动运行 Docker 命令

```bash
# 创建数据卷
docker volume create dockerun-data

# 运行容器
docker run -d \
  --name dockerun \
  -p 8080:8080 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v dockerun-data:/app/data \
  --restart unless-stopped \
  dockerun:latest

# 查看日志
docker logs -f dockerun

# 停止容器
docker stop dockerun

# 启动容器
docker start dockerun

# 删除容器
docker rm -f dockerun
```

---

### 使用 Docker Compose

创建 `docker-compose.yml`：

```yaml
version: '3.8'

services:
  dockerun:
    image: dockerun:latest
    container_name: dockerun
    ports:
      - "8080:8080"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - dockerun-data:/app/data
    restart: unless-stopped
    environment:
      - TZ=Asia/Shanghai
      - GIN_MODE=release

volumes:
  dockerun-data:
```

运行：

```bash
docker-compose up -d
```

---

## 🎯 高级用法

### 自定义端口

```bash
# 方式1：环境变量
DOCKERUN_PORT=9000 ./run.sh start

# 方式2：Docker 命令
docker run -d --name dockerun -p 9000:8080 ... dockerun:latest
```

### 自定义网络配置

```bash
docker run -d \
  --name dockerun \
  -p 8080:8080 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v dockerun-data:/app/data \
  -e DOCKER_NETWORK_NAME=my-net \
  -e DOCKER_NETWORK_SUBNET=172.99.0.0/16 \
  -e DOCKER_NETWORK_GATEWAY=172.99.0.1 \
  dockerun:latest
```

### 数据持久化

```bash
# 方式1：命名卷（推荐）
-v dockerun-data:/app/data

# 方式2：绑定挂载
-v /path/on/host:/app/data

# 备份数据
docker run --rm \
  -v dockerun-data:/data \
  -v $(pwd):/backup \
  alpine tar czf /backup/dockerun-backup.tar.gz /data

# 恢复数据
docker run --rm \
  -v dockerun-data:/data \
  -v $(pwd):/backup \
  alpine sh -c "cd /data && tar xzf /backup/dockerun-backup.tar.gz --strip 1"
```

### 推送到镜像仓库

```bash
# Docker Hub
docker tag dockerun:latest myusername/dockerun:latest
docker push myusername/dockerun:latest

# 私有仓库
docker tag dockerun:latest registry.example.com/dockerun:latest
docker push registry.example.com/dockerun:latest

# 使用 build.sh 自动推送
DOCKER_REGISTRY=registry.example.com ./build.sh v1.0.0
# 脚本会询问是否推送
```

---

## 📊 构建产物

### 目录结构

```
.
├── backend/
│   └── main              # Go 二进制文件（Linux amd64）
├── frontend/
│   └── dist/             # 前端构建产物
│       ├── index.html
│       ├── assets/
│       └── ...
└── templates/            # 部署脚本模板
```

### 镜像大小优化

当前镜像大小约 **50-80 MB**（取决于依赖）

**优化技巧**：

1. **使用 Alpine 基础镜像**（已应用）
   ```dockerfile
   FROM alpine:latest
   ```

2. **Go 构建时使用 ldflags**（已应用）
   ```bash
   go build -ldflags="-s -w" -o main .
   ```

3. **前端构建优化**
   - 启用 Vite 的代码分割
   - 压缩资源文件
   - Tree-shaking 去除未使用代码

4. **.dockerignore 优化**（已应用）
   - 排除源代码
   - 只包含构建产物

---

## 🔍 故障排查

### 问题1：构建失败 - pnpm: command not found

**原因**：pnpm 未安装

**解决**：
```bash
npm install -g pnpm
```

---

### 问题2：构建失败 - go: command not found

**原因**：Go 未安装或未配置 PATH

**解决**：
```bash
# macOS
brew install go

# Linux
wget https://go.dev/dl/go1.21.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
```

---

### 问题3：容器启动失败 - permission denied: /var/run/docker.sock

**原因**：Docker socket 权限不足

**解决**：
```bash
# 方式1：将用户添加到 docker 组
sudo usermod -aG docker $USER
newgrp docker

# 方式2：修改 socket 权限（不推荐）
sudo chmod 666 /var/run/docker.sock
```

---

### 问题4：前端访问 404

**原因**：前端路由配置问题

**检查**：
```bash
# 进入容器检查文件
docker exec -it dockerun sh
ls -la /app/static

# 查看日志
docker logs dockerun
```

---

### 问题5：后端 CGO 构建失败

**原因**：SQLite 需要 CGO 支持

**解决**：
```bash
# 确保 CGO_ENABLED=1
CGO_ENABLED=1 GOOS=linux go build ...

# macOS 交叉编译到 Linux 可能需要额外工具
brew install FiloSottile/musl-cross/musl-cross
```

---

### 问题6：镜像体积过大

**检查**：
```bash
docker history dockerun:latest

# 分析镜像
docker run --rm -v /var/run/docker.sock:/var/run/docker.sock \
  wagoodman/dive:latest dockerun:latest
```

---

## 📝 最佳实践

### 开发流程

```bash
# 1. 开发阶段
make dev-frontend   # 前端热重载
make dev-backend    # 后端热重载

# 2. 测试阶段
./build.sh dev      # 构建测试镜像
./run.sh start      # 启动测试

# 3. 生产发布
./build.sh v1.0.0   # 构建生产镜像
# 推送到仓库
docker push registry.example.com/dockerun:v1.0.0
```

### 版本管理

```bash
# 版本号规范：v{major}.{minor}.{patch}
./build.sh v1.0.0   # 正式版本
./build.sh v1.0.0-beta.1   # 测试版本
./build.sh v1.0.0-rc.1     # 候选版本

# 同时打 latest 标签
docker tag dockerun:v1.0.0 dockerun:latest
```

### CI/CD 集成

```yaml
# .github/workflows/build.yml
name: Build and Push

on:
  push:
    tags:
      - 'v*'

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Build Docker image
        run: |
          docker build -f Dockerfile.multistage \
            -t ${{ secrets.REGISTRY }}/dockerun:${GITHUB_REF#refs/tags/} .
      
      - name: Push to registry
        run: |
          echo ${{ secrets.REGISTRY_PASSWORD }} | \
            docker login -u ${{ secrets.REGISTRY_USERNAME }} --password-stdin
          docker push ${{ secrets.REGISTRY }}/dockerun:${GITHUB_REF#refs/tags/}
```

---

## 🎯 性能调优

### 构建速度优化

```bash
# 1. 使用构建缓存
docker build --cache-from dockerun:latest -t dockerun:latest .

# 2. 并行构建
make -j4 build

# 3. 使用 BuildKit
DOCKER_BUILDKIT=1 docker build -t dockerun:latest .
```

### 运行时优化

```bash
# 限制资源使用
docker run -d \
  --name dockerun \
  --memory="512m" \
  --cpus="1.0" \
  -p 8080:8080 \
  dockerun:latest

# 启用只读根文件系统
docker run -d \
  --name dockerun \
  --read-only \
  --tmpfs /tmp \
  -v dockerun-data:/app/data \
  dockerun:latest
```

---

## 📚 相关文档

- [Docker 官方文档](https://docs.docker.com/)
- [Go 交叉编译指南](https://go.dev/doc/install/source#environment)
- [Vite 构建优化](https://vitejs.dev/guide/build.html)
- [Alpine Linux 包管理](https://wiki.alpinelinux.org/wiki/Alpine_Package_Keeper)

---

## 💬 获取帮助

如果遇到问题：

1. 查看构建日志
2. 查看容器日志：`./run.sh logs`
3. 检查镜像：`docker inspect dockerun:latest`
4. 提交 Issue 到 GitHub

---

**祝构建顺利！🎉**

