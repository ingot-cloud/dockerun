# Dockerun 发布工作区测试结果

日期：2026-11-01  
环境：Cursor Cloud VM（Ubuntu 24.04，`linux/amd64`）  
产品代码：未修改  
测试助手：仅 `artifacts/` 下脚本与隔离 compose 覆盖文件

## 总览

| 结果 | 数量 |
|---|---|
| pass | 32（含复测通过） |
| skip | 2 |
| fail（最终） | 0 |
| blocked | 0（Docker 安装后可用） |

关键路径均已实际跑通：**隔离栈启动**、**创建应用（空白 + 模板）**、**内置仓库发布/部署**、**内置仓库推送自动发布**、**部署记录 / CLI**、**离线打包-导入-部署**、**官方 Docker 集成测试**。

首次 API 脚本有两处测试助手错误（Compose `healthcheck.test` 未加引号；`POST /api/bundles/publish` 误传 `bundle_id`）。修正后复测全部通过，不视为产品缺陷。

## 环境与启动

Cloud VM 初始状态：无 Docker、无 systemd、`go 1.22.2`、Node 22、pnpm 10。已安装 Docker Engine 29.9.2 / Compose v5.6.1，并用 `GOTOOLCHAIN=auto` 下载 Go 1.26.0。`dockerd` 因无 systemd 手动启动。

隔离约定：

| 项 | 值 |
|---|---|
| 本机测试服务 | `127.0.0.1:18080`，`DATA_DIR=/tmp/dockerun-isolated-2222` |
| Compose 项目 | `dockerun-test-2222`，镜像 `dockerun:test-2222`，端口 `127.0.0.1:18081`，数据 `/var/lib/dockerun-test-2222` |
| Docker project | 部署产生 `dockerun-a1-e1`；官方测试使用 `dockerun-a987654-*` 并自行清理 |

启动证据：

```text
# 本机隔离后端
GET http://127.0.0.1:18080/health  -> 200 {"status":"ok"}
GET /api/system/docker-status      -> 200 {"available":true}

# docker compose -f artifacts/docker-compose.test-2222.yml up -d --no-build --pull never
GET http://127.0.0.1:18081/health                 -> 200 {"status":"ok"}
GET http://127.0.0.1:18081/api/auth/check-setup   -> 200 {"require_setup":true}
GET http://127.0.0.1:18081/                       -> 200 text/html lang="zh-CN"
```

本机二进制：`/workspace/bin/dockerun-server`、`/workspace/bin/dockerun`（`go build`，未改产品源码）。  
Compose 镜像构建成功：`Image dockerun:test-2222 Built`（287MB）。

## 用例结果

### S0 启动 / 初始化

| 用例 | 结果 | 证据 |
|---|---|---|
| S0.3 health | **pass** | HTTP 200 `{"status":"ok"}` |
| S0.4 setup | **pass** | `POST /api/auth/setup` 200「初始化设置成功」 |
| S0.4 login | **pass** | `POST /api/auth/login` 200，用户 `admin` |
| S0.4 automation token | **pass** | `POST /api/auth/tokens` 201，scopes `pull push deploy` |
| S0 docker-status | **pass** | 200 `available: true` |
| S0 compose 栈 | **pass** | 18081 health/setup/UI 见上 |

### S1 概览

| 用例 | 结果 | 证据 |
|---|---|---|
| S1.1 overview | **pass** | `GET /api/overview` 200，初始 counts 全 0 |
| S1.2 计数增长 | **pass** | applications 0→4，bundles 0→1，deployments 0→1，image_events 0→2；后续部署继续增加 |

### S2 创建应用

| 用例 | 结果 | 证据 |
|---|---|---|
| S2.1 空白应用 | **pass** | `POST /api/applications` 201，id=1 `blank-app-2222` |
| S2.2 空白环境 | **pass** | `PUT .../environments` 200，`test` / `target_id=1` / 内置仓库 |
| S2.3 redis 模板 | **pass** | `POST /api/managed-templates/redis/applications` 201，未创建部署任务 |
| S2.3 mysql 模板 | **pass** | 同上，201，镜像参数改写为 `127.0.0.1:18080/mysql:8.0.44` |
| S2.3 web-app 模板 | **pass** | 201；nginx-proxy 见 skip |
| S3.1 nginx-proxy | **skip** | 受管模板仅有 `web-app/mysql/redis/nacos-standalone/nacos-cluster`。`templates/nginx-proxy` 是旧脚本目录 |

### S3 应用模板

| 用例 | 结果 | 证据 |
|---|---|---|
| S3.1 列表 | **pass** | `GET /api/managed-templates` 200，5 个内置模板 |
| S3.3 另存自定义 | **pass**（复测） | `POST /api/applications/1/template` 201，`custom-1` / `custom-demo-2222@1.0.0` |
| S3.4 列表出现 custom | **pass**（复测） | `custom_ids: ["custom-1"]` |

### S4 发布 / 部署 / 内置仓库

