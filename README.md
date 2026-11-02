# Dockerun

当前版本：`2.0.0`。默认安装镜像标签为 `dockerun:2.0.0`，手动构建及离线打包见[构建与交付](docs/BUILD_AND_DEPLOY.md#构建-dockerun-镜像)。

Dockerun 是面向自托管 Docker 环境的轻量部署与运行管理平台，将 **Docker 资源管理、私有镜像仓库、应用发布和离线交付** 集成到同一个中文网页中。适合在单台服务器或少量固定主机上管理业务服务、中间件及测试、生产等部署环境。

一个 Dockerun 容器内包含管理网页、Go API、内嵌的 CNCF Distribution v3.1.2、Docker CLI 和 Compose。默认使用 SQLite，无需另装数据库或启动独立 Registry；也可以连接 MySQL 和已有外部镜像仓库。远程主机通过主动连接控制台的 Agent 执行发布。

## 可以做什么

| 能力 | 使用方式 |
| --- | --- |
| Docker 资源管理 | 查看和操作容器、日志、控制台、镜像、网络及数据卷 |
| 内置与外部镜像仓库 | 使用网页浏览镜像；通过标准 `docker login`、`push`、`pull` 交付镜像 |
| 应用与环境发布 | 保存部署命令或 Compose、运行参数、配置文件及前后置脚本，按环境预览和发布 |
| 应用模板 | 从单服务应用、MySQL、Redis、Nacos 等模板开始，保存自定义模板 |
| 版本与任务追踪 | 保留镜像身份、配置快照、执行日志和健康结果，将已有版本晋升到其他环境 |
| 离线交付 | 导入 `docker save` 镜像归档或包含镜像、Compose、非敏感配置的完整发布包，支持分块续传及服务器目录导入 |
| 自动化与多主机 | 使用受限令牌、CLI、镜像自动发布和 Agent，按固定主机组顺序执行任务 |
| 用户与维护 | 管理员维护系统；普通用户按无权限、只读、读写访问内置仓库；支持完整备份和仓库维护 |

发布前先准备镜像和检查资源，受管 Compose 发布会锁定 digest 或 Image ID。普通容器替换可能短暂停机；自动恢复仅适用于明确无状态且开启策略的环境，不回滚数据库及业务数据卷。固定主机组用于顺序发布，不提供跨主机集群编排或零停机保证。协议兼容范围见[验证记录](docs/VALIDATION.md)。

## 部署 Dockerun

### 环境要求

- 运行中的 Linux Docker Engine，支持 API 1.44+（Docker 25+），以及 Docker Compose v2。
- 交付镜像支持 `linux/amd64`、`linux/arm64`，镜像与目标主机架构需匹配。容器内置 Docker CLI 29.1.3 和 Compose。
- 可使用 Docker socket，并能读写持久化目录 `/var/lib/dockerun`。该目录在容器和宿主机上必须保持同一绝对路径，以便宿主 Docker 找到 Compose 的配置及挂载文件。
- 从源码构建镜像需要联网拉取基础镜像和依赖；仅运行交付镜像无需安装 Node、pnpm 或 Go。目标主机仍须预先安装 Docker Engine。

Docker socket 赋予 Dockerun 管理宿主容器和挂载目录的能力，应只向可信管理员开放管理入口。正式部署使用 HTTPS，并在开放入口前完成管理员初始化。

### 从源码构建并启动

以下以 Linux 服务器首次部署为例，命令在仓库根目录执行：

```bash
git clone https://github.com/ingot-cloud/dockerun.git
cd dockerun
cp .env.example .env
sudo install -d -m 700 /var/lib/dockerun
```

编辑 `.env`，将 `PUBLIC_URL` 改为浏览器、Docker 客户端及 Docker daemon 都能访问的实际入口。首次在受控网络内用 HTTP 体验时，可使用：

```dotenv
PUBLIC_URL=http://服务器IP:8080
DOCKERUN_IMAGE=dockerun:2.0.0
DATABASE_TYPE=sqlite
```

将示例中的“服务器IP”替换为实际 IP。仅在服务器本机访问时可以使用 `http://localhost:8080`；远程客户端不能使用服务器的 `localhost`。`PUBLIC_URL` 只能是完整根地址，不能带 `/dockerun` 等路径。

```bash
docker compose config --quiet
docker compose build
docker compose up -d --no-build --pull never
docker compose ps
docker compose logs --tail=100 dockerun
curl --fail http://127.0.0.1:8080/health
```

在浏览器打开配置的地址，例如 `http://服务器IP:8080`。首次访问为 `admin` 设置密码，再登录管理页面；没有预设管理员密码。`/health` 返回 `{"status":"ok"}` 说明 HTTP 服务可用，登录后还需在系统页面确认 Docker 连接正常。

默认 Compose 将宿主 `8080` 映射到容器 `8080`，网页、`/api/`、`/v2/`、`/registry/token`、Agent 和 Webhook 共用入口。修改宿主端口时，同时修改 Compose 的 `ports` 与 `PUBLIC_URL`。

### 正式部署与离线安装

- **HTTPS 反向代理**：将 `PUBLIC_URL` 设为 `https://dockerun.example.com`，代理整个入口，并保留 Registry 认证头、流式日志及 WebSocket。Nginx 示例见[部署指南](docs/DEPLOYMENT.md#https-反向代理)。只改 `PUBLIC_URL` 不会自动启用 TLS。
- **服务直接提供 HTTPS**：同时设置 `TLS_CERT`、`TLS_KEY`，配置证书及对应的 HTTPS 健康检查，见[直接启用 TLS](docs/DEPLOYMENT.md#直接启用-tls)。
- **离线安装**：在联网机器准备同平台的 Dockerun 镜像，`docker save` 后交付到目标机，执行 `docker load`，再按实际镜像标签设置 `DOCKERUN_IMAGE` 并启动；完整命令见[离线安装](docs/DEPLOYMENT.md#离线安装)。

HTTP 环境浏览网页无需调整 Docker 配置，但通过 Docker 登录、拉取或推送 HTTP 仓库时，相关 daemon 和构建器还需配置该地址为 `insecure-registries`。正式环境使用可信 HTTPS；本地调试步骤见[开发指南](docs/DEVELOPMENT.md#本地-http-仓库调试)。

## 开始使用

### 管理 Docker 资源

管理员登录后，在容器、镜像、网络和卷页面查看宿主机资源。容器详情提供日志、进程、统计和控制台；镜像支持拉取、导入、导出和标签管理。这里操作的是所连接 Docker Engine 的实际资源。

### 发布第一个应用

1. 打开「发布工作区 → 应用模板」，选择模板并填写应用名称、首个环境、目标主机及运行参数；也可以自行创建应用和环境。
2. 准备镜像：在线发布使用可访问的镜像地址、内置或外部仓库；离线发布先导入镜像归档或完整发布包。
3. 进入「发布应用」，选择部署命令或 Compose，检查环境参数、`configs/` 配置文件、前后置脚本及健康检查。
4. 点击「预览发布」，核对镜像、目标和变更，再「确认发布」。创建模板、保存配置、导入材料及预览本身不会启动业务容器。
5. 在发布任务中查看日志和结果；运行状态与健康状态分别展示。没有健康检查时，只能确认服务已启动。

可操作的 Nginx Compose 示例见[快速开始](docs/QUICKSTART.md#发布一个-compose-应用)。应用模板默认使用内置仓库，需先将镜像推送到所选仓库，或改用按镜像地址拉取的 Compose 配置。中间件默认手动更新，数据卷按环境保留。

### 使用内置镜像仓库

网页和内置仓库共用账号。以已配置可信 HTTPS 的 `dockerun.example.com` 为例：

```bash
docker login -u admin dockerun.example.com
docker pull nginx:1.28-alpine
docker tag nginx:1.28-alpine dockerun.example.com/demo/nginx:1.28-alpine
docker push dockerun.example.com/demo/nginx:1.28-alpine
```

`docker login` 输入初始化时设置的密码，或以相应账号创建、具有所需权限的自动化令牌。仓库地址不包含 `https://`；非默认端口需写入地址。普通用户需要管理员授予仓库权限，不能部署应用或管理宿主资源。内置仓库、宿主本地镜像和离线材料分别存储，导入镜像不等于推送入库。详见[仓库指南](docs/REGISTRY_GUIDE.md)。

## 日常运维

```bash
docker compose ps                         # 查看状态
docker compose logs -f --tail=100 dockerun # 跟踪日志，Ctrl+C 退出查看
docker compose restart dockerun           # 重启服务
docker compose stop dockerun              # 停止服务
docker compose up -d --no-build --pull never # 使用已有镜像启动
docker compose down                       # 移除服务容器，保留绑定目录数据
```

从「仓库与维护」创建完整备份并下载，另行备份业务数据卷、外部仓库和安装配置。升级前先备份并保留旧镜像，再构建或导入新镜像、更新 `DOCKERUN_IMAGE` 并重新启动。不要只复制运行中的 SQLite 文件作为完整备份。恢复要求新的空目录，详见[备份、恢复及升级](docs/DEPLOYMENT.md#备份与恢复)。

## 本地开发与 CLI

源码工具链固定为 Go 1.26.0、Node 22.17.0、pnpm 10.12.4；Go 可使用支持自动工具链下载的版本引导，`GOTOOLCHAIN` 不要设为 `local`。先启动 Docker Desktop 或 Docker Engine，再执行：

```bash
make install-deps
# 终端一：后端 API、内置仓库与执行器，127.0.0.1:8080
make dev-backend
# 终端二：前端热更新，localhost:5999
make dev-frontend
```

访问 [http://localhost:5999](http://localhost:5999)。开发数据位于 `data/local`，开发命令不自动加载部署用的 `.env`。本地 Make 默认 `PUBLIC_URL=http://host.docker.internal:8080`，适合 Docker Desktop；Linux 本机使用 `make dev-backend PUBLIC_URL=http://localhost:8080`。端口、代理、Go 下载代理和 HTTP 仓库配置见[开发指南](docs/DEVELOPMENT.md)。

```bash
make build
make run
```

构建产生 `bin/dockerun-server` 和 `bin/dockerun`；`make run` 在 [http://localhost:8080](http://localhost:8080) 提供构建后的网页。CLI 使用示例：

```bash
export DOCKERUN_URL=https://dockerun.example.com
# 通过当前进程环境设置 DOCKERUN_TOKEN，使用个人账号创建的 deploy 令牌。
# 先在网页创建 gateway 应用及 test、prod 环境。
./bin/dockerun deploy gateway --env test --file compose.yaml --build --wait
./bin/dockerun package gateway --file compose.yaml --platform linux/amd64 --output gateway-release.tar.gz
./bin/dockerun deploy gateway --env prod --bundle gateway-release.tar.gz --wait
./bin/dockerun promote gateway --release 12 --to prod --wait
```

示例中的应用、文件和版本 ID 需替换为实际值。在线构建要求本机已有 Docker、Compose / Buildx 和仓库登录配置；离线打包要求已准备全部目标平台镜像。普通用户不能取得 deploy 权限。详见[一体化操作指南](docs/INTEGRATED_DEPLOYMENT.md)。

## 文档导航

| 文档 | 内容 |
| --- | --- |
| [快速开始](docs/QUICKSTART.md) | 首次启动、初始化和第一个应用 |
| [部署指南](docs/DEPLOYMENT.md) | HTTPS、离线安装、持久化、备份与升级 |
| [配置参考](docs/CONFIGURATION.md) | 环境变量、密钥及 Compose 配置方式 |
| [一体化操作指南](docs/INTEGRATED_DEPLOYMENT.md) | 应用发布、自动发布、离线包、CLI、Agent 及接口 |
| [开发指南](docs/DEVELOPMENT.md) / [构建与交付](docs/BUILD_AND_DEPLOY.md) | 开发启动、检查、二进制与多架构镜像 |
| [安全说明](docs/SECURITY.md) / [故障排查](docs/TROUBLESHOOTING.md) | 权限、凭据、认证与常见错误 |
| [完整文档索引](docs/README.md) | 专题指南、历史设计及维护记录 |
| [路线与实施记录](ROADMAP.md) / [验证记录](docs/VALIDATION.md) | 实施进度、验证依据及已知边界 |

MIT License · Ingot Cloud
