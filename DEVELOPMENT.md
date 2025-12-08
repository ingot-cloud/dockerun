# 开发指南

本文档介绍如何在本地开发和运行 DockeRun 项目。

## 开发模式

### 方式一：前后端分离运行（推荐）

这是最常用的开发模式，前端和后端分别运行在不同端口，支持热重载。

#### 1. 启动后端

```bash
# 终端 1
cd backend
go run main.go

# 后端将运行在 http://localhost:8080
# 提供 API 服务：/api/*
```

#### 2. 启动前端

```bash
# 终端 2
cd frontend
pnpm install  # 首次需要
pnpm dev

# 前端将运行在 http://localhost:5173
# Vite 会自动代理 /api 请求到后端 8080 端口
```

#### 3. 访问

打开浏览器访问：`http://localhost:5173`

**优点：**
- ✅ 前端支持热重载（HMR）
- ✅ 后端修改后自动重启（使用 air）
- ✅ 开发体验最佳

### 方式二：集成模式（生产预览）

前端构建后，由后端统一服务，模拟生产环境。

#### 1. 构建前端

```bash
# 使用 Makefile
make build

# 或手动构建
cd frontend
pnpm run build

# 复制到后端
rm -rf ../backend/static
cp -r dist ../backend/static
```

#### 2. 启动后端

```bash
cd backend
go run main.go
```

#### 3. 访问

打开浏览器访问：`http://localhost:8080`

**优点：**
- ✅ 模拟生产环境
- ✅ 测试完整的部署流程
- ✅ 单端口访问

**缺点：**
- ❌ 前端无热重载
- ❌ 每次修改需重新构建

## 快速命令

### 使用 Makefile

```bash
# 安装所有依赖
make install-deps

# 开发模式 - 启动后端
make dev-backend

# 开发模式 - 启动前端（另一个终端）
make dev-frontend

# 构建整个项目
make build

# 运行构建后的程序
make run

# 清理构建产物
make clean

# Docker 构建
make docker-build

# Docker 运行
make docker-run
```

### 后端命令

```bash
cd backend

# 运行
go run main.go

# 构建
go build -o dockerun main.go

# 测试
go test ./...

# 格式化代码
go fmt ./...

# 更新依赖
go mod tidy

# 查看依赖
go mod graph
```

### 前端命令

```bash
cd frontend

# 安装依赖
pnpm install

# 开发模式
pnpm dev

# 构建
pnpm build

# 预览构建结果
pnpm preview

# 类型检查
pnpm type-check

# 更新依赖
pnpm update

# 查看依赖树
pnpm list
```

## 目录结构

```
dockerun/
├── backend/              # Go 后端
│   ├── internal/
│   │   ├── config/      # 配置
│   │   ├── database/    # 数据库
│   │   ├── docker/      # Docker 操作
│   │   ├── handler/     # API 处理器
│   │   ├── middleware/  # 中间件
│   │   ├── models/      # 数据模型
│   │   └── router/      # 路由
│   ├── static/          # 前端构建产物（gitignore）
│   ├── data/            # 数据库文件（gitignore）
│   ├── go.mod
│   └── main.go
├── frontend/            # Vue3 前端
│   ├── src/
│   │   ├── api/        # API 接口
│   │   ├── layouts/    # 布局组件
│   │   ├── router/     # 路由
│   │   ├── stores/     # 状态管理
│   │   ├── views/      # 页面组件
│   │   ├── App.vue
│   │   └── main.ts
│   ├── dist/           # 构建产物（gitignore）
│   ├── package.json
│   └── vite.config.ts
├── scripts/            # 工具脚本
├── Dockerfile
├── docker-compose.yml
└── Makefile
```

## 环境变量

### 后端环境变量

创建 `.env` 文件（可选）：

```bash
# 服务器配置
SERVER_ADDR=:8080
GIN_MODE=debug

# 数据库
DATABASE_PATH=./data/dockerun.db

# JWT 密钥
JWT_SECRET=your-secret-key

# Docker
# DOCKER_HOST=unix:///var/run/docker.sock
```

### 前端环境变量

前端使用 Vite 的代理配置，无需额外配置。

## 调试技巧

### 后端调试

#### 使用 Delve

```bash
# 安装 delve
go install github.com/go-delve/delve/cmd/dlv@latest

# 启动调试
cd backend
dlv debug

# 在代码中设置断点
(dlv) break main.main
(dlv) continue
```

#### 使用 VS Code

在 `.vscode/launch.json` 中配置：

