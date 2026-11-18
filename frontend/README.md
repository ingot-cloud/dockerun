# DockeRun 前端

基于 Vue 3 + TypeScript + Pinia + Element Plus 的现代化前端应用。

## 技术栈

- **框架**: Vue 3.4+
- **语言**: TypeScript 5.3+
- **构建工具**: Vite 5.0+
- **状态管理**: Pinia 2.1+
- **UI组件库**: Element Plus 2.5+
- **路由**: Vue Router 4.2+
- **HTTP客户端**: Axios 1.6+
- **包管理器**: pnpm 8+
- **Node版本**: 20+

## 环境要求

确保你的开发环境满足以下要求：

```bash
# 检查Node版本（需要20+）
node --version

# 检查pnpm版本（需要8+）
pnpm --version

# 如果没有安装pnpm
npm install -g pnpm
```

## 快速开始

### 安装依赖

```bash
# 使用pnpm安装（推荐）
pnpm install

# 或者使用npm
npm install
```

### 开发模式

```bash
# 启动开发服务器
pnpm dev

# 访问 http://localhost:5173
```

### 生产构建

```bash
# 类型检查
pnpm type-check

# 构建生产版本
pnpm build

# 预览生产构建
pnpm preview
```

## 项目结构

```
src/
├── api/              # API接口定义
│   ├── request.ts    # Axios封装
│   ├── auth.ts       # 认证API
│   ├── docker.ts     # Docker管理API
│   ├── deploy.ts     # 部署脚本API
│   └── system.ts     # 系统API
├── stores/           # Pinia状态管理
│   └── auth.ts       # 认证状态
├── router/           # Vue Router配置
│   └── index.ts      # 路由定义
├── layouts/          # 布局组件
│   └── MainLayout.vue # 主布局
├── views/            # 页面组件
│   ├── Login.vue     # 登录页
│   ├── Setup.vue     # 初始设置页
│   ├── Containers.vue # 容器管理
│   ├── Images.vue    # 镜像管理
│   ├── Networks.vue  # 网络管理
│   ├── Volumes.vue   # 卷管理
│   ├── DeployScripts.vue # 部署脚本
│   └── System.vue    # 系统设置
├── App.vue           # 根组件
└── main.ts           # 应用入口
```

## 开发指南

### 代码规范

- 使用 TypeScript 编写所有代码
- 遵循 Vue 3 Composition API 风格
- 使用 ESLint 进行代码检查
- 组件命名使用 PascalCase
- 文件名使用 PascalCase（组件）或 camelCase（工具函数）

### API 调用

所有 API 调用都经过统一的 axios 实例：

```typescript
import { dockerApi } from '@/api/docker'

// 调用API
const containers = await dockerApi.containers.list()
```

### 状态管理

使用 Pinia 进行状态管理：

```typescript
import { useAuthStore } from '@/stores/auth'

const authStore = useAuthStore()
console.log(authStore.isAuthenticated)
```

### 路由导航

```typescript
import { useRouter } from 'vue-router'

const router = useRouter()
router.push('/containers')
```

## pnpm 特性

### 为什么使用 pnpm？

1. **磁盘空间高效** - 通过硬链接节省磁盘空间
2. **安装速度快** - 比 npm/yarn 更快的安装速度
3. **严格的依赖管理** - 避免幽灵依赖问题
4. **Monorepo 支持** - 原生支持工作区

### 常用命令

```bash
# 安装依赖
pnpm install

# 添加依赖
pnpm add <package>
pnpm add -D <package>  # 开发依赖

# 更新依赖
pnpm update
pnpm update <package>

# 删除依赖
pnpm remove <package>

# 查看依赖树
pnpm list
pnpm list --depth=1

# 清理缓存
pnpm store prune
```

### 配置文件

- `.npmrc` - pnpm 配置
- `pnpm-workspace.yaml` - 工作区配置

## 环境变量

开发环境通过 Vite 的代理配置访问后端 API：

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

## 构建优化

- 代码分割
- 组件懒加载
- Tree Shaking
- 静态资源优化
- Gzip 压缩

## 浏览器支持

- Chrome (最新)
- Firefox (最新)
- Safari (最新)
- Edge (最新)

## 常见问题

### 1. pnpm 安装依赖失败

```bash
# 清理缓存后重试
pnpm store prune
pnpm install
```

### 2. TypeScript 类型错误

```bash
# 运行类型检查
pnpm type-check

# 重启 IDE 的 TypeScript 服务
```

### 3. 开发服务器无法访问后端

确保后端服务运行在 `http://localhost:8080`，或修改 `vite.config.ts` 中的代理配置。

## 贡献指南

1. Fork 项目
2. 创建特性分支
3. 提交变更
4. 推送到分支
5. 创建 Pull Request

## 许可证

[Apache License, Version 2.0](../LICENSE)

