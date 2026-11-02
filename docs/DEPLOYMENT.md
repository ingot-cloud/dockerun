# 部署与运维指南

本文部署的是 Dockerun 控制台。业务应用的发布、离线包和 Agent 操作见[一体化操作指南](INTEGRATED_DEPLOYMENT.md)，首次体验见[快速开始](QUICKSTART.md)。所有 Compose 命令在安装目录执行。

## 安装方式与前置条件

使用 Linux Docker Engine（API 1.44+，Docker 25+）和 Docker Compose v2。支持 `linux/amd64`、`linux/arm64`；构建机与目标平台不同时，使用具有对应平台构建能力的 Buildx builder。

默认交付为一个容器，内含网页、API、Distribution、Docker CLI、Compose 和数据库备份工具。默认 SQLite，不需要独立 Registry 或数据库容器。MySQL 可选，见[数据库配置](MYSQL_SETUP.md)。Docker Engine 必须提前安装，运行时不会自动安装依赖或从 CDN 加载前端资源。

```bash
git clone https://github.com/ingot-cloud/dockerun.git
cd dockerun
cp .env.example .env
```

编辑 `.env`，确定 `PUBLIC_URL`、`DOCKERUN_IMAGE` 和数据库类型，再选择下面的 HTTPS 方案。`PUBLIC_URL` 必须是客户端及目标 Docker daemon 可达的根地址，不含路径、凭据或查询参数。已有反向代理时，不设置 `TLS_CERT`、`TLS_KEY`，由代理终止 TLS。

```bash
docker compose config --quiet
docker compose build
docker compose up -d --no-build --pull never
docker compose ps
docker compose logs --tail=100 dockerun
```

