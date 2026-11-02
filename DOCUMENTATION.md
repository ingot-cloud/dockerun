# Dockerun 文档维护说明

项目入口是 [README.md](README.md)，完整导航是 [docs/README.md](docs/README.md)。文档按当前安装与使用、专题资料和历史记录组织，避免将早期设计或脚本命令当成现版本操作步骤。

## 文档职责

| 文档 | 维护内容 |
| --- | --- |
| [README](README.md) | 项目定位、核心能力、可执行的部署入口、首次使用和文档导航 |
| [快速开始](docs/QUICKSTART.md) | 新用户从安装到首次应用发布的最短流程 |
| [部署指南](docs/DEPLOYMENT.md) | 正式 HTTPS、离线安装、持久化、备份恢复和升级 |
| [配置参考](docs/CONFIGURATION.md) | 后端变量、密钥、Compose 插值及开发默认值 |
| [一体化操作指南](docs/INTEGRATED_DEPLOYMENT.md) | 当前应用发布、权限、自动化、离线包、Agent 和接口语义 |
| [开发指南](docs/DEVELOPMENT.md) / [构建交付](docs/BUILD_AND_DEPLOY.md) | 工具链、启动、检查、产物和脚本参数 |
| [项目架构](docs/PROJECT_SUMMARY.md) | 当前组件职责和源码导航，不维护易失真的文件数量统计 |
| [验证记录](docs/VALIDATION.md) | 实际执行的检查、日期、结果和验证边界 |
| [更新日志](CHANGELOG.md) / [路线记录](ROADMAP.md) | 历史版本与分阶段实施情况，保留当时语境 |

## 更新依据

维护文档时先核对实际实现：

- 后端环境变量及密钥：`backend/internal/config/config.go`。
- 容器默认值、挂载、端口及健康检查：`Dockerfile`、`docker-compose.yml`、`.env.example`。
- 本地数据、端口、构建产物与依赖命令：根目录及后端 `Makefile`、`frontend/vite.config.ts`、`frontend/package.json`、`backend/go.mod`。
- 当前版本和交付脚本的环境变量、镜像标签及归档名：`VERSION`、`build.sh`、`run.sh`。
- 实际路由、权限及 CLI 参数：`backend/internal/router/`、`middleware/`、`backend/cmd/dockerun/`。

代码与配置变更时同步修改对应操作指南和 README 摘要。历史资料保留时明确标注适用范围，当前流程在文档索引中优先展示。没有实际执行的测试或交付演练，不写成已经通过。

## 文档检查

1. 检查项目内相对链接及章节锚点，示例路径应存在或明确为需要创建的交付材料。
2. 核对命令与工具参数，区分宿主、Dockerun 容器、开发机和远程 Agent 的执行位置。
3. 使用 `docker compose config --quiet` 检查安装配置，避免输出包含真实凭据的完整配置。
4. 核对工具链、默认端口、镜像标签、数据目录、HTTPS 健康检查及 `.env` 加载方式。
5. 示例使用占位域名和非敏感内容，明确替换要求，不写入真实密码、令牌或密钥。
6. 真实部署及恢复验证使用隔离资源，保留原业务数据；结果只记录实际完成的范围。

新增项目说明使用简体中文，代码标识符、接口字段、品牌、命令及资源原始名称按项目约定保留。前端相关变更遵循 [Dockerun 前端规范](.agents/skills/dockerun-frontend/SKILL.md)。
