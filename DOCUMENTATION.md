# DockeRun 文档整理说明

## 📝 整理日期
2026-01-17

## 👥 团队信息
- **团队**: Ingot Cloud
- **维护者**: JY

---

## 📁 文档结构

### 根目录（3个核心文档）

```
dockerun/
├── README.md          ✅ 项目入口说明
├── CHANGELOG.md       ✅ 更新日志
└── ROADMAP.md         ✅ 开发路线图
```

### docs/ 目录（详细文档）

```
dockerun/docs/
├── README.md                              # 文档索引
│
├── 快速开始
│   ├── QUICKSTART.md                      # 快速开始指南
│   ├── DEPLOYMENT.md                       # 部署指南
│   └── CONFIGURATION.md                    # 配置说明
│
├── 功能指南
│   ├── DEPLOY_SCRIPT_GUIDE.md              # 部署脚本指南
│   ├── DEPLOY_SCRIPT_DOCKER_MANAGER_GUIDE.md # Docker 管理脚本
│   ├── WEBHOOK_GUIDE.md                    # Webhook 自动部署
│   ├── WEBHOOK_REFACTOR_GUIDE.md           # Webhook 重构说明
│   ├── REGISTRY_GUIDE.md                   # Registry 管理
│   ├── NETWORK_GUIDE.md                    # 网络管理
│   └── TEMPLATE_GUIDE.md                   # 模板使用
│
├── 高级特性
│   ├── CODE_EDITOR_FEATURE.md              # 代码编辑器
│   ├── AGENT_ARCHITECTURE_DESIGN.md        # Agent 架构设计
│   ├── UI_OPTIMIZATION_GUIDE.md            # UI 优化指南
│   └── UI_DESIGN.md                        # UI 设计说明
│
├── 安全与运维
│   ├── SECURITY.md                         # 安全指南
│   ├── REGISTRY_PASSWORD_ENCRYPTION.md     # 密码加密
│   ├── DEPLOYMENT_PERMISSIONS.md           # 部署权限
│   ├── TROUBLESHOOTING.md                  # 故障排查
│   ├── MYSQL_SETUP.md                      # MySQL 配置
│   ├── INSECURE_REGISTRY.md                # HTTP Registry
│   ├── QUICK_FIX_INSECURE_REGISTRY.md      # Registry 快速修复
│   └── SELF_SIGNED_CERTIFICATE.md          # 自签名证书
│
└── 构建与开发
    ├── BUILD_AND_DEPLOY.md                 # 构建和部署
    ├── DEVELOPMENT.md                      # 开发指南
    ├── PROJECT_SUMMARY.md                  # 项目总结
    ├── OPTIMIZATION_SUMMARY.md             # 优化总结
    └── PNPM_MIGRATION.md                   # PNPM 迁移
```

---

## 📚 文档说明

### 核心文档（根目录）

#### 1. README.md
**用途**: 项目入口和概览
- 项目简介
- 核心特性
- 快速开始
- 技术栈
- 基本配置
- 文档索引

**目标读者**: 所有用户

#### 2. CHANGELOG.md
**用途**: 版本更新记录
- 版本历史
- 新增功能
- Bug 修复
- 技术改进
- 破坏性变更

**目标读者**: 所有用户

**格式**: 遵循 Keep a Changelog 标准

#### 3. ROADMAP.md
**用途**: 未来规划
- 开发计划
- 功能路线图
- 版本里程碑
- 时间表
- 技术演进

**目标读者**: 开发者、贡献者、项目规划者

---

### 详细文档（docs/ 目录）

#### 快速开始类
- **面向**: 新用户、运维人员
- **特点**: 简明扼要、步骤清晰
- **目的**: 快速上手和部署

#### 功能指南类
- **面向**: 功能使用者
- **特点**: 详细完整、示例丰富
- **目的**: 深入了解和高效使用

#### 高级特性类
- **面向**: 高级用户、开发者
- **特点**: 技术深度、架构设计
- **目的**: 理解原理、扩展功能

#### 安全与运维类
- **面向**: 运维人员、安全工程师
- **特点**: 最佳实践、问题解决
- **目的**: 安全部署、稳定运行

#### 构建与开发类
- **面向**: 开发者、贡献者
- **特点**: 技术细节、开发规范
- **目的**: 参与开发、代码贡献

---

## 🎯 文档使用建议

### 新手用户
1. 阅读 `README.md` 了解项目
2. 按照 `docs/QUICKSTART.md` 快速部署
3. 根据需求查阅功能指南

### 运维人员
1. `README.md` - 项目概览
2. `docs/DEPLOYMENT.md` - 生产部署
3. `docs/SECURITY.md` - 安全配置
4. `docs/TROUBLESHOOTING.md` - 问题排查

### 开发者
1. `README.md` - 项目介绍
2. `docs/DEVELOPMENT.md` - 开发环境
3. `docs/PROJECT_SUMMARY.md` - 架构了解
4. `ROADMAP.md` - 未来规划

### 贡献者
1. `ROADMAP.md` - 了解方向
2. `docs/DEVELOPMENT.md` - 开发规范
3. `docs/BUILD_AND_DEPLOY.md` - 构建流程
4. `CHANGELOG.md` - 版本历史

---

## ✨ 文档维护原则

### 1. 简洁原则
- 根目录仅保留 3 个核心文档
- 详细文档统一放在 docs/ 目录
- 避免文档冗余和重复

### 2. 分类原则
- 按照用途和读者分类
- 文件命名清晰明确
- 目录结构合理清晰

### 3. 更新原则
- 代码变更同步更新文档
- 重大变更记录在 CHANGELOG
- 版本发布前检查文档完整性

### 4. 规范原则
- 使用 Markdown 格式
- 遵循统一的文档模板
- 包含必要的代码示例

---

## 📋 文档检查清单

### 每次发布前
- [ ] README.md 功能列表是否最新
- [ ] CHANGELOG.md 是否记录所有变更
- [ ] ROADMAP.md 进度是否更新
- [ ] 新功能是否有对应文档
- [ ] 配置变更是否更新文档
- [ ] API 变更是否更新文档
- [ ] 示例代码是否可运行

### 文档质量标准
- [ ] 语法正确无误
- [ ] 步骤清晰可操作
- [ ] 代码示例完整
- [ ] 截图清晰准确
- [ ] 链接有效可访问
- [ ] 格式统一规范

---

## 🔄 文档更新流程

1. **功能开发** → 编写功能文档
2. **代码审查** → 审查文档完整性
3. **功能发布** → 更新 CHANGELOG
4. **版本规划** → 更新 ROADMAP
5. **定期维护** → 检查文档时效性

---

## 📞 联系方式

如发现文档问题或需要补充，请：
- 提交 Issue: https://github.com/ingot-cloud/dockerun/issues
- 提交 PR: https://github.com/ingot-cloud/dockerun/pulls

---

**Maintained by Ingot Cloud Team - JY**