| 用例 | 结果 | 证据 |
|---|---|---|
| S4.5 docker login 内置仓库 | **pass** | `docker login 127.0.0.1:18080` → Login Succeeded |
| S4.5 push demo:v1/v2/v3 | **pass** | digest `sha256:e6b408afd0fcc6786c31e80825f4d2c6ca76934aa3e9280152addb23202ffd5d` |
| S4.1 保存部署配置 | **pass**（复测） | `PUT /deployment-config` 200 |
| S4.2 创建仓库版本 | **pass**（复测） | `POST /releases` 201，id=3，source=registry |
| S4.3 预检查 | **pass** | `POST /deployments/preflight` 200，`valid: true` |
| S4.4 确认部署 | **pass** | `POST /deployments` 202 → 任务 #2 `success`「发布成功，健康检查通过」 |
| S4.5 开启自动发布 | **pass**（复测） | `PATCH .../automation` 200，`repository=dockerun-test/demo` `tag_pattern=v*` |
| S4.5 推送触发自动部署 | **pass** | 推送 `demo:v3` 后任务 #3 `automatic=true` `image-event-3-env-1` → success |
| S4.6 外部仓库 | **skip** | 无外部仓库凭据 |
| S4 builtin catalog | **pass** | `GET /api/builtin/catalog` 200，仓库 `dockerun-test/demo` |

### S5 部署记录

| 用例 | 结果 | 证据 |
|---|---|---|
| S5.1 列表 | **pass** | `GET /api/deployments` 200 |
| S5.2 详情 | **pass** | 任务 #2 200，`status=success` |
| S5.3 CLI status | **pass** | `dockerun status` 退出码 0；`dockerun status 2` 退出码 0 |

### S6 离线发布包

| 用例 | 结果 | 证据 |
|---|---|---|
| S6.1 package | **pass** | `dockerun package` 退出码 0，产物 25 971 446 字节 |
| S6.2 import | **pass** | `dockerun import --wait` 退出码 0，bundle id=1 `status=ready`，`local_loaded=true` |
| S6.3 确认版本 | **pass** | `POST /api/bundles/1/version` 201，release `offline-v1` source=bundle |
| S6.4 从包部署 | **pass** | `dockerun deploy blank-app-2222 --env test --bundle ... --wait` 退出码 0，project `dockerun-a1-e1` 健康 |
| S6.5 显式入库 | **pass**（复测） | `POST /api/bundles/publish {"release_id":1}` 201，生成 `offline-v1-registry`，镜像 `127.0.0.1:18080/offline/app-1/image-0:release-1` |

### S7 官方 Docker 集成测试

```bash
cd backend
DOCKERUN_DOCKER_TESTS=1 go test ./internal/release -run TestDockerLocalAndOfflineDeployment -count=1 -v
```

**pass**：`--- PASS: TestDockerLocalAndOfflineDeployment (15.67s)`  
覆盖本机 Compose 部署、失败准备不影响旧服务、离线打包/校验/加载及严格 Image ID 部署。测试自建 `dockerun-a987654-*` 并清理。日志：`artifacts/evidence/s7_go_docker_test.log`。

## Cloud VM 未能覆盖的内容

| 项 | 说明 |
|---|---|
| 外部 Registry | 无凭据，按计划 skip |
| nginx-proxy 受管模板 | 发布工作区受管模板不含此项；旧 `templates/nginx-proxy` 未当发布模板执行 |
| MySQL/Redis 实际容器部署 | 模板创建已验证且明确不启动容器；完整中间件镜像未拉取，避免占用与宿主端口。实际部署使用 `nginx:1.28-alpine` 经内置仓库/离线包 |
| 浏览器完整 UI 矩阵 | Compose 栈已返回中文工作台 HTML（`lang="zh-CN"`），未做点击级浏览器回归 |
| systemd 管理的 Docker | 平台无 systemd，`dockerd` 手动启动 |
| 多架构 / 物理断网离线机 | 仅 linux/amd64，且 VM 可访问外网以下载镜像与工具链 |
| MySQL 后端 / OCI conformance | 非本次发布工作区范围 |

## 测试助手问题（非产品缺陷）

1. 首次 Compose 片段写了 `healthcheck.test: [CMD, true]`，服务端正确拒绝（400，`must be a string`）。改为 `[CMD, 'true']` 后通过。  
2. `POST /api/bundles/publish` 需要 `release_id`，误传 `bundle_id` 得到 400「请选择已导入发布包的版本」。改正后 201。

## 清理

测试结束后已停止并删除：

- Compose 项目 `dockerun-test-2222`
- 部署 project `dockerun-a1-e1`
- 本机 `dockerun-server`（18080）
- 隔离数据目录 `/tmp/dockerun-isolated-2222`、`/var/lib/dockerun-test-2222`
- 测试镜像标签 `dockerun:test-2222`、`127.0.0.1:18080/dockerun-test/demo:*`、`127.0.0.1:18080/offline/app-1/image-0:release-1`
- `docker logout 127.0.0.1:18080`

未改动仓库外业务容器或外部 Registry。

## 产物

- `artifacts/TEST_PLAN.md`
- `artifacts/TEST_RESULTS.md`（本文件）
- `artifacts/run_release_workspace_tests.py`（主 API/CLI 脚本）
- `artifacts/docker-compose.test-2222.yml`
- `artifacts/evidence/`（HTTP/CLI/日志；令牌已打码）
