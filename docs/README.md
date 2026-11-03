# Dockerun 文档索引

[项目 README](../README.md) 提供项目介绍、部署入口和使用概览。首次安装按下面的当前指南操作；专题及历史资料保留原有设计背景，不代替当前配置和发布流程。

## 安装与使用

| 文档 | 内容 |
| --- | --- |
| [快速开始](QUICKSTART.md) | 启动控制台、初始化管理员和发布首个 Compose 应用 |
| [部署与运维](DEPLOYMENT.md) | HTTPS、离线安装、数据目录、备份恢复和升级 |
| [单容器部署脚本](DEPLOY_SCRIPT.md) | 沿用 MySQL 与 nginx-proxy，独立环境配置和容器升级 |
| [数据存储与迁移](DATA_STORAGE.md) | 命名卷、目录绑定、查看数据和停机迁移 |
| [配置参考](CONFIGURATION.md) | 环境变量、自动生成密钥、Compose 与开发配置差异 |
| [一体化操作与接口](INTEGRATED_DEPLOYMENT.md) | 应用与环境、模板、发布、自动化、离线包、CLI 和 Agent |
| [仓库指南](REGISTRY_GUIDE.md) | 内置及外部仓库、登录推送、权限、删除和维护 |
| [网络指南](NETWORK_GUIDE.md) | Compose 网络、已有网络和历史配置边界 |

## 安全与维护

| 文档 | 内容 |
| --- | --- |
| [安全与权限](SECURITY.md) | 管理员与普通用户权限、令牌、HTTPS 和执行边界 |
| [外部仓库凭据加密](REGISTRY_PASSWORD_ENCRYPTION.md) | AES-GCM、持久化密钥及旧凭据兼容 |
| [MySQL 元数据库](MYSQL_SETUP.md) | 专用数据库配置、切换与恢复 |
| [故障排查](TROUBLESHOOTING.md) | 启动、镜像标签、Docker、认证、代理、发布和备份 |
| [验证记录](VALIDATION.md) | 复现方法、已验证范围和待验收事项；保留各次记录日期 |

## 开发与架构

| 文档 | 内容 |
| --- | --- |
| [本地开发](DEVELOPMENT.md) | 固定工具链、5999 前端、8080 后端、Docker Desktop 和开发检查 |
| [构建与交付](BUILD_AND_DEPLOY.md) | 二进制、多平台镜像、build.sh 和离线安装材料 |
| [项目架构](PROJECT_SUMMARY.md) | 当前组件职责、源码路径和数据流转 |
| [UI 设计规范](UI_DESIGN.md) | 工作台与页面设计约定 |
| [Dockerun 前端规范](../.agents/skills/dockerun-frontend/SKILL.md) | 本仓库前端样式、交互与中文要求 |
| [路线与实施记录](../ROADMAP.md) | 分阶段计划与实施情况 |
| [更新日志](../CHANGELOG.md) | 历史版本变更记录 |
| [文档维护说明](../DOCUMENTATION.md) | 文档职责、维护依据和检查方法 |

## 专题与历史资料

下列文档包含旧脚本、独立外部 Registry、早期架构或优化记录。使用其中命令前核对适用场景；涉及当前安装、端口、数据目录、权限及发布行为时，以本索引上方指南和源码为准。

| 资料 | 适用范围 |
| --- | --- |
| [历史脚本指南](DEPLOY_SCRIPT_GUIDE.md) / [Docker 管理脚本](DEPLOY_SCRIPT_DOCKER_MANAGER_GUIDE.md) | 独立脚本与旧执行模式；新应用使用发布工作区 |
| [应用模板格式](../templates/README.md) | 模板目录结构、参数、配置文件与开发验证 |
| [Webhook 专题](WEBHOOK_GUIDE.md) / [Webhook 重构记录](WEBHOOK_REFACTOR_GUIDE.md) | 外部回调及历史架构；当前自动发布需按环境显式授权 |
| [Agent 架构设计](AGENT_ARCHITECTURE_DESIGN.md) | 历史方案；实际 Agent 安装与约束见一体化操作指南 |
| [HTTP Registry](INSECURE_REGISTRY.md) / [HTTP 快速排查](QUICK_FIX_INSECURE_REGISTRY.md) | 外部仓库与历史 HTTP 排查；内置仓库调试见本地开发指南 |
| [自签名证书](SELF_SIGNED_CERTIFICATE.md) | 外部 Registry 证书专题；内置引擎 TLS 见部署指南 |
| [部署权限专题](DEPLOYMENT_PERMISSIONS.md) | 历史执行环境排查；实际权限以当前服务和 Agent 身份为准 |
| [代码编辑器](CODE_EDITOR_FEATURE.md) | 编辑器集成专题，当前发布编辑流程见一体化操作指南 |
| [界面优化](UI_OPTIMIZATION_GUIDE.md) / [优化总结](OPTIMIZATION_SUMMARY.md) | 历史实现与优化背景 |
| [pnpm 迁移](PNPM_MIGRATION.md) | 迁移记录；当前固定版本见开发指南 |
| [辅助脚本目录](../scripts/README.md) | 现有脚本适用范围及当前运维入口 |

发现文档问题可提交 [Issue](https://github.com/ingot-cloud/dockerun/issues) 或 [Pull Request](https://github.com/ingot-cloud/dockerun/pulls)。
