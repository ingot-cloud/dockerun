# Dockerun 发布工作区测试计划

日期：2026-11-01  
环境：Cursor Cloud VM（Ubuntu 24.04，隔离数据目录与端口）  
范围：发布工作区（概览、应用、模板、发布/部署、部署记录、离线发布包、内置仓库）  
原则：优先运行与验证；不改产品代码。仅在必要时增加测试辅助脚本（不进入 `backend/` / `frontend/`）。

## 隔离约定

| 项 | 值 |
|---|---|
| Git 分支 | `cursor/release-workspace-test-2222` |
| Compose 项目 | `dockerun-test-2222` |
| 数据目录 | `/tmp/dockerun-isolated-2222` |
| 后端监听 | `127.0.0.1:18080` |
| 前端开发端口 | `15173`（仅在 compose 不可用时） |
| PUBLIC_URL | `http://127.0.0.1:18080` |
| 测试账号 | 仅本机 SQLite 初始化的 `admin`，不触达外部仓库凭据 |

清理：测试结束后停止本项目进程 / compose，删除隔离数据目录，移除本轮创建的 Docker project / 测试镜像。

## 启动策略

1. 优先：安装 Docker Engine 后 `docker compose` 构建并启动隔离栈（覆盖端口与数据卷）。
2. 回退：`make install-deps` + 本机 `make dev-backend` / `make build` + `bin/dockerun-server`，`DATA_DIR` 指向隔离目录。
3. 外部 Registry：无凭据则跳过；内置仓库与离线包为必测路径（Docker 可用时）。

## 用例

### S0 环境与启动

- S0.1 读取 README / Makefile / docker-compose / `docs/VALIDATION.md` / `docs/INTEGRATED_DEPLOYMENT.md`
- S0.2 检查 Docker、Go、Node、pnpm
- S0.3 启动隔离栈；`GET /health` 返回 200
- S0.4 初始化管理员、登录、签发含 `pull/push/deploy` 的自动化令牌

### S1 概览

- S1.1 `GET /api/overview`：`counts.applications/deployments/bundles/image_events` 与 `environments`
- S1.2 创建资源后再读一次，计数应增加

### S2 创建应用

- S2.1 空白应用：`POST /api/applications` `{name, description}` → 201
- S2.2 为空白应用添加环境（本机 `target_id=1`，内置仓库）
- S2.3 从受管模板创建：`mysql`、`redis`、`web-app`（文档中的 nginx-proxy 属于旧 `templates/` 脚本目录，不在 `GET /api/managed-templates`；会单独记录）
- S2.4 创建不部署：模板创建接口不得启动容器

### S3 应用模板

- S3.1 `GET /api/managed-templates` 列出内置模板及字段
- S3.2 从模板创建应用（见 S2.3）
- S3.3 保存空白应用部署配置后 `POST /api/applications/:id/template` 另存自定义模板
- S3.4 列表中出现 `custom-*`

### S4 发布 / 部署

- S4.1 保存部署配置：`PUT /api/applications/:id/deployment-config`（不创建任务）
- S4.2 创建版本：`POST /api/applications/:id/releases`（本机 / 内置仓库 / 命令 / 离线）
- S4.3 预检查：`POST /api/deployments/preflight`
- S4.4 确认部署：`POST /api/deployments` → 202，查询任务状态
- S4.5 内置仓库自动发布：向内置 Registry 推送匹配镜像，确认是否生成版本/任务（需 Docker + `docker login`）
- S4.6 外部仓库：无凭据则标记 skip

### S5 部署记录

- S5.1 `GET /api/deployments` 列表
- S5.2 `GET /api/deployments/:id` 详情、日志、节点状态
- S5.3 CLI：`dockerun status` / `dockerun status <id>`

### S6 离线发布包

- S6.1 CLI `dockerun package` 生成完整包（需本机已有同平台镜像）
- S6.2 分块上传：`GET/PUT /api/bundles/uploads/:sha256` + `POST .../complete` 或 `dockerun import --wait`
- S6.3 `POST /api/bundles/:id/version` 确认待发布版本
- S6.4 `dockerun deploy <app> --env <env> --bundle <file> --wait` 或 API 部署
- S6.5 可选：`POST /api/bundles/publish` 显式入库（不作为外部仓库替代）

### S7 现有 Docker 集成测试

```bash
cd backend
DOCKERUN_DOCKER_TESTS=1 go test ./internal/release -run TestDockerLocalAndOfflineDeployment -count=1 -v
```

前置：同平台 `nginx:1.28-alpine`；测试自建 `dockerun-a987654-*` project 并清理。

## 判定

- **pass**：HTTP 状态、关键字段、CLI 退出码符合预期；部署任务在 Docker 可用时达到 success 或文档允许的健康状态。
- **fail**：接口/CLI 返回错误、数据不一致、任务失败且非环境限制。
- **blocked**：Cloud VM 缺少 Docker / 无法拉取镜像 / 无外部仓库凭据等环境限制，步骤本身未执行或无法完成。
- **skip**：计划内明确跳过（外部 Registry）。

证据写入 `artifacts/TEST_RESULTS.md`：命令、关键日志、HTTP 状态码、时间戳。
