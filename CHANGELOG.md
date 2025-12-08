# 更新日志

所有重要的项目变更都会记录在此文件中。

格式基于 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/)，
项目遵循 [语义化版本](https://semver.org/lang/zh-CN/)。

## [1.0.2] - 2024-12-08

### UI 重大升级

#### Portainer 风格深色主题
- 🎨 完全重新设计 UI，采用 Portainer 的深色专业风格
- 🌙 全局深色主题
  - 主背景：`#1e1e1e`
  - 卡片背景：`#2a2d31`
  - 侧边栏：`#23252a`
  - 主题色：`#4a9eff`
- ✨ 重新设计登录页面
  - 渐变深色背景
  - 半透明卡片效果
  - DOCKERUN Logo 大写样式
  - 现代化输入框和按钮
- ✨ 重新设计初始设置页面
  - 与登录页面统一风格
  - 引导式密码设置流程
- ✨ 重新设计主布局
  - 深色侧边栏 + "COMMUNITY EDITION" 标签
  - 图标化菜单导航
  - 用户下拉菜单
  - 统一的深色配色
- ✨ 重新设计容器列表页面
  - Portainer 风格表格
  - 状态图标（绿色/灰色圆点）
  - 快捷操作按钮（日志/统计/终端）
  - 批量操作按钮栏
  - 端口链接显示
  - 所有权信息

#### 交互优化
- 🎯 改进 Hover 效果（按钮上移+阴影）
- 🎯 优化 Focus 状态（蓝色边框+阴影）
- 🎯 侧边栏激活状态（左侧蓝边）
- 🎯 表格行 Hover 高亮
- 🎯 平滑过渡动画（0.3s）

#### 组件样式
- 📦 深色主题按钮（Primary/Success/Warning/Danger）
- 📦 深色输入框（半透明背景）
- 📦 深色下拉菜单
- 📦 深色对话框
- 📦 深色表格
- 📦 自定义滚动条样式

### 功能修复

#### 首次运行流程优化
- 🔧 修复首次运行不提示设置密码的问题
- 🔧 优化密码为空检测逻辑
- 🔧 自动跳转到设置页面
- ✅ 确保首次运行必须设置密码

### 新增文档

- 📚 添加 UI_DESIGN.md UI 设计规范文档
  - 设计理念
  - 页面布局说明
  - 组件样式规范
  - 颜色规范
  - 交互效果
  - 响应式设计
  - 可访问性指南

## [1.0.1] - 2024-12-08

### 优化

#### 前端构建优化
- 🚀 升级 Node.js 版本要求至 20+
- 📦 使用 pnpm 10.12.4 替代 npm 作为包管理器
- ⚡ 添加 pnpm 配置文件（.npmrc）
- 🔧 更新 Dockerfile 使用 pnpm 构建
- 📝 更新所有文档中的命令为 pnpm
- ✅ 添加 engines 字段约束 Node 20+ 和 pnpm 10+
- 🛠️ 添加 type-check 脚本

#### 后端依赖修复
- 🔧 升级 Docker SDK 到 v25.0.5 解决兼容性问题
- 🔧 修复 Docker SDK API 类型变更
  - `image.ListOptions` → `types.ImageListOptions`
  - `image.PullOptions` → `types.ImagePullOptions`
  - `image.RemoveOptions` → `types.ImageRemoveOptions`
  - `network.ListOptions` → `types.NetworkListOptions`
  - `network.InspectOptions` → `types.NetworkInspectOptions`
  - `network.CreateOptions` → `types.NetworkCreate`
- 📦 更新 `github.com/docker/go-connections` 到 v0.5.0
- ✅ 修复 `go.sum` 缺失问题
- ✅ 修复 `distribution/reference` 包冲突

### 新增

- 📚 添加 TROUBLESHOOTING.md 问题排查指南
- 📚 添加 frontend/README.md 前端项目文档
- 📚 添加 frontend/PNPM_MIGRATION.md 迁移指南
- 📚 添加 OPTIMIZATION_SUMMARY.md 优化总结

## [1.0.0] - 2024-12-08

### 新增功能

#### Docker资源管理
- ✨ 容器管理功能
  - 查看所有容器列表（运行中和已停止）
  - 启动、停止、重启容器
  - 查看容器实时日志
  - 查看容器统计信息
  - 删除容器（支持强制删除）
  
- ✨ 镜像管理功能
  - 查看镜像列表
  - 拉取Docker镜像（实时显示进度）
  - 删除镜像
  - 镜像打标签
  
- ✨ 网络管理功能
  - 查看Docker网络列表
  - 创建自定义网络（支持bridge、overlay等驱动）
  - 删除网络
  - 连接容器到网络
  - 断开容器与网络的连接
  
- ✨ 卷管理功能
  - 查看卷列表
  - 创建命名卷
  - 删除卷

#### 部署脚本管理
- ✨ 脚本CRUD操作
  - 创建部署脚本
  - 编辑脚本内容
  - 删除脚本
  - 脚本列表展示
  
- ✨ 脚本执行功能
  - 一键执行Shell脚本
  - 支持Docker Compose部署
  - 实时查看执行输出
  - 脚本与容器关联
  - 脚本状态跟踪（idle、running、stopped、failed）

#### Docker环境管理
- ✨ 系统信息检测
  - 自动检测操作系统类型
  - 检测系统架构（x86_64、aarch64）
  - 检测Linux发行版和版本
  - 检测网络连接状态
  - 检测Docker和Docker Compose安装状态
  
- ✨ Docker自动安装
  - 在线安装（Ubuntu/Debian/CentOS/RHEL/Fedora）
  - 离线RPM包安装
  - 离线DEB包安装
  - 离线二进制文件安装
  - 安装进度实时反馈

#### 安全认证
- ✨ 用户认证系统
  - JWT Token认证
  - 首次运行强制设置管理员密码
  - 密码加密存储（bcrypt）
  - Token自动刷新
  - 登录状态持久化
  
- ✨ 权限管理
  - 所有API接口都需要认证
  - 用户信息管理
  - 密码修改功能
  - 安全退出登录

#### 用户界面
- ✨ 现代化Web界面
  - 基于Vue 3和Element Plus
  - 响应式设计
  - 深色侧边栏导航
  - 实时数据更新
  - 友好的错误提示
  - 加载状态指示
  
- ✨ 页面组件
  - 登录页面
  - 初始设置页面
  - 容器管理页面
  - 镜像管理页面
  - 网络管理页面
  - 卷管理页面
  - 部署脚本管理页面
  - 系统设置页面

#### 开发工具
- ✨ 构建和部署工具
  - Dockerfile多阶段构建
  - Docker Compose配置
  - Makefile便捷命令
  - 自动化脚本
    - install-docker.sh - Docker自动安装
    - backup.sh - 数据备份
    - restore.sh - 数据恢复
    - update.sh - 系统更新

#### 文档
- 📝 完整的项目文档
  - README.md - 项目说明
  - QUICKSTART.md - 快速开始指南
  - DEPLOYMENT.md - 详细部署指南
  - CHANGELOG.md - 更新日志
  - requirement.md - 需求文档

### 技术栈

#### 后端
- Go 1.21
- Gin Web框架
- GORM ORM
- SQLite数据库
- Docker Go SDK
- JWT认证

#### 前端
- Vue 3
- TypeScript
- Pinia状态管理
- Element Plus UI库
- Vite构建工具
- Vue Router
- Axios HTTP客户端

### 性能优化
- ⚡ 前端静态资源优化
- ⚡ Docker镜像分层构建优化
- ⚡ 数据库连接池
- ⚡ API响应缓存

### 安全增强
- 🔒 JWT Token认证
- 🔒 密码bcrypt加密
- 🔒 API接口权限验证
- 🔒 CORS跨域配置
- 🔒 SQL注入防护（GORM）

### 已知问题
- 暂无

### 计划功能
- [ ] WebSocket终端（进入容器命令行）
- [ ] 容器实时资源监控图表
- [ ] 多用户管理
- [ ] 角色权限系统
- [ ] 容器编排模板
- [ ] 日志搜索和过滤
- [ ] 系统监控面板
- [ ] Docker镜像构建
- [ ] 容器备份和恢复
- [ ] 国际化支持（i18n）

## 版本说明

### 版本号规则

版本格式：`主版本号.次版本号.修订号`

- **主版本号**：不兼容的API修改
- **次版本号**：向下兼容的功能性新增
- **修订号**：向下兼容的问题修正

### 更新类型

- `新增` - 新功能
- `变更` - 对现有功能的变更
- `弃用` - 即将移除的功能
- `移除` - 已移除的功能
- `修复` - 问题修复
- `安全` - 安全相关修复

---

[1.0.0]: https://github.com/your-repo/releases/tag/v1.0.0