仓库没有在此文档中约定可直接拉取的公共发布镜像。使用自行构建或交付的实际标签，不要把示例域名或版本当作已发布地址。仅内网 HTTP 体验可使用 `PUBLIC_URL=http://服务器IP:8080`，但 Docker 仓库客户端还需配置 HTTP 信任，见[开发指南](DEVELOPMENT.md#本地-http-仓库调试)。

## HTTPS 反向代理

已有 nginx-proxy、acme-companion 和 MySQL 的安装，可使用根目录 `deploy.sh` 与独立环境配置，沿用原域名、网络和数据库。参数、代理上传配置及旧版本升级步骤见[单容器部署脚本](DEPLOY_SCRIPT.md)。该入口使用 `docker create/start` 管理容器，与下面的 Compose 安装方式择一使用。

以下假定 Nginx 在同一宿主机运行，域名为 `dockerun.example.com`，DNS 已指向服务器，并已准备覆盖该域名的证书和私钥。

在 `.env` 设置：

```dotenv
PUBLIC_URL=https://dockerun.example.com
DOCKERUN_IMAGE=dockerun:2.0.0
DATABASE_TYPE=sqlite
```

将 `docker-compose.yml` 原有的端口映射**替换**为以下内容，避免绕过代理直接访问管理入口；不要在覆盖文件中简单追加另一条映射：

```yaml
services:
  dockerun:
    ports:
      - "127.0.0.1:8080:8080"
```

以下是 Nginx `http` 上下文中的配置示例，替换证书路径后使用。若 Nginx 在独立容器或其他主机中运行，需按实际网络修改上游地址，容器内的 `127.0.0.1` 不指向宿主 Dockerun。

```nginx
map $http_upgrade $connection_upgrade {
    default upgrade;
    '' close;
}

server {
    listen 80;
    server_name dockerun.example.com;
    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl;
    server_name dockerun.example.com;
    ssl_certificate /etc/nginx/certs/dockerun.fullchain.pem;
    ssl_certificate_key /etc/nginx/certs/dockerun.key;

    client_max_body_size 0;
    client_body_timeout 3600s;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Host $http_host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection $connection_upgrade;
        proxy_request_buffering off;
        proxy_buffering off;
        proxy_read_timeout 3600s;
        proxy_send_timeout 3600s;
    }
}
```

代理整个根入口，覆盖网页、`/api/`、`/v2/`、`/registry/token`、`/agent/` 和 `/webhooks/`。保留客户端的 `Authorization` 和上游的 `WWW-Authenticate`、`Location` 响应头，不为 Registry 路由追加独立网页登录认证。上传大小、超时和流式响应设置也需在前置网关中保持一致。

```bash
# 在安装 Nginx 的宿主机检查配置后加载
sudo nginx -t
sudo systemctl reload nginx
# 在安装目录应用 Compose 配置
docker compose up -d --no-build --pull never
curl --fail https://dockerun.example.com/health
```

首次从受控入口初始化 `admin`。初始化后，`curl -i https://dockerun.example.com/v2/` 未带认证应返回带 `WWW-Authenticate` 的 `401`，其中 realm 应为 `https://dockerun.example.com/registry/token`。再执行 `docker login -u admin dockerun.example.com` 验证仓库认证。

## 直接启用 TLS

服务也可直接提供 HTTPS。将证书链和私钥放到数据卷或绑定目录中的 `keys/` 下，容器内路径例如 `/var/lib/dockerun/keys/tls.crt`、`tls.key`，并限制私钥读取权限。保持仓库默认的 `8080:8080` 映射，在 `.env` 设置：

```dotenv
PUBLIC_URL=https://dockerun.example.com:8080
TLS_CERT=/var/lib/dockerun/keys/tls.crt
TLS_KEY=/var/lib/dockerun/keys/tls.key
```

证书需覆盖实际域名。当前 Dockerfile 默认用 HTTP 探测 `/health`，直接启用 TLS 时，必须在 `docker-compose.yml` 的 `dockerun` 服务中增加 HTTPS 健康检查，否则服务正常也会显示不健康：

```yaml
services:
  dockerun:
    healthcheck:
      test: ["CMD", "curl", "--fail", "--resolve", "dockerun.example.com:8080:127.0.0.1", "https://dockerun.example.com:8080/health"]
      interval: 30s
      timeout: 5s
      retries: 3
```

此处域名用于证书校验，`--resolve` 仅让容器内探测连接回环地址。私有 CA 场景需将 CA 证书放入容器可读位置，在探测命令中追加 `--cacert` 及 CA 路径，并让浏览器、Docker daemon、Buildx 和 Agent 分别信任该 CA。不要以跳过校验替代证书信任。

```bash
docker compose config --quiet
docker compose up -d --no-build --pull never
curl --fail https://dockerun.example.com:8080/health
docker login -u admin dockerun.example.com:8080
```

## 离线安装

Dockerun 自身镜像归档与业务应用的完整发布包是两种材料。安装控制台用 `docker save` / `docker load`；业务发布包由 Dockerun CLI 打包并在网页或 CLI 导入。

### 联网机器准备

以下为目标 `linux/amd64` 的示例；ARM64 将平台改为 `linux/arm64`、文件名中的 `amd64` 改为 `arm64`，镜像标签保持 `dockerun:2.0.0`。

```bash
docker buildx build --platform linux/amd64 --tag dockerun:2.0.0 --load .
mkdir -p dist/dockerun-install
docker save --output dist/dockerun-install/dockerun-2.0.0-amd64.tar dockerun:2.0.0
cp docker-compose.yml docker-compose.bind.yml .env.example dist/dockerun-install/
```

交付 `dist/dockerun-install/` 以及实际使用的代理配置、证书和说明。模板及前端静态资源已在镜像中，离线目标无需源码或构建工具链。安装配置中的密码、私钥和环境密钥应通过受控渠道交付。

### 目标机器启动

目标已安装可用的 Docker Engine 和 Compose v2。进入交付目录：

```bash
docker load --input dockerun-2.0.0-amd64.tar
cp .env.example .env
```

编辑 `.env`，设置真实入口及交付标签：

```dotenv
PUBLIC_URL=https://dockerun.example.com
DOCKERUN_IMAGE=dockerun:2.0.0
DATABASE_TYPE=sqlite
```

按前文配置 HTTPS 代理或直接 TLS，再启动：

```bash
docker image inspect dockerun:2.0.0 --format '{{.Os}}/{{.Architecture}}'
docker compose config --quiet
docker compose up -d --no-build --pull never
docker compose ps
```

即使交付目录没有 Dockerfile，只要镜像已加载且标签匹配，`--no-build --pull never` 也不会构建或联网拉取。业务应用的镜像仍需单独准备，不能据此认为所有应用依赖都已离线就绪。

## 持久化与权限

默认使用 `dockerun-data` 命名卷，由 Docker 管理宿主存储位置；容器内数据目录始终为 `/var/lib/dockerun`。新安装无需创建宿主目录，在 `.env` 用 `DATA_VOLUME` 自定义卷名。

沿用已有宿主目录时，在 `.env` 设置：

```dotenv
COMPOSE_FILE=docker-compose.yml:docker-compose.bind.yml
DATA_DIR=/ingot-data/dockerun
```

确保该目录存在、可写且包含原数据，再运行 `docker compose config` 核对挂载。覆盖文件将宿主目录绑定到固定容器路径，并传入 `HOST_DATA_DIR`，后端在应用发布时转换 Docker 挂载源。**已有 Compose 目录绑定安装升级时必须保留原目录并启用这个覆盖文件，不能直接切到默认空卷。** `deploy.sh` 的旧配置只设置 `DATA_DIR` 时自动沿用绑定模式。

| 数据 | 默认位置或用途 |
| --- | --- |
| SQLite 元数据 | `/var/lib/dockerun/dockerun.db` |
| `keys/` | 自动生成的网页、Registry、HTTP 状态和凭据加密密钥；也可放置 TLS 文件 |
| `registry/` | 内置 Registry 镜像内容和上传状态 |
| `uploads/`、`releases/` | 已上传归档与解包的发布材料 |
| `jobs/` | 任务快照、渲染配置和恢复材料 |
| `backups/` | 完整备份及迁移前数据库快照 |
| `imports/` | 服务器目录导入入口，默认只读取该目录内的文件 |

MySQL 模式同样需要持久化文件数据。两种模式都保留密钥、仓库及任务材料；更改卷名或宿主目录不会自动复制原数据。数据查看、模式切换和注意事项见[数据存储指南](DATA_STORAGE.md)。

Docker socket 提供高权限管理能力。不要通过开放 socket 的全局写权限解决连接问题，按服务身份配置访问。当前安装方式按单个控制台实例设计，不要通过增加副本共享同一数据目录实现高可用。

## 备份与恢复

管理员从「仓库与维护」创建完整备份，等待任务成功并下载文件。备份窗口暂停相关本机执行及导入，Registry 只读；任务离开页面后继续运行。SQLite 使用一致性快照，MySQL 使用 `mysqldump --single-transaction`。

完整备份包含元数据、密钥、内置仓库、发布材料和任务材料；**不包含业务数据卷、外部 Registry 内容及整个安装目录**。另外保存业务备份、`.env`、Compose、代理配置和显式设置的外部密钥；`imports/` 中尚未导入的文件及历史备份也应按需另存。

恢复时先停止 Dockerun，目标必须是新的空目录或空卷。下面使用命名卷；备份文件实际位于宿主 `/backup/dockerun/dockerun-backup.tar.gz`，镜像标签与安装一致，`dockerun-restored` 必须是尚未使用的新卷名：

```bash
docker compose stop dockerun
docker volume create dockerun-restored
docker run --rm --pull never --network none --entrypoint dockerun \
  --mount type=bind,src=/backup/dockerun,dst=/restore,readonly \
  --mount type=volume,src=dockerun-restored,dst=/var/lib/dockerun \
  dockerun:2.0.0 restore \
  --file /restore/dockerun-backup.tar.gz --data-dir /var/lib/dockerun
```

SQLite 恢复成功后，将 `.env` 的 `DATA_VOLUME` 改为 `dockerun-restored`，再运行 `docker compose up -d --no-build --pull never`；保留原卷以便核对。目录绑定恢复时，先创建新宿主空目录，将示例的第二个挂载改为 `type=bind,src=/ingot-data/dockerun-restored,dst=/var/lib/dockerun`，成功后保持 bind 覆盖文件，将 `.env` 的 `DATA_DIR` 指向恢复目录。使用 `deploy.sh` 时修改对应的 `DATA_VOLUME` 或 `DATA_DIR` 后执行 `./deploy.sh update`。

MySQL 恢复命令会解出卷或目录中的 `metadata.sql`，并返回需手动恢复数据库的提示及非零退出码；这一步只恢复材料。将 SQL 恢复到专用数据库、配置正确的 `MYSQL_DSN` 后再切换存储并启动。可通过只读辅助容器取出 SQL，见[数据存储指南](DATA_STORAGE.md)。保留原安装通过环境变量提供的密钥，否则令牌或保存的仓库凭据可能失效。恢复详情见[数据库指南](MYSQL_SETUP.md)。

## 升级与检查

1. 暂停新的发布操作，完成并下载完整备份，同时备份业务数据及安装配置。
2. 保留当前源码版本、安装配置和旧镜像标签。联网环境检出计划升级的代码版本后构建新标签；离线环境导入交付的新镜像。
3. 在 `.env` 更新 `DOCKERUN_IMAGE`，执行 `docker compose config --quiet` 和 `docker compose up -d --no-build --pull never`。
4. 检查服务日志、`/health`、网页登录、仓库登录，以及应用与任务记录。

启动可能执行数据库迁移。失败时核对日志；仅切回旧镜像不保证旧程序能读取迁移后的数据库，需要时使用升级前完整备份恢复。旧 `scripts/backup.sh`、`restore.sh`、`update.sh` 基于旧目录和 Compose 命令，不作为当前一体化版本的运维入口。

更多排查见[故障排查](TROUBLESHOOTING.md)，完整变量见[配置参考](CONFIGURATION.md)。
