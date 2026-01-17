# DockeRun - 现代化 Docker 管理平台

[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.23+-00ADD8?logo=go)](https://golang.org)
[![Vue Version](https://img.shields.io/badge/vue-3.x-4FC08D?logo=vue.js)](https://vuejs.org)

> **开发团队**: Ingot Cloud  
> **核心开发**: JY  
> **项目类型**: 开源 Docker 管理平台

一个功能强大、界面现代的 Docker 管理平台，提供容器管理、镜像管理、Webhook 自动部署、代码编辑器等企业级功能。

## ✨ 核心特性

### 🐳 Docker 资源管理
- **容器管理**: 启动/停止/重启、日志查看、命令行进入
- **镜像管理**: 拉取/删除/标签、导入/导出镜像
- **网络管理**: 创建/删除网络、容器网络连接
- **卷管理**: 数据卷的完整生命周期管理
- **私有仓库**: Docker Registry 集成与管理

### 🚀 部署脚本管理
- **智能编辑器**: 基于 Monaco Editor 的代码编辑器
- **语法高亮**: 支持 Shell、YAML、JSON、Python、Go 等 20+ 种语言
- **脚本类型**: Shell 脚本、Docker Compose
- **模板系统**: 内置常用服务的部署模板
- **文件管理**: 支持多文件配置和环境变量

### 🔄 Webhook 自动部署
- **Registry 集成**: 支持 Docker Hub、Harbor、GitLab Registry
- **智能匹配**: 镜像名称和标签的通配符匹配
- **自动化流程**: 
  - 自动清理旧容器和镜像
  - 自动拉取最新镜像
  - 执行关联的部署脚本
- **安全可靠**: HMAC-SHA256 签名验证、异步任务队列

### 🎨 现代化界面
- **暗色主题**: 专业的暗色设计风格
- **响应式布局**: 适配各种屏幕尺寸
- **统一组件**: 一致的 UI/UX 体验
- **实时反馈**: WebSocket 实时日志输出

### 🔒 安全特性
- **AES 加密**: 敏感信息（密码）AES-256-CBC 加密
- **JWT 认证**: Token 身份验证
- **权限控制**: 完整的权限管理体系

## 🏗️ 技术栈

### 后端
- **Go 1.23+** - 高性能后端语言
- **Gin** - Web 框架
- **GORM** - ORM 框架
- **SQLite** - 嵌入式数据库（支持 MySQL）
- **Docker SDK** - Docker API 集成

### 前端
- **Vue 3** - 渐进式 JavaScript 框架
- **TypeScript** - 类型安全
- **Pinia** - 状态管理
- **Element Plus** - UI 组件库
- **Monaco Editor** - 代码编辑器
- **Vite** - 构建工具

## 🚀 快速开始

### 使用 Docker 运行（推荐）

```bash
# 1. 克隆仓库
git clone https://github.com/ingot-cloud/dockerun.git
cd dockerun

# 2. 启动服务
docker-compose up -d

# 3. 访问应用
# 浏览器打开: http://localhost:8080
# 首次运行需要设置管理员密码
```

### 使用预构建脚本

```bash
# 运行部署脚本
chmod +x run.sh
./run.sh

# 停止服务
docker stop dockerun
```

### 手动构建

```bash
# 构建多架构镜像（支持 AMD64 和 ARM64）
./build.sh

# 推送到私有仓库
docker push your-registry.com/dockerun:latest
```

## 📦 配置说明

### 环境变量

| 变量名 | 说明 | 默认值 |
|--------|------|--------|
| `SERVER_ADDR` | 服务器监听地址 | `:8080` |
| `DATABASE_TYPE` | 数据库类型 | `sqlite` |
| `DATABASE_PATH` | SQLite 数据库路径 | `./data/dockerun.db` |
| `MYSQL_DSN` | MySQL 连接字符串 | - |
| `JWT_SECRET` | JWT 密钥 | `your-secret-key` |
| `AES_KEY` | AES 加密密钥 | 自动生成 |
| `AES_IV` | AES 初始化向量 | 自动生成 |
| `NETWORK_NAME` | 默认 Docker 网络 | `ingot-net` |

### Docker Compose 配置示例

```yaml
version: '3.8'

services:
  dockerun:
    image: docker-registry.ingotcloud.top/ingot/dockerun:0.1.0
    container_name: dockerun
    restart: always
    ports:
      - "8080:8080"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - /ingot-data/dockerun:/app/data
      - /ingot-data/dockerun:/ingot-data/dockerun
    environment:
      - GIN_MODE=release
      - JWT_SECRET=your-custom-secret-key
      - NETWORK_NAME=ingot-net
    networks:
      - ingot-net

networks:
  ingot-net:
    external: true
```

## 📖 文档

### 用户指南
- [更新日志](./CHANGELOG.md) - 版本更新记录
- [开发路线图](./ROADMAP.md) - 未来规划

### 详细文档（docs 目录）
- **快速开始**: 快速上手指南
- **部署指南**: 生产环境部署
- **Webhook 配置**: 自动部署配置
- **模板使用**: 部署模板说明
- **安全指南**: AES 加密与最佳实践
- **故障排查**: 常见问题解决

更多文档请查看 [docs](./docs/) 目录。

## 🎯 开发路线图

查看 [ROADMAP.md](./ROADMAP.md) 了解未来规划：

- ✅ 基础容器管理
- ✅ 部署脚本管理
- ✅ Webhook 自动部署
- ✅ 代码编辑器集成
- 🚧 Agent 分布式架构
- 🚧 监控和日志系统
- 📋 RBAC 权限控制
- 📋 多租户支持

## 🤝 贡献

欢迎贡献代码、提交 Issue 和 Pull Request！

### 开发环境搭建

```bash
# 1. 安装依赖
cd frontend && pnpm install
cd backend && go mod download

# 2. 启动开发服务
# 终端 1: 后端
cd backend && go run main.go

# 终端 2: 前端
cd frontend && pnpm run dev

# 3. 访问开发服务器
# 打开 http://localhost:5173
```

## 📄 许可证

本项目基于 [MIT License](./LICENSE) 开源。

## 👥 团队

**Ingot Cloud 团队**
- **核心开发**: JY
- **项目主页**: https://github.com/ingot-cloud/dockerun

## 📮 联系我们

- 提交 Issue: https://github.com/ingot-cloud/dockerun/issues
- 功能建议: 欢迎在 Issues 中讨论

---

**Made with ❤️ by Ingot Cloud Team**
