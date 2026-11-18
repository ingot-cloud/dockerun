# Dockerun

[English](README.md) · **简体中文**

Dockerun 是面向自托管环境的轻量 Docker 管理与发布控制台，将私有镜像仓库、应用发布和离线交付集成在同一个界面中，适合在单台 Docker 主机或少量固定目标主机上部署和运维应用。

Dockerun 镜像内包含中文网页控制台、Go API、内嵌的 CNCF Distribution 镜像仓库、Docker CLI 和 Docker Compose。元数据默认保存在 SQLite，也可以连接已有 MySQL 实例。

[功能](#功能) · [从源码部署](#从源码部署) · [发布第一个应用](#发布第一个应用) · [本地开发](#本地开发) · [文档](#文档) · [许可证](#许可证)

## 功能

| 模块 | 支持的操作 |
| --- | --- |
| 容器 | 启动、停止、重启、强制停止、暂停、恢复和删除容器。查看实时日志、配置、进程和资源用量，打开执行或附加控制台。可重命名容器、调整重启策略、从容器创建镜像。 |
| 本地镜像 | 拉取、打标签、导入、导出和删除镜像，查看镜像信息、构建历史和使用该镜像的容器。 |
| 网络与卷 | 创建、查看和删除 Docker 网络及数据卷，连接或断开容器与网络，查看资源使用关系。 |
| 内置与外部镜像仓库 | 用标准 `docker login`、`push` 和 `pull` 访问内置私有仓库。浏览仓库、标签与摘要，在保留策略允许时删除镜像清单。可接入已有仓库、测试连接并加密保存凭据。 |
| 应用与部署环境 | 使用 Docker Compose 或 Bash 命令部署应用，定义配置文件及前后置脚本。按环境设置运行参数、独立脚本、关联容器及检查命令，提交前预览变更。 |
| 应用模板 | 从单服务应用、MySQL、Redis、Nacos、Kafka、Registry、Portainer、Nginx Proxy 和 Ingot 服务等预设开始，也可以另存自定义模板，预览、导入或导出标准 ZIP 模板包。 |
| 发布版本与部署记录 | 保存版本化的部署定义，查看任务进度、逐机结果、日志和健康检查。可重新部署历史版本或晋升到其他环境，核对实际运行镜像与 Image ID。无状态 Compose 环境满足条件且启用策略后支持自动恢复。 |
| 自动发布与远程主机 | 配置授权的镜像推送触发规则和外部仓库 Webhook，在 CI/CD 中使用受限令牌调用 CLI。支持本机或由 Agent 主动连接控制台的远程发布。固定主机组按顺序部署，失败时暂停后续目标。 |
| 离线交付 | 导入 `docker save` 归档，或包含镜像、Compose 和非敏感配置的完整发布包。支持分块续传及从配置的服务器目录导入，可通过 CLI 按目标平台打包。 |
| 账号与维护 | 管理用户、仓库权限、自动化令牌和 Agent 凭据。查看系统与 Docker 状态和审计事件。可手动运行历史脚本、下载完整备份、通过 CLI 恢复，以及在只读维护期间清理未引用的仓库镜像层。 |

Docker 资源页面操作当前连接的 Docker Engine，远程部署由 Agent 执行。固定主机组逐台发布，替换容器可能短暂停机。恢复涵盖部署定义和镜像，业务数据卷需单独备份。[一体化操作指南](docs/INTEGRATED_DEPLOYMENT.md) · [验证与兼容范围](docs/VALIDATION.md)

## 从源码部署

### 环境要求

- Linux Docker Engine 支持 API 1.44 或更高版本（Docker 25+），并安装 Docker Compose v2。
- 目标平台为 Linux `amd64` 或 `arm64`。跨平台构建需使用支持目标平台的构建器。
- 构建机能访问 Docker socket，并可联网下载基础镜像与依赖。Go、Node.js 和 pnpm 由镜像构建阶段安装。

### 构建并启动

以下命令在构建和部署 Dockerun 的宿主机上执行。

```bash
git clone https://github.com/ingot-cloud/dockerun.git
cd dockerun
cp .env.example .env
```

编辑 `.env`。在服务器本机使用 HTTP 体验时，可以这样配置。

```dotenv
PUBLIC_URL=http://localhost:8080
DOCKERUN_IMAGE=dockerun:2.0.0
DATABASE_TYPE=sqlite
```

从其他机器访问时，将 `PUBLIC_URL` 中的主机名改为服务器的实际域名或 IP。这个地址同时用于仓库认证，使用浏览器、Docker 客户端和 Docker daemon 均可访问的根地址，并按需带上端口。正式环境的 HTTPS 配置见下文。

```bash
docker compose config --quiet
docker compose build
docker compose up -d --wait --no-build --pull never
docker compose ps
docker compose logs --tail=100 dockerun
curl --fail http://127.0.0.1:8080/health
```

在浏览器打开配置的地址，首次访问时为 `admin` 设置密码并登录，然后在「系统信息」核对 Docker 连接状态。默认使用宿主机的 `8080` 端口，调整时同步更新 Compose 端口映射和相应的访问或代理地址。

### 正式环境、持久化与离线安装

- **HTTPS。** 使用受信任的证书，将 `PUBLIC_URL` 设为 HTTPS 根地址。反向代理应转发整个站点，包括 API、仓库、令牌接口、Webhook 和 Agent 连接，并保留流式响应、WebSocket 升级及仓库认证头。若由 Dockerun 直接提供 HTTPS，则配置 `TLS_CERT`、`TLS_KEY` 和相应的健康检查。HTTP 调试适用于受控网络，通过 Docker daemon 或构建器访问 HTTP 仓库时还需配置对应的 `insecure-registries`。Docker socket 赋予 Dockerun 操作宿主 Docker 资源的权限，仅向可信用户开放管理入口。[HTTPS 配置](docs/DEPLOYMENT.md#https-反向代理) · [安全说明](docs/SECURITY.md)
- **持久化。** Compose 默认将 `dockerun-data` 命名卷挂载到 `/var/lib/dockerun`，保存元数据、密钥、内置仓库及发布材料。绑定宿主目录时，将 `docker-compose.bind.yml` 与主 Compose 文件合用，并将 `DATA_DIR` 指向已有目录。使用已有 MySQL 时配置 `DATABASE_TYPE=mysql` 和 `MYSQL_DSN`，同时保留数据挂载。沿用 MySQL 和 nginx-proxy 的安装可使用 `deploy.sh`。[数据存储](docs/DATA_STORAGE.md) · [MySQL](docs/MYSQL_SETUP.md) · [部署脚本](docs/DEPLOY_SCRIPT.md)
- **离线安装控制台。** 在联网机器上为目标架构构建 Dockerun 镜像，用 `docker save` 导出。将归档、Compose 文件、`.env.example`、`LICENSE` 和 `NOTICE` 一并交付。在目标机执行 `docker load`，把 `DOCKERUN_IMAGE` 设为实际标签，然后运行 `docker compose up -d --no-build --pull never`。业务应用的镜像单独准备。[离线安装指南](docs/DEPLOYMENT.md#离线安装)
- **备份与升级。** 升级前在「仓库与维护」创建并下载完整备份，同时备份业务数据卷、外部仓库及安装配置。保留旧镜像标签，恢复时使用新的空目录或空卷。[备份与恢复](docs/DEPLOYMENT.md#备份与恢复) · [升级检查](docs/DEPLOYMENT.md#升级与检查)

## 发布第一个应用

1. 以管理员身份登录，创建应用及首个环境，也可以从「应用模板」开始。按模板支持的范围选择本机、Agent 或固定主机组。
2. 准备镜像。根据部署方式选择镜像地址、已配置仓库、目标主机已有镜像或离线材料。
3. 打开「发布应用」，检查 Compose 或部署命令、运行参数、`configs/` 下的配置文件、脚本和健康检查。
4. 预览并核对目标及变更，确认后开始发布。在「部署记录」跟踪日志、逐机结果和健康状态。

完整的 Compose 示例、镜像登录与推送方式及后续更新步骤见[快速开始](docs/QUICKSTART.md#发布一个-compose-应用)和[仓库指南](docs/REGISTRY_GUIDE.md)。

## 本地开发

使用 Go `1.26.0`、Node.js `22.17.0` 和 pnpm `10.12.4`。启动 Docker Desktop 或 Docker Engine 后，在仓库根目录执行以下命令。

```bash
make install-deps
# 终端一
make dev-backend
# 终端二
make dev-frontend
```

Vite 前端地址为 `http://localhost:5999`，后端监听 `127.0.0.1:8080`，开发数据位于 `data/local`。Make 默认使用 `PUBLIC_URL=http://host.docker.internal:8080`，适用于 Docker Desktop。本机 Linux Docker Engine 可使用 `make dev-backend PUBLIC_URL=http://localhost:8080`。

```bash
make build
make run
```

`make build` 产出 `bin/dockerun-server` 和 `bin/dockerun` CLI。`make run` 在 `http://localhost:8080` 提供构建后的网页。使用 CLI 连接服务时，设置 `DOCKERUN_URL`，通过 `DOCKERUN_TOKEN` 提供具有部署权限的令牌。CLI 支持 `deploy`、`package`、`import`、`promote`、`rollback` 和 `status`。[开发指南](docs/DEVELOPMENT.md) · [CLI 与 Agent](docs/INTEGRATED_DEPLOYMENT.md#cli-与-agent)

## 文档

| 文档 | 内容 |
| --- | --- |
| [快速开始](docs/QUICKSTART.md) | 首次启动、管理员初始化及 Compose 发布示例 |
| [部署指南](docs/DEPLOYMENT.md) · [配置参考](docs/CONFIGURATION.md) | HTTPS、持久化、环境变量、离线安装、备份与升级 |
| [一体化操作与接口](docs/INTEGRATED_DEPLOYMENT.md) · [模板格式](templates/README.md) | 发布、自动化、离线包、CLI、Agent 和模板包 |
| [仓库指南](docs/REGISTRY_GUIDE.md) · [安全说明](docs/SECURITY.md) | 内置与外部仓库、账号、令牌和权限边界 |
| [开发指南](docs/DEVELOPMENT.md) · [构建与交付](docs/BUILD_AND_DEPLOY.md) | 本地开发、验证、二进制与多平台镜像 |
| [故障排查](docs/TROUBLESHOOTING.md) · [验证记录](docs/VALIDATION.md) · [文档索引](docs/README.md) | 常见问题、已验证范围及其他专题指南 |

## 许可证

Dockerun 采用 [Apache License, Version 2.0](LICENSE)。[NOTICE](NOTICE) 保留历史 MIT 许可证声明，并标明 Nacos 建表脚本的第三方许可。
