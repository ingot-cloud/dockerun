# 问题排查指南

本文档记录了项目开发过程中遇到的问题和解决方案。

## Go 依赖问题

### 问题 1: Missing go.sum entries

**错误信息：**
```
missing go.sum entry for module providing package golang.org/x/crypto/bcrypt
missing go.sum entry for module providing package gorm.io/gorm
...
```

**原因：**
- `go.sum` 文件缺失或不完整
- 依赖未正确下载

**解决方案：**
```bash
cd backend
go mod tidy
```

### 问题 2: Docker SDK 版本兼容性

**错误信息：**
```
# github.com/docker/distribution/reference
reference_deprecated.go:122:19: undefined: reference.SplitHostname
```

**原因：**
- Docker SDK v24 与某些依赖版本不兼容
- `github.com/docker/distribution` 包版本冲突

**解决方案：**

1. **升级 Docker SDK 到 v25**

```go
// go.mod
require (
    github.com/docker/docker v25.0.5+incompatible
    github.com/docker/go-connections v0.5.0
    // ...
)
```

2. **更新 distribution/reference 包**

```bash
go get github.com/distribution/reference@latest
go mod tidy
```

### 问题 3: Docker SDK API 变更

**错误信息：**
```
undefined: image.ListOptions
undefined: image.PullOptions
undefined: network.ListOptions
```

**原因：**
- Docker SDK v25 的类型定义从子包移到了主 types 包

**解决方案：**

更新代码中的类型引用：

```go
// 修改前
import "github.com/docker/docker/api/types/image"
Client.ImageList(ctx, image.ListOptions{All: true})

// 修改后
import "github.com/docker/docker/api/types"
Client.ImageList(ctx, types.ImageListOptions{All: true})
```

**具体变更：**
- `image.ListOptions` → `types.ImageListOptions`
- `image.PullOptions` → `types.ImagePullOptions`
- `image.RemoveOptions` → `types.ImageRemoveOptions`
- `network.ListOptions` → `types.NetworkListOptions`
- `network.InspectOptions` → `types.NetworkInspectOptions`
- `network.CreateOptions` → `types.NetworkCreate`

## 前端问题

### 问题 1: pnpm 未安装

**错误信息：**
```
command not found: pnpm
```

**解决方案：**
```bash
npm install -g pnpm
```

### 问题 2: Node 版本过低

**错误信息：**
```
The engine "node" is incompatible with this module
```

**解决方案：**

确保 Node.js 版本 >= 20.0.0

```bash
# 检查版本
node --version

# 升级 Node.js (使用 nvm)
nvm install 20
nvm use 20

# 或使用官方安装包
# https://nodejs.org/
```

### 问题 3: pnpm-lock.yaml 缺失

**错误信息：**
```
ERR_PNPM_FROZEN_LOCKFILE_WITH_NO_LOCKFILE
```

**解决方案：**
```bash
cd frontend
pnpm install
```

## Docker 构建问题

### 问题 1: Docker 构建失败 - Go 依赖

**错误信息：**
```
go: finding module for package ...
build failed
```

**解决方案：**

确保在构建前运行了 `go mod tidy`：

```dockerfile
# Dockerfile 中
COPY backend/go.mod backend/go.sum ./
RUN go mod download
```

### 问题 2: Docker 构建失败 - pnpm

**错误信息：**
```
corepack: command not found
```

**解决方案：**

使用支持 corepack 的 Node.js 镜像：

```dockerfile
FROM node:20-alpine AS frontend-builder
RUN corepack enable && corepack prepare pnpm@latest --activate
```

### 问题 3: 前端构建产物路径错误

**错误信息：**
```
COPY failed: file not found in build context
```

**解决方案：**

确保 Dockerfile 中的路径正确：

```dockerfile
COPY --from=frontend-builder /frontend/dist ./static
```

## 运行时问题

### 问题 1: 无法连接到 Docker

**错误信息：**
```
Cannot connect to the Docker daemon
```

**解决方案：**

1. 确保 Docker 服务运行：
```bash
systemctl status docker
```

2. 确保挂载了 Docker socket：
```yaml
volumes:
  - /var/run/docker.sock:/var/run/docker.sock
```

