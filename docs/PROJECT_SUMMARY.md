# 项目架构概览

Dockerun 将 Docker 资源管理、内置镜像仓库、应用发布和离线交付集成到一个控制台。项目介绍和部署入口见[项目 README](../README.md)，以下用于了解当前源码职责，不按历史文件数量或规划中的能力描述项目。

## 组件与职责

| 组件 | 主要职责 |
| --- | --- |
| Vue 3 + TypeScript 前端 | 中文工作台、资源管理、发布编辑器、任务与维护页面 |
| Go + Gin API | 登录鉴权、管理接口、网页静态资源、WebSocket / 流式日志 |
| GORM + SQLite / MySQL | 用户、应用、环境、版本、任务、审计与系统配置 |
| 内嵌 Distribution v3.1.2 | 标准 Registry 镜像内容读写，使用 Dockerun 用户与令牌鉴权 |
| 发布执行器 | 镜像准备、预检查、命令或 Compose 执行、状态核对、日志及恢复 |
| 离线包模块 | 打包、分块上传、校验、加载镜像、配置与材料保存 |
| CLI | 构建发布、打包、导入、晋升、状态、恢复和 Agent 启动 |
| Agent | 主动连接控制台，领取绑定主机的任务并回传日志及结果 |
| 备份与维护模块 | 元数据一致性备份、文件归档、Registry 只读与 GC |

## 数据与执行边界

应用保存共享部署定义，环境保存目标主机、运行参数和策略。新版本保存镜像身份及定义快照，提交的任务再固定当次执行配置。保存配置、导入材料和预览不会启动业务容器。

仓库内容、Docker Engine 本地镜像缓存与离线材料是独立存储。仓库来源在准备版本时解析摘要，严格离线流程使用包内镜像并核对 Image ID；按地址拉取的来源在实际执行时确定镜像身份。

本机任务在 Dockerun 执行环境运行，远程任务由 Agent 执行。固定主机组逐机发布，失败时暂停后续目标。任务状态持久化，不随浏览器关闭而停止；无法确认的中断执行需要人工核对。

自动恢复仅用于明确无状态且开启策略的受管 Compose 环境，不回滚业务数据卷、数据库或脚本的外部副作用。命令部署不自动回滚。

## 源码导航

| 路径 | 内容 |
| --- | --- |
| `backend/main.go` | 初始化配置、数据库、Docker、Registry、执行器与 HTTP 服务 |
| `backend/cmd/dockerun/` | CLI 命令入口 |
| `backend/internal/config/`、`database/`、`models/` | 配置、存储与模型 |
| `backend/internal/router/`、`middleware/`、`handler/` | 路由、鉴权和 API 行为 |
| `backend/internal/builtin/`、`identity/` | 内置仓库与统一身份 |
| `backend/internal/release/`、`bundle/`、`agent/`、`backup/` | 发布、离线材料、远程执行及备份 |
| `frontend/src/` | 页面、组件、接口及状态 |
| `templates/`、`scripts/` | 随镜像分发资源和独立辅助脚本 |

## 交付与持久化

Dockerfile 使用 Go 1.26.0、Node 22.17.0、pnpm 10.12.4 多阶段构建，运行镜像包含 Docker CLI 29.1.3、Compose 和所需工具。支持 Linux AMD64 / ARM64，每个完整业务发布包对应一个平台。

容器内数据根目录固定为 `/var/lib/dockerun`，默认使用命名卷，也支持宿主目录绑定。Docker SDK、Compose 和应用脚本的默认 Docker 连接会转换数据目录内的绑定源，读取文件仍使用容器内路径；源码开发使用仓库 `data/local`。MySQL 只替换元数据库，不替代 Registry 及材料文件存储。初始化密钥自动持久化，恢复必须保留匹配的密钥与元数据。

## 继续阅读

- [一体化操作与接口](INTEGRATED_DEPLOYMENT.md)：当前行为、权限、应用模板、发布及 Agent。
- [开发指南](DEVELOPMENT.md)：启动、工具链、代理与检查。
- [构建与交付](BUILD_AND_DEPLOY.md)：二进制、镜像和离线交付。
- [验证记录](VALIDATION.md)：实际验证范围、协议边界及待验收项目。
- [路线与实施记录](../ROADMAP.md)：阶段计划与完成记录；历史设计方案不能直接当成当前安装步骤。