```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Launch Backend",
      "type": "go",
      "request": "launch",
      "mode": "debug",
      "program": "${workspaceFolder}/backend",
      "cwd": "${workspaceFolder}/backend"
    }
  ]
}
```

### 前端调试

使用浏览器的开发者工具：

1. 打开 Chrome DevTools（F12）
2. Sources 标签查看源码
3. 设置断点
4. 刷新页面

### 日志调试

#### 后端日志

Gin 默认会输出请求日志：

```
[GIN] 2024/12/08 - 17:03:01 | 200 | 336.125µs | ::1 | GET "/api/auth/check-setup"
```

添加自定义日志：

```go
import "log"

log.Println("Debug message")
log.Printf("User: %s", username)
```

#### 前端日志

使用 console：

```typescript
console.log('Debug:', data)
console.error('Error:', error)
console.table(users)
```

## 热重载

### 后端热重载（Air）

```bash
# 安装 air
go install github.com/cosmtrek/air@latest

# 在 backend 目录创建 .air.toml
cd backend
air init

# 启动
air
```

### 前端热重载

Vite 自带热重载，运行 `pnpm dev` 即可。

## API 测试

### 使用 curl

```bash
# 检查初始设置
curl http://localhost:8080/api/auth/check-setup

# 登录
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"your-password"}'

# 列出容器（需要 token）
curl http://localhost:8080/api/containers \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### 使用 Postman

导入 API 集合测试各个接口。

## 数据库管理

### 查看数据库

```bash
# 安装 sqlite3
# macOS: brew install sqlite3
# Ubuntu: apt-get install sqlite3

# 连接数据库
cd backend
sqlite3 data/dockerun.db

# 查看表
.tables

# 查看用户
SELECT * FROM users;

# 退出
.quit
```

### 重置数据库

```bash
# 删除数据库文件
rm backend/data/dockerun.db

# 重启后端，会自动创建新数据库
cd backend
go run main.go
```

## 常见问题

### 1. 端口被占用

```bash
# 查找占用端口的进程
lsof -i :8080
lsof -i :5173

# 终止进程
kill -9 <PID>
```

### 2. 前端无法连接后端

检查 Vite 代理配置：

```typescript
// vite.config.ts
server: {
  proxy: {
    '/api': {
      target: 'http://localhost:8080',
      changeOrigin: true
    }
  }
}
```

### 3. Docker 连接失败

确保：
- Docker 服务运行中
- Docker socket 可访问
- 当前用户有权限

```bash
# 检查 Docker 状态
systemctl status docker

# 测试 Docker
docker ps

# 添加用户到 docker 组
sudo usermod -aG docker $USER
```

### 4. 前端构建失败

```bash
# 清理缓存
cd frontend
rm -rf node_modules pnpm-lock.yaml .vite
pnpm install
pnpm run build
```

### 5. Go 模块问题

```bash
cd backend
go clean -modcache
go mod tidy
```

## 代码规范

### Go 代码

- 遵循 Go 官方规范
- 使用 `gofmt` 格式化
- 使用 `golint` 检查
- 注释使用英文

### TypeScript/Vue 代码

- 使用 ESLint
- 组件名使用 PascalCase
- 文件名使用 PascalCase（组件）或 camelCase
- 使用 TypeScript 类型

## Git 工作流

```bash
# 创建功能分支
git checkout -b feature/your-feature

# 提交代码
git add .
git commit -m "feat: add your feature"

# 推送
git push origin feature/your-feature

# 创建 Pull Request
```

## 性能优化

### 前端优化

1. 组件懒加载
2. 路由懒加载
3. 图片懒加载
4. 使用虚拟滚动
5. 防抖节流

### 后端优化

1. 数据库索引
2. 缓存查询结果
3. 使用连接池
4. 避免 N+1 查询

## 部署前检查

```bash
# 1. 运行测试
cd backend && go test ./...
cd frontend && pnpm test

# 2. 类型检查
cd frontend && pnpm type-check

# 3. 构建测试
make build

# 4. 本地运行测试
./bin/dockerun

# 5. Docker 构建测试
make docker-build
make docker-run
```

## 获取帮助

- 查看 [README.md](README.md)
- 查看 [TROUBLESHOOTING.md](TROUBLESHOOTING.md)
- 查看 [DEPLOYMENT.md](DEPLOYMENT.md)
- 提交 Issue

## 更新日志

- 2024-12-08: 添加开发指南
- 2024-12-08: 添加前后端集成模式说明

