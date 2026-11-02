# 单容器部署脚本

根目录 `deploy.sh` 用于在目标 Docker 主机上安装和管理 Dockerun，适合沿用已有 MySQL、nginx-proxy 和 acme-companion 的部署。镜像需先构建、导入或拉取；脚本不构建镜像、不执行 `git pull`，也不修改现有代理容器。

## 文件与配置

| 文件 | 用途 |
| --- | --- |
| `deploy.sh` | 部署和管理命令 |
| `deploy.env.example` | 可提交的配置示例，不含真实密码 |
| `deploy.env` | 实际部署参数；Git 忽略且排除在镜像构建上下文之外 |

首次使用复制示例；已有 `deploy.env` 时直接编辑，勿覆盖原配置：

```bash
cp deploy.env.example deploy.env
chmod 600 deploy.env
```

配置每行一个 `KEY=value`，可用单引号或双引号包住整个值。引号内的 `$`、反斜杠和空格按字面量保留，不执行命令替换、转义或变量插值。注释单独占一行，不支持跨行值。该格式由部署脚本读取，不应使用 `source` 或直接交给 Compose、Docker 的 `--env-file`。

已导出的同名环境变量优先，包括空值，例如禁用宿主端口映射：

```bash
HOST_PORT= ./deploy.sh update
# 使用另一份配置文件
CONFIG_FILE=/etc/dockerun/deploy.env ./deploy.sh check
```

自定义配置文件应保存在仓库之外，或命名为根目录的 `deploy.prod.env` 等 `deploy.*.env` 文件；这些名称同样被 Git 和镜像构建上下文忽略。不要在共享终端启用 `bash -x`，其中可能出现配置凭据。

## 部署参数

| 变量 | 说明 |
| --- | --- |
| `CONTAINER_NAME` | 默认 `dockerun`；升级时与原容器名称一致 |
| `IMAGE_NAME` | 默认 `dockerun:2.0.0`，必须与构建、加载或推送的标签一致 |
| `PULL_POLICY` | 默认 `never`，只使用本地镜像；`missing` 缺失时拉取，`always` 部署前拉取 |
| `DOCKER_NETWORK` | 已存在的 Docker 网络，默认 `ingot-net`；与 nginx-proxy 共享 |
| `ADDITIONAL_NETWORKS` | 可选的附加网络，用空格分隔，例如 MySQL 所在网络 |
| `DOCKER_SOCKET` | 默认 `/var/run/docker.sock`；须是目标主机的 Unix socket |
| `DATA_MOUNT_TYPE` | 新安装默认 `volume`；可选 `bind`，旧配置只填 `DATA_DIR` 时自动选 `bind` |
| `DATA_VOLUME` | 命名卷名称，默认 `dockerun-data`；首次部署不存在时创建 |
| `DATA_DIR` | 仅在 `bind` 模式表示宿主机目录，默认 `/var/lib/dockerun`；容器内始终挂载到 `/var/lib/dockerun` |
| `PUBLIC_URL` | 客户端与 Docker daemon 可达的完整根地址，例如 `https://dockerun.example.com` |
| `VIRTUAL_HOST` | nginx-proxy 使用的单个域名，与 `PUBLIC_URL` 的域名一致 |
| `LETSENCRYPT_HOST` | acme-companion 2.6 自动证书域名；与 `VIRTUAL_HOST` 一致，已有证书可留空 |
| `LETSENCRYPT_EMAIL` | 可选的证书联系邮箱，未设置时沿用现有 companion 配置 |
| `HOST_PORT` | 可选宿主端口；示例默认留空，仅通过 nginx-proxy 访问 |
| `HOST_BIND_IP` | 发布宿主端口时使用的 IPv4 地址，默认 `127.0.0.1` |
| `DATABASE_TYPE` | 示例为 `mysql`；新安装也可选择 `sqlite` |
| `MYSQL_DSN` | MySQL 模式必填；升级时填写原数据库连接，不要改连空数据库 |
| `STARTUP_TIMEOUT` | 健康检查等待秒数，默认 600，包含首次数据库备份和迁移时间 |
| `STOP_TIMEOUT` | 停止容器的等待秒数，默认 30 |
| `LOG_TAIL` | 日志初始显示行数，默认 100 |
| `RESTART_POLICY` | 默认 `unless-stopped` |

签名和凭据密钥默认生成并保存在容器 `/var/lib/dockerun/keys`，随数据卷或目录持久化。原安装显式设置过 `JWT_SECRET`、`REGISTRY_SECRET`、`REGISTRY_HTTP_SECRET`、`CREDENTIALS_SECRET`、`AES_KEY` 或 `AES_IV` 时，继续在配置中填写原值。它们会按名称传给 Docker，不把值拼到命令行。

使用 MySQL 仍需保留整个数据卷或绑定目录，其中包含内置仓库镜像、密钥、上传材料和任务文件。原来使用 `/ingot-data/dockerun` 的安装继续设置 `DATA_MOUNT_TYPE=bind`、`DATA_DIR=/ingot-data/dockerun`；旧配置未设置模式时同样按 bind 处理，无需搬迁。