3. 检查权限：
```bash
ls -la /var/run/docker.sock
```

### 问题 2: 数据库初始化失败

**错误信息：**
```
Failed to initialize database
```

**解决方案：**

1. 检查数据目录权限：
```bash
mkdir -p data
chmod 755 data
```

2. 检查环境变量：
```bash
export DATABASE_PATH=./data/dockerun.db
```

### 问题 3: JWT 认证失败

**错误信息：**
```
认证令牌无效
```

**解决方案：**

1. 检查 JWT_SECRET 配置：
```bash
export JWT_SECRET=your-secret-key
```

2. 清除浏览器缓存和 localStorage

3. 重新登录获取新 token

## 开发环境问题

### 问题 1: 端口冲突

**错误信息：**
```
bind: address already in use
```

**解决方案：**

1. 查找占用端口的进程：
```bash
# 后端端口 8080
lsof -i :8080

# 前端端口 5173
lsof -i :5173
```

2. 终止进程或更改端口：
```bash
# 终止进程
kill -9 <PID>

# 或修改配置使用其他端口
export SERVER_ADDR=:8081
```

### 问题 2: 热重载不工作

**前端热重载失败**

```bash
cd frontend
rm -rf node_modules .vite
pnpm install
pnpm dev
```

**后端热重载失败**

使用 `air` 工具：

```bash
go install github.com/cosmtrek/air@latest
cd backend
air
```

### 问题 3: CORS 错误

**错误信息：**
```
Access to fetch blocked by CORS policy
```

**解决方案：**

1. 确保后端 CORS 配置正确（已在 router.go 中配置）

2. 开发环境使用 Vite 代理：
```typescript
// vite.config.ts
server: {
  proxy: {
    '/api': 'http://localhost:8080'
  }
}
```

## 性能问题

### 问题 1: 前端构建慢

**解决方案：**

1. 使用 pnpm 而非 npm（已优化）

2. 启用缓存：
```bash
pnpm install --prefer-offline
```

3. 使用更快的镜像源：
```bash
pnpm config set registry https://registry.npmmirror.com/
```

### 问题 2: Docker 镜像体积大

**解决方案：**

1. 使用多阶段构建（已实现）

2. 使用 alpine 基础镜像

3. 清理不必要的依赖：
```dockerfile
RUN apk add --no-cache ca-certificates
```

## 常用调试命令

### Go 后端

```bash
# 查看依赖树
go mod graph

# 清理缓存
go clean -modcache

# 查看导入的包
go list -m all

# 详细构建日志
go build -v

# 运行测试
go test ./...
```

### 前端

```bash
# 清理依赖
pnpm store prune

# 查看依赖树
pnpm list

# 查看过期依赖
pnpm outdated

# 交互式更新
pnpm update -i

# 检查类型
pnpm type-check
```

### Docker

```bash
# 查看容器日志
docker logs dockerun

# 进入容器
docker exec -it dockerun sh

# 查看镜像层
docker history dockerun:latest

# 清理无用镜像
docker system prune -a

# 查看构建缓存
docker builder prune
```

## 获取帮助

如果遇到未列出的问题：

1. 查看详细日志
2. 搜索 GitHub Issues
3. 查阅相关文档：
   - [Go Modules](https://go.dev/ref/mod)
   - [Docker SDK](https://pkg.go.dev/github.com/docker/docker)
   - [pnpm](https://pnpm.io/)
   - [Vite](https://vitejs.dev/)

4. 提交 Issue 并包含：
   - 错误信息
   - 操作系统和版本
   - Go/Node/Docker 版本
   - 完整的错误堆栈

## 版本信息

当前测试通过的版本：

| 组件 | 版本 |
|-----|------|
| Go | 1.21+ |
| Node.js | 20.0.0+ |
| pnpm | 10.12.4+ |
| Docker | 25.0.0+ |
| Docker Compose | 2.0.0+ |
| Docker SDK | v25.0.5 |

## 更新记录

- 2024-12-08: 修复 Docker SDK v25 兼容性问题
- 2024-12-08: 升级到 pnpm 10.12.4
- 2024-12-08: 初始版本

