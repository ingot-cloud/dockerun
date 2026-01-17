# 更新日志 (Changelog)

所有重要的项目变更都会记录在此文件中。

格式基于 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/)，
版本遵循 [Semantic Versioning](https://semver.org/lang/zh-CN/)。

---

## [Unreleased] - 开发中

### 计划中
- Agent 分布式架构支持
- 监控和日志系统
- RBAC 权限控制
- 多租户支持

---

## [0.2.0] - 2026-01-17

### 新增
- ✨ **代码编辑器集成**
  - 基于 Monaco Editor（VS Code 编辑器）
  - 支持 20+ 种语言的语法高亮
  - 智能语言识别（Shell、YAML、JSON、Python、Go 等）
  - 代码折叠、行号、小地图、自动补全
  - 暗色主题，与整体 UI 风格一致
  - 主脚本编辑器高度 500px，文件编辑器 600px

### 修复
- 🐛 **部署脚本文件保存问题**
  - 修复创建/更新部署脚本时关联文件未保存到数据库的问题
  - 使用事务确保脚本和文件的原子性创建
  - 执行脚本时从数据库正确读取并创建物理文件

### 优化
- 🎨 **前端 UI 优化**
  - 统一所有对话框使用 `FormDialog` 组件
  - Webhooks 页面的 URL 对话框和日志对话框样式统一
  - 优化文件编辑对话框宽度（80%）和保存按钮

### 技术细节
- 添加 `@guolao/vue-monaco-editor` 和 `monaco-editor` 依赖
- 新增 `CodeEditor.vue` 可复用组件
- 添加 `getFileLanguage()` 智能语言识别函数
- 优化 `CreateDeployScript` 和 `UpdateDeployScript` 的文件处理逻辑

---

## [0.1.0] - 2026-01-16

### 新增
- 🎨 **UI 大幅优化**
  - 全新暗色主题设计系统
  - 统一的设计变量（颜色、间距、字体）
  - 可复用组件：`PageContainer`、`ContentCard`、`DataTable`、`FormDialog`
  - 重构所有页面样式，确保一致性
  - 优化所有对话框的 margin 和 padding

- 🔄 **Webhook 自动部署功能**
  - 支持 Docker Registry Webhook 集成
  - 智能匹配镜像名称和标签（支持通配符）
  - 单个 Webhook 可触发多个部署脚本
  - 异步任务队列处理（不阻塞 Registry）
  - HMAC-SHA256 签名验证
  - 完整的执行日志记录
  - 自动清理旧容器和镜像
  - 支持 Docker Hub、Harbor、GitLab Registry 等

- 🔐 **AES 密码加密**
  - 用户登录密码 AES-256-CBC 加密
  - Registry 密码 AES-256-CBC 加密存储
  - 前后端加密传输
  - 动态密钥生成和管理

- 📦 **镜像导入导出**
  - 支持镜像导出为 tar 文件
  - 支持从 tar 文件导入镜像
  - 进度显示和错误处理

- 🎯 **Favicon 图标**
  - Docker 主题的 SVG 图标
  - 支持多尺寸（16x16, 32x32, 48x48）

### 功能增强
- 📝 **部署脚本增强**
  - 添加 Webhook 自动部署配置
  - 支持镜像名称和标签匹配
  - 自动部署开关
  - 工作目录、环境变量、关联文件管理

- 🖥️ **前端组件化**
  - 抽象通用页面容器组件
  - 抽象通用卡片组件
  - 抽象通用表格组件
  - 抽象通用对话框组件

### 技术改进
- 🏗️ **多架构支持**
  - 支持 AMD64 和 ARM64 架构
  - 本地交叉编译（CGO_ENABLED=0）
  - Docker Buildx 多架构镜像构建
  - 纯 Go SQLite 驱动（github.com/glebarez/sqlite）

- 🔧 **构建优化**
  - 优化 `build.sh` 脚本
  - 本地预编译前后端
  - Dockerfile 使用预编译产物
  - 减少构建时间和镜像大小

- 📊 **数据库优化**
  - 应用层处理外键关系（无 GORM 外键）
  - 添加 Webhook 相关表（webhook_configs、webhook_logs）
  - 优化索引和查询性能

### 修复
- 🐛 修复前端 TypeScript 类型错误
- 🐛 修复 Docker 模板目录挂载问题
- 🐛 修复部署脚本 Docker 命令找不到的问题
- 🐛 修复跨架构镜像运行问题
- 🐛 修复 GPG 密钥验证问题

### 文档
- 📚 新增 `WEBHOOK_GUIDE.md` - Webhook 功能详细指南
- 📚 新增 `WEBHOOK_REFACTOR_GUIDE.md` - Webhook 重构说明
- 📚 新增 `REGISTRY_PASSWORD_ENCRYPTION.md` - 密码加密说明
- 📚 新增 `SECURITY.md` - 安全指南
- 📚 新增 `DEPLOYMENT_PERMISSIONS.md` - 权限问题排查
- 📚 新增 `UI_OPTIMIZATION_GUIDE.md` - UI 优化说明
- 📚 新增 `CODE_EDITOR_FEATURE.md` - 代码编辑器功能说明

---

## [0.0.1] - 2026-01-10

### 初始版本
- 🐳 **Docker 资源管理**
  - 容器管理（启动、停止、重启、删除、日志）
  - 镜像管理（拉取、删除、标签）
  - 网络管理（创建、删除、连接）
  - 卷管理（创建、删除、查看）

- 📝 **部署脚本管理**
  - Shell 脚本支持
  - Docker Compose 支持
  - 脚本执行和日志查看
  - 脚本模板系统

- 🔐 **用户认证**
  - JWT Token 认证
  - 首次运行设置密码
  - 密码修改功能

- 🎨 **前端界面**
  - Vue 3 + TypeScript
  - Element Plus UI
  - 响应式布局
  - 现代化设计

- 🏗️ **技术栈**
  - 后端：Go + Gin + GORM + SQLite
  - 前端：Vue 3 + TypeScript + Pinia + Vite
  - Docker SDK 集成

---

## 版本说明

### 版本号规则
- **主版本号（Major）**: 重大架构变更或不兼容的 API 变更
- **次版本号（Minor）**: 新功能添加，向后兼容
- **修订号（Patch）**: Bug 修复和小改进

### 图标含义
- ✨ 新增功能
- 🐛 Bug 修复
- 🔧 配置/工具优化
- 📚 文档更新
- 🎨 UI/样式改进
- ⚡ 性能优化
- 🔐 安全相关
- 🗑️ 移除功能
- 🚧 开发中

---

**Maintained by Ingot Cloud Team - JY**