如果未指定模式、`DATA_DIR` 和 `DATA_VOLUME`，脚本会检查同名现有容器挂载到 `/var/lib/dockerun` 的数据源，并继续使用原目录或卷；没有该挂载时才使用默认命名卷。显式设置的配置优先，不自动复制数据。查看数据和迁移方法见[数据存储指南](DATA_STORAGE.md)，应用发布的路径处理见[脚本说明](INTEGRATED_DEPLOYMENT.md#应用和环境的部署脚本)。

## 沿用 nginx-proxy

Dockerun 的网页、API、内置仓库和令牌入口统一监听容器内 `8080`。脚本设置 `VIRTUAL_PORT=8080`，由现有 nginx-proxy 负责 HTTPS；不在 Dockerun 中启用第二层 TLS，也不需要额外开放 `5000`。

域名 DNS 应已指向代理服务器，`DOCKER_NETWORK` 与代理共享网络。MySQL 如果位于不同网络，将其网络加入 `ADDITIONAL_NETWORKS`。脚本检查网络存在，不自动创建同名空网络或改动代理。

在 nginx-proxy 已挂载到 `/etc/nginx/vhost.d` 的宿主目录中，创建以实际域名命名的文件，例如 `dockerun.example.com`，适配镜像和发布包上传：

```nginx
client_max_body_size 0;
client_body_timeout 3600s;
proxy_request_buffering off;
proxy_buffering off;
proxy_read_timeout 3600s;
proxy_send_timeout 3600s;
```

修改已经被 include 的域名文件后，检查并 reload 即可。新增文件时，先按现有 nginx-proxy 部署方式触发 docker-gen 重新生成配置，确认 `nginx -T` 中出现对应 include 后再加载；仅 reload 不会补出缺少的 include。不要覆盖生成的 `location /`：

```bash
docker exec nginx-proxy nginx -t
docker exec nginx-proxy nginx -s reload
```

域名级自定义文件及共享网络遵循 [nginx-proxy 1.8 文档](https://github.com/nginx-proxy/nginx-proxy/blob/1.8.0/docs/README.md#per-virtual_host)，自动证书参数沿用 [acme-companion 2.6 文档](https://github.com/nginx-proxy/acme-companion/blob/v2.6.0/README.md#step-3---proxied-containers)。代理整个根入口，保留认证头和 WebSocket，不给 `/v2/` 另加一套网页登录认证。

部署后用实际域名验证：

```bash
curl --fail https://dockerun.example.com/health
curl -i https://dockerun.example.com/v2/
docker login dockerun.example.com
```

`/v2/` 未认证返回带 `WWW-Authenticate` 的 `401` 是正常结果，其认证地址应是该域名下的 `/registry/token`。仓库使用 Dockerun 网页账号或有对应权限的自动化令牌。

## 安装与升级

新安装准备镜像、修改 `deploy.env` 后执行：

```bash
./deploy.sh check
./deploy.sh start
./deploy.sh status
```

从旧版本升级时，先独立备份原 MySQL、原文件数据目录和部署配置，在数据库副本验证迁移。不要同时让新旧 Dockerun 实例连接同一原数据库进行升级测试；旧版 `scripts/backup.sh` 不会备份 MySQL。新版本包含首次迁移前的 SQL 备份，但不能代替独立备份和恢复验证。迁移会保留旧脚本记录并关闭旧脚本自动部署，新自动发布规则在应用环境中配置。

确认 `CONTAINER_NAME`、`MYSQL_DSN` 和数据卷或宿主目录与原安装一致，加载目标架构的镜像，然后执行：

```bash
docker load --input docker-packages/dockerun-2.0.0-amd64.tar
./deploy.sh check
./deploy.sh update
```

ARM64 使用对应的 `dockerun-2.0.0-arm64.tar`。若原镜像也使用其他仓库前缀，修改 `IMAGE_NAME` 或先把加载的镜像重新标记为配置的名称。

`check` 不拉取镜像、不创建数据卷或目录、不启动容器，也不连接或迁移 MySQL；它验证配置、本地镜像及其与 Docker 服务的平台匹配、Docker socket 和网络。`update` 先准备镜像、创建尚未启动的新容器并连接附加网络，再停止原容器，将其保留为 `容器名-previous-时间-进程号`，最后启动新版并等待 HTTP 健康检查。原容器、镜像及持久化数据不会被自动删除。

若新版本启动或健康检查失败，会保留失败现场和停止的原容器，不会自动切回旧程序，因为数据库可能已经迁移。先核对日志和迁移结果，必要时配合数据库及文件备份恢复。只恢复旧容器或旧镜像不等于恢复数据库。健康检查通过仅确认 HTTP 服务就绪；MySQL 中的原用户和配置、Docker 连接、代理及仓库登录仍需实际核对。

`start` 与 `restart` 沿用已有容器的原配置，不应用新镜像或新环境变量；每次更改配置后使用 `update`。切换到新镜像前的检查失败不会停止原容器。被保留的历史容器会继续占用镜像引用，确认升级和备份后按其完整名称手动清理。

## 管理与验证

```bash
./deploy.sh stop
./deploy.sh restart
./deploy.sh logs
./deploy.sh shell
./deploy.sh remove
./deploy.sh remove-image
```

`remove` 仅删除配置指向的容器，不删除命名卷、持久化目录或 MySQL 数据。`remove-image` 只删除配置的镜像；仍被容器引用时由 Docker 拒绝，不自动停止或删除容器。

脚本回归测试使用临时 Unix socket 和模拟 Docker，不连接真实 Docker 或数据库：

```bash
bash -n deploy.sh
PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover -s scripts/tests -p 'test_deploy.py' -v
```
